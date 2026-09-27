package domain

import (
	"fmt"
	"sort"
	"strings"
)

// Fragment is one pre-recordable replica: stable player id + source text.
type Fragment struct {
	ID   string `json:"frag_id"`
	Text string `json:"text"`
}

// Fskey encodes UTF-8 bytes as a unique, path-safe fragment component.
func Fskey(key string) string {
	const hexDigits = "0123456789abcdef"
	var b strings.Builder
	b.Grow(2 + 2*len(key))
	b.WriteString("k-")
	for i := range len(key) {
		c := key[i]
		b.WriteByte(hexDigits[c>>4])
		b.WriteByte(hexDigits[c&0xf])
	}
	return b.String()
}

// moods enumerates composed|worried|tense in stable order.
var moods = []string{"composed", "worried", "tense"}

// genericStyles are the GENERIC styles enumerated for every mood/variant.
var genericStyles = []string{"dont_know", "mishear", "ack"}

// Enumerate lists fragments in deterministic order: opening, sorted facts ×
// explicitly nonempty answer styles, GENERIC variants, SLOW_DOWN, and URGE
// over critical slots. Urge speech comes from urgeSlots, falling back to slots.
func Enumerate(sc ScenarioSnapshot, slots, urgeSlots map[string]string, generic map[string]map[string][]string, slowdown map[string]string, urge map[string]string) []Fragment {
	out := []Fragment{}
	out = append(out, Fragment{ID: fmt.Sprintf("a/%s/opening.wav", sc.ID), Text: sc.Opening})

	keys := make([]string, 0, len(sc.Facts))
	byKey := make(map[string]FactSnapshot, len(sc.Facts))
	for _, f := range sc.Facts {
		keys = append(keys, f.Key)
		byKey[f.Key] = f
	}
	sort.Strings(keys)
	for _, key := range keys {
		f := byKey[key]
		styles := map[string]string{}
		for s, t := range f.Answers {
			if t != "" {
				styles[s] = t
			}
		}
		names := make([]string, 0, len(styles))
		for s := range styles {
			names = append(names, s)
		}
		sort.Strings(names)
		for _, s := range names {
			out = append(out, Fragment{
				ID:   fmt.Sprintf("a/%s/%s/%s.wav", sc.ID, Fskey(f.Key), s),
				Text: styles[s],
			})
		}
	}

	for _, style := range genericStyles {
		byMood, ok := generic[style]
		if !ok {
			continue
		}
		for _, mood := range moods {
			for i, text := range byMood[mood] {
				out = append(out, Fragment{
					ID:   fmt.Sprintf("common/%s/%s_%d.wav", style, mood, i),
					Text: text,
				})
			}
		}
	}
	for _, mood := range moods {
		if t, ok := slowdown[mood]; ok {
			out = append(out, Fragment{ID: fmt.Sprintf("common/slow_down/%s.wav", mood), Text: t})
		}
	}
	crit := map[string]bool{}
	for _, c := range sc.Critical {
		if f, ok := byKey[c]; ok {
			crit[f.Slot] = true
		}
	}
	slotIDs := make([]string, 0, len(crit))
	for s := range crit {
		slotIDs = append(slotIDs, s)
	}
	sort.Strings(slotIDs)
	for _, mood := range moods {
		tmpl, ok := urge[mood]
		if !ok {
			continue
		}
		for _, slot := range slotIDs {
			what, known := slots[slot]
			if !known {
				continue
			}
			if spoken := urgeSlots[slot]; spoken != "" {
				what = spoken
			}
			if strings.TrimSpace(what) == "" {
				continue
			}
			out = append(out, Fragment{
				ID:   fmt.Sprintf("common/urge/%s/%s.wav", mood, slot),
				Text: strings.ReplaceAll(tmpl, "{what}", what),
			})
		}
	}
	return out
}
