package scoring_test

import (
	"testing"

	"traineebox/internal/tickets/application/scoring"
	"traineebox/internal/tickets/domain/models"

	"github.com/google/uuid"
)

func TestScorePerfectMatch(t *testing.T) {
	typeID := uuid.New()
	tag := uuid.New()
	svc := uuid.New()
	ref := models.ReferenceAnswer{
		IncidentTypeID:     typeID,
		TagIDs:             []uuid.UUID{tag},
		ServiceIDs:         []uuid.UUID{svc},
		ApplicantLastName:  "Ivanov",
		ApplicantFirstName: "Ivan",
		CallerNumber:       "112",
		DictatedNumber:     "101",
	}
	ans := models.NewAnswer(&typeID, []uuid.UUID{tag}, []uuid.UUID{svc}, "Ivanov", "Ivan", "112", "101", "notes ignored")
	score, err := scoring.Score(ref, ans)
	if err != nil {
		t.Fatal(err)
	}
	if score != 100 {
		t.Fatalf("score = %d", score)
	}
}

func TestScorePartialAndNotesIgnored(t *testing.T) {
	typeID := uuid.New()
	ref := models.ReferenceAnswer{
		IncidentTypeID:    typeID,
		ApplicantLastName: "Ivanov",
	}
	wrong := uuid.New()
	ans := models.NewAnswer(&wrong, nil, nil, "Ivanov", "", "", "", "whatever")
	score, err := scoring.Score(ref, ans)
	if err != nil {
		t.Fatal(err)
	}
	// type mismatch, last name match, empty tags/services match → 3 of 4? 
	// components: type, tags, services, lastName = 4; matched: tags, services, lastName = 3 → 75
	if score != 75 {
		t.Fatalf("score = %d", score)
	}
}
