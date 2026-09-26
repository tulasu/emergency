package value_objects

const (
	StatusOriginating = "originating"
	StatusRinging     = "ringing"
	StatusAnswered    = "answered"
	StatusCompleted   = "completed"
	StatusNoAnswer    = "no_answer"
	StatusFailed      = "failed"
	StatusTimedOut    = "timed_out"
)

func IsTerminal(status string) bool {
	switch status {
	case StatusCompleted, StatusNoAnswer, StatusFailed, StatusTimedOut:
		return true
	}
	return false
}
