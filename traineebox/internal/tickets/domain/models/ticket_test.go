package models_test

import (
	"testing"
	"time"

	"traineebox/internal/tickets/application/scoring"
	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

func TestTicketAvailabilityAndAttempts(t *testing.T) {
	title, _ := value_objects.NewTicketTitle("T1")
	max := 2
	from := time.Now().UTC().Add(-time.Hour)
	until := time.Now().UTC().Add(time.Hour)
	ticket, err := models.NewTicket(uuid.New(), title, "body", uuid.New(), &max, &from, &until, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := ticket.CanStartAttempt(now, 0, false); err != nil {
		t.Fatal(err)
	}
	if err := ticket.CanStartAttempt(now, 0, true); err != errs.ErrAttemptInProgress {
		t.Fatalf("got %v", err)
	}
	if err := ticket.CanStartAttempt(now, 2, false); err != errs.ErrAttemptsExhausted {
		t.Fatalf("got %v", err)
	}
	past := until.Add(time.Minute)
	if err := ticket.CanStartAttempt(past, 0, false); err != errs.ErrUnavailable {
		t.Fatalf("got %v", err)
	}
}

func TestAnswerTagsRequireType(t *testing.T) {
	tagID := uuid.New()
	ans := models.NewAnswer(nil, []uuid.UUID{tagID}, nil, "", "", "", "", "")
	if err := ans.ValidateTagsAgainstType(map[uuid.UUID]struct{}{tagID: {}}); err != errs.ErrInvalidTags {
		t.Fatalf("got %v", err)
	}
	typeID := uuid.New()
	ans = models.NewAnswer(&typeID, []uuid.UUID{tagID}, nil, "", "", "", "", "")
	if err := ans.ValidateTagsAgainstType(map[uuid.UUID]struct{}{tagID: {}}); err != nil {
		t.Fatal(err)
	}
	foreign := uuid.New()
	ans = models.NewAnswer(&typeID, []uuid.UUID{foreign}, nil, "", "", "", "", "")
	if err := ans.ValidateTagsAgainstType(map[uuid.UUID]struct{}{tagID: {}}); err != errs.ErrInvalidTags {
		t.Fatalf("got %v", err)
	}
}

func TestAttemptExpireScoresDraft(t *testing.T) {
	typeID := uuid.New()
	ref := models.ReferenceAnswer{
		TicketID:       uuid.New(),
		IncidentTypeID: typeID,
		TagIDs:         nil,
		ServiceIDs:     nil,
	}
	deadline := time.Now().UTC().Add(-time.Second)
	attempt := models.NewAttempt(ref.TicketID, uuid.New(), 1, time.Now().UTC().Add(-time.Minute), &deadline)
	ansType := typeID
	_ = attempt.SaveDraft(models.NewAnswer(&ansType, nil, nil, "", "", "", "", ""), time.Now().UTC().Add(-time.Minute))
	// SaveDraft after expire should fail; reset status for Expire test by constructing fresh
	attempt = models.NewAttempt(ref.TicketID, uuid.New(), 1, time.Now().UTC().Add(-time.Minute), &deadline)
	attempt.Answer = models.NewAnswer(&ansType, nil, nil, "", "", "", "", "")
	now := time.Now().UTC()
	if !attempt.IsExpired(now) {
		t.Fatal("expected expired")
	}
	if err := attempt.Expire(now, ref, scoring.Score); err != nil {
		t.Fatal(err)
	}
	if attempt.Status != value_objects.AttemptStatusTimedOut {
		t.Fatalf("status = %s", attempt.Status)
	}
	if attempt.Score == nil || *attempt.Score < 1 {
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
