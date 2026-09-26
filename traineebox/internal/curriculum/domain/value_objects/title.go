package value_objects

import (
	"strings"
	"unicode/utf8"

	"traineebox/internal/curriculum/domain/errs"
)

type Title string

func NewTitle(s string) (Title, error) {
	trimmed := strings.TrimSpace(s)
	n := utf8.RuneCountInString(trimmed)
	if n < 1 || n > 256 {
		return "", errs.ErrInvalidInput
	}
	return Title(trimmed), nil
}

func (t Title) String() string { return string(t) }
