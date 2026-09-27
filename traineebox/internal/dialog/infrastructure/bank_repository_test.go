package infrastructure

import (
	"context"
	"testing"

	"traineebox/internal/testkit"
)

func TestReplaceBankPreservesUrgeForRetainedSlot(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	ctx := context.Background()
	repo := NewBankRepository(pool)
	if _, err := pool.Exec(ctx, `DELETE FROM slot_questions`); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceBank(ctx, "v1", map[string]string{"addr.street": "Street", "obsolete": "Old"}, map[string][]string{
		"addr.street": {"Where is it?"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO slot_questions (slot_id, question, source) VALUES ('addr.street', 'Say street now', 'urge'), ('obsolete', 'Say old now', 'urge')`); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceBank(ctx, "v2", map[string]string{"addr.street": "Street updated"}, map[string][]string{
		"addr.street": {"Which street?"},
	}); err != nil {
		t.Fatal(err)
	}
	urges, err := repo.ListSlotUrges(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(urges) != 1 || urges["addr.street"] != "Say street now" {
		t.Fatalf("urges after reload = %v", urges)
	}
	var source string
	if err := pool.QueryRow(ctx, `SELECT source FROM slot_questions WHERE slot_id = 'addr.street' AND question = 'Which street?'`).Scan(&source); err != nil {
		t.Fatal(err)
	}
	if source == "urge" {
		t.Fatal("new bank question incorrectly marked as an urge")
	}
}
