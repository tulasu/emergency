package value_objects_test

import (
	"testing"

	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"
)

func TestParseMemberRole(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    value_objects.MemberRole
		wantErr error
	}{
		{in: "owner", want: value_objects.MemberRoleOwner},
		{in: "teacher", want: value_objects.MemberRoleTeacher},
		{in: "student", want: value_objects.MemberRoleStudent},
		{in: "admin", wantErr: errs.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got, err := value_objects.ParseMemberRole(tt.in)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("err = %v", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q err %v", got, err)
			}
		})
	}
}

func TestNewGroupName(t *testing.T) {
	t.Parallel()
	if _, err := value_objects.NewGroupName("  "); err != errs.ErrInvalidInput {
		t.Fatalf("empty err = %v", err)
	}
	got, err := value_objects.NewGroupName("  Alpha  ")
	if err != nil || got.String() != "Alpha" {
		t.Fatalf("got %q err %v", got, err)
	}
}
