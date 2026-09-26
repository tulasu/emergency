package abilities

import (
	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/value_objects"
)

func ManageJob(role value_objects.AccountRole) error {
	if role == value_objects.AccountRoleAdmin || role == value_objects.AccountRoleTeacher {
		return nil
	}
	return errs.ErrForbidden
}
