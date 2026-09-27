package audio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

func TestEnsureRetriesTransientServiceFailure(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"pending"}`))
	}))
	defer server.Close()

	result, err := (Client{BaseURL: server.URL, ServiceToken: "secret"}).EnsureWithRetry(context.Background(), EnsureRequest{
		TicketID: uuid.New(), ScenarioDigest: "digest", Scenario: json.RawMessage(`{"id":"s"}`),
		Slots: map[string]string{"addr.street": "Street"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusAccepted || result.Status != "pending" {
		t.Fatalf("result = %+v", result)
	}
	if calls.Load() != 3 {
		t.Fatalf("attempts = %d, want 3", calls.Load())
	}
}

func TestCanonicalizeStabilizesValidatedScenario(t *testing.T) {
	slots := map[string]bool{"addr.street": true}
	first, err := Canonicalize([]byte(`{"facts":[{"answers":{"plain":"x"},"slot":"addr.street","key":"fact"}],"opening":"hello","id":"s"}`), slots)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Canonicalize([]byte(`{"id":"s","opening":"hello","facts":[{"key":"fact","slot":"addr.street","answers":{"plain":"x"}}]}`), slots)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest || string(first.JSON) != string(second.JSON) {
		t.Fatalf("canonical digests differ: %q != %q", first.Digest, second.Digest)
	}
}

func TestCanonicalizePreservesFactNumbers(t *testing.T) {
	slots := map[string]bool{"caller.phone": true}
	withNumbers, err := Canonicalize([]byte(`{"id":"s","opening":"hello","facts":[{"key":"phone","slot":"caller.phone","answers":{"plain":"nine hundred sixteen"},"numbers":[916,126,34,56]}]}`), slots)
	if err != nil {
		t.Fatal(err)
	}
	withoutNumbers, err := Canonicalize([]byte(`{"id":"s","opening":"hello","facts":[{"key":"phone","slot":"caller.phone","answers":{"plain":"nine hundred sixteen"}}]}`), slots)
	if err != nil {
		t.Fatal(err)
	}
	if withNumbers.Digest == withoutNumbers.Digest {
		t.Fatal("numbers were erased from canonical digest")
	}
	var snapshot struct {
		Facts []struct {
			Numbers []int64 `json:"numbers"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(withNumbers.JSON, &snapshot); err != nil {
		t.Fatal(err)
	}
	want := []int64{916, 126, 34, 56}
	if len(snapshot.Facts) != 1 || len(snapshot.Facts[0].Numbers) != len(want) {
		t.Fatalf("numbers in canonical snapshot = %+v, want %v", snapshot.Facts, want)
	}
	for i, n := range want {
		if snapshot.Facts[0].Numbers[i] != n {
			t.Fatalf("numbers[%d] = %d, want %d", i, snapshot.Facts[0].Numbers[i], n)
		}
	}
}
