package domain_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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
	if got := domain.Fskey("fire#1/what"); got != "fire_1_what" {
		t.Fatalf("fskey = %q", got)
	}
}

func TestEnumerateShapes(t *testing.T) {
	f := loadFixture(t)
	raw := []byte(`{"id":"demo_call","opening":"У нас мусорный контейнер горит...",
		"critical":["fire_what"],
		"facts":[
			{"key":"fire_what","slot":"fire.what","answers":{"plain":"Горит мусорный контейнер у дома."}},
			{"key":"addr","slot":"addr.street","answers":{"plain":"Ленина 5.","short":"Ленина."}}
		]}`)
	sc, err := domain.Validate(raw, map[string]bool{"fire.what": true, "addr.street": true})
	if err != nil {
		t.Fatal(err)
	}
	slots := map[string]string{"fire.what": "что горит", "addr.street": "адрес"}
	got := domain.Enumerate(sc, slots, f.Generic, f.Slowdown, f.Urge)
	byID := map[string]string{}
	for _, fr := range got {
		byID[fr.ID] = fr.Text
	}
	// opening + facts×(plain,short,confirm) + GENERIC(8) + SLOW_DOWN(3) + URGE(3 moods × 1 critical slot)
	for _, want := range []string{
		"a/demo_call/opening.wav",
		"a/demo_call/fire_what/plain.wav",
		"a/demo_call/fire_what/short.wav",
		"a/demo_call/addr/short.wav",
		"a/demo_call/addr/confirm.wav",
		"common/dont_know/composed_0.wav",
		"common/slow_down/composed.wav",
		"common/urge/composed/fire.what.wav",
	} {
		if _, ok := byID[want]; !ok {
			t.Fatalf("missing fragment %s (got %d)", want, len(got))
		}
	}
	// opening(1) + facts(2×plain,short,confirm) + GENERIC(17) + SLOW_DOWN(3) + URGE(3×1 slot)
	if n := len(got); n != 1+2*3+17+3+3 {
		t.Fatalf("fragment count = %d", n)
	}
}
