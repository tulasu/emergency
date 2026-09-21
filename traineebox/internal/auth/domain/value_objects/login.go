package value_objects

import (
	"regexp"

	"traineebox/internal/auth/domain/errs"
)

var loginPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,64}$`)

type Login string

func NewLogin(s string) (Login, error) {
	if !loginPattern.MatchString(s) {
		return "", errs.ErrInvalidInput
	}
	return Login(s), nil
}

func (l Login) String() string { return string(l) }
