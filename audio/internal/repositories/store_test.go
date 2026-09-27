package repositories_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"audio/internal/domain"
	"audio/internal/repositories"
	"audio/migrations"
	"audio/pkg/hash"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func openTestDB(t *testing.T) (*repositories.Store, *pgxpool.Pool) {
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
	_, _ = pool.Exec(ctx, `TRUNCATE refs, synth_queue, manifests, blobs`)
	return repositories.New(pool), pool
}

func testFragment(text, voice string) ([]domain.Fragment, [][32]byte) {
	frags := []domain.Fragment{{ID: "a/t/opening.wav", Text: text}}
	sums := [][32]byte{hash.Texthash(hash.Normalize(text), voice, 8000, hash.Model)}
	return frags, sums
}

func TestEnsureFinishDeleteRefcount(t *testing.T) {
	st, pool := openTestDB(t)
	ctx := context.Background()
	initialHash := hash.Texthash("initial orphan", "kseniya", 8000, hash.Model)
	if _, err := pool.Exec(ctx, `INSERT INTO blobs (hash, voice, rate, bytes, dur_s) VALUES ($1, 'kseniya', 8000, 1, 1)`, initialHash[:]); err != nil {
		t.Fatal(err)
	}
	var initialOrphaned bool
	if err := pool.QueryRow(ctx, `SELECT unreferenced_since IS NOT NULL FROM blobs WHERE hash = $1`, initialHash[:]).Scan(&initialOrphaned); err != nil {
		t.Fatal(err)
	}
	if !initialOrphaned {
		t.Fatal("newly inserted unreferenced blob has no orphan timestamp")
	}
	ticket := uuid.New()
	frags := []domain.Fragment{
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
	if _, missing2, _, err := st.Ensure(ctx, ticket, "d1", "kseniya", frags, sums); err != nil || len(missing2) != 2 {
		t.Fatalf("re-ensure = %v %v", missing2, err)
	}
	items, err := st.Claim(ctx, 10)
	if err != nil || len(items) != 2 {
		t.Fatalf("claim = %d %v", len(items), err)
	}
	for _, it := range items {
		if err := st.FinishItem(ctx, it, 8000, 48044, 3.0); err != nil {
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
	var refcount int
	var referencedAtNull bool
	if err := pool.QueryRow(ctx, `SELECT refcount, unreferenced_since IS NULL FROM blobs WHERE hash = $1`, sums[0][:]).Scan(&refcount, &referencedAtNull); err != nil {
		t.Fatal(err)
	}
	if refcount != 1 || !referencedAtNull {
		t.Fatalf("referenced blob = refcount:%d unreferenced:%t", refcount, referencedAtNull)
	}
	if _, _, matched, err := st.Ensure(ctx, ticket, "d1", "kseniya", frags, sums); err != nil || !matched {
		t.Fatalf("digest match = %v %v", matched, err)
	}
	n, err := st.DeleteTicket(ctx, ticket)
	if err != nil || n != 2 {
		t.Fatalf("delete = %d %v", n, err)
	}
	var orphaned bool
	if err := pool.QueryRow(ctx, `SELECT refcount, unreferenced_since IS NOT NULL FROM blobs WHERE hash = $1`, sums[0][:]).Scan(&refcount, &orphaned); err != nil {
		t.Fatal(err)
	}
	if refcount != 0 || !orphaned {
		t.Fatalf("orphan blob = refcount:%d unreferenced:%t", refcount, orphaned)
	}
}

func TestClaimIsExclusiveAndFailureReleasesLease(t *testing.T) {
	st, _ := openTestDB(t)
	ctx := context.Background()
	frags, sums := testFragment("exclusive", "kseniya")
	if _, _, _, err := st.Ensure(ctx, uuid.New(), "claim", "kseniya", frags, sums); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	claimed := make(chan []repositories.QueueItem, 2)
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			items, err := st.Claim(ctx, 1)
			errs <- err
			claimed <- items
		}()
	}
	close(start)
	var item repositories.QueueItem
	count := 0
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
		items := <-claimed
		count += len(items)
		if len(items) == 1 {
			item = items[0]
		}
	}
	if count != 1 {
		t.Fatalf("concurrent claims = %d, want 1", count)
	}
	if err := st.FailItem(ctx, item, "retry me"); err != nil {
		t.Fatal(err)
	}
	retry, err := st.Claim(ctx, 1)
	if err != nil || len(retry) != 1 || retry[0].Attempts != 2 || retry[0].ClaimToken == item.ClaimToken {
		t.Fatalf("released retry = %#v, %v", retry, err)
	}
}

func TestExpiredFinalLeasePoisonsAndFencesWorker(t *testing.T) {
	st, pool := openTestDB(t)
	ctx := context.Background()
	frags, sums := testFragment("expired lease", "kseniya")
	ticket := uuid.New()
	if _, _, _, err := st.Ensure(ctx, ticket, "expired", "kseniya", frags, sums); err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		items, err := st.Claim(ctx, 1)
		if err != nil || len(items) != 1 {
			t.Fatalf("claim attempt %d = %#v, %v", attempt, items, err)
		}
		if err := st.FailItem(ctx, items[0], "retry"); err != nil {
			t.Fatal(err)
		}
	}
	items, err := st.Claim(ctx, 1)
	if err != nil || len(items) != 1 || items[0].Attempts != 3 {
		t.Fatalf("final lease = %#v, %v", items, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE synth_queue SET claim_until = now() - interval '1 second' WHERE hash = $1`, sums[0][:]); err != nil {
		t.Fatal(err)
	}
	if err := st.FinishItem(ctx, items[0], 8000, 1, 1); !errors.Is(err, repositories.ErrClaimLost) {
		t.Fatalf("late worker finish = %v, want ErrClaimLost", err)
	}
	recovered, err := st.Claim(ctx, 1)
	if err != nil || len(recovered) != 0 {
		t.Fatalf("lease recovery claim = %#v, %v", recovered, err)
	}
	m, err := st.GetManifest(ctx, ticket)
	if err != nil || m.Status != "error" || m.Error != "synthesis lease expired" {
		t.Fatalf("expired lease manifest = %#v, %v", m, err)
	}
}

func TestActiveFinalLeaseStaysPendingUntilFailure(t *testing.T) {
	st, _ := openTestDB(t)
	ctx := context.Background()
	frags, sums := testFragment("active final lease", "kseniya")
	if _, _, _, err := st.Ensure(ctx, uuid.New(), "active-owner", "kseniya", frags, sums); err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		items, err := st.Claim(ctx, 1)
		if err != nil || len(items) != 1 {
			t.Fatalf("claim attempt %d = %#v, %v", attempt, items, err)
		}
		if err := st.FailItem(ctx, items[0], "retry"); err != nil {
			t.Fatal(err)
		}
	}
	items, err := st.Claim(ctx, 1)
	if err != nil || len(items) != 1 || items[0].Attempts != 3 {
		t.Fatalf("active final claim = %#v, %v", items, err)
	}
	waiting := uuid.New()
	status, _, matched, err := st.Ensure(ctx, waiting, "active-waiter", "kseniya", frags, sums)
	if err != nil || matched || status != "pending" {
		t.Fatalf("ensure during active final lease = %s %t %v", status, matched, err)
	}
	if err := st.FailItem(ctx, items[0], "final failure"); err != nil {
		t.Fatal(err)
	}
	m, err := st.GetManifest(ctx, waiting)
	if err != nil || m.Status != "error" || m.Error != "final failure" {
		t.Fatalf("waiter after final failure = %#v, %v", m, err)
	}
}

func TestEnsureSameDigestDifferentVoiceRefreshesQueue(t *testing.T) {
	st, _ := openTestDB(t)
	ctx := context.Background()
	ticket := uuid.New()
	frags, first := testFragment("voice refresh", "kseniya")
	if _, _, _, err := st.Ensure(ctx, ticket, "same-digest", "kseniya", frags, first); err != nil {
		t.Fatal(err)
	}
	_, second := testFragment("voice refresh", "alena")
	status, missing, matched, err := st.Ensure(ctx, ticket, "same-digest", "alena", frags, second)
	if err != nil || matched || status != "pending" || len(missing) != 1 {
		t.Fatalf("voice refresh ensure = %s %v %t %v", status, missing, matched, err)
	}
	m, err := st.GetManifest(ctx, ticket)
	if err != nil || m.Voice != "alena" {
		t.Fatalf("voice refresh manifest = %#v, %v", m, err)
	}
	items, err := st.Claim(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.Hash == second[0] {
			found = item.Voice == "alena"
		}
	}
	if !found {
		t.Fatalf("requested voice hash was not claimed with alena: %#v", items)
	}
}

func TestPoisonPersistsForExistingAndNewManifests(t *testing.T) {
	st, _ := openTestDB(t)
	ctx := context.Background()
	frags, sums := testFragment("poison", "kseniya")
	ticket := uuid.New()
	if _, _, _, err := st.Ensure(ctx, ticket, "poison-1", "kseniya", frags, sums); err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		items, err := st.Claim(ctx, 1)
		if err != nil || len(items) != 1 {
			t.Fatalf("claim attempt %d = %#v, %v", attempt, items, err)
		}
		errText := "retryable synthesis error"
		if attempt == 3 {
			errText = "fatal synthesis error"
		}
		if err := st.FailItem(ctx, items[0], errText); err != nil {
			t.Fatal(err)
		}
	}
	m, err := st.GetManifest(ctx, ticket)
	if err != nil || m.Status != "error" || m.Error != "fatal synthesis error" {
		t.Fatalf("poisoned manifest = %#v, %v", m, err)
	}
	status, _, matched, err := st.Ensure(ctx, ticket, "poison-1", "kseniya", frags, sums)
	if err != nil || !matched || status != "error" {
		t.Fatalf("poison re-ensure = %s %t %v", status, matched, err)
	}
	newTicket := uuid.New()
	status, _, matched, err = st.Ensure(ctx, newTicket, "poison-2", "kseniya", frags, sums)
	if err != nil || matched || status != "error" {
		t.Fatalf("new poisoned manifest = %s %t %v", status, matched, err)
	}
	m, err = st.GetManifest(ctx, newTicket)
	if err != nil || m.Error != "fatal synthesis error" {
		t.Fatalf("new poison text = %#v, %v", m, err)
	}
}

func TestSweepHoldsLockUntilS3CallbackAndDelete(t *testing.T) {
	st, pool := openTestDB(t)
	ctx := context.Background()
	frags, sums := testFragment("sweep race", "kseniya")
	owner := uuid.New()
	if _, _, _, err := st.Ensure(ctx, owner, "owner", "kseniya", frags, sums); err != nil {
		t.Fatal(err)
	}
	items, err := st.Claim(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("claim owner = %#v, %v", items, err)
	}
	if err := st.FinishItem(ctx, items[0], 8000, 48044, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DeleteTicket(ctx, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE blobs SET unreferenced_since = now() - interval '1 hour' WHERE hash = $1`, sums[0][:]); err != nil {
		t.Fatal(err)
	}
	callbackStarted := make(chan struct{})
	releaseDelete := make(chan struct{})
	sweepDone := make(chan error, 1)
	go func() {
		deleted, _, err := st.SweepOrphans(ctx, 0, 1, func(context.Context, repositories.Orphan) error {
			close(callbackStarted)
			<-releaseDelete
			return nil
		})
		if err == nil && deleted != 1 {
			err = fmt.Errorf("deleted %d blobs, want 1", deleted)
		}
		sweepDone <- err
	}()
	select {
	case <-callbackStarted:
	case <-time.After(time.Second):
		t.Fatal("sweep did not acquire orphan lock")
	}
	type ensureResult struct {
		status  string
		missing []string
		err     error
	}
	ensureDone := make(chan ensureResult, 1)
	go func() {
		status, missing, _, err := st.Ensure(ctx, uuid.New(), "racer", "kseniya", frags, sums)
		ensureDone <- ensureResult{status, missing, err}
	}()
	select {
	case result := <-ensureDone:
		t.Fatalf("ensure escaped locked sweep: %#v", result)
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseDelete)
	if err := <-sweepDone; err != nil {
		t.Fatal(err)
	}
	result := <-ensureDone
	if result.err != nil || result.status != "pending" || len(result.missing) != 1 {
		t.Fatalf("ensure after sweep = %#v", result)
	}
}
