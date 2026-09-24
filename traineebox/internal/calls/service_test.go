package calls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeStore fakes the repo: no DB, all persistence in memory.
type fakeStore struct {
	calls       map[uuid.UUID]Call
	createCalls int
	channel     map[uuid.UUID]string
	status      map[uuid.UUID]string
	active      Call
	hasActive   bool
	turns       map[uuid.UUID][]Turn
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		calls:   map[uuid.UUID]Call{},
		channel: map[uuid.UUID]string{},
		status:  map[uuid.UUID]string{},
		turns:   map[uuid.UUID][]Turn{},
	}
}

func (f *fakeStore) ActiveForAttempt(_ context.Context, _ uuid.UUID) (Call, bool, error) {
	return f.active, f.hasActive, nil
}

func (f *fakeStore) Create(_ context.Context, c Call) error {
	f.createCalls++
	f.calls[c.ID] = c
	f.status[c.ID] = c.Status
	return nil
}

func (f *fakeStore) FindByID(_ context.Context, id uuid.UUID) (Call, error) {
	c, ok := f.calls[id]
	if !ok {
		return Call{}, ErrNotFound
	}
	c.Status = f.status[id]
	c.ChannelID = f.channel[id]
	return c, nil
}

func (f *fakeStore) ListByAttempt(_ context.Context, _ uuid.UUID) ([]Call, error) {
	return nil, nil
}

func (f *fakeStore) ListExpired(_ context.Context, _ time.Time) ([]Call, error) {
	return nil, nil
}

func (f *fakeStore) SetChannelID(_ context.Context, id uuid.UUID, channelID string) error {
	f.channel[id] = channelID
	return nil
}

func (f *fakeStore) SetStatus(_ context.Context, id uuid.UUID, status string) error {
	f.status[id] = status
	return nil
}

func (f *fakeStore) SaveTurns(_ context.Context, callID uuid.UUID, turns []Turn) error {
	f.turns[callID] = turns
	return nil
}

// fakeDialog answers /sessions/open: 400 when the digest smells of drift.
func fakeDialog(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			SessionID  string `json:"session_id"`
			BankDigest string `json:"bank_digest"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in.BankDigest == "drift" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"scenario drift"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
}

type fakeARI struct {
	srv           *httptest.Server
	originateHits int
}

func fakeARIBuilder(t *testing.T) *fakeARI {
	t.Helper()
	f := &fakeARI{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			f.originateHits++
			_, _ = w.Write([]byte(`{"id":"ch-1"}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	return f
}

func testService(fs *fakeStore, ariURL, dialogURL string) *Service {
	return &Service{
		Calls:     fs,
		ARI:       &ARI{BaseURL: ariURL, Client: http.DefaultClient},
		DialogURL: dialogURL,
		LoadAttempt: func(_ context.Context, attemptID, _ uuid.UUID) (uuid.UUID, uuid.UUID, *time.Time, string, error) {
			return uuid.New(), uuid.New(), nil, "in_progress", nil
		},
		LoadTicket: func(_ context.Context, _ uuid.UUID) (string, string, string, error) {
			return "sc1", `{"id":"sc1"}`, "abc123", nil
		},
	}
}

func TestRequestCallDriftNeverOriginates(t *testing.T) {
	fs := newFakeStore()
	ari := fakeARIBuilder(t)
	defer ari.srv.Close()
	dlg := fakeDialog(t)
	defer dlg.Close()
	svc := testService(fs, ari.srv.URL, dlg.URL)
	svc.LoadTicket = func(_ context.Context, _ uuid.UUID) (string, string, string, error) {
		return "sc1", `{"id":"sc1"}`, "drift", nil
	}
	_, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	})
	if !errors.Is(err, ErrBadSnapshot) {
		t.Fatalf("drift = %v, want ErrBadSnapshot", err)
	}
	if ari.originateHits != 0 {
		t.Fatalf("originate hits = %d, want 0 (phone never rings on drift)", ari.originateHits)
	}
}

func TestRequestCallSuccess(t *testing.T) {
	fs := newFakeStore()
	ari := fakeARIBuilder(t)
	defer ari.srv.Close()
	dlg := fakeDialog(t)
	defer dlg.Close()
	svc := testService(fs, ari.srv.URL, dlg.URL)
	call, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if call.Status != StatusRinging {
		t.Fatalf("status = %s, want ringing", call.Status)
	}
	if ari.originateHits != 1 {
		t.Fatalf("originate hits = %d, want 1", ari.originateHits)
	}
	if fs.channel[call.ID] != "ch-1" {
		t.Fatalf("channel = %q, want ch-1", fs.channel[call.ID])
	}
}

func TestRequestCallRejectedInputs(t *testing.T) {
	fs := newFakeStore()
	ari := fakeARIBuilder(t)
	defer ari.srv.Close()
	dlg := fakeDialog(t)
	defer dlg.Close()
	svc := testService(fs, ari.srv.URL, dlg.URL)

	// Endpoint charset: ARI/dialplan injection never reaches PJSIP/.
	for _, to := range []string{"a;b", "a,b", "a b", "a\nb"} {
		if _, err := svc.RequestCall(context.Background(), RequestCallInput{
			AttemptID: uuid.New(), ActorID: uuid.New(), To: to,
		}); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("to %q = %v, want ErrInvalidInput", to, err)
		}
	}
	if fs.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0 (rejected before persist)", fs.createCalls)
	}
	// Unresolved actor never dials.
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.Nil, To: "op_test",
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("nil actor = %v, want ErrInvalidInput", err)
	}
	// Empty digest would skip the digest gate: fail closed, no originate.
	svc.LoadTicket = func(_ context.Context, _ uuid.UUID) (string, string, string, error) {
		return "sc1", `{"id":"sc1"}`, "", nil
	}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	}); !errors.Is(err, ErrBadSnapshot) {
		t.Fatalf("empty digest = %v, want ErrBadSnapshot", err)
	}
	if ari.originateHits != 0 {
		t.Fatalf("originate hits = %d, want 0", ari.originateHits)
	}
}

func TestRequestCallAttemptGating(t *testing.T) {
	fs := newFakeStore()
	ari := fakeARIBuilder(t)
	defer ari.srv.Close()
	dlg := fakeDialog(t)
	defer dlg.Close()
	svc := testService(fs, ari.srv.URL, dlg.URL)

	// Repeat after answered → 409.
	fs.hasActive = true
	fs.active = Call{Status: StatusAnswered}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("repeat = %v, want ErrConflict", err)
	}
	fs.hasActive = false

	// Submitted attempt → 409.
	svc.LoadAttempt = func(_ context.Context, attemptID, _ uuid.UUID) (uuid.UUID, uuid.UUID, *time.Time, string, error) {
		return uuid.New(), uuid.New(), nil, "submitted", nil
	}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("submitted = %v, want ErrConflict", err)
	}

	// Past deadline → 410.
	past := time.Now().UTC().Add(-time.Minute)
	svc.LoadAttempt = func(_ context.Context, attemptID, _ uuid.UUID) (uuid.UUID, uuid.UUID, *time.Time, string, error) {
		return uuid.New(), uuid.New(), &past, "in_progress", nil
	}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	}); !errors.Is(err, ErrGone) {
		t.Fatalf("expired = %v, want ErrGone", err)
	}
	if ari.originateHits != 0 {
		t.Fatalf("originate hits = %d, want 0", ari.originateHits)
	}
}

func TestOnEventUnknownIs400(t *testing.T) {
	fs := newFakeStore()
	svc := &Service{Calls: fs}
	id := uuid.New()
	fs.calls[id] = Call{ID: id, Status: StatusRinging}
	fs.status[id] = StatusRinging
	_, err := svc.OnEvent(context.Background(), id, "mystery")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("unknown event = %v, want ErrInvalidInput", err)
	}
	if statusOf(t, mapCallError(err)) != http.StatusBadRequest {
		t.Fatalf("unknown event maps to %d, want 400", statusOf(t, mapCallError(err)))
	}
}

func TestMapCallErrorStatuses(t *testing.T) {
	cases := map[error]int{
		ErrBadSnapshot:  http.StatusBadRequest,
		ErrInvalidInput: http.StatusBadRequest,
		ErrConflict:     http.StatusConflict,
		ErrGone:         http.StatusGone,
		ErrNotFound:     http.StatusNotFound,
	}
	for err, want := range cases {
		if got := statusOf(t, mapCallError(err)); got != want {
			t.Fatalf("%v maps to %d, want %d", err, got, want)
		}
	}
	// Wrapped sentinels (fmt %w) map the same; plain lookalikes do not.
	if got := statusOf(t, mapCallError(fmt.Errorf("originate: %w", ErrConflict))); got != http.StatusConflict {
		t.Fatalf("wrapped conflict maps to %d, want 409", got)
	}
	plain := mapCallError(errors.New(ErrConflict.Error()))
	var se interface{ GetStatus() int }
	if errors.As(plain, &se) && se.GetStatus() == http.StatusConflict {
		t.Fatal("unwrapped lookalike must not map to 409")
	}
}

func statusOf(t *testing.T, err error) int {
	t.Helper()
	var se interface{ GetStatus() int }
	if !errors.As(err, &se) {
		t.Fatalf("error %v has no status", err)
	}
	return se.GetStatus()
}
