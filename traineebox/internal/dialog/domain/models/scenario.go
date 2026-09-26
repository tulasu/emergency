package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"traineebox/internal/dialog/domain/errs"
)

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

var forbiddenTop = []string{
	"aliases", "overrides", "since", "urge", "mood", "preset", "presets",
	"drift", "preview", "recording_path", "meta", "pin",
}

var allowedTop = map[string]bool{
	"id": true, "opening": true, "mode": true, "critical": true, "facts": true,
}

const (
	MaxOpeningLen = 2000
	MaxFacts      = 256
)

func Validate(raw []byte, knownSlots map[string]bool) (ScenarioSnapshot, error) {
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return ScenarioSnapshot{}, fmt.Errorf("%w: bad scenario json: %v", errs.ErrInvalidInput, err)
	}
	for _, k := range forbiddenTop {
		if _, ok := probe[k]; ok {
			return ScenarioSnapshot{}, fmt.Errorf("%w: forbidden key %q in snapshot", errs.ErrInvalidInput, k)
		}
	}
	for k := range probe {
		if !allowedTop[k] {
			return ScenarioSnapshot{}, fmt.Errorf("%w: unknown key %q in snapshot", errs.ErrInvalidInput, k)
		}
	}
	if facts, ok := probe["facts"].([]any); ok {
		for _, item := range facts {
			if m, ok := item.(map[string]any); ok {
				if _, ok := m["audio"]; ok {
					return ScenarioSnapshot{}, fmt.Errorf("%w: Fact.audio forbidden in snapshot", errs.ErrInvalidInput)
				}
			}
		}
	}
	var sc ScenarioSnapshot
	if err := json.Unmarshal(raw, &sc); err != nil {
		return ScenarioSnapshot{}, fmt.Errorf("%w: bad scenario json: %v", errs.ErrInvalidInput, err)
	}
	if mode := sc.Mode; mode != "" && mode != "voice" && mode != "text" {
		if mode == "both" {
			return ScenarioSnapshot{}, fmt.Errorf("%w: mode=both forbidden", errs.ErrInvalidInput)
		}
		return ScenarioSnapshot{}, fmt.Errorf("%w: bad mode %q", errs.ErrInvalidInput, mode)
	}
	if sc.ID == "" {
		return ScenarioSnapshot{}, fmt.Errorf("%w: scenario.id required", errs.ErrInvalidInput)
	}
	if strings.TrimSpace(sc.Opening) == "" {
		return ScenarioSnapshot{}, fmt.Errorf("%w: scenario.opening required", errs.ErrInvalidInput)
	}
	if len([]rune(sc.Opening)) > MaxOpeningLen {
		return ScenarioSnapshot{}, fmt.Errorf("%w: scenario.opening too long", errs.ErrInvalidInput)
	}
	if len(sc.Facts) == 0 {
		return ScenarioSnapshot{}, fmt.Errorf("%w: scenario.facts empty", errs.ErrInvalidInput)
	}
	if len(sc.Facts) > MaxFacts {
		return ScenarioSnapshot{}, fmt.Errorf("%w: scenario.facts too many", errs.ErrInvalidInput)
	}
	keys := map[string]bool{}
	for _, f := range sc.Facts {
		if f.Key == "" || f.Slot == "" {
			return ScenarioSnapshot{}, fmt.Errorf("%w: fact key/slot required", errs.ErrInvalidInput)
		}
		if !knownSlots[f.Slot] {
			return ScenarioSnapshot{}, fmt.Errorf("%w: unknown slot %q for fact %q", errs.ErrInvalidInput, f.Slot, f.Key)
		}
		if len(f.Answers) == 0 || strings.TrimSpace(f.Answers["plain"]) == "" {
			return ScenarioSnapshot{}, fmt.Errorf("%w: fact %q needs answers.plain", errs.ErrInvalidInput, f.Key)
		}
		if d := f.Disclosure; d != "" && d != "volunteered" && d != "on_request" {
			return ScenarioSnapshot{}, fmt.Errorf("%w: fact %q: bad disclosure %q", errs.ErrInvalidInput, f.Key, d)
		}
		if keys[f.Key] {
			return ScenarioSnapshot{}, fmt.Errorf("%w: duplicate fact %q", errs.ErrInvalidInput, f.Key)
		}
		keys[f.Key] = true
	}
	for _, c := range sc.Critical {
		if !keys[c] {
			return ScenarioSnapshot{}, fmt.Errorf("%w: critical %q not a fact", errs.ErrInvalidInput, c)
		}
	}
	for _, f := range sc.Facts {
		for _, r := range f.Requires {
			if !keys[r] {
				return ScenarioSnapshot{}, fmt.Errorf("%w: fact %q requires unknown %q", errs.ErrInvalidInput, f.Key, r)
			}
		}
	}
	return sc, nil
}
