package models_test

import (
	"testing"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"

	"github.com/google/uuid"
)

func TestValidateTagSelection(t *testing.T) {
	typeID := uuid.New()
	whereGroupID := uuid.New()
	streetID := uuid.New()
	transportID := uuid.New()
	signGroupID := uuid.New()
	flameID := uuid.New()
	smellID := uuid.New()

	groups := []models.IncidentTagGroup{
		{
			ID: whereGroupID, IncidentTypeID: typeID, Code: "where", Title: "Где",
			SelectionMode: models.TagSelectionSingle, SortOrder: 0,
			Tags: []models.IncidentTag{
				{ID: streetID, IncidentTypeID: typeID, GroupID: whereGroupID, Code: "street", Title: "Улица"},
				{ID: transportID, IncidentTypeID: typeID, GroupID: whereGroupID, Code: "transport", Title: "Транспорт"},
			},
		},
		{
			ID: signGroupID, IncidentTypeID: typeID, Code: "sign_street", Title: "Признак",
			SelectionMode: models.TagSelectionSingle, ParentTagID: &streetID, SortOrder: 1,
			Tags: []models.IncidentTag{
				{ID: flameID, IncidentTypeID: typeID, GroupID: signGroupID, Code: "flame", Title: "Пламя"},
				{ID: smellID, IncidentTypeID: typeID, GroupID: signGroupID, Code: "smell", Title: "Запах"},
			},
		},
	}

	t.Run("ok single and visible child", func(t *testing.T) {
		err := models.ValidateTagSelection(&typeID, []uuid.UUID{streetID, flameID}, groups)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("reject two in single group", func(t *testing.T) {
		err := models.ValidateTagSelection(&typeID, []uuid.UUID{streetID, transportID}, groups)
		if err != errs.ErrInvalidTagSelection {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("reject child without parent", func(t *testing.T) {
		err := models.ValidateTagSelection(&typeID, []uuid.UUID{flameID}, groups)
		if err != errs.ErrInvalidTagSelection {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("reject foreign tag", func(t *testing.T) {
		err := models.ValidateTagSelection(&typeID, []uuid.UUID{uuid.New()}, groups)
		if err != errs.ErrInvalidTags {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("tags without type", func(t *testing.T) {
		err := models.ValidateTagSelection(nil, []uuid.UUID{streetID}, groups)
		if err != errs.ErrInvalidTags {
			t.Fatalf("got %v", err)
		}
	})
}
