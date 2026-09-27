package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"traineebox/internal/calls/domain/errs"
	"traineebox/internal/calls/domain/models"
	"traineebox/internal/calls/domain/value_objects"

	"github.com/google/uuid"
)

type fakeStore struct {
	calls       map[uuid.UUID]models.Call
	createCalls int
	channel     map[uuid.UUID]string
	status      map[uuid.UUID]string
	active      models.Call
	hasActive   bool
	turns       map[uuid.UUID][]models.Turn
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		calls:   map[uuid.UUID]models.Call{},
		channel: map[uuid.UUID]string{},
		status:  map[uuid.UUID]string{},
		turns:   map[uuid.UUID][]models.Turn{},
	}
}

func (f *fakeStore) ActiveForAttempt(_ context.Context, _ uuid.UUID) (models.Call, bool, error) {
	return f.active, f.hasActive, nil
}

func (f *fakeStore) Create(_ context.Context, c models.Call) error {
	f.createCalls++
	f.calls[c.ID] = c
	f.status[c.ID] = c.Status
	return nil
}

func (f *fakeStore) FindByID(_ context.Context, id uuid.UUID) (models.Call, error) {
	c, ok := f.calls[id]
	if !ok {
		return models.Call{}, errs.ErrNotFound
	}
	c.Status = f.status[id]
	c.ChannelID = f.channel[id]
	return c, nil
}

func (f *fakeStore) ListByAttempt(_ context.Context, _ uuid.UUID) ([]models.Call, error) {
	return nil, nil
}

func (f *fakeStore) ListExpired(_ context.Context, _ time.Time) ([]models.Call, error) {
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

func (f *fakeStore) SaveTurns(_ context.Context, callID uuid.UUID, turns []models.Turn) error {
	f.turns[callID] = turns
	return nil
}

type fakeDialer struct {
	originateHits int
	channelID     string
	openErr       error
}

func (f *fakeDialer) Originate(_, _ string) (string, error) {
	f.originateHits++
	if f.channelID == "" {
		return "ch-1", nil
	}
	return f.channelID, nil
}

func (f *fakeDialer) Hangup(string) error { return nil }

type fakeSessions struct {
	openErr error
	audio   *AudioReference
}

func (f *fakeSessions) Open(_ context.Context, call models.Call, _ string, audio *AudioReference) error {
	f.audio = audio
	if f.openErr != nil {
		return f.openErr
	}
	if call.BankDigest == "drift" {
		return errs.ErrBadSnapshot
	}
	return nil
}

func (f *fakeSessions) Close(context.Context, string) error { return nil }

func testService(fs *fakeStore, dialer *fakeDialer, sessions *fakeSessions) *Service {
	return &Service{
		Calls:    fs,
		Dialer:   dialer,
		Sessions: sessions,
		LoadAttempt: func(_ context.Context, _, _ uuid.UUID) (uuid.UUID, uuid.UUID, *time.Time, string, error) {
			return uuid.New(), uuid.New(), nil, "in_progress", nil
		},
		LoadTicket: func(_ context.Context, _ uuid.UUID) (TicketSnapshot, error) {
			return TicketSnapshot{ScenarioID: "sc1", ScenarioJSON: `{"id":"sc1"}`, BankDigest: "abc123"}, nil
		},
	}
}

func TestRequestCallDriftNeverOriginates(t *testing.T) {
	fs := newFakeStore()
	dialer := &fakeDialer{}
	svc := testService(fs, dialer, &fakeSessions{})
	svc.LoadTicket = func(_ context.Context, _ uuid.UUID) (TicketSnapshot, error) {
		return TicketSnapshot{ScenarioID: "sc1", ScenarioJSON: `{"id":"sc1"}`, BankDigest: "drift"}, nil
	}
	_, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	})
	if !errors.Is(err, errs.ErrBadSnapshot) {
		t.Fatalf("drift = %v, want ErrBadSnapshot", err)
	}
	if dialer.originateHits != 0 {
		t.Fatalf("originate hits = %d, want 0", dialer.originateHits)
	}
}

func TestRequestCallSuccess(t *testing.T) {
	fs := newFakeStore()
	dialer := &fakeDialer{}
	svc := testService(fs, dialer, &fakeSessions{})
	call, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if call.Status != value_objects.StatusRinging {
		t.Fatalf("status = %s, want ringing", call.Status)
	}
	if dialer.originateHits != 1 {
		t.Fatalf("originate hits = %d, want 1", dialer.originateHits)
	}
	if fs.channel[call.ID] != "ch-1" {
		t.Fatalf("channel = %q, want ch-1", fs.channel[call.ID])
	}
}

func TestRequestCallForwardsOnlyReadyAudio(t *testing.T) {
	fs := newFakeStore()
	dialer := &fakeDialer{}
	sessions := &fakeSessions{}
	svc := testService(fs, dialer, sessions)
	ticketID := uuid.New()
	svc.LoadAttempt = func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (uuid.UUID, uuid.UUID, *time.Time, string, error) {
		return ticketID, uuid.New(), nil, "in_progress", nil
	}
	svc.LoadTicket = func(_ context.Context, _ uuid.UUID) (TicketSnapshot, error) {
		return TicketSnapshot{
			ScenarioID: "sc1", ScenarioJSON: `{"id":"sc1"}`, BankDigest: "bank",
			AudioDigest: "audio-digest", AudioStatus: "ready",
		}, nil
	}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test"}); err != nil {
		t.Fatal(err)
	}
	if sessions.audio == nil || sessions.audio.TicketID != ticketID || sessions.audio.Digest != "audio-digest" {
		t.Fatalf("audio = %+v", sessions.audio)
	}

	sessions.audio = &AudioReference{}
	svc.LoadTicket = func(_ context.Context, _ uuid.UUID) (TicketSnapshot, error) {
		return TicketSnapshot{ScenarioID: "sc1", ScenarioJSON: `{"id":"sc1"}`, BankDigest: "bank", AudioDigest: "audio-digest", AudioStatus: "pending"}, nil
	}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test"}); err != nil {
		t.Fatal(err)
	}
	if sessions.audio != nil {
		t.Fatalf("pending audio = %+v, want nil", sessions.audio)
	}
}

func TestRequestCallRejectedInputs(t *testing.T) {
	fs := newFakeStore()
	dialer := &fakeDialer{}
	svc := testService(fs, dialer, &fakeSessions{})

	for _, to := range []string{"a;b", "a,b", "a b", "a\nb"} {
		if _, err := svc.RequestCall(context.Background(), RequestCallInput{
			AttemptID: uuid.New(), ActorID: uuid.New(), To: to,
		}); !errors.Is(err, errs.ErrInvalidInput) {
			t.Fatalf("to %q = %v, want ErrInvalidInput", to, err)
		}
	}
	if fs.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", fs.createCalls)
	}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.Nil, To: "op_test",
	}); !errors.Is(err, errs.ErrInvalidInput) {
		t.Fatalf("nil actor = %v, want ErrInvalidInput", err)
	}
	svc.LoadTicket = func(_ context.Context, _ uuid.UUID) (TicketSnapshot, error) {
		return TicketSnapshot{ScenarioID: "sc1", ScenarioJSON: `{"id":"sc1"}`}, nil
	}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	}); !errors.Is(err, errs.ErrBadSnapshot) {
		t.Fatalf("empty digest = %v, want ErrBadSnapshot", err)
	}
	if dialer.originateHits != 0 {
		t.Fatalf("originate hits = %d, want 0", dialer.originateHits)
	}
}

func TestRequestCallAttemptGating(t *testing.T) {
	fs := newFakeStore()
	dialer := &fakeDialer{}
	svc := testService(fs, dialer, &fakeSessions{})

	svc.LoadAttempt = func(_ context.Context, _, _ uuid.UUID) (uuid.UUID, uuid.UUID, *time.Time, string, error) {
		return uuid.New(), uuid.New(), nil, "submitted", nil
	}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	}); !errors.Is(err, errs.ErrConflict) {
		t.Fatalf("submitted = %v, want ErrConflict", err)
	}

	past := time.Now().UTC().Add(-time.Minute)
	svc.LoadAttempt = func(_ context.Context, _, _ uuid.UUID) (uuid.UUID, uuid.UUID, *time.Time, string, error) {
		return uuid.New(), uuid.New(), &past, "in_progress", nil
	}
	if _, err := svc.RequestCall(context.Background(), RequestCallInput{
		AttemptID: uuid.New(), ActorID: uuid.New(), To: "op_test",
	}); !errors.Is(err, errs.ErrGone) {
		t.Fatalf("expired = %v, want ErrGone", err)
	}
	if dialer.originateHits != 0 {
		t.Fatalf("originate hits = %d, want 0", dialer.originateHits)
	}
}

func TestOnEventUnknownIsInvalid(t *testing.T) {
	fs := newFakeStore()
	svc := &Service{Calls: fs}
	id := uuid.New()
	fs.calls[id] = models.Call{ID: id, Status: value_objects.StatusRinging}
	fs.status[id] = value_objects.StatusRinging
	_, err := svc.OnEvent(context.Background(), id, "mystery")
	if !errors.Is(err, errs.ErrInvalidInput) {
		t.Fatalf("unknown event = %v, want ErrInvalidInput", err)
	}
}
