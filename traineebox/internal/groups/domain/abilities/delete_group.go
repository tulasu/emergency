package abilities

import (
	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"
)

func DeleteGroup(actor value_objects.MemberRole, admin bool) error {
	if admin || actor == value_objects.MemberRoleOwner {
		return nil
	}
	return errs.ErrForbidden
}
