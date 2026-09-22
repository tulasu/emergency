package ability

import (
	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"
)

func AssignMemberRole(account value_objects.AccountRole, desired value_objects.MemberRole) error {
	switch account {
	case value_objects.AccountRoleTeacher:
		if desired == value_objects.MemberRoleOwner || desired == value_objects.MemberRoleTeacher {
			return nil
		}
	case value_objects.AccountRoleStudent:
		if desired == value_objects.MemberRoleStudent {
			return nil
		}
	}
	return errs.ErrForbidden
}
