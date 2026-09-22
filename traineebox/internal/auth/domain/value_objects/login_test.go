package value_objects_test

import (
	"strings"
	"testing"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/value_objects"
)

func TestNewLogin(t *testing.T) {
	t.Parallel()

	valid64 := strings.Repeat("a", 64)
	invalid65 := strings.Repeat("a", 65)

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "min length", input: "abc"},
		{name: "with underscore and dash", input: "user_name-1"},
		{name: "with dot", input: "user.name"},
		{name: "max length", input: valid64},
		{name: "too short", input: "ab", wantErr: errs.ErrInvalidInput},
		{name: "empty", input: "", wantErr: errs.ErrInvalidInput},
		{name: "too long", input: invalid65, wantErr: errs.ErrInvalidInput},
		{name: "space", input: "ab c", wantErr: errs.ErrInvalidInput},
		{name: "special char", input: "ab@c", wantErr: errs.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := value_objects.NewLogin(tt.input)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.String() != tt.input {
				t.Fatalf("login = %q, want %q", got, tt.input)
			}
		})
	}
}
