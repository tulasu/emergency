package abilities

import (
	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/value_objects"
)

func ManageTicket(role value_objects.AccountRole) error {
	if role == value_objects.AccountRoleAdmin || role == value_objects.AccountRoleTeacher {
		return nil
	}
	return errs.ErrForbidden
}

func ViewTicket(role value_objects.AccountRole) error {
	if role == value_objects.AccountRoleAdmin || role == value_objects.AccountRoleTeacher || role == value_objects.AccountRoleStudent {
		return nil
	}
	return errs.ErrForbidden
}

func StartAttempt(role value_objects.AccountRole) error {
	if role == value_objects.AccountRoleStudent {
		return nil
	}
	return errs.ErrForbidden
}

func GrantAttempt(role value_objects.AccountRole) error {
	if role == value_objects.AccountRoleAdmin || role == value_objects.AccountRoleTeacher {
		return nil
	}
	return errs.ErrForbidden
}

func ViewOwnAttempt(role value_objects.AccountRole) error {
	if role == value_objects.AccountRoleAdmin || role == value_objects.AccountRoleTeacher || role == value_objects.AccountRoleStudent {
		return nil
	}
	return errs.ErrForbidden
}

func ViewAnyAttempt(role value_objects.AccountRole) error {
	if role == value_objects.AccountRoleAdmin || role == value_objects.AccountRoleTeacher {
		return nil
	}
	return errs.ErrForbidden
}
