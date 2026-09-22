package ability

import (
	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"
)

func RemoveMember(
	actor value_objects.MemberRole,
	admin bool,
	target value_objects.MemberRole,
	ownerCount int,
) error {
	if target == value_objects.MemberRoleOwner && ownerCount <= 1 {
		return errs.ErrForbidden
	}
	if admin {
		return nil
	}
	switch actor {
	case value_objects.MemberRoleOwner:
		return nil
	case value_objects.MemberRoleTeacher:
		if target == value_objects.MemberRoleStudent {
			return nil
		}
		return errs.ErrForbidden
	default:
		return errs.ErrForbidden
	}
}
