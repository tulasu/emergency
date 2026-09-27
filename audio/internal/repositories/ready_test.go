package repositories_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestReadyManifestsForHashIncludesSharedTickets(t *testing.T) {
	st, _ := openTestDB(t)
	ctx := context.Background()
	frags, sums := testFragment("shared synthesis", "kseniya")
	firstTicket := uuid.New()
	secondTicket := uuid.New()

	if _, _, _, err := st.Ensure(ctx, firstTicket, "first-digest", "kseniya", frags, sums); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := st.Ensure(ctx, secondTicket, "second-digest", "kseniya", frags, sums); err != nil {
		t.Fatal(err)
	}
	items, err := st.Claim(ctx, 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("claim = %#v, %v", items, err)
	}
	if err := st.FinishItem(ctx, items[0], 8000, 48044, 3); err != nil {
		t.Fatal(err)
	}

	manifests, err := st.ReadyManifestsForHash(ctx, sums[0])
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[uuid.UUID]string, len(manifests))
	for _, manifest := range manifests {
		got[manifest.TicketID] = manifest.Digest
	}
	want := map[uuid.UUID]string{
		firstTicket:  "first-digest",
		secondTicket: "second-digest",
	}
	if len(got) != len(want) {
		t.Fatalf("ready manifests = %#v, want %#v", got, want)
	}
	for ticketID, digest := range want {
		if got[ticketID] != digest {
			t.Fatalf("ticket %s digest = %q, want %q", ticketID, got[ticketID], digest)
		}
	}
}
