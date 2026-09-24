package calls

import "testing"

func TestBlocksRecall(t *testing.T) {
	for _, st := range []string{StatusOriginating, StatusRinging, StatusAnswered, StatusCompleted} {
		if !(Call{Status: st}).BlocksRecall() {
			t.Fatalf("%s should block recall (409)", st)
		}
	}
	// no_answer/failed leave attempt in_progress so recall is allowed
	for _, st := range []string{StatusNoAnswer, StatusFailed, StatusTimedOut} {
		if (Call{Status: st}).BlocksRecall() {
			t.Fatalf("%s should allow recall", st)
		}
	}
}

func TestTerminalNeverReverts(t *testing.T) {
	for _, st := range []string{StatusCompleted, StatusNoAnswer, StatusFailed, StatusTimedOut} {
		if !IsTerminal(st) {
			t.Fatalf("%s should be terminal", st)
		}
	}
	for _, st := range []string{StatusOriginating, StatusRinging, StatusAnswered} {
		if IsTerminal(st) {
			t.Fatalf("%s should not be terminal", st)
		}
	}
}
