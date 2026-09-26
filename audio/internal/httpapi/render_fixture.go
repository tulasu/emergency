package httpapi

// renderFixture is replaced by the generated dialog render constants
// (audio/internal/enumerate/testdata/render_fixture.json, see
// audio/tools/export_render_fixture.py). The checked-in fallback below
// mirrors dialog/core/dialog/render.py so the service boots without
// regeneration; TestEnumerateParity fails on drift.
var renderFixture = struct {
	Generic  map[string]map[string][]string
	Slowdown map[string]string
	Urge     map[string]string
}{
	Generic: map[string]map[string][]string{
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
	},
	Slowdown: map[string]string{
		"composed": "Подождите, не так быстро.",
		"worried":  "Помедленнее, я не успеваю!",
		"tense":    "Да подождите вы, сразу всё спрашиваете!",
	},
	Urge: map[string]string{
		"composed": "Вы записали, {what}?",
		"worried":  "Вы запишите, {what}!",
		"tense":    "Вы вообще записали, {what}?!",
	},
}
