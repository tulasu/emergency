package store_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"audio/internal/enumerate"
	"audio/internal/hash"
	"audio/internal/store"
	"audio/migrations"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func openTestDB(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("AUDIO_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AUDIO_TEST_DATABASE_URL unset (live-postgres test)")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	st := store.New(pool)
	_, _ = pool.Exec(ctx, `TRUNCATE refs, synth_queue, manifests, blobs`)
	return st
}

func TestEnsureFinishDeleteRefcount(t *testing.T) {
	st := openTestDB(t)
	ctx := context.Background()
	ticket := uuid.New()
	frags := []enumerate.Fragment{
		{ID: "a/t/opening.wav", Text: "Алло."},
		{ID: "a/t/x/plain.wav", Text: "Да."},
	}
	sums := make([][32]byte, len(frags))
	for i, f := range frags {
		sums[i] = hash.Texthash(hash.Normalize(f.Text), "kseniya", 8000, hash.Model)
	}
	status, missing, matched, err := st.Ensure(ctx, ticket, "d1", "kseniya", frags, sums)
	if err != nil {
		t.Fatal(err)
	}
	if matched || status != "pending" || len(missing) != 2 {
		t.Fatalf("ensure = %s %v %v", status, missing, matched)
	}
	// Re-ensure is idempotent: same missing, no duplicate queue rows.
	if _, missing2, _, err := st.Ensure(ctx, ticket, "d1", "kseniya", frags, sums); err != nil || len(missing2) != 2 {
		t.Fatalf("re-ensure = %v %v", missing2, err)
	}
	// Claim + finish both: blobs stored, refs backfilled, manifest ready.
	items, err := st.Claim(ctx, 10)
	if err != nil || len(items) != 2 {
		t.Fatalf("claim = %d %v", len(items), err)
	}
	for _, it := range items {
		if err := st.FinishItem(ctx, it, "kseniya", 8000, 48044, 3.0); err != nil {
			t.Fatal(err)
		}
	}
	m, err := st.GetManifest(ctx, ticket)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "ready" || len(m.Fragments) != 2 {
		t.Fatalf("manifest = %+v", m)
	}
	if _, _, matched, err := st.Ensure(ctx, ticket, "d1", "kseniya", frags, sums); err != nil || !matched {
		t.Fatalf("digest match = %v %v", matched, err)
	}
	// Refcount never below zero: delete drops refs, blobs stay orphan.
	n, err := st.DeleteTicket(ctx, ticket)
	if err != nil || n != 2 {
		t.Fatalf("delete = %d %v", n, err)
	}
	q, b, r, err := st.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if q != 0 || b != 2 || r != 0 {
		t.Fatalf("counts = %d %d %d", q, b, r)
	}
}
