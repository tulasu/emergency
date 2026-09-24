package models_test

import (
	"testing"

	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/value_objects"

	"github.com/google/uuid"
)

func TestJobStateTransitions(t *testing.T) {
	job, err := models.NewJob(uuid.New(), uuid.New(), "пожар")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != value_objects.JobStatusQueued {
		t.Fatalf("status=%s", job.Status)
	}

	job.Status = value_objects.JobStatusReady
	job.DraftTitle = "T"
	job.DraftReference.IncidentTypeCode = "101"
	if err := job.ApplyDraft("Title", "Body", models.DraftReference{
		IncidentTypeCode: "101",
		TagCodes:         []string{"a"},
	}); err != nil {
		t.Fatal(err)
	}

	job.Status = value_objects.JobStatusFailed
	if err := job.MarkRetry(); err != nil {
		t.Fatal(err)
	}
	if job.Status != value_objects.JobStatusQueued {
		t.Fatalf("after retry %s", job.Status)
	}

	job.Status = value_objects.JobStatusPublished
	if err := job.Cancel(); err != errs.ErrInvalidState {
		t.Fatalf("cancel published: %v", err)
	}
}
