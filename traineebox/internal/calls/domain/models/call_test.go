package models_test

import (
	"testing"

	"traineebox/internal/calls/domain/models"
	"traineebox/internal/calls/domain/value_objects"
)

func TestBlocksRecall(t *testing.T) {
	for _, st := range []string{
		value_objects.StatusOriginating, value_objects.StatusRinging,
		value_objects.StatusAnswered, value_objects.StatusCompleted,
	} {
		if !(models.Call{Status: st}).BlocksRecall() {
			t.Fatalf("%s should block recall (409)", st)
		}
	}
	for _, st := range []string{
		value_objects.StatusNoAnswer, value_objects.StatusFailed, value_objects.StatusTimedOut,
	} {
		if (models.Call{Status: st}).BlocksRecall() {
			t.Fatalf("%s should allow recall", st)
		}
	}
}

func TestTerminalNeverReverts(t *testing.T) {
	for _, st := range []string{
		value_objects.StatusCompleted, value_objects.StatusNoAnswer,
		value_objects.StatusFailed, value_objects.StatusTimedOut,
	} {
		if !value_objects.IsTerminal(st) {
			t.Fatalf("%s should be terminal", st)
		}
	}
	for _, st := range []string{
		value_objects.StatusOriginating, value_objects.StatusRinging, value_objects.StatusAnswered,
	} {
		if value_objects.IsTerminal(st) {
			t.Fatalf("%s should not be terminal", st)
		}
	}
}
