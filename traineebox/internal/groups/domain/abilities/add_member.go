package abilities

import (
	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"
)

func AddMember(
	actor value_objects.MemberRole,
	admin bool,
	account value_objects.AccountRole,
	desired value_objects.MemberRole,
	already bool,
) error {
	if already {
		return errs.ErrConflict
	}
	if desired == value_objects.MemberRoleOwner {
		return errs.ErrForbidden
	}
	if err := AssignMemberRole(account, desired); err != nil {
		return err
	}
	if admin {
		return nil
	}
	switch actor {
	case value_objects.MemberRoleOwner:
		return nil
	case value_objects.MemberRoleTeacher:
		if desired == value_objects.MemberRoleStudent {
			return nil
		}
		return errs.ErrForbidden
	default:
		return errs.ErrForbidden
	}
}
