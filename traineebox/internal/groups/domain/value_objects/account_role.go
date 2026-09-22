package value_objects

import "traineebox/internal/groups/domain/errs"

type AccountRole string

const (
	AccountRoleAdmin   AccountRole = "admin"
	AccountRoleTeacher AccountRole = "teacher"
	AccountRoleStudent AccountRole = "student"
)

func ParseAccountRole(s string) (AccountRole, error) {
	switch AccountRole(s) {
	case AccountRoleAdmin, AccountRoleTeacher, AccountRoleStudent:
		return AccountRole(s), nil
	default:
		return "", errs.ErrInvalidInput
	}
}

func (r AccountRole) String() string { return string(r) }
