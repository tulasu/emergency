package value_objects

import (
	"strings"
	"unicode/utf8"

	"traineebox/internal/tickets/domain/errs"
)

type TicketTitle string

const (
	MinTicketTitleLength = 1
	MaxTicketTitleLength = 256
)

func NewTicketTitle(s string) (TicketTitle, error) {
	trimmed := strings.TrimSpace(s)
	n := utf8.RuneCountInString(trimmed)
	if n < MinTicketTitleLength || n > MaxTicketTitleLength {
		return "", errs.ErrInvalidInput
	}
	return TicketTitle(trimmed), nil
}

func (t TicketTitle) String() string { return string(t) }
