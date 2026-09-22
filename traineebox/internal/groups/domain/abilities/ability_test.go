package abilities_test

import (
	"errors"
	"testing"

	"traineebox/internal/groups/domain/abilities"
	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"
)

func TestAssignMemberRole(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		account value_objects.AccountRole
		desired value_objects.MemberRole
		wantErr error
	}{
		{name: "teacher owner", account: value_objects.AccountRoleTeacher, desired: value_objects.MemberRoleOwner},
		{name: "teacher teacher", account: value_objects.AccountRoleTeacher, desired: value_objects.MemberRoleTeacher},
		{name: "teacher student", account: value_objects.AccountRoleTeacher, desired: value_objects.MemberRoleStudent, wantErr: errs.ErrForbidden},
		{name: "student student", account: value_objects.AccountRoleStudent, desired: value_objects.MemberRoleStudent},
		{name: "student teacher", account: value_objects.AccountRoleStudent, desired: value_objects.MemberRoleTeacher, wantErr: errs.ErrForbidden},
		{name: "admin teacher", account: value_objects.AccountRoleAdmin, desired: value_objects.MemberRoleTeacher, wantErr: errs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := abilities.AssignMemberRole(tt.account, tt.desired)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddMember(t *testing.T) {
	t.Parallel()
	err := abilities.AddMember(
		value_objects.MemberRoleTeacher,
		false,
		value_objects.AccountRoleTeacher,
		value_objects.MemberRoleTeacher,
		false,
	)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("co-teacher add teacher: %v", err)
	}

	err = abilities.AddMember(
		value_objects.MemberRoleTeacher,
		false,
		value_objects.AccountRoleStudent,
		value_objects.MemberRoleStudent,
		false,
	)
	if err != nil {
		t.Fatalf("co-teacher add student: %v", err)
	}

	err = abilities.AddMember(
		value_objects.MemberRoleOwner,
		false,
		value_objects.AccountRoleStudent,
		value_objects.MemberRoleStudent,
		true,
	)
	if !errors.Is(err, errs.ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
}

func TestRemoveMemberLastOwner(t *testing.T) {
	t.Parallel()
	err := abilities.RemoveMember(value_objects.MemberRoleOwner, false, value_objects.MemberRoleOwner, 1)
	if !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("last owner: %v", err)
	}
	err = abilities.RemoveMember(value_objects.MemberRoleOwner, false, value_objects.MemberRoleOwner, 2)
	if err != nil {
		t.Fatalf("two owners: %v", err)
	}
}
