package errs

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("conflict")
	ErrInvalidInput        = errors.New("invalid input")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrUserBlocked         = errors.New("user blocked")
	ErrUnavailable         = errors.New("ticket unavailable")
	ErrAttemptsExhausted   = errors.New("attempts exhausted")
	ErrAttemptInProgress   = errors.New("attempt already in progress")
	ErrAttemptNotActive    = errors.New("attempt not active")
	ErrNoReferenceAnswer   = errors.New("no reference answer")
	ErrInvalidTags         = errors.New("tags do not belong to incident type")
	ErrInvalidTagSelection = errors.New("invalid tag selection")
)
