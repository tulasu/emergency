package abilities

import (
	"traineebox/internal/curriculum/domain/errs"
	"traineebox/internal/curriculum/domain/value_objects"
)

func ManageLibrary(role value_objects.AccountRole) error {
	if role.IsStaff() {
		return nil
	}
	return errs.ErrForbidden
}

func ViewLibrary(role value_objects.AccountRole) error {
	if role.IsStaff() {
		return nil
	}
	return errs.ErrForbidden
}

func AssignModule(role value_objects.AccountRole) error {
	if role.IsStaff() {
		return nil
	}
	return errs.ErrForbidden
}

func ViewOwnCurriculum(role value_objects.AccountRole) error {
	if role == value_objects.AccountRoleStudent || role.IsStaff() {
		return nil
	}
	return errs.ErrForbidden
}
