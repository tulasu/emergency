package scoring_test

import (
	"testing"

	"traineebox/internal/tickets/application/scoring"
	"traineebox/internal/tickets/domain/models"
)

func TestScorePerfectMatch(t *testing.T) {
	typeCode := "fire"
	ref := models.ReferenceAnswer{
		IncidentTypeCode:   typeCode,
		TagCodes:           []string{"building"},
		ServiceCodes:       []string{"fire_service"},
		ApplicantLastName:  "Ivanov",
		ApplicantFirstName: "Ivan",
		CallerNumber:       "112",
		DictatedNumber:     "101",
	}
	ans := models.NewAnswer(&typeCode, []string{"building"}, []string{"fire_service"}, "Ivanov", "Ivan", "112", "101", "notes ignored")
	score, err := scoring.Score(ref, ans)
	if err != nil {
		t.Fatal(err)
	}
	if score != 100 {
		t.Fatalf("score = %d", score)
	}
}

func TestScorePartialAndNotesIgnored(t *testing.T) {
	typeCode := "fire"
	ref := models.ReferenceAnswer{
		IncidentTypeCode:  typeCode,
		ApplicantLastName: "Ivanov",
	}
	wrong := "other"
	ans := models.NewAnswer(&wrong, nil, nil, "Ivanov", "", "", "", "whatever")
	score, err := scoring.Score(ref, ans)
	if err != nil {
		t.Fatal(err)
	}
	// components: type, tags, services, lastName = 4; matched: tags, services, lastName = 3 → 75
	if score != 75 {
		t.Fatalf("score = %d", score)
	}
}
