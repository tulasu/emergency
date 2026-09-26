package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"traineebox/internal/calls/domain/errs"
	"traineebox/internal/calls/domain/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CallRepository struct {
	pool *pgxpool.Pool
}

func NewCallRepository(pool *pgxpool.Pool) *CallRepository {
	return &CallRepository{pool: pool}
}

func (r *CallRepository) Create(ctx context.Context, c models.Call) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO attempt_calls (id, attempt_id, ticket_id, user_id, scenario_id, bank_digest, channel_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		c.ID, c.AttemptID, c.TicketID, c.UserID, c.ScenarioID, c.BankDigest, c.ChannelID, c.Status)
	return err
}

const callCols = `id, attempt_id, ticket_id, user_id, scenario_id, bank_digest, channel_id, status, created_at, updated_at`

func scanCall(row interface{ Scan(dest ...any) error }) (models.Call, error) {
	var c models.Call
	err := row.Scan(&c.ID, &c.AttemptID, &c.TicketID, &c.UserID, &c.ScenarioID, &c.BankDigest, &c.ChannelID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *CallRepository) ActiveForAttempt(ctx context.Context, attemptID uuid.UUID) (models.Call, bool, error) {
	c, err := scanCall(r.pool.QueryRow(ctx,
		`SELECT `+callCols+` FROM attempt_calls WHERE attempt_id = $1
		 AND status IN ('originating','ringing','answered','completed')
		 ORDER BY created_at DESC LIMIT 1`,
		attemptID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Call{}, false, nil
		}
		return models.Call{}, false, err
	}
	return c, true, nil
}

func (r *CallRepository) FindByID(ctx context.Context, id uuid.UUID) (models.Call, error) {
	c, err := scanCall(r.pool.QueryRow(ctx, `SELECT `+callCols+` FROM attempt_calls WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Call{}, errs.ErrNotFound
		}
		return models.Call{}, err
	}
	return c, nil
}

func (r *CallRepository) ListByAttempt(ctx context.Context, attemptID uuid.UUID) ([]models.Call, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+callCols+` FROM attempt_calls WHERE attempt_id = $1 ORDER BY created_at`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Call
	for rows.Next() {
		c, err := scanCall(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CallRepository) SetChannelID(ctx context.Context, id uuid.UUID, channelID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE attempt_calls SET channel_id = $2, updated_at = now() WHERE id = $1`, id, channelID)
	return err
}

func (r *CallRepository) ListExpired(ctx context.Context, now time.Time) ([]models.Call, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.`+strings.ReplaceAll(callCols, ", ", ", c.")+`
		 FROM attempt_calls c JOIN ticket_attempts a ON a.id = c.attempt_id
		 WHERE c.status IN ('originating','ringing','answered')
		 AND a.deadline_at IS NOT NULL AND a.deadline_at < $1`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Call
	for rows.Next() {
		c, err := scanCall(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CallRepository) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE attempt_calls SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	return err
}

func (r *CallRepository) SaveTurns(ctx context.Context, callID uuid.UUID, turns []models.Turn) error {
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
