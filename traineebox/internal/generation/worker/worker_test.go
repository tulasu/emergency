package worker

import (
	"testing"

	"traineebox/internal/generation/application"
)

func TestFromEnvDisabled(t *testing.T) {
	t.Setenv("TICKETGEN_URL", "")
	w := FromEnv(application.DriveJob{WorkerID: "w1"})
	if w.Enabled {
		t.Fatal("worker enabled without TICKETGEN_URL")
	}
}

func TestFromEnvEnabled(t *testing.T) {
	t.Setenv("TICKETGEN_URL", "http://ticketgen")
	t.Setenv("TICKETGEN_POLL_SECONDS", "5")
	w := FromEnv(application.DriveJob{WorkerID: "w1"})
	if !w.Enabled {
		t.Fatal("worker disabled with TICKETGEN_URL set")
	}
	if w.PollInterval.Seconds() != 5 {
		t.Fatalf("poll = %s, want 5s", w.PollInterval)
	}
}
