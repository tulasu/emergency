package errs

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrNoWorkers    = errors.New("no dialog workers")
	ErrEmptyBank    = errors.New("empty slots would wipe bank canon")
)
