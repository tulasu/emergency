package worker

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"traineebox/internal/generation/application"
)

type Worker struct {
	Drive        application.DriveJob
	Enabled      bool
	PollInterval time.Duration
}

func FromEnv(drive application.DriveJob) *Worker {
	if strings.TrimSpace(os.Getenv("TICKETGEN_URL")) == "" {
		return &Worker{Drive: drive, Enabled: false, PollInterval: 2 * time.Second}
	}
	poll := 2 * time.Second
	if v, err := strconv.Atoi(os.Getenv("TICKETGEN_POLL_SECONDS")); err == nil && v > 0 {
		poll = time.Duration(v) * time.Second
	}
	return &Worker{Drive: drive, Enabled: true, PollInterval: poll}
}

func (w *Worker) Loop(ctx context.Context) {
	if !w.Enabled {
		log.Print("generation worker disabled (TICKETGEN_URL unset)")
		return
	}
	log.Printf("generation worker %s driving", w.Drive.WorkerID)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		drove, err := w.Drive.Execute(ctx)
		if err != nil {
			log.Printf("generation worker: %v", err)
		}
		if !drove {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.PollInterval):
			}
		}
	}
}
