package value_objects

import (
	"strings"
	"unicode/utf8"

	"traineebox/internal/tickets/domain/errs"
)

type Notes string

const MaxNotesLength = 1000

func NewNotes(s string) (Notes, error) {
	if utf8.RuneCountInString(s) > MaxNotesLength {
		return "", errs.ErrInvalidInput
	}
	return Notes(s), nil
}

func (n Notes) String() string { return string(n) }

func (n Notes) Trimmed() string { return strings.TrimSpace(string(n)) }
