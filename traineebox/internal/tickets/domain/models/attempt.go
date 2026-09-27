package models

import (
	"time"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type Scorer func(ref ReferenceAnswer, answer Answer) (int, error)

type Attempt struct {
	ID            uuid.UUID
	VariantID     uuid.UUID
	UserID        uuid.UUID
	GrantedBy     uuid.UUID
	AttemptNo     int
	Status        value_objects.AttemptStatus
	AvailableFrom *time.Time
	StartedAt     *time.Time
	DeadlineAt    *time.Time
	FinishedAt    *time.Time
	Score         *int
	Answers       map[uuid.UUID]Answer
	Report        Report
}

func NewAvailableAttempt(variantID, userID, grantedBy uuid.UUID, attemptNo int) Attempt {
	return Attempt{
		ID:        uuid.New(),
		VariantID: variantID,
		UserID:    userID,
		GrantedBy: grantedBy,
		AttemptNo: attemptNo,
		Status:    value_objects.AttemptStatusAvailable,
		Answers:   map[uuid.UUID]Answer{},
		Report:    EmptyReport(),
	}
}

func NewAvailableAttemptWithSchedule(variantID, userID, grantedBy uuid.UUID, attemptNo int, availableFrom, deadline *time.Time) Attempt {
	a := NewAvailableAttempt(variantID, userID, grantedBy, attemptNo)
	a.AvailableFrom = availableFrom
	a.DeadlineAt = deadline
	return a
}

func (a Attempt) IsExpired(now time.Time) bool {
	return a.Status == value_objects.AttemptStatusInProgress &&
		a.DeadlineAt != nil &&
		!now.Before(*a.DeadlineAt)
}

func (a *Attempt) Start(now time.Time, deadline *time.Time) error {
	if a.Status != value_objects.AttemptStatusAvailable {
		if a.Status == value_objects.AttemptStatusInProgress {
			return errs.ErrAttemptInProgress
		}
		return errs.ErrAttemptNotActive
	}
	a.Status = value_objects.AttemptStatusInProgress
	a.StartedAt = &now
	a.DeadlineAt = deadline
	return nil
}

func (a *Attempt) SaveDraft(ticketID uuid.UUID, answer Answer, now time.Time) error {
	if a.Status != value_objects.AttemptStatusInProgress {
		return errs.ErrAttemptNotActive
	}
	if a.IsExpired(now) {
		return errs.ErrAttemptNotActive
	}
	if a.Answers == nil {
		a.Answers = map[uuid.UUID]Answer{}
	}
	a.Answers[ticketID] = answer
	return nil
}

func (a *Attempt) Submit(now time.Time, report Report, score int) error {
	if a.Status != value_objects.AttemptStatusInProgress {
		return errs.ErrAttemptNotActive
	}
	if a.IsExpired(now) {
		return errs.ErrAttemptNotActive
	}
	if score < 1 {
		score = 1
	}
	if score > 100 {
		score = 100
	}
	a.Status = value_objects.AttemptStatusSubmitted
	a.FinishedAt = &now
	a.Score = &score
	a.Report = report
	return nil
}

func (a *Attempt) Expire(now time.Time, report Report, score int) error {
	if a.Status != value_objects.AttemptStatusInProgress {
		return nil
	}
	if score < 1 {
		score = 1
	}
	if score > 100 {
		score = 100
	}
	a.Status = value_objects.AttemptStatusTimedOut
	a.FinishedAt = &now
	a.Score = &score
	a.Report = report
	return nil
}

func (a Attempt) AnswerFor(ticketID uuid.UUID) Answer {
	if ans, ok := a.Answers[ticketID]; ok {
		return ans
	}
	return EmptyAnswer()
}
