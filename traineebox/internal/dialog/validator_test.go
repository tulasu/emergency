package dialog

import "testing"

func TestValidateOK(t *testing.T) {
	raw := []byte(`{"id":"bilet04_call01","opening":"Alio","critical":["a"],
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	sc, err := Validate(raw, map[string]bool{"addr.street": true})
	if err != nil {
		t.Fatal(err)
	}
	if sc.ID != "bilet04_call01" || len(sc.Facts) != 1 {
		t.Fatalf("bad snapshot: %+v", sc)
	}
}

func TestValidateRejectsDrift(t *testing.T) {
	// unknown slot → 400 on open, call never starts (AD-7)
	raw := []byte(`{"id":"x","opening":"hi","facts":[{"key":"a","slot":"nope","answers":{"plain":"x"}}]}`)
	if _, err := Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected unknown-slot error")
	}
	// dangling critical → 400
	raw = []byte(`{"id":"x","opening":"hi","critical":["ghost"],
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	if _, err := Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected dangling-critical error")
	}
}

func TestValidateParityWithPython(t *testing.T) {
	// mode=both forbidden (spec E: PUT never stores what open rejects)
	raw := []byte(`{"id":"x","mode":"both","opening":"hi",
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	if _, err := Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected mode=both error")
	}
	// unknown mode forbidden
	raw = []byte(`{"id":"x","mode":"video","opening":"hi",
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	if _, err := Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected bad-mode error")
	}
	// Fact.audio forbidden in snapshot
	raw = []byte(`{"id":"x","opening":"hi",
		"facts":[{"key":"a","slot":"addr.street","audio":{"plain":"f.wav"},"answers":{"plain":"x"}}]}`)
	if _, err := Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected Fact.audio error")
	}
	// forbidden top-level keys (aliases/overrides/since/urge/mood/...)
	for _, k := range []string{"aliases", "overrides", "since", "urge", "mood", "recording_path", "meta", "pin"} {
		raw = []byte(`{"id":"x","opening":"hi","` + k + `":{},
			"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
		if _, err := Validate(raw, map[string]bool{"addr.street": true}); err == nil {
			t.Fatalf("expected forbidden-key error for %s", k)
		}
	}
	// whitespace opening rejected (spec O)
	raw = []byte(`{"id":"x","opening":"   ",
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	if _, err := Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected whitespace-opening error")
	}
	// unknown disclosure rejected (never 500 in open)
	raw = []byte(`{"id":"x","opening":"hi",
		"facts":[{"key":"a","slot":"addr.street","disclosure":"sometimes","answers":{"plain":"x"}}]}`)
	if _, err := Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected bad-disclosure error")
	}
	// non-JSON rejected
	if _, err := Validate([]byte(`{oops`), map[string]bool{}); err == nil {
		t.Fatal("expected bad-json error")
	}
}
