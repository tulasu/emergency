package abilities

import (
	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/value_objects"
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

func ViewTicket(actor value_objects.MemberRole, admin bool) error {
	if admin {
		return nil
	}
	switch actor {
	case value_objects.MemberRoleOwner, value_objects.MemberRoleTeacher, value_objects.MemberRoleStudent:
		return nil
	default:
		return errs.ErrForbidden
	}
}

func StartAttempt(actor value_objects.MemberRole, admin bool) error {
	if admin {
		return errs.ErrForbidden
	}
	if actor == value_objects.MemberRoleStudent {
		return nil
	}
	return errs.ErrForbidden
}

func ViewOwnAttempt(actor value_objects.MemberRole, admin bool) error {
	if admin {
		return nil
	}
	switch actor {
	case value_objects.MemberRoleOwner, value_objects.MemberRoleTeacher, value_objects.MemberRoleStudent:
		return nil
	default:
		return errs.ErrForbidden
	}
}

func ListGroupAttempts(actor value_objects.MemberRole, admin bool) error {
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
