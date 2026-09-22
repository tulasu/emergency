package models

import (
	"time"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type Ticket struct {
	ID              uuid.UUID
	GroupID         uuid.UUID
	Title           value_objects.TicketTitle
	Body            string
	MaxAttempts     *int
	AvailableFrom   *time.Time
	AvailableUntil  *time.Time
	DurationSeconds *int
	CreatedBy       uuid.UUID
	CreatedAt       time.Time
}

func NewTicket(
	groupID uuid.UUID,
	title value_objects.TicketTitle,
	body string,
	createdBy uuid.UUID,
	maxAttempts *int,
	availableFrom, availableUntil *time.Time,
	durationSeconds *int,
) (Ticket, error) {
	if maxAttempts != nil && *maxAttempts <= 0 {
		return Ticket{}, errs.ErrInvalidInput
	}
	if durationSeconds != nil && *durationSeconds <= 0 {
		return Ticket{}, errs.ErrInvalidInput
	}
	if availableFrom != nil && availableUntil != nil && availableFrom.After(*availableUntil) {
		return Ticket{}, errs.ErrInvalidInput
	}
	return Ticket{
		ID:              uuid.New(),
		GroupID:         groupID,
		Title:           title,
		Body:            body,
		MaxAttempts:     maxAttempts,
		AvailableFrom:   availableFrom,
		AvailableUntil:  availableUntil,
		DurationSeconds: durationSeconds,
		CreatedBy:       createdBy,
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (t Ticket) IsAvailableAt(now time.Time) bool {
	if t.AvailableFrom != nil && now.Before(*t.AvailableFrom) {
		return false
	}
	if t.AvailableUntil != nil && now.After(*t.AvailableUntil) {
		return false
	}
	return true
}

func (t Ticket) CanStartAttempt(now time.Time, finishedCount int, hasInProgress bool) error {
	if hasInProgress {
		return errs.ErrAttemptInProgress
	}
	if !t.IsAvailableAt(now) {
		return errs.ErrUnavailable
	}
	if t.MaxAttempts != nil && finishedCount >= *t.MaxAttempts {
		return errs.ErrAttemptsExhausted
	}
	return nil
}

func (t Ticket) DeadlineFor(startedAt time.Time) *time.Time {
	if t.DurationSeconds == nil {
		return nil
	}
	d := startedAt.Add(time.Duration(*t.DurationSeconds) * time.Second)
	return &d
}
