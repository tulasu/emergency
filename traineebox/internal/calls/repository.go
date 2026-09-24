package calls

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository persists calls; dialog writes turns only via internal endpoints on close.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, c Call) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO attempt_calls (id, attempt_id, ticket_id, user_id, scenario_id, bank_digest, channel_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		c.ID, c.AttemptID, c.TicketID, c.UserID, c.ScenarioID, c.BankDigest, c.ChannelID, c.Status)
	return err
}

const callCols = `id, attempt_id, ticket_id, user_id, scenario_id, bank_digest, channel_id, status, created_at, updated_at`

// ActiveForAttempt returns a call that blocks recall (409), if any.
func (r *Repository) ActiveForAttempt(ctx context.Context, attemptID uuid.UUID) (Call, bool, error) {
	var c Call
	err := r.pool.QueryRow(ctx,
		`SELECT `+callCols+` FROM attempt_calls WHERE attempt_id = $1
		 AND status IN ('originating','ringing','answered','completed')
		 ORDER BY created_at DESC LIMIT 1`,
		attemptID).Scan(&c.ID, &c.AttemptID, &c.TicketID, &c.UserID, &c.ScenarioID, &c.BankDigest, &c.ChannelID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Call{}, false, nil
		}
		return Call{}, false, err
	}
	return c, true, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (Call, error) {
	var c Call
	err := r.pool.QueryRow(ctx,
		`SELECT `+callCols+` FROM attempt_calls WHERE id = $1`, id).
		Scan(&c.ID, &c.AttemptID, &c.TicketID, &c.UserID, &c.ScenarioID, &c.BankDigest, &c.ChannelID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Call{}, ErrNotFound // unknown callId → 404, never raw DB 500 (spec P)
		}
		return Call{}, err
	}
	return c, nil
}

func (r *Repository) ListByAttempt(ctx context.Context, attemptID uuid.UUID) ([]Call, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+callCols+` FROM attempt_calls WHERE attempt_id = $1 ORDER BY created_at`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Call
	for rows.Next() {
		var c Call
		if err := rows.Scan(&c.ID, &c.AttemptID, &c.TicketID, &c.UserID, &c.ScenarioID, &c.BankDigest, &c.ChannelID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetChannelID persists the ARI channel id from originate for real hangup (spec F).
func (r *Repository) SetChannelID(ctx context.Context, id uuid.UUID, channelID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE attempt_calls SET channel_id = $2, updated_at = now() WHERE id = $1`, id, channelID)
	return err
}

// ListExpired returns live calls whose attempt deadline passed (spec Q).
func (r *Repository) ListExpired(ctx context.Context, now time.Time) ([]Call, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.`+strings.ReplaceAll(callCols, ", ", ", c.")+`
		 FROM attempt_calls c JOIN ticket_attempts a ON a.id = c.attempt_id
		 WHERE c.status IN ('originating','ringing','answered')
		 AND a.deadline_at IS NOT NULL AND a.deadline_at < $1`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Call
	for rows.Next() {
		var c Call
		if err := rows.Scan(&c.ID, &c.AttemptID, &c.TicketID, &c.UserID, &c.ScenarioID, &c.BankDigest, &c.ChannelID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE attempt_calls SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	return err
}

// SaveTurns stores partial turns on close (deadline race keeps partials).
// One transaction: a close never leaves half its turns behind.
func (r *Repository) SaveTurns(ctx context.Context, callID uuid.UUID, turns []Turn) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, t := range turns {
		if _, err := tx.Exec(ctx,
			`INSERT INTO call_turns (call_id, n, utterance, reply, style)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (call_id, n) DO UPDATE SET utterance = EXCLUDED.utterance, reply = EXCLUDED.reply, style = EXCLUDED.style`,
			callID, t.N, t.Utterance, t.Reply, t.Style); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListTurns(ctx context.Context, callID uuid.UUID) ([]Turn, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT n, utterance, reply, style FROM call_turns WHERE call_id = $1 ORDER BY n`, callID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Turn
	for rows.Next() {
		var t Turn
		if err := rows.Scan(&t.N, &t.Utterance, &t.Reply, &t.Style); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
