// Package store is the audio PostgreSQL repository (direct pgx, no sqlc).
// Invariant: application code NEVER writes blobs.refcount — only the
// refs_adjust_blob_refcount trigger does (insert +1/clear flag,
// delete −1/flag at 0). Queue PK (hash) is the cross-worker singleflight.
package store

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"audio/internal/enumerate"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound maps to 404, never an accidental 500.
var ErrNotFound = errors.New("not found")

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

// QueueItem is one claimed synth job.
type QueueItem struct {
	Hash     [32]byte
	TicketID uuid.UUID
	FragID   string
	Text     string
	Attempts int
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

// New wraps pool (created by internal/postgres).
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func hexOf(sum [32]byte) string { return hex.EncodeToString(sum[:]) }

// Ensure upserts the manifest + refs for existing blobs + queue for missing.
// Idempotent: stored digest == digest returns current status + missing with
// no writes. matched reports the digest hit (caller maps ready→200).
func (s *Store) Ensure(ctx context.Context, ticketID uuid.UUID, digest, voice string, frags []enumerate.Fragment, hashes [][32]byte) (status string, missing []string, matched bool, err error) {
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

	var storedDigest, storedStatus string
	err = tx.QueryRow(ctx, `SELECT scenario_digest, status FROM manifests WHERE ticket_id = $1`, ticketID).Scan(&storedDigest, &storedStatus)
	if err == nil && storedDigest == digest {
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

	if len(newHashes) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM refs WHERE ticket_id = $1 AND hash <> ALL($2)`, ticketID, newHashes); err != nil {
			return "", nil, false, fmt.Errorf("drop stale refs: %w", err)
		}
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
			INSERT INTO synth_queue (hash, ticket_id, frag_id, text)
			SELECT h, $2, f, t FROM unnest($1::bytea[], $3::text[], $4::text[]) AS u(h, f, t)
			WHERE NOT EXISTS (SELECT 1 FROM blobs WHERE hash = h)
			ON CONFLICT DO NOTHING`, newHashes, ticketID, fragIDs, texts); err != nil {
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
	if err := tx.Commit(ctx); err != nil {
		return "", nil, false, err
	}
	return "pending", m, false, nil
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

// Claim takes up to batch queue rows (attempts<3), bumping attempts upfront
// as the lease: a crashed worker's item retries, at most 3 total.
func (s *Store) Claim(ctx context.Context, batch int) ([]QueueItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT hash, ticket_id, frag_id, text, attempts FROM synth_queue
		WHERE attempts < 3 ORDER BY created_at LIMIT $1 FOR UPDATE SKIP LOCKED`, batch)
	if err != nil {
		return nil, err
	}
	items := []QueueItem{}
	var raws [][]byte
	for rows.Next() {
		var it QueueItem
		var h []byte
		if err := rows.Scan(&h, &it.TicketID, &it.FragID, &it.Text, &it.Attempts); err != nil {
			rows.Close()
			return nil, err
		}
		copy(it.Hash[:], h)
		items = append(items, it)
		raws = append(raws, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		_ = tx.Commit(ctx)
		return nil, nil
	}
	if _, err := tx.Exec(ctx, `UPDATE synth_queue SET attempts = attempts + 1 WHERE hash = ANY($1)`, raws); err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Attempts++
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return items, nil
}

// FinishItem stores the blob, backfills refs for every non-ready manifest
// pointing at the hash (owner + dedup sharers), drops the queue row and
// recomputes affected manifest statuses. Idempotent via ON CONFLICT.
func (s *Store) FinishItem(ctx context.Context, item QueueItem, voice string, rate, bytes int, durS float64) error {
	raw := make([]byte, 32)
	copy(raw, item.Hash[:])
	hexDigest := hexOf(item.Hash)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO blobs (hash, voice, rate, bytes, dur_s) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING`, raw, voice, rate, bytes, durS); err != nil {
		return fmt.Errorf("insert blob: %w", err)
	}
	rows, err := tx.Query(ctx, `
		INSERT INTO refs (ticket_id, hash)
		SELECT m.ticket_id, $1 FROM manifests m
		WHERE m.status IN ('pending', 'partial', 'error')
		  AND EXISTS (SELECT 1 FROM jsonb_each_text(m.fragments) e WHERE e.value = $2)
		ON CONFLICT DO NOTHING RETURNING ticket_id`, raw, hexDigest)
	if err != nil {
		return fmt.Errorf("backfill refs: %w", err)
	}
	affected := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		affected = append(affected, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM synth_queue WHERE hash = $1`, raw); err != nil {
		return fmt.Errorf("dequeue: %w", err)
	}
	for _, id := range affected {
		if err := s.recomputeLocked(ctx, tx, id, ""); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// FailItem records a synth/s3/wav failure. attempts was already bumped by
// Claim; at >=3 the item is poisoned: manifests referencing it go error with
// redacted text. If the blob actually made it (S3 ok, DB failed), finish instead.
func (s *Store) FailItem(ctx context.Context, item QueueItem, redactedErr string) error {
	raw := make([]byte, 32)
	copy(raw, item.Hash[:])
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM blobs WHERE hash = $1`, raw).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		// Blob stored despite the error path — finish, don't poison.
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `DELETE FROM synth_queue WHERE hash = $1`, raw); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT ticket_id FROM manifests WHERE status IN ('pending','partial')`)
		if err != nil {
			return err
		}
		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		for _, id := range ids {
			if err := s.recomputeLocked(ctx, tx, id, ""); err != nil {
				return err
			}
		}
		return tx.Commit(ctx)
	}
	if item.Attempts < 3 {
		return nil // stays queued for retry
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	hexDigest := hexOf(item.Hash)
	rows, err := tx.Query(ctx, `SELECT ticket_id FROM manifests WHERE status IN ('pending', 'partial')`)
	if err != nil {
		return err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		var found int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM manifests m, jsonb_each_text(m.fragments) e
			WHERE m.ticket_id = $1 AND e.value = $2`, id, hexDigest).Scan(&found); err != nil {
			return err
		}
		if found == 0 {
			continue
		}
		if err := s.recomputeLocked(ctx, tx, id, redactedErr); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// recomputeLocked sets manifest status from stored blobs: poisoned queue row
// (attempts>=3, any owner) → error, else missing blobs → pending, else ready.
func (s *Store) recomputeLocked(ctx context.Context, tx pgx.Tx, ticketID uuid.UUID, redactedErr string) error {
	var missing, poisoned int
	if err := tx.QueryRow(ctx, `
		WITH frags AS (
			SELECT e.value AS h FROM manifests m, jsonb_each_text(m.fragments) e WHERE m.ticket_id = $1
		)
		SELECT
			(SELECT count(*) FROM frags f WHERE NOT EXISTS (SELECT 1 FROM blobs b WHERE b.hash = decode(f.h, 'hex'))),
			(SELECT count(*) FROM frags f WHERE EXISTS (SELECT 1 FROM synth_queue q WHERE encode(q.hash, 'hex') = f.h AND q.attempts >= 3))`,
		ticketID).Scan(&missing, &poisoned); err != nil {
		return err
	}
	status := "ready"
	errText := ""
	switch {
	case poisoned > 0:
		status, errText = "error", redactedErr
	case missing > 0:
		status = "pending"
	}
	_, err := tx.Exec(ctx, `UPDATE manifests SET status = $2, error = $3, updated_at = now() WHERE ticket_id = $1`,
		ticketID, status, errText)
	return err
}

// SweepOrphans selects up to limit blobs past TTL for S3-first deletion.
func (s *Store) SweepOrphans(ctx context.Context, ttl time.Duration, limit int) ([]Orphan, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT hash, voice, rate, bytes FROM blobs
		WHERE refcount = 0 AND unreferenced_since < now() - make_interval(secs => $1)
		LIMIT $2 FOR UPDATE SKIP LOCKED`, ttl.Seconds(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Orphan{}
	for rows.Next() {
		var o Orphan
		var h []byte
		if err := rows.Scan(&h, &o.Voice, &o.Rate, &o.Bytes); err != nil {
			return nil, err
		}
		copy(o.Hash[:], h)
		out = append(out, o)
	}
	return out, rows.Err()
}

// DeleteBlobs removes blob rows after their S3 keys are gone.
func (s *Store) DeleteBlobs(ctx context.Context, hashes [][32]byte) (int64, error) {
	if len(hashes) == 0 {
		return 0, nil
	}
	raws := make([][]byte, len(hashes))
	for i, h := range hashes {
		b := make([]byte, 32)
		copy(b, h[:])
		raws[i] = b
	}
	res, err := s.pool.Exec(ctx, `DELETE FROM blobs WHERE hash = ANY($1)`, raws)
	return res.RowsAffected(), err
}

// Counts serves /health: queue depth, blobs, refs.
func (s *Store) Counts(ctx context.Context) (queue, blobs, refs int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM synth_queue),
		       (SELECT count(*) FROM blobs),
		       (SELECT count(*) FROM refs)`).Scan(&queue, &blobs, &refs)
	return queue, blobs, refs, err
}
