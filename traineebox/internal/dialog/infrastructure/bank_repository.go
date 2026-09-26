package infrastructure

import (
	"context"
	"fmt"

	"traineebox/internal/dialog/domain/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BankRepository struct {
	pool *pgxpool.Pool
}

func NewBankRepository(pool *pgxpool.Pool) *BankRepository {
	return &BankRepository{pool: pool}
}

func (r *BankRepository) ListSlotIDs(ctx context.Context) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM dialog_slots`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (r *BankRepository) ReplaceBank(ctx context.Context, version string, slots map[string]string, questions map[string][]string) error {
	if version == "" {
		return fmt.Errorf("bank version required")
	}
	if len(slots) == 0 {
		return fmt.Errorf("empty slots would wipe bank canon")
	}
	digest := models.BankDigest(slots, questions)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM slot_questions`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM dialog_slots`); err != nil {
		return err
	}
	for id, label := range slots {
		family := id
		for i := 0; i < len(id); i++ {
			if id[i] == '.' {
				family = id[:i]
				break
			}
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO dialog_slots (id, label, family) VALUES ($1, $2, $3)`,
			id, label, family); err != nil {
			return err
		}
	}
	for slot, qs := range questions {
		for _, q := range qs {
			if _, err := tx.Exec(ctx,
				`INSERT INTO slot_questions (slot_id, question) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				slot, q); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO bank_state (version, digest) VALUES ($1, $2)
		 ON CONFLICT (version) DO UPDATE SET digest = EXCLUDED.digest, updated_at = now()`,
		version, digest); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *BankRepository) BankSnapshot(ctx context.Context) (slots map[string]string, questions map[string][]string, err error) {
	slots = map[string]string{}
	srows, err := r.pool.Query(ctx, `SELECT id, label FROM dialog_slots`)
	if err != nil {
		return nil, nil, err
	}
	defer srows.Close()
	for srows.Next() {
		var id, label string
		if err := srows.Scan(&id, &label); err != nil {
			return nil, nil, err
		}
		slots[id] = label
	}
	if err := srows.Err(); err != nil {
		return nil, nil, err
	}
	questions = map[string][]string{}
	qrows, err := r.pool.Query(ctx, `SELECT slot_id, question FROM slot_questions ORDER BY slot_id, question`)
	if err != nil {
		return nil, nil, err
	}
	defer qrows.Close()
	for qrows.Next() {
		var slot, q string
		if err := qrows.Scan(&slot, &q); err != nil {
			return nil, nil, err
		}
		questions[slot] = append(questions[slot], q)
	}
	return slots, questions, qrows.Err()
}

func (r *BankRepository) BankVersion(ctx context.Context) (version, digest string, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT version, digest FROM bank_state ORDER BY updated_at DESC LIMIT 1`).
		Scan(&version, &digest)
	return version, digest, err
}
