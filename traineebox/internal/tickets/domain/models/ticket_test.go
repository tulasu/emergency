package models_test

import (
	"testing"
	"time"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

func TestAnswerTagsRequireType(t *testing.T) {
	typeCode := "fire"
	groups := []models.IncidentTagGroup{{
		IncidentTypeCode: typeCode, Code: "g", Title: "G",
		SelectionMode: models.TagSelectionMulti,
		Tags: []models.IncidentTag{
			{IncidentTypeCode: typeCode, GroupCode: "g", Code: "t", Title: "T"},
		},
	}}
	ans := models.NewAnswer(nil, []string{"t"}, nil, "", "", "", "", "")
	if err := ans.ValidateTagSelection(groups); err != errs.ErrInvalidTags {
		t.Fatalf("got %v", err)
	}
	ans = models.NewAnswer(&typeCode, []string{"t"}, nil, "", "", "", "", "")
	if err := ans.ValidateTagSelection(groups); err != nil {
		t.Fatal(err)
	}
	ans = models.NewAnswer(&typeCode, []string{"foreign"}, nil, "", "", "", "", "")
	if err := ans.ValidateTagSelection(groups); err != errs.ErrInvalidTags {
		t.Fatalf("got %v", err)
	}
}

func TestAttemptExpireScoresDraft(t *testing.T) {
	deadline := time.Now().UTC().Add(-time.Second)
	started := time.Now().UTC().Add(-time.Minute)
	attempt := models.NewAvailableAttempt(uuid.New(), uuid.New(), uuid.New(), 1)
	attempt.Status = value_objects.AttemptStatusInProgress
	attempt.StartedAt = &started
	attempt.DeadlineAt = &deadline
	ticketID := uuid.New()
	typeCode := "fire"
	attempt.Answers = map[uuid.UUID]models.Answer{
		ticketID: models.NewAnswer(&typeCode, nil, nil, "", "", "", "", ""),
	}
	now := time.Now().UTC()
	if !attempt.IsExpired(now) {
		t.Fatal("expected expired")
	}
	report := models.Report{OverallScore: 42}
	if err := attempt.Expire(now, report, 42); err != nil {
		t.Fatal(err)
	}
	if attempt.Status != value_objects.AttemptStatusTimedOut {
		t.Fatalf("status = %s", attempt.Status)
	}
	if attempt.Score == nil || *attempt.Score != 42 {
		t.Fatalf("score = %v", attempt.Score)
	}
}

func TestNotesLimit(t *testing.T) {
	long := make([]rune, 1001)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := value_objects.NewNotes(string(long)); err != errs.ErrInvalidInput {
		t.Fatalf("got %v", err)
	}
}

func TestAttemptStartOnce(t *testing.T) {
	attempt := models.NewAvailableAttempt(uuid.New(), uuid.New(), uuid.New(), 1)
	now := time.Now().UTC()
	if err := attempt.Start(now, nil); err != nil {
		t.Fatal(err)
	}
	if err := attempt.Start(now, nil); err != errs.ErrAttemptInProgress {
		t.Fatalf("got %v", err)
	}
}
