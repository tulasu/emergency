package value_objects

import (
	"strings"

	"traineebox/internal/groups/domain/errs"
)

type GroupName string

func NewGroupName(s string) (GroupName, error) {
	name := strings.TrimSpace(s)
	if len(name) < 1 || len(name) > 128 {
		return "", errs.ErrInvalidInput
	}
	return GroupName(name), nil
}

func (n GroupName) String() string { return string(n) }
