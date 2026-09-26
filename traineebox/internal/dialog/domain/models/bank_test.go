package models_test

import (
	"testing"

	"traineebox/internal/dialog/domain/models"
)

func TestBankDigestParityFixture(t *testing.T) {
	slots := map[string]string{
		"addr.street":  "Улица, дом",
		"common.floor": `Этаж "второй"`,
	}
	questions := map[string][]string{
		"addr.street":  {"Назовите адрес", `Какой "точный" адрес?`},
		"common.floor": {"этаж"},
	}
	if got := models.BankDigest(slots, questions); got != "5174a6af863e067c" {
		t.Fatalf("BankDigest = %s, want 5174a6af863e067c", got)
	}
}
