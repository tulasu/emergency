package domain_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"audio/internal/domain"
)

type fixture struct {
	Generic  map[string]map[string][]string `json:"generic"`
	Slowdown map[string]string              `json:"slow_down"`
	Urge     map[string]string              `json:"urge"`
}

func loadFixture(t *testing.T) fixture {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", "render_fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// TestEnumerateParity fails on drift between dialog render.py constants and
// the Go enumerate inputs (spec 06). Regenerate the fixture with
// audio/tools/export_render_fixture.py, never hand-edit.
func TestEnumerateParity(t *testing.T) {
	f := loadFixture(t)

	wantGeneric := map[string]map[string][]string{
		"dont_know": {
			"composed": {"Не знаю, не видела.", "Затрудняюсь сказать.", "Этого я не знаю."},
			"worried":  {"Не знаю я! Не смотрела.", "Откуда мне знать?"},
			"tense":    {"Да не знаю я!", "Я не разглядывала, мне не до того!"},
		},
		"mishear": {
			"composed": {"Что? Повторите, пожалуйста.", "Не расслышала."},
			"worried":  {"Что вы говорите? Повторите!", "Плохо слышно, ещё раз."},
			"tense":    {"Не слышу вас! Громче!"},
		},
		"ack": {
			"composed": {"Хорошо, поняла.", "Да, поняла вас."},
			"worried":  {"Хорошо... я жду.", "Поняла, только быстрее."},
			"tense":    {"Ладно, только быстрее, пожалуйста!"},
		},
	}
	if !reflect.DeepEqual(f.Generic, wantGeneric) {
		t.Fatalf("GENERIC drift: regenerate fixture via audio/tools/export_render_fixture.py")
	}
	wantSlow := map[string]string{
		"composed": "Подождите, не так быстро.",
		"worried":  "Помедленнее, я не успеваю!",
		"tense":    "Да подождите вы, сразу всё спрашиваете!",
	}
	if !reflect.DeepEqual(f.Slowdown, wantSlow) {
		t.Fatalf("SLOW_DOWN drift: regenerate fixture")
	}
	wantUrge := map[string]string{
		"composed": "Вы записали, {what}?",
		"worried":  "Вы запишите, {what}!",
		"tense":    "Вы вообще записали, {what}?!",
	}
	if !reflect.DeepEqual(f.Urge, wantUrge) {
		t.Fatalf("URGE drift: regenerate fixture")
	}
}

func TestFskey(t *testing.T) {
	cases := map[string]string{
		"":       "k-",
		"addr#2": "k-616464722332",
		"addr_2": "k-616464725f32",
		"addr/2": "k-616464722f32",
		"улица":  "k-d183d0bbd0b8d186d0b0",
	}
	seen := map[string]bool{}
	for key, want := range cases {
		got := domain.Fskey(key)
		if got != want {
			t.Errorf("Fskey(%q) = %q, want %q", key, got, want)
		}
		if seen[got] {
			t.Errorf("duplicate fragment component for %q: %q", key, got)
		}
		seen[got] = true
	}
}

func TestEnumerateShapes(t *testing.T) {
	f := loadFixture(t)
	raw := []byte(`{"id":"demo_call","opening":"У нас мусорный контейнер горит...",
		"critical":["fire_what","addr"],
		"facts":[
			{"key":"fire_what","slot":"fire.what","answers":{"plain":"Горит мусорный контейнер у дома.","short":"","confirm":""}},
			{"key":"addr","slot":"addr.street","answers":{"plain":"Ленина 5.","short":"Ленина.","confirm":"Да, Ленина 5."}}
		]}`)
	sc, err := domain.Validate(raw, map[string]bool{"fire.what": true, "addr.street": true})
	if err != nil {
		t.Fatal(err)
	}
	slots := map[string]string{"fire.what": "что горит", "addr.street": "адрес"}
	urgeSlots := map[string]string{"fire.what": "пожар"}
	got := domain.Enumerate(sc, slots, urgeSlots, f.Generic, f.Slowdown, f.Urge)
	byID := map[string]string{}
	for _, fr := range got {
		if _, exists := byID[fr.ID]; exists {
			t.Fatalf("duplicate fragment %s", fr.ID)
		}
		byID[fr.ID] = fr.Text
	}
	for _, want := range []string{
		"a/demo_call/opening.wav",
		"a/demo_call/k-666972655f77686174/plain.wav",
		"a/demo_call/k-61646472/short.wav",
		"a/demo_call/k-61646472/confirm.wav",
		"common/dont_know/composed_0.wav",
		"common/slow_down/composed.wav",
		"common/urge/composed/fire.what.wav",
		"common/urge/composed/addr.street.wav",
	} {
		if _, ok := byID[want]; !ok {
			t.Fatalf("missing fragment %s (got %d)", want, len(got))
		}
	}
	for _, style := range []string{"short", "confirm"} {
		if _, ok := byID["a/demo_call/k-666972655f77686174/"+style+".wav"]; ok {
			t.Fatalf("empty %s answer should use the plain fragment", style)
		}
	}
	if text := byID["a/demo_call/k-61646472/confirm.wav"]; text != "Да, Ленина 5." {
		t.Fatalf("explicit confirm text = %q", text)
	}
	for slot, what := range map[string]string{"fire.what": "пожар", "addr.street": "адрес"} {
		id := "common/urge/composed/" + slot + ".wav"
		if want := strings.ReplaceAll(f.Urge["composed"], "{what}", what); byID[id] != want {
			t.Errorf("urge %s = %q, want %q", id, byID[id], want)
		}
	}
	// opening(1) + facts(plain; plain,short,confirm) + GENERIC(17) + SLOW_DOWN(3) + URGE(3×2 slots)
	if n := len(got); n != 1+1+3+17+3+6 {
		t.Fatalf("fragment count = %d", n)
	}
}

func TestEnumerateDistinctFactKeysAndMissingConfirm(t *testing.T) {
	sc := domain.ScenarioSnapshot{
		ID: "demo", Opening: "Начало",
		Facts: []domain.FactSnapshot{
			{Key: "addr#2", Slot: "addr", Answers: map[string]string{"plain": "Первый"}},
			{Key: "addr_2", Slot: "addr", Answers: map[string]string{"plain": "Второй", "confirm": "Верно"}},
			{Key: "addr/2", Slot: "addr", Answers: map[string]string{"plain": "Третий"}},
		},
	}
	got := domain.Enumerate(sc, nil, nil, nil, nil, nil)
	want := map[string]string{
		"a/demo/opening.wav":                "Начало",
		"a/demo/k-616464722332/plain.wav":   "Первый",
		"a/demo/k-616464725f32/plain.wav":   "Второй",
		"a/demo/k-616464725f32/confirm.wav": "Верно",
		"a/demo/k-616464722f32/plain.wav":   "Третий",
	}
	if len(got) != len(want) {
		t.Fatalf("fragments = %v; want exactly %d", got, len(want))
	}
	for _, fr := range got {
		if text, ok := want[fr.ID]; !ok || text != fr.Text {
			t.Fatalf("unexpected fragment %+v", fr)
		}
		delete(want, fr.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing fragments: %v", want)
	}
}
