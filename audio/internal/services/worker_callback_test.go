package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"audio/internal/config"
	"audio/internal/repositories"

	"github.com/google/uuid"
)

func TestReadyCallbackSkipsNonReadyAndRetriesTransientFailure(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	callback := newReadyCallback(config.Config{TraineeBoxURL: server.URL, InternalToken: "service-token"})
	callback.retryDelay = 0
	manifest := repositories.Manifest{TicketID: uuid.New(), Digest: "digest"}
	for _, status := range []string{"pending", "partial", "error"} {
		manifest.Status = status
		if err := callback.notify(context.Background(), manifest); err != nil {
			t.Fatalf("notify %s: %v", status, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("non-ready callback requests = %d", calls.Load())
	}
	manifest.Status = "ready"
	if err := callback.notify(context.Background(), manifest); err != nil {
		t.Fatalf("notify ready: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("ready callback attempts = %d, want 2", calls.Load())
	}
}
