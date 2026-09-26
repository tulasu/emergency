package value_objects

import "traineebox/internal/tickets/domain/errs"

type AttemptStatus string

const (
	AttemptStatusAvailable  AttemptStatus = "available"
	AttemptStatusInProgress AttemptStatus = "in_progress"
	AttemptStatusSubmitted  AttemptStatus = "submitted"
	AttemptStatusTimedOut   AttemptStatus = "timed_out"
)

func ParseAttemptStatus(s string) (AttemptStatus, error) {
	switch AttemptStatus(s) {
	case AttemptStatusAvailable, AttemptStatusInProgress, AttemptStatusSubmitted, AttemptStatusTimedOut:
		return AttemptStatus(s), nil
	default:
		return "", errs.ErrInvalidInput
	}
}

func (s AttemptStatus) String() string { return string(s) }

func (s AttemptStatus) IsOpen() bool {
	return s == AttemptStatusAvailable || s == AttemptStatusInProgress
}

func (s AttemptStatus) IsFinished() bool {
	return s == AttemptStatusSubmitted || s == AttemptStatusTimedOut
}
