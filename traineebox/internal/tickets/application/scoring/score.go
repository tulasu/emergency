package scoring

import (
	"traineebox/internal/tickets/domain/models"

	"github.com/google/uuid"
)

// Score compares answer to reference. Notes are ignored.
// Each present reference component contributes equal weight; result is clamped to [1, 100].
func Score(ref models.ReferenceAnswer, answer models.Answer) (int, error) {
	type component struct {
		ok bool
	}
	var parts []component

	parts = append(parts, component{ok: answer.IncidentTypeID != nil && *answer.IncidentTypeID == ref.IncidentTypeID})
	parts = append(parts, component{ok: setEqual(ref.TagIDs, answer.TagIDs)})
	parts = append(parts, component{ok: setEqual(ref.ServiceIDs, answer.ServiceIDs)})

	if ref.ApplicantLastName != "" {
		parts = append(parts, component{ok: answer.ApplicantLastName == ref.ApplicantLastName})
	}
	if ref.ApplicantFirstName != "" {
		parts = append(parts, component{ok: answer.ApplicantFirstName == ref.ApplicantFirstName})
	}
	if ref.CallerNumber != "" {
		parts = append(parts, component{ok: answer.CallerNumber == ref.CallerNumber})
	}
	if ref.DictatedNumber != "" {
		parts = append(parts, component{ok: answer.DictatedNumber == ref.DictatedNumber})
	}

	if len(parts) == 0 {
		return 100, nil
	}
	matched := 0
	for _, p := range parts {
		if p.ok {
			matched++
		}
	}
	score := (matched * 100) / len(parts)
	if score < 1 {
		score = 1
	}
	if score > 100 {
		score = 100
	}
	return score, nil
}

func setEqual(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[uuid.UUID]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := set[id]; !ok {
			return false
		}
	}
	return true
}
