package models_test

import (
	"testing"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
)

func TestValidateTagSelection(t *testing.T) {
	typeCode := "101"
	groups := []models.IncidentTagGroup{
		{
			IncidentTypeCode: typeCode, Code: "where", Title: "Где",
			SelectionMode: models.TagSelectionSingle, SortOrder: 0,
			Tags: []models.IncidentTag{
				{IncidentTypeCode: typeCode, GroupCode: "where", Code: "street", Title: "Улица"},
				{IncidentTypeCode: typeCode, GroupCode: "where", Code: "transport", Title: "Транспорт"},
			},
		},
		{
			IncidentTypeCode: typeCode, Code: "sign_street", Title: "Признак",
			SelectionMode: models.TagSelectionSingle, ParentTagCode: "street", SortOrder: 1,
			Tags: []models.IncidentTag{
				{IncidentTypeCode: typeCode, GroupCode: "sign_street", Code: "flame", Title: "Пламя"},
				{IncidentTypeCode: typeCode, GroupCode: "sign_street", Code: "smell", Title: "Запах"},
			},
		},
	}

	t.Run("ok single and visible child", func(t *testing.T) {
		err := models.ValidateTagSelection(&typeCode, []string{"street", "flame"}, groups)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("reject two in single group", func(t *testing.T) {
		err := models.ValidateTagSelection(&typeCode, []string{"street", "transport"}, groups)
		if err != errs.ErrInvalidTagSelection {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("reject child without parent", func(t *testing.T) {
		err := models.ValidateTagSelection(&typeCode, []string{"flame"}, groups)
		if err != errs.ErrInvalidTagSelection {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("reject foreign tag", func(t *testing.T) {
		err := models.ValidateTagSelection(&typeCode, []string{"unknown"}, groups)
		if err != errs.ErrInvalidTags {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("tags without type", func(t *testing.T) {
		err := models.ValidateTagSelection(nil, []string{"street"}, groups)
		if err != errs.ErrInvalidTags {
			t.Fatalf("got %v", err)
		}
	})
}
