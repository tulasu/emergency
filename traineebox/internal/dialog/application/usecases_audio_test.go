package application

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"traineebox/internal/audio"

	"github.com/google/uuid"
)

type audioTestBank struct{}

func (audioTestBank) ListSlotIDs(context.Context) (map[string]bool, error) {
	return map[string]bool{"addr.street": true}, nil
}
func (audioTestBank) ListSlotLabels(context.Context) (map[string]string, error) {
	return map[string]string{"addr.street": "Street", "addr.house": "House"}, nil
}
func (audioTestBank) ListSlotUrges(context.Context) (map[string]string, error) {
	return map[string]string{"addr.street": "Say street now"}, nil
}
func (audioTestBank) ReplaceBank(context.Context, string, map[string]string, map[string][]string) error {
	return nil
}
func (audioTestBank) BankSnapshot(context.Context) (map[string]string, map[string][]string, error) {
	return nil, nil, nil
}
func (audioTestBank) BankVersion(context.Context) (string, string, error) { return "", "", nil }

type audioTestStore struct {
	mu      sync.Mutex
	digest  string
	status  string
	updated chan struct{}
}

func (s *audioTestStore) Save(_ context.Context, _ uuid.UUID, _ string, _ string, digest, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.digest = digest
	s.status = status
	return nil
}

func (s *audioTestStore) UpdateAudioStatus(_ context.Context, _ uuid.UUID, digest, status string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	updated := s.digest == digest && (s.status == "pending" || status == "ready")
	if updated {
		s.status = status
	}
	if s.updated != nil {
		close(s.updated)
		s.updated = nil
	}
	return updated, nil
}

func (s *audioTestStore) State() (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.digest, s.status
}

type audioTestEnsurer struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
	done    chan struct{}
}

func (e *audioTestEnsurer) Configured() bool { return true }

func (e *audioTestEnsurer) EnsureWithRetry(_ context.Context, _ audio.EnsureRequest) (audio.EnsureResult, error) {
	e.mu.Lock()
	e.calls++
	call := e.calls
	e.mu.Unlock()
	if call == 1 {
		close(e.started)
		<-e.release
		close(e.done)
		return audio.EnsureResult{StatusCode: 200, Status: "ready"}, nil
	}
	return audio.EnsureResult{StatusCode: 202, Status: "pending"}, nil
}

func TestPutScenarioStaleEnsureCannotMarkReplacementReady(t *testing.T) {
	updated := make(chan struct{})
	store := &audioTestStore{updated: updated}
	ensurer := &audioTestEnsurer{started: make(chan struct{}), release: make(chan struct{}), done: make(chan struct{})}
	put := PutScenario{Bank: audioTestBank{}, Store: store, Audio: ensurer}
	ticketID := uuid.New()
	first, err := put.Execute(context.Background(), ticketID, []byte(`{"id":"s","opening":"first","facts":[{"key":"street","slot":"addr.street","answers":{"plain":"Main"}}]}`), "v1", true)
	if err != nil || first.AudioStatus != "pending" {
		t.Fatalf("first = %+v, %v", first, err)
	}
	<-ensurer.started
	second, err := put.Execute(context.Background(), ticketID, []byte(`{"id":"s","opening":"second","facts":[{"key":"street","slot":"addr.street","answers":{"plain":"Main"}}]}`), "v2", true)
	if err != nil || second.AudioStatus != "pending" {
		t.Fatalf("second = %+v, %v", second, err)
	}
	secondDigest, status := store.State()
	if status != "pending" || secondDigest == "" {
		t.Fatalf("replacement state = %q, %q", secondDigest, status)
	}
	close(ensurer.release)
	<-ensurer.done
	<-updated
	if digest, status := store.State(); digest != secondDigest || status != "pending" {
		t.Fatalf("stale completion overwrote replacement: %q, %q", digest, status)
	}
}

func TestPutScenarioPrerenderDisabledClearsAudioState(t *testing.T) {
	store := &audioTestStore{digest: "old", status: "ready"}
	ensurer := &audioTestEnsurer{started: make(chan struct{}), release: make(chan struct{}), done: make(chan struct{})}
	put := PutScenario{Bank: audioTestBank{}, Store: store, Audio: ensurer}
	result, err := put.Execute(context.Background(), uuid.New(), []byte(`{"id":"s","opening":"hello","facts":[{"key":"street","slot":"addr.street","answers":{"plain":"Main"}}]}`), "v", false)
	if err != nil || result.AudioStatus != "none" {
		t.Fatalf("result = %+v, %v", result, err)
	}
	if digest, status := store.State(); digest != "" || status != "none" {
		t.Fatalf("state = %q, %q", digest, status)
	}
	ensurer.mu.Lock()
	calls := ensurer.calls
	ensurer.mu.Unlock()
	if calls != 0 {
		t.Fatalf("ensure calls = %d", calls)
	}
}

type fixedAudioEnsurer struct {
	result audio.EnsureResult
}

func (fixedAudioEnsurer) Configured() bool { return true }

func (e fixedAudioEnsurer) EnsureWithRetry(context.Context, audio.EnsureRequest) (audio.EnsureResult, error) {
	return e.result, nil
}

func TestPutScenarioMarksAudioManifestError(t *testing.T) {
	updated := make(chan struct{})
	store := &audioTestStore{updated: updated}
	put := PutScenario{
		Bank:  audioTestBank{},
		Store: store,
		Audio: fixedAudioEnsurer{result: audio.EnsureResult{StatusCode: http.StatusAccepted, Status: "error"}},
	}
	result, err := put.Execute(context.Background(), uuid.New(), []byte(`{"id":"s","opening":"hello","facts":[{"key":"street","slot":"addr.street","answers":{"plain":"Main"}}]}`), "v", true)
	if err != nil || result.AudioStatus != "pending" {
		t.Fatalf("result = %+v, %v", result, err)
	}
	select {
	case <-updated:
	case <-time.After(time.Second):
		t.Fatal("audio status was not updated")
	}
	if _, status := store.State(); status != "error" {
		t.Fatalf("audio_status = %q, want error", status)
	}
}

type requestAudioEnsurer struct {
	requests chan audio.EnsureRequest
	result   audio.EnsureResult
}

func (e requestAudioEnsurer) Configured() bool { return true }

func (e requestAudioEnsurer) EnsureWithRetry(_ context.Context, req audio.EnsureRequest) (audio.EnsureResult, error) {
	e.requests <- req
	return e.result, nil
}

func TestPutScenarioReusedAudioReadyAndSpokenUrges(t *testing.T) {
	store := &audioTestStore{updated: make(chan struct{})}
	updated := store.updated
	ensurer := requestAudioEnsurer{
		requests: make(chan audio.EnsureRequest, 1),
		result:   audio.EnsureResult{StatusCode: http.StatusAccepted, Status: "ready"},
	}
	put := PutScenario{Bank: audioTestBank{}, Store: store, Audio: ensurer}
	_, err := put.Execute(context.Background(), uuid.New(), []byte(`{"id":"s","opening":"hello","facts":[{"key":"street","slot":"addr.street","answers":{"plain":"Main"}}]}`), "v", true)
	if err != nil {
		t.Fatal(err)
	}
	req := <-ensurer.requests
	<-updated
	if _, status := store.State(); status != "ready" {
		t.Fatalf("reused audio status = %q, want ready", status)
	}
	if req.Slots["addr.street"] != "Street" || req.UrgeSlots["addr.street"] != "Say street now" || req.UrgeSlots["addr.house"] != "House" {
		t.Fatalf("ensure slot mappings = labels %v, urges %v", req.Slots, req.UrgeSlots)
	}
}
