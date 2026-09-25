package application

import (
	"strings"
	"unicode/utf8"

	"traineebox/internal/auth/domain/errs"
)

func normalizeFullName(s string, allowEmpty bool) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		if allowEmpty {
			return "", nil
		}
		return "", errs.ErrInvalidInput
	}
	if utf8.RuneCountInString(s) > 128 {
		return "", errs.ErrInvalidInput
	}
	return s, nil
}
