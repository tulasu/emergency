package models_test

import (
	"testing"

	"traineebox/internal/dialog/domain/models"
)

func TestValidateOK(t *testing.T) {
	raw := []byte(`{"id":"bilet04_call01","opening":"Alio","critical":["a"],
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	sc, err := models.Validate(raw, map[string]bool{"addr.street": true})
	if err != nil {
		t.Fatal(err)
	}
	if sc.ID != "bilet04_call01" || len(sc.Facts) != 1 {
		t.Fatalf("bad snapshot: %+v", sc)
	}
}

func TestValidateRejectsDrift(t *testing.T) {
	raw := []byte(`{"id":"x","opening":"hi","facts":[{"key":"a","slot":"nope","answers":{"plain":"x"}}]}`)
	if _, err := models.Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected unknown-slot error")
	}
	raw = []byte(`{"id":"x","opening":"hi","critical":["ghost"],
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	if _, err := models.Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected dangling-critical error")
	}
}

func TestValidateParityWithPython(t *testing.T) {
	raw := []byte(`{"id":"x","mode":"both","opening":"hi",
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	if _, err := models.Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected mode=both error")
	}
	raw = []byte(`{"id":"x","mode":"video","opening":"hi",
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	if _, err := models.Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected bad-mode error")
	}
	raw = []byte(`{"id":"x","opening":"hi",
		"facts":[{"key":"a","slot":"addr.street","audio":{"plain":"f.wav"},"answers":{"plain":"x"}}]}`)
	if _, err := models.Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected Fact.audio error")
	}
	for _, k := range []string{"aliases", "overrides", "since", "urge", "mood", "recording_path", "meta", "pin"} {
		raw = []byte(`{"id":"x","opening":"hi","` + k + `":{},
			"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
		if _, err := models.Validate(raw, map[string]bool{"addr.street": true}); err == nil {
			t.Fatalf("expected forbidden-key error for %s", k)
		}
	}
	raw = []byte(`{"id":"x","opening":"   ",
		"facts":[{"key":"a","slot":"addr.street","answers":{"plain":"x"}}]}`)
	if _, err := models.Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected whitespace-opening error")
	}
	raw = []byte(`{"id":"x","opening":"hi",
		"facts":[{"key":"a","slot":"addr.street","disclosure":"sometimes","answers":{"plain":"x"}}]}`)
	if _, err := models.Validate(raw, map[string]bool{"addr.street": true}); err == nil {
		t.Fatal("expected bad-disclosure error")
	}
	if _, err := models.Validate([]byte(`{oops`), map[string]bool{}); err == nil {
		t.Fatal("expected bad-json error")
	}
}
