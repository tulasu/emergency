// Package repositories owns audio PostgreSQL persistence (direct pgx).
// Only the refs_adjust_blob_refcount trigger writes blobs.refcount.
package repositories

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"audio/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound maps to 404, never an accidental 500.
var ErrNotFound = errors.New("not found")

// ErrClaimLost reports a stale worker lease; it must not be failed again.
var ErrClaimLost = errors.New("synthesis claim lost")

// Manifest is the read-only {frag_id: hex(hash)} cache for dialog open.
type Manifest struct {
	TicketID  uuid.UUID
	Digest    string
	Voice     string
	Status    string
	Fragments map[string]string
	Error     string
	UpdatedAt time.Time
}

// QueueItem is one exclusively claimed synth job.
type QueueItem struct {
	Hash       [32]byte
	TicketID   uuid.UUID
	FragID     string
	Voice      string
	Text       string
	Attempts   int
	ClaimToken uuid.UUID
}

// Orphan is a sweep candidate with the fields needed for the S3 key.
type Orphan struct {
	Hash  [32]byte
	Voice string
	Rate  int
	Bytes int
}

// Store wraps one pgx pool.
type Store struct {
	pool *pgxpool.Pool
}

// New wraps the audio PostgreSQL pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func hexOf(sum [32]byte) string { return hex.EncodeToString(sum[:]) }

// Ensure upserts the manifest + refs for existing blobs + queue for missing.
// Idempotent: stored digest and voice match returns current status + missing
// with no writes. matched reports the cache hit (caller maps ready→200).
func (s *Store) Ensure(ctx context.Context, ticketID uuid.UUID, digest, voice string, frags []domain.Fragment, hashes [][32]byte) (status string, missing []string, matched bool, err error) {
	fragMap := make(map[string]string, len(frags))
	newHashes := make([][]byte, len(hashes))
	for i, f := range frags {
		fragMap[f.ID] = hexOf(hashes[i])
		h := make([]byte, 32)
		copy(h, hashes[i][:])
		newHashes[i] = h
	}
	rawFrag, err := json.Marshal(fragMap)
	if err != nil {
		return "", nil, false, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", nil, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var storedDigest, storedStatus, storedVoice string
	err = tx.QueryRow(ctx, `SELECT scenario_digest, status, voice FROM manifests WHERE ticket_id = $1`, ticketID).Scan(&storedDigest, &storedStatus, &storedVoice)
	if err == nil && storedDigest == digest && storedVoice == voice {
		m, err := s.missingLocked(ctx, tx, ticketID)
		if err != nil {
			return "", nil, false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", nil, false, err
		}
		return storedStatus, m, true, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", nil, false, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM refs WHERE ticket_id = $1 AND hash <> ALL($2)`, ticketID, newHashes); err != nil {
		return "", nil, false, fmt.Errorf("drop stale refs: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO manifests (ticket_id, scenario_digest, voice, status, fragments, error, updated_at)
		VALUES ($1, $2, $3, 'pending', $4, '', now())
		ON CONFLICT (ticket_id) DO UPDATE SET
			scenario_digest = EXCLUDED.scenario_digest,
			voice = EXCLUDED.voice,
			status = 'pending', fragments = EXCLUDED.fragments,
			error = '', updated_at = now()`,
		ticketID, digest, voice, rawFrag); err != nil {
		return "", nil, false, fmt.Errorf("upsert manifest: %w", err)
	}
	if len(newHashes) > 0 {
		// Lock reusable blobs through the ref insert, so a concurrent sweep
		// cannot delete S3 after this transaction decides a blob is available.
		locked, err := tx.Query(ctx, `SELECT hash FROM blobs WHERE hash = ANY($1) FOR KEY SHARE`, newHashes)
		if err != nil {
			return "", nil, false, fmt.Errorf("lock reusable blobs: %w", err)
		}
		for locked.Next() {
		}
		if err := locked.Err(); err != nil {
			locked.Close()
			return "", nil, false, fmt.Errorf("lock reusable blobs: %w", err)
		}
		locked.Close()
		// Refs only for blobs already stored (FK RESTRICT); the rest queue.
		if _, err := tx.Exec(ctx, `
			INSERT INTO refs (ticket_id, hash)
			SELECT $1, h FROM unnest($2::bytea[]) AS h
			WHERE EXISTS (SELECT 1 FROM blobs WHERE hash = h)
			ON CONFLICT DO NOTHING`, ticketID, newHashes); err != nil {
			return "", nil, false, fmt.Errorf("insert refs: %w", err)
		}
		fragIDs := make([]string, len(frags))
		texts := make([]string, len(frags))
		for i, f := range frags {
			fragIDs[i] = f.ID
			texts[i] = f.Text
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO synth_queue (hash, ticket_id, frag_id, voice, text)
			SELECT h, $2, f, $3, t FROM unnest($1::bytea[], $4::text[], $5::text[]) AS u(h, f, t)
			WHERE NOT EXISTS (SELECT 1 FROM blobs WHERE hash = h)
			ON CONFLICT DO NOTHING`, newHashes, ticketID, voice, fragIDs, texts); err != nil {
			return "", nil, false, fmt.Errorf("enqueue: %w", err)
		}
	}
	m, err := s.missingLocked(ctx, tx, ticketID)
	if err != nil {
		return "", nil, false, err
	}
	if len(m) == 0 {
		if _, err := tx.Exec(ctx, `UPDATE manifests SET status = 'ready', updated_at = now() WHERE ticket_id = $1`, ticketID); err != nil {
			return "", nil, false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", nil, false, err
		}
		return "ready", nil, false, nil
	}
	if err := s.recomputeLocked(ctx, tx, ticketID); err != nil {
		return "", nil, false, err
	}
	if err := tx.QueryRow(ctx, `SELECT status FROM manifests WHERE ticket_id = $1`, ticketID).Scan(&status); err != nil {
		return "", nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", nil, false, err
	}
	return status, m, false, nil
}

// missingLocked lists fragment hexes with no blob row (caller holds tx).
func (s *Store) missingLocked(ctx context.Context, tx pgx.Tx, ticketID uuid.UUID) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT e.value FROM manifests m, jsonb_each_text(m.fragments) e
		WHERE m.ticket_id = $1 AND NOT EXISTS (SELECT 1 FROM blobs WHERE hash = decode(e.value, 'hex'))`,
		ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func scanManifest(row pgx.Row) (Manifest, error) {
	var m Manifest
	var rawFrag []byte
	err := row.Scan(&m.TicketID, &m.Digest, &m.Voice, &m.Status, &rawFrag, &m.Error, &m.UpdatedAt)
	if err != nil {
		return Manifest{}, err
	}
	m.Fragments = map[string]string{}
	if err := json.Unmarshal(rawFrag, &m.Fragments); err != nil {
		return Manifest{}, fmt.Errorf("decode fragments: %w", err)
	}
	return m, nil
}

// GetManifest returns the manifest or ErrNotFound.
func (s *Store) GetManifest(ctx context.Context, ticketID uuid.UUID) (Manifest, error) {
	m, err := scanManifest(s.pool.QueryRow(ctx,
		`SELECT ticket_id, scenario_digest, voice, status, fragments, error, updated_at
		 FROM manifests WHERE ticket_id = $1`, ticketID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Manifest{}, ErrNotFound
	}
	return m, err
}

// BlobVoice returns voice/rate for the S3 key, or ErrNotFound.
func (s *Store) BlobVoice(ctx context.Context, hash [32]byte) (voice string, rate int, err error) {
	h := make([]byte, 32)
	copy(h, hash[:])
	err = s.pool.QueryRow(ctx, `SELECT voice, rate FROM blobs WHERE hash = $1`, h).Scan(&voice, &rate)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, ErrNotFound
	}
	return voice, rate, err
}

// DeleteTicket drops refs + manifest; blobs turn orphan under the sweep.
// Unknown tickets are ErrNotFound (404), not silent success.
func (s *Store) DeleteTicket(ctx context.Context, ticketID uuid.UUID) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	res, err := tx.Exec(ctx, `DELETE FROM refs WHERE ticket_id = $1`, ticketID)
	if err != nil {
		return 0, err
	}
	mres, err := tx.Exec(ctx, `DELETE FROM manifests WHERE ticket_id = $1`, ticketID)
	if err != nil {
		return 0, err
	}
	if mres.RowsAffected() == 0 {
		return 0, ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

const expiredLeaseError = "synthesis lease expired"

// recoverExpiredLocked poisons terminal leases whose workers died. It runs
// before each claim so no manifest can remain pending after its final lease.
func (s *Store) recoverExpiredLocked(ctx context.Context, tx pgx.Tx, limit int) error {
	rows, err := tx.Query(ctx, `
		SELECT hash FROM synth_queue
		WHERE attempts >= 3 AND claim_until < now() AND error = ''
		ORDER BY claim_until LIMIT $1 FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return err
	}
	expired := [][32]byte{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return err
		}
		var sum [32]byte
		copy(sum[:], raw)
		expired = append(expired, sum)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, sum := range expired {
		if _, err := tx.Exec(ctx, `
			UPDATE synth_queue
			SET claim_token = NULL, claim_until = NULL, error = $2
			WHERE hash = $1 AND attempts >= 3 AND claim_until < now() AND error = ''`, sum[:], expiredLeaseError); err != nil {
			return err
		}
		ids, err := s.affectedManifestIDs(ctx, tx, sum)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := s.recomputeLocked(ctx, tx, id); err != nil {
				return err
			}
		}
	}
	return nil
}

// Claim takes up to batch unclaimed or expired queue rows. A claim token
// fences stale workers; the ten-minute lease exceeds one bounded worker pass.
func (s *Store) Claim(ctx context.Context, batch int) ([]QueueItem, error) {
	if batch <= 0 {
		return nil, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.recoverExpiredLocked(ctx, tx, batch); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT hash, ticket_id, frag_id, voice, text, attempts FROM synth_queue
		WHERE attempts < 3 AND (claim_until IS NULL OR claim_until < now())
		ORDER BY created_at LIMIT $1 FOR UPDATE SKIP LOCKED`, batch)
	if err != nil {
		return nil, err
	}
	items := []QueueItem{}
	for rows.Next() {
		var it QueueItem
		var raw []byte
		if err := rows.Scan(&raw, &it.TicketID, &it.FragID, &it.Voice, &it.Text, &it.Attempts); err != nil {
			rows.Close()
			return nil, err
		}
		copy(it.Hash[:], raw)
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		items[i].ClaimToken = uuid.New()
		if _, err := tx.Exec(ctx, `
			UPDATE synth_queue
			SET attempts = attempts + 1, claim_token = $2,
			    claim_until = now() + interval '10 minutes'
			WHERE hash = $1`, items[i].Hash[:], items[i].ClaimToken); err != nil {
			return nil, err
		}
		items[i].Attempts++
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return items, nil
}

// claimLocked locks and verifies a queue item before a terminal operation.
func claimLocked(ctx context.Context, tx pgx.Tx, item QueueItem) (int, error) {
	var attempts int
	err := tx.QueryRow(ctx, `
		SELECT attempts FROM synth_queue
		WHERE hash = $1 AND claim_token = $2 AND claim_until > now() FOR UPDATE`, item.Hash[:], item.ClaimToken).Scan(&attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrClaimLost
	}
	return attempts, err
}

func (s *Store) affectedManifestIDs(ctx context.Context, tx pgx.Tx, sum [32]byte) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx, `
		SELECT ticket_id FROM manifests
		WHERE EXISTS (SELECT 1 FROM jsonb_each_text(fragments) e WHERE e.value = $1)`, hexOf(sum))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) finishLocked(ctx context.Context, tx pgx.Tx, item QueueItem, rate, bytes int, durS float64) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO blobs (hash, voice, rate, bytes, dur_s) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING`, item.Hash[:], item.Voice, rate, bytes, durS); err != nil {
		return fmt.Errorf("insert blob: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO refs (ticket_id, hash)
		SELECT m.ticket_id, $1 FROM manifests m
		WHERE EXISTS (SELECT 1 FROM jsonb_each_text(m.fragments) e WHERE e.value = $2)
		ON CONFLICT DO NOTHING`, item.Hash[:], hexOf(item.Hash)); err != nil {
		return fmt.Errorf("backfill refs: %w", err)
	}
	res, err := tx.Exec(ctx, `DELETE FROM synth_queue WHERE hash = $1 AND claim_token = $2`, item.Hash[:], item.ClaimToken)
	if err != nil {
		return fmt.Errorf("dequeue: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrClaimLost
	}
	ids, err := s.affectedManifestIDs(ctx, tx, item.Hash)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.recomputeLocked(ctx, tx, id); err != nil {
			return err
		}
	}
	return nil
}

// FinishItem stores the claimed blob, backfills every referring manifest, and
// releases the exclusive queue claim by removing it.
func (s *Store) FinishItem(ctx context.Context, item QueueItem, rate, bytes int, durS float64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := claimLocked(ctx, tx, item); err != nil {
		return err
	}
	if err := s.finishLocked(ctx, tx, item, rate, bytes, durS); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// FailItem releases a matching failed lease for retry. The third failure is
// poisoned with its redacted text; recomputation reads that persisted text.
func (s *Store) FailItem(ctx context.Context, item QueueItem, redactedErr string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	attempts, err := claimLocked(ctx, tx, item)
	if err != nil {
		return err
	}
	var blobExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM blobs WHERE hash = $1)`, item.Hash[:]).Scan(&blobExists); err != nil {
		return err
	}
	if blobExists {
		if err := s.finishLocked(ctx, tx, item, 0, 0, 0); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if attempts < 3 {
		_, err := tx.Exec(ctx, `
			UPDATE synth_queue SET claim_token = NULL, claim_until = NULL
			WHERE hash = $1 AND claim_token = $2`, item.Hash[:], item.ClaimToken)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE synth_queue SET claim_token = NULL, claim_until = NULL, error = $3
		WHERE hash = $1 AND claim_token = $2`, item.Hash[:], item.ClaimToken, redactedErr); err != nil {
		return err
	}
	ids, err := s.affectedManifestIDs(ctx, tx, item.Hash)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.recomputeLocked(ctx, tx, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// recomputeLocked sets manifest status from stored blobs. Poison status and
// text are owned by synth_queue so later Ensure calls remain idempotent.
func (s *Store) recomputeLocked(ctx context.Context, tx pgx.Tx, ticketID uuid.UUID) error {
	var missing int
	var poisoned bool
	var poisonErr string
	if err := tx.QueryRow(ctx, `
		WITH frags AS (
			SELECT e.value AS h FROM manifests m, jsonb_each_text(m.fragments) e WHERE m.ticket_id = $1
		)
		SELECT
			(SELECT count(*) FROM frags f WHERE NOT EXISTS (SELECT 1 FROM blobs b WHERE b.hash = decode(f.h, 'hex'))),
			EXISTS (SELECT 1 FROM frags f JOIN synth_queue q ON encode(q.hash, 'hex') = f.h WHERE q.attempts >= 3 AND q.claim_token IS NULL),
			COALESCE((
				SELECT q.error FROM frags f JOIN synth_queue q ON encode(q.hash, 'hex') = f.h
				WHERE q.attempts >= 3 AND q.claim_token IS NULL ORDER BY q.created_at LIMIT 1
			), '')`, ticketID).Scan(&missing, &poisoned, &poisonErr); err != nil {
		return err
	}
	status, errText := "ready", ""
	switch {
	case poisoned:
		status, errText = "error", poisonErr
	case missing > 0:
		status = "pending"
	}
	_, err := tx.Exec(ctx, `UPDATE manifests SET status = $2, error = $3, updated_at = now() WHERE ticket_id = $1`,
		ticketID, status, errText)
	return err
}

// SweepOrphans calls remove and deletes each successful candidate while its
// row lock is held. This prevents a new ref from racing S3-first deletion.
func (s *Store) SweepOrphans(ctx context.Context, ttl time.Duration, limit int, remove func(context.Context, Orphan) error) (deleted, bytes int64, err error) {
	if limit <= 0 {
		return 0, 0, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT hash, voice, rate, bytes FROM blobs
		WHERE refcount = 0 AND unreferenced_since < now() - make_interval(secs => $1)
		ORDER BY unreferenced_since LIMIT $2 FOR UPDATE SKIP LOCKED`, ttl.Seconds(), limit)
	if err != nil {
		return 0, 0, err
	}
	orphans := []Orphan{}
	for rows.Next() {
		var o Orphan
		var raw []byte
		if err := rows.Scan(&raw, &o.Voice, &o.Rate, &o.Bytes); err != nil {
			rows.Close()
			return 0, 0, err
		}
		copy(o.Hash[:], raw)
		orphans = append(orphans, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	for _, o := range orphans {
		if err := remove(ctx, o); err != nil {
			continue
		}
		res, err := tx.Exec(ctx, `DELETE FROM blobs WHERE hash = $1 AND refcount = 0`, o.Hash[:])
		if err != nil {
			return 0, 0, err
		}
		if res.RowsAffected() == 1 {
			deleted++
			bytes += int64(o.Bytes)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return deleted, bytes, nil
}

// Counts serves /health: queue depth, blobs, refs.
func (s *Store) Counts(ctx context.Context) (queue, blobs, refs int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM synth_queue),
		       (SELECT count(*) FROM blobs),
		       (SELECT count(*) FROM refs)`).Scan(&queue, &blobs, &refs)
	return queue, blobs, refs, err
}
