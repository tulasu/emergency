package abilities

import (
	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/value_objects"
)

func ManageTicket(actor value_objects.MemberRole, admin bool) error {
	if admin {
		return nil
	}
	switch actor {
	case value_objects.MemberRoleOwner, value_objects.MemberRoleTeacher:
		return nil
	default:
		return errs.ErrForbidden
	}
}
