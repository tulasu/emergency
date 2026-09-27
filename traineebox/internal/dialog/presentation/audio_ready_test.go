package presentation

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type audioStatusStub struct {
	ticketID uuid.UUID
	digest   string
	status   string
	writes   int
}

func (s *audioStatusStub) UpdateAudioStatus(_ context.Context, ticketID uuid.UUID, digest, status string) (bool, error) {
	if ticketID != s.ticketID || digest != s.digest {
		return false, nil
	}
	s.status = status
	s.writes++
	return true, nil
}

func TestAudioReadyRequiresServiceTokenAndCurrentDigest(t *testing.T) {
	ticketID := uuid.New()
	store := &audioStatusStub{ticketID: ticketID, digest: "current", status: "pending"}
	api := &API{audioStatus: store, token: "shared-token"}

	badToken := &audioReadyIn{ServiceToken: "wrong-token"}
	badToken.Body.TicketID = ticketID.String()
	badToken.Body.ScenarioDigest = "current"
	if _, err := api.audioReady(context.Background(), badToken); err == nil {
		t.Fatal("invalid service token was accepted")
	}
	if store.status != "pending" || store.writes != 0 {
		t.Fatalf("invalid token changed status: %+v", store)
	}

	stale := &audioReadyIn{ServiceToken: "shared-token"}
	stale.Body.TicketID = ticketID.String()
	stale.Body.ScenarioDigest = "old"
	if _, err := api.audioReady(context.Background(), stale); err != nil {
		t.Fatalf("stale callback: %v", err)
	}
	if store.status != "pending" || store.writes != 0 {
		t.Fatalf("stale callback changed status: %+v", store)
	}

	current := &audioReadyIn{ServiceToken: "shared-token"}
	current.Body.TicketID = ticketID.String()
	current.Body.ScenarioDigest = "current"
	if _, err := api.audioReady(context.Background(), current); err != nil {
		t.Fatalf("current callback: %v", err)
	}
	if store.status != "ready" || store.writes != 1 {
		t.Fatalf("current callback did not mark ready: %+v", store)
	}
}
