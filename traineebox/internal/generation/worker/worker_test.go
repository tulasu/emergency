package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	authvo "traineebox/internal/auth/domain/value_objects"
	genmodels "traineebox/internal/generation/domain/models"
	geninfra "traineebox/internal/generation/infrastructure"
	"traineebox/internal/testkit"
	ticketsmodels "traineebox/internal/tickets/domain/models"
	ticketsrepos "traineebox/internal/tickets/domain/repositories"

	"github.com/google/uuid"
)

// fakeCatalog is the catalog surface: zero groups, one known type.
type fakeCatalog struct {
	failType bool
}

func (f *fakeCatalog) ListIncidentTypes(_ context.Context) ([]ticketsmodels.IncidentType, error) {
	return []ticketsmodels.IncidentType{{Code: "101", Title: "Пожар"}}, nil
}

func (f *fakeCatalog) ListTagGroupsByType(_ context.Context, _ string) ([]ticketsmodels.IncidentTagGroup, error) {
	return nil, nil
}

func (f *fakeCatalog) ListServices(_ context.Context) ([]ticketsmodels.EmergencyService, error) {
	return nil, nil
}

func (f *fakeCatalog) FindIncidentTypeByCode(_ context.Context, code string) (ticketsmodels.IncidentType, error) {
	if f.failType {
		return ticketsmodels.IncidentType{}, errCatalogDown
	}
	return ticketsmodels.IncidentType{Code: code, Title: "Пожар"}, nil
}

func (f *fakeCatalog) ServiceExists(_ context.Context, _ []string) (bool, error) {
	return true, nil
}

func (f *fakeCatalog) RecommendServices(_ context.Context, _ string, _ []string) ([]string, error) {
	return nil, nil
}

type catalogErr string

func (e catalogErr) Error() string { return string(e) }

const errCatalogDown = catalogErr("catalog down")

var _ ticketsrepos.CatalogRepository = (*fakeCatalog)(nil)

type ticketgenFake struct {
	hits     atomic.Int32
	fail500  bool
	scenario string
}

func (f *ticketgenFake) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		if f.fail500 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		scenario := f.scenario
		if scenario == "" {
			scenario = "Горит бак у дома 5."
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"draft_title":   "Пожар",
			"scenario_text": scenario,
			"draft_reference": map[string]any{
				"incident_type_code":   "101",
				"tag_codes":            []string{},
				"service_codes":        []string{},
				"applicant_last_name":  "Петров",
				"applicant_first_name": "Петр",
				"caller_number":        "79001112233",
				"dictated_number":      "79001112233",
			},
		})
	})
}

func driveSetup(t *testing.T) (context.Context, *geninfra.JobRepository, uuid.UUID, uuid.UUID) {
	t.Helper()
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	ctx := context.Background()
	teacher := testkit.SeedUser(t, pool, "driveowner", "password1", authvo.RoleTeacher)
	groupID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO groups (id, name) VALUES ($1, 'Drive')`, groupID); err != nil {
		t.Fatal(err)
	}
	return ctx, geninfra.NewJobRepository(pool), groupID, teacher.ID
}

func driveJob(t *testing.T, ctx context.Context, jobs *geninfra.JobRepository, groupID, teacherID uuid.UUID, prompt string) uuid.UUID {
	t.Helper()
	job, err := genmodels.NewJob(groupID, teacherID, prompt)
	if err != nil {
		t.Fatal(err)
	}
	if err := jobs.Create(ctx, job); err != nil {
		t.Fatal(err)
	}
	return job.ID
}

func driveWorker(jobs *geninfra.JobRepository, cat *fakeCatalog, ticketgenURL string, dialogURLs []string) *Worker {
	return &Worker{
		Jobs: jobs, Catalog: cat, TicketgenURL: ticketgenURL, DialogURLs: dialogURLs,
		WorkerID: "w1", LeaseSeconds: 60, PollInterval: time.Millisecond,
	}
}

func TestDriveOnceReady(t *testing.T) {
	ctx, jobs, groupID, teacherID := driveSetup(t)
	tg := &ticketgenFake{}
	srv := httptest.NewServer(tg.handler())
	defer srv.Close()
	id := driveJob(t, ctx, jobs, groupID, teacherID, "пожар")

	w := driveWorker(jobs, &fakeCatalog{}, srv.URL, nil)
	drove, err := w.DriveOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !drove {
		t.Fatal("drove = false, want true")
	}
	// building_dialog → checking_dialog → ready: two CAS bumps past the claim.
	got, err := jobs.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status.String() != "ready" {
		t.Fatalf("status = %s, want ready", got.Status)
	}
	if got.Version != 4 {
		t.Fatalf("version = %d, want 4 (claim + checking + ready)", got.Version)
	}
	if got.DraftTitle != "Пожар" {
		t.Fatalf("draft title = %q", got.DraftTitle)
	}
	if tg.hits.Load() != 1 {
		t.Fatalf("ticketgen hits = %d, want 1 (no retry on success)", tg.hits.Load())
	}
}

func TestDriveOnceCheckFailed(t *testing.T) {
	ctx, jobs, groupID, teacherID := driveSetup(t)
	tg := &ticketgenFake{}
	srv := httptest.NewServer(tg.handler())
	defer srv.Close()
	id := driveJob(t, ctx, jobs, groupID, teacherID, "пожар")

	w := driveWorker(jobs, &fakeCatalog{failType: true}, srv.URL, nil)
	if drove, err := w.DriveOnce(ctx); err != nil || !drove {
		t.Fatalf("drove = %v, err = %v", drove, err)
	}
	got, err := jobs.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status.String() != "failed" {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.Version != 4 {
		t.Fatalf("version = %d, want 4 (claim + checking + failed)", got.Version)
	}
	if got.ErrorMessage == "" {
		t.Fatal("error_message empty, want the check error")
	}
}

func TestDriveOnceTicketgenRetryThenFailed(t *testing.T) {
	ctx, jobs, groupID, teacherID := driveSetup(t)
	tg := &ticketgenFake{fail500: true}
	srv := httptest.NewServer(tg.handler())
	defer srv.Close()
	id := driveJob(t, ctx, jobs, groupID, teacherID, "пожар")

	w := driveWorker(jobs, &fakeCatalog{}, srv.URL, nil)
	if drove, err := w.DriveOnce(ctx); err != nil || !drove {
		t.Fatalf("drove = %v, err = %v", drove, err)
	}
	if tg.hits.Load() != 2 {
		t.Fatalf("ticketgen hits = %d, want 2 (one retry with backoff)", tg.hits.Load())
	}
	got, err := jobs.FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status.String() != "failed" {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.Version != 3 {
		t.Fatalf("version = %d, want 3 (claim + failed, never reached checking)", got.Version)
	}
}

func TestDriveOnceDialogLint(t *testing.T) {
	snapshot := `{"id":"s1","opening":"Алло","critical":["k1"],` +
		`"facts":[{"key":"k1","slot":"a.b","answers":{"plain":"x"}}]}`
	for _, tc := range []struct {
		name        string
		unreachable []string
		wantStatus  string
	}{
		{"clean", nil, "ready"},
		{"unreachable_critical", []string{"k1"}, "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, jobs, groupID, teacherID := driveSetup(t)
			tg := &ticketgenFake{scenario: snapshot}
			tsrv := httptest.NewServer(tg.handler())
			defer tsrv.Close()
			var lintHits atomic.Int32
			dsrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				lintHits.Add(1)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id": "s1", "facts": 1, "critical": []string{"k1"},
					"unreachable": tc.unreachable,
				})
			}))
			defer dsrv.Close()
			id := driveJob(t, ctx, jobs, groupID, teacherID, "пожар")

			w := driveWorker(jobs, &fakeCatalog{}, tsrv.URL, []string{dsrv.URL})
			if drove, err := w.DriveOnce(ctx); err != nil || !drove {
				t.Fatalf("drove = %v, err = %v", drove, err)
			}
			if lintHits.Load() != 1 {
				t.Fatalf("dialog lint hits = %d, want 1", lintHits.Load())
			}
			got, err := jobs.FindByID(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status.String() != tc.wantStatus {
				t.Fatalf("status = %s, want %s (error=%q)", got.Status, tc.wantStatus, got.ErrorMessage)
			}
		})
	}
}

func TestTruncRedact(t *testing.T) {
	// Rune-safe: 2000 Cyrillic chars (4000 bytes) truncate to 1000 runes.
	if got := trunc(strings.Repeat("ж", 2000)); len([]rune(got)) != 1000 {
		t.Fatalf("trunc runes = %d, want 1000", len([]rune(got)))
	}
	if got := trunc("short"); got != "short" {
		t.Fatalf("trunc = %q", got)
	}
	// PII digit runs never reach logs / error_message.
	if got := redact("prompt 79001234567 draft"); strings.Contains(got, "79001234567") {
		t.Fatalf("redact = %q, want number masked", got)
	}
}
