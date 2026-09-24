package dialog

import "testing"

// BankDigest parity with dialog/bank_source.digest_of on fixed fixtures
// (Cyrillic + quotes). The same slots/questions must hash to the same hex
// on both sides, byte-for-byte (spec J).
func TestBankDigestParityFixture(t *testing.T) {
	slots := map[string]string{
		"addr.street":  "Улица, дом",
		"common.floor": `Этаж "второй"`,
	}
	questions := map[string][]string{
		"addr.street":  {"Назовите адрес", `Какой "точный" адрес?`},
		"common.floor": {"этаж"},
	}
	if got := BankDigest(slots, questions); got != "5174a6af863e067c" {
		t.Fatalf("BankDigest = %s, want 5174a6af863e067c", got)
	}
}
