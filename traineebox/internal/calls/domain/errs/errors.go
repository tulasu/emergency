package errs

import "errors"

var (
	ErrConflict     = errors.New("conflict: call already active")
	ErrGone         = errors.New("gone: attempt deadline passed")
	ErrBadSnapshot  = errors.New("bad snapshot: scenario drift")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrForbidden    = errors.New("forbidden")
)
