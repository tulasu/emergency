package value_objects_test

import (
	"testing"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/value_objects"
)

func TestParseRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    value_objects.Role
		wantErr error
	}{
		{name: "admin", input: "admin", want: value_objects.RoleAdmin},
		{name: "teacher", input: "teacher", want: value_objects.RoleTeacher},
		{name: "student", input: "student", want: value_objects.RoleStudent},
		{name: "empty", input: "", wantErr: errs.ErrInvalidInput},
		{name: "unknown", input: "guest", wantErr: errs.ErrInvalidInput},
		{name: "case sensitive", input: "Admin", wantErr: errs.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := value_objects.ParseRole(tt.input)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Fatalf("role = %q, want %q", got, tt.want)
			}
		})
	}
}
