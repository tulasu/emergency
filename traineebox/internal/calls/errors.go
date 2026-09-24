package calls

import "errors"

var (
	ErrConflict     = errors.New("conflict: call already active")
	ErrGone         = errors.New("gone: attempt deadline passed")
	ErrBadSnapshot  = errors.New("bad snapshot: scenario drift")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

// IsTerminal reports states that never transition again (spec P).
func IsTerminal(status string) bool {
	switch status {
	case StatusCompleted, StatusNoAnswer, StatusFailed, StatusTimedOut:
		return true
	}
	return false
}
