package models

import (
	"time"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type Scorer func(ref ReferenceAnswer, answer Answer) (int, error)

type Attempt struct {
	ID         uuid.UUID
	TicketID   uuid.UUID
	UserID     uuid.UUID
	AttemptNo  int
	Status     value_objects.AttemptStatus
	StartedAt  time.Time
	DeadlineAt *time.Time
	FinishedAt *time.Time
	Score      *int
	Answer     Answer
}

func NewAttempt(ticketID, userID uuid.UUID, attemptNo int, startedAt time.Time, deadline *time.Time) Attempt {
	return Attempt{
		ID:         uuid.New(),
		TicketID:   ticketID,
		UserID:     userID,
		AttemptNo:  attemptNo,
		Status:     value_objects.AttemptStatusInProgress,
		StartedAt:  startedAt,
		DeadlineAt: deadline,
		Answer:     EmptyAnswer(),
	}
}

func (a Attempt) IsExpired(now time.Time) bool {
	return a.Status == value_objects.AttemptStatusInProgress &&
		a.DeadlineAt != nil &&
		!now.Before(*a.DeadlineAt)
}

func (a *Attempt) SaveDraft(answer Answer, now time.Time) error {
	if a.Status != value_objects.AttemptStatusInProgress {
		return errs.ErrAttemptNotActive
	}
	if a.IsExpired(now) {
		return errs.ErrAttemptNotActive
	}
	a.Answer = answer
	return nil
}

func (a *Attempt) Submit(now time.Time, ref ReferenceAnswer, score Scorer) error {
	if a.Status != value_objects.AttemptStatusInProgress {
		return errs.ErrAttemptNotActive
	}
	if a.IsExpired(now) {
		return errs.ErrAttemptNotActive
	}
	s, err := score(ref, a.Answer)
	if err != nil {
		return err
	}
	if s < 1 {
		s = 1
	}
	if s > 100 {
		s = 100
	}
	a.Status = value_objects.AttemptStatusSubmitted
	a.FinishedAt = &now
	a.Score = &s
	return nil
}

func (a *Attempt) Expire(now time.Time, ref ReferenceAnswer, score Scorer) error {
	if a.Status != value_objects.AttemptStatusInProgress {
		return nil
	}
	s, err := score(ref, a.Answer)
	if err != nil {
		return err
	}
	if s < 1 {
		s = 1
	}
	if s > 100 {
		s = 100
	}
	a.Status = value_objects.AttemptStatusTimedOut
	a.FinishedAt = &now
	a.Score = &s
	return nil
}
