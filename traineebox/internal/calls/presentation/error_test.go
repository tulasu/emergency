package presentation

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"traineebox/internal/calls/domain/errs"
)

func TestMapCallErrorStatuses(t *testing.T) {
	cases := map[error]int{
		errs.ErrBadSnapshot:  http.StatusBadRequest,
		errs.ErrInvalidInput: http.StatusBadRequest,
		errs.ErrConflict:     http.StatusConflict,
		errs.ErrGone:         http.StatusGone,
		errs.ErrNotFound:     http.StatusNotFound,
		errs.ErrForbidden:    http.StatusForbidden,
	}
	for err, want := range cases {
		if got := statusOf(t, mapCallError(err)); got != want {
			t.Fatalf("%v maps to %d, want %d", err, got, want)
		}
	}
	if got := statusOf(t, mapCallError(fmt.Errorf("originate: %w", errs.ErrConflict))); got != http.StatusConflict {
		t.Fatalf("wrapped conflict maps to %d, want 409", got)
	}
	plain := mapCallError(errors.New(errs.ErrConflict.Error()))
	var se interface{ GetStatus() int }
	if errors.As(plain, &se) && se.GetStatus() == http.StatusConflict {
		t.Fatal("unwrapped lookalike must not map to 409")
	}
}

func statusOf(t *testing.T, err error) int {
	t.Helper()
	var se interface{ GetStatus() int }
	if !errors.As(err, &se) {
		t.Fatalf("error %v has no status", err)
	}
	return se.GetStatus()
}
