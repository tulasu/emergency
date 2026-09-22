package value_objects

import "traineebox/internal/tickets/domain/errs"

type MemberRole string

const (
	MemberRoleOwner   MemberRole = "owner"
	MemberRoleTeacher MemberRole = "teacher"
	MemberRoleStudent MemberRole = "student"
)

func ParseMemberRole(s string) (MemberRole, error) {
	switch MemberRole(s) {
	case MemberRoleOwner, MemberRoleTeacher, MemberRoleStudent:
		return MemberRole(s), nil
	default:
		return "", errs.ErrInvalidInput
	}
}

func (r MemberRole) String() string { return string(r) }
