package value_objects

import "traineebox/internal/tickets/domain/errs"

type AttemptStatus string

const (
	AttemptStatusInProgress AttemptStatus = "in_progress"
	AttemptStatusSubmitted  AttemptStatus = "submitted"
	AttemptStatusTimedOut   AttemptStatus = "timed_out"
)

func ParseAttemptStatus(s string) (AttemptStatus, error) {
	switch AttemptStatus(s) {
	case AttemptStatusInProgress, AttemptStatusSubmitted, AttemptStatusTimedOut:
		return AttemptStatus(s), nil
	default:
		return "", errs.ErrInvalidInput
	}
}

func (s AttemptStatus) String() string { return string(s) }
