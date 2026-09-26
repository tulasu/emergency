package abilities_test

import (
	"testing"

	"traineebox/internal/tickets/domain/abilities"
	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/value_objects"
)

func TestManageTicket(t *testing.T) {
	if err := abilities.ManageTicket(value_objects.AccountRoleTeacher); err != nil {
		t.Fatal(err)
	}
	if err := abilities.ManageTicket(value_objects.AccountRoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := abilities.ManageTicket(value_objects.AccountRoleStudent); err != errs.ErrForbidden {
		t.Fatalf("got %v", err)
	}
}

func TestStartAttempt(t *testing.T) {
	if err := abilities.StartAttempt(value_objects.AccountRoleStudent); err != nil {
		t.Fatal(err)
	}
	if err := abilities.StartAttempt(value_objects.AccountRoleTeacher); err != errs.ErrForbidden {
		t.Fatalf("got %v", err)
	}
	if err := abilities.StartAttempt(value_objects.AccountRoleAdmin); err != errs.ErrForbidden {
		t.Fatalf("admin start got %v", err)
	}
}
