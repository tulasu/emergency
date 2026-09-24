package dialog

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository is the single-writer bank store (AD-1/AD-2).
// Hot call path never hits DB; dialog workers pull immutable snapshots.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListSlotIDs returns known slot ids for validator (facts[].slot must be in set).
func (r *Repository) ListSlotIDs(ctx context.Context) (map[string]bool, error) {
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

// ReplaceBank atomically swaps the immutable bank snapshot.
// In-flight calls finish on old digest (AD-8).
// Empty slots would wipe the canon — rejected (spec J).
// Digest is recomputed here, never trusted from the caller (spec J).
func (r *Repository) ReplaceBank(ctx context.Context, version string, slots map[string]string, questions map[string][]string) error {
	if version == "" {
		return fmt.Errorf("bank version required")
	}
	if len(slots) == 0 {
		return fmt.Errorf("empty slots would wipe bank canon")
	}
	digest := BankDigest(slots, questions)
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

// BankSnapshot reads the canon for digest recompute + worker fan-out.
func (r *Repository) BankSnapshot(ctx context.Context) (slots map[string]string, questions map[string][]string, err error) {
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

// BankDigest recomputes the canon digest byte-identical to dialog digest_of:
// Python json.dumps(obj, ensure_ascii=False, sort_keys=True) with default
// separators (', ', ': ') — Go's encoder emits no spaces, so the canon is
// built by hand. Fresh DB: digest of empty canon, never caller-supplied.
func BankDigest(slots map[string]string, questions map[string][]string) string {
	canonQ := map[string][]string{}
	for k, v := range questions {
		cp := append([]string{}, v...)
		sort.Strings(cp)
		canonQ[k] = cp
	}
	var buf bytes.Buffer
	buf.WriteString(`{"questions": {`)
	writeStrMapList(&buf, canonQ)
	buf.WriteString(`}, "slots": {`)
	writeStrMap(&buf, slots)
	buf.WriteString(`}}`)
	sum := sha256.Sum256(buf.Bytes())
	return fmt.Sprintf("%x", sum)[:16]
}

func writeStrMap(buf *bytes.Buffer, m map[string]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			buf.WriteString(", ")
		}
		writeJSONString(buf, k)
		buf.WriteString(": ")
		writeJSONString(buf, m[k])
	}
}

func writeStrMapList(buf *bytes.Buffer, m map[string][]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			buf.WriteString(", ")
		}
		writeJSONString(buf, k)
		buf.WriteString(": [")
		for j, v := range m[k] {
			if j > 0 {
				buf.WriteString(", ")
			}
			writeJSONString(buf, v)
		}
		buf.WriteString("]")
	}
}

// writeJSONString escapes like Python json.dumps(ensure_ascii=False):
// quote, backslash and C0 controls; raw UTF-8 otherwise.
func writeJSONString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
}
func (r *Repository) BankVersion(ctx context.Context) (version, digest string, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT version, digest FROM bank_state ORDER BY updated_at DESC LIMIT 1`).
		Scan(&version, &digest)
	return version, digest, err
}
