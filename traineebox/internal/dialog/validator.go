package dialog

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ScenarioSnapshot is the JSON canon owned by traineebox (AD-7).
// Dialog rejects `open` with 400 on drift: unknown slot, missing opening,
// dangling critical/requires. Mirrors dialog/validator.py: mode=both,
// Fact.audio, runtime meta and forbidden keys are rejected here so PUT
// never stores what open would refuse with 400 (spec E).
type ScenarioSnapshot struct {
	ID         string         `json:"id"`
	Opening    string         `json:"opening"`
	Mode       string         `json:"mode"`
	Critical   []string       `json:"critical"`
	Facts      []FactSnapshot `json:"facts"`
	Disclosure string         `json:"-"`
}

type FactSnapshot struct {
	Key        string            `json:"key"`
	Slot       string            `json:"slot"`
	Answers    map[string]string `json:"answers"`
	Requires   []string          `json:"requires,omitempty"`
	Disclosure string            `json:"disclosure,omitempty"`
}

// forbiddenTop mirrors Python FORBIDDEN_KEYS: never in a stored snapshot.
var forbiddenTop = []string{
	"aliases", "overrides", "since", "urge", "mood", "preset", "presets",
	"drift", "preview", "recording_path", "meta", "pin",
}

// Snapshot shape canon: unknown top-level keys are drift, not data (AD-7).
// Mirrored in dialog/validator.py — keep identical.
var allowedTop = map[string]bool{
	"id": true, "opening": true, "mode": true, "critical": true, "facts": true,
}

// Size caps mirrored in dialog/validator.py — keep identical.
const (
	MaxOpeningLen = 2000
	MaxFacts      = 256
)

// Validate checks snapshot against known slot ids.
// facts[].slot must be in dialog_slots; critical/requires must link existing facts.
func Validate(raw []byte, knownSlots map[string]bool) (ScenarioSnapshot, error) {
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return ScenarioSnapshot{}, fmt.Errorf("bad scenario json: %w", err)
	}
	for _, k := range forbiddenTop {
		if _, ok := probe[k]; ok {
			return ScenarioSnapshot{}, fmt.Errorf("forbidden key %q in snapshot", k)
		}
	}
	for k := range probe {
		if !allowedTop[k] {
			return ScenarioSnapshot{}, fmt.Errorf("unknown key %q in snapshot", k)
		}
	}
	if facts, ok := probe["facts"].([]any); ok {
		for _, item := range facts {
			if m, ok := item.(map[string]any); ok {
				if _, ok := m["audio"]; ok {
					return ScenarioSnapshot{}, fmt.Errorf("Fact.audio forbidden in snapshot")
				}
			}
		}
	}
	var sc ScenarioSnapshot
	if err := json.Unmarshal(raw, &sc); err != nil {
		return ScenarioSnapshot{}, fmt.Errorf("bad scenario json: %w", err)
	}
	if mode := sc.Mode; mode != "" && mode != "voice" && mode != "text" {
		if mode == "both" {
			return ScenarioSnapshot{}, fmt.Errorf("mode=both forbidden")
		}
		return ScenarioSnapshot{}, fmt.Errorf("bad mode %q", mode)
	}
	if sc.ID == "" {
		return ScenarioSnapshot{}, fmt.Errorf("scenario.id required")
	}
	if strings.TrimSpace(sc.Opening) == "" {
		return ScenarioSnapshot{}, fmt.Errorf("scenario.opening required")
	}
	if len([]rune(sc.Opening)) > MaxOpeningLen {
		return ScenarioSnapshot{}, fmt.Errorf("scenario.opening too long")
	}
	if len(sc.Facts) == 0 {
		return ScenarioSnapshot{}, fmt.Errorf("scenario.facts empty")
	}
	if len(sc.Facts) > MaxFacts {
		return ScenarioSnapshot{}, fmt.Errorf("scenario.facts too many")
	}
	keys := map[string]bool{}
	for _, f := range sc.Facts {
		if f.Key == "" || f.Slot == "" {
			return ScenarioSnapshot{}, fmt.Errorf("fact key/slot required")
		}
		if !knownSlots[f.Slot] {
			return ScenarioSnapshot{}, fmt.Errorf("unknown slot %q for fact %q", f.Slot, f.Key)
		}
		if len(f.Answers) == 0 || strings.TrimSpace(f.Answers["plain"]) == "" {
			return ScenarioSnapshot{}, fmt.Errorf("fact %q needs answers.plain", f.Key)
		}
		if d := f.Disclosure; d != "" && d != "volunteered" && d != "on_request" {
			return ScenarioSnapshot{}, fmt.Errorf("fact %q: bad disclosure %q", f.Key, d)
		}
		if keys[f.Key] {
			return ScenarioSnapshot{}, fmt.Errorf("duplicate fact %q", f.Key)
		}
		keys[f.Key] = true
	}
	for _, c := range sc.Critical {
		if !keys[c] {
			return ScenarioSnapshot{}, fmt.Errorf("critical %q not a fact", c)
		}
	}
	for _, f := range sc.Facts {
		for _, r := range f.Requires {
			if !keys[r] {
				return ScenarioSnapshot{}, fmt.Errorf("fact %q requires unknown %q", f.Key, r)
			}
		}
	}
	return sc, nil
}
