package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/value_objects"

	"github.com/google/uuid"
)

type fakeJobs struct {
	byID map[uuid.UUID]models.Job
}

func newFakeJobs(job models.Job) *fakeJobs {
	return &fakeJobs{byID: map[uuid.UUID]models.Job{job.ID: job}}
}

func (f *fakeJobs) Create(_ context.Context, job models.Job) error {
	f.byID[job.ID] = job
	return nil
}

func (f *fakeJobs) FindByID(_ context.Context, id uuid.UUID) (models.Job, error) {
	job, ok := f.byID[id]
	if !ok {
		return models.Job{}, errs.ErrNotFound
	}
	return job, nil
}

func (f *fakeJobs) ListByVariant(context.Context, uuid.UUID) ([]models.Job, error) {
	return nil, nil
}

func (f *fakeJobs) SaveCAS(_ context.Context, job models.Job, expectedStatus string, expectedVersion int) error {
	cur, ok := f.byID[job.ID]
	if !ok || cur.Status.String() != expectedStatus || cur.Version != expectedVersion {
		return errs.ErrConflict
	}
	job.Version = expectedVersion + 1
	f.byID[job.ID] = job
	return nil
}

func (f *fakeJobs) Delete(context.Context, uuid.UUID) error { return nil }

func (f *fakeJobs) ClaimNext(_ context.Context, workerID string, _ int) (models.Job, bool, error) {
	for id, job := range f.byID {
		if job.Status == value_objects.JobStatusQueued {
			job.Status = value_objects.JobStatusBuildingDialog
			job.Version++
			job.ClaimedBy = workerID
			f.byID[id] = job
			return job, true, nil
		}
	}
	return models.Job{}, false, nil
}

type fakeCatalog struct {
	failType bool
}

func (f *fakeCatalog) IncidentTypeExists(_ context.Context, _ string) error {
	if f.failType {
		return errors.New("catalog down")
	}
	return nil
}

func (f *fakeCatalog) ValidateTags(context.Context, string, []string) error { return nil }

func (f *fakeCatalog) ServicesExist(context.Context, []string) (bool, error) {
	return true, nil
}

type fakeDrafts struct {
	hits     int
	fail     bool
	scenario string
}

func (f *fakeDrafts) Draft(context.Context, string) (string, string, models.DraftReference, error) {
	f.hits++
	if f.fail {
		return "", "", models.DraftReference{}, errors.New("ticketgen draft: 500")
	}
	scenario := f.scenario
	if scenario == "" {
		scenario = "Горит бак у дома 5."
	}
	return "Пожар", scenario, models.DraftReference{
		IncidentTypeCode:   "101",
		TagCodes:           []string{},
		ServiceCodes:       []string{},
		ApplicantLastName:  "Петров",
		ApplicantFirstName: "Петр",
		CallerNumber:       "79001112233",
		DictatedNumber:     "79001112233",
	}, nil
}

type fakeLinter struct {
	hits        int
	unreachable []string
}

func (f *fakeLinter) Lint(context.Context, string) ([]string, bool, error) {
	f.hits++
	return f.unreachable, false, nil
}

func queuedJob() models.Job {
	job, err := models.NewJob(uuid.New(), uuid.New(), uuid.New(), "пожар")
	if err != nil {
		panic(err)
	}
	return job
}

func TestDriveJobReady(t *testing.T) {
	job := queuedJob()
	jobs := newFakeJobs(job)
	drafts := &fakeDrafts{}
	uc := DriveJob{Jobs: jobs, Catalog: &fakeCatalog{}, Drafts: drafts, WorkerID: "w1", LeaseSeconds: 60}
	drove, err := uc.Execute(context.Background())
	if err != nil || !drove {
		t.Fatalf("drove = %v, err = %v", drove, err)
	}
	got, err := jobs.FindByID(context.Background(), job.ID)
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
	if drafts.hits != 1 {
		t.Fatalf("ticketgen hits = %d, want 1", drafts.hits)
	}
}

func TestDriveJobCheckFailed(t *testing.T) {
	job := queuedJob()
	jobs := newFakeJobs(job)
	uc := DriveJob{Jobs: jobs, Catalog: &fakeCatalog{failType: true}, Drafts: &fakeDrafts{}, WorkerID: "w1"}
	if drove, err := uc.Execute(context.Background()); err != nil || !drove {
		t.Fatalf("drove = %v, err = %v", drove, err)
	}
	got, _ := jobs.FindByID(context.Background(), job.ID)
	if got.Status.String() != "failed" {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.Version != 4 {
		t.Fatalf("version = %d, want 4", got.Version)
	}
	if got.ErrorMessage == "" {
		t.Fatal("error_message empty")
	}
}

func TestDriveJobTicketgenRetryThenFailed(t *testing.T) {
	job := queuedJob()
	jobs := newFakeJobs(job)
	drafts := &fakeDrafts{fail: true}
	uc := DriveJob{Jobs: jobs, Catalog: &fakeCatalog{}, Drafts: drafts, WorkerID: "w1"}
	if drove, err := uc.Execute(context.Background()); err != nil || !drove {
		t.Fatalf("drove = %v, err = %v", drove, err)
	}
	if drafts.hits != 2 {
		t.Fatalf("ticketgen hits = %d, want 2", drafts.hits)
	}
	got, _ := jobs.FindByID(context.Background(), job.ID)
	if got.Status.String() != "failed" {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.Version != 3 {
		t.Fatalf("version = %d, want 3", got.Version)
	}
}

func TestDriveJobDialogLint(t *testing.T) {
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
			job := queuedJob()
			jobs := newFakeJobs(job)
			linter := &fakeLinter{unreachable: tc.unreachable}
			uc := DriveJob{
				Jobs: jobs, Catalog: &fakeCatalog{},
				Drafts: &fakeDrafts{scenario: snapshot}, Linter: linter, WorkerID: "w1",
			}
			if drove, err := uc.Execute(context.Background()); err != nil || !drove {
				t.Fatalf("drove = %v, err = %v", drove, err)
			}
			if linter.hits != 1 {
				t.Fatalf("lint hits = %d, want 1", linter.hits)
			}
			got, _ := jobs.FindByID(context.Background(), job.ID)
			if got.Status.String() != tc.wantStatus {
				t.Fatalf("status = %s, want %s (error=%q)", got.Status, tc.wantStatus, got.ErrorMessage)
			}
		})
	}
}

func TestTruncRedact(t *testing.T) {
	if got := trunc(strings.Repeat("ж", 2000)); len([]rune(got)) != 1000 {
		t.Fatalf("trunc runes = %d, want 1000", len([]rune(got)))
	}
	if got := trunc("short"); got != "short" {
		t.Fatalf("trunc = %q", got)
	}
	if got := redact("prompt 79001234567 draft"); strings.Contains(got, "79001234567") {
		t.Fatalf("redact = %q, want number masked", got)
	}
}
