package abilities_test

import (
	"testing"

	"traineebox/internal/tickets/domain/abilities"
	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/value_objects"
)

func TestManageTicket(t *testing.T) {
	if err := abilities.ManageTicket(value_objects.MemberRoleOwner, false); err != nil {
		t.Fatal(err)
	}
	if err := abilities.ManageTicket(value_objects.MemberRoleTeacher, false); err != nil {
		t.Fatal(err)
	}
	if err := abilities.ManageTicket(value_objects.MemberRoleStudent, false); err != errs.ErrForbidden {
		t.Fatalf("got %v", err)
	}
	if err := abilities.ManageTicket("", true); err != nil {
		t.Fatal(err)
	}
}

func TestStartAttempt(t *testing.T) {
	if err := abilities.StartAttempt(value_objects.MemberRoleStudent, false); err != nil {
		t.Fatal(err)
	}
	if err := abilities.StartAttempt(value_objects.MemberRoleTeacher, false); err != errs.ErrForbidden {
		t.Fatalf("got %v", err)
	}
	if err := abilities.StartAttempt("", true); err != errs.ErrForbidden {
		t.Fatalf("admin start got %v", err)
	}
}
