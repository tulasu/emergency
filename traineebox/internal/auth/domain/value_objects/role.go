package value_objects

import "traineebox/internal/auth/domain/errs"

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleAdmin, RoleTeacher, RoleStudent:
		return Role(s), nil
	default:
		return "", errs.ErrInvalidInput
	}
}
