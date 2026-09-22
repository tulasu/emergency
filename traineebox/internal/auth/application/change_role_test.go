package application

import (
	"context"
	"testing"
	"time"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

func TestChangeRole(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := uuid.New()
	users := newFakeUsers(models.User{
		ID:           id,
		Login:        value_objects.Login("alice"),
		PasswordHash: value_objects.PasswordHash("hash"),
		Role:         value_objects.RoleStudent,
		CreatedAt:    time.Now().UTC(),
	})
	uc := ChangeRole{Users: users}

	t.Run("invalid role", func(t *testing.T) {
		err := uc.Execute(ctx, id, "guest")
		if err != errs.ErrInvalidInput {
			t.Fatalf("err = %v, want %v", err, errs.ErrInvalidInput)
		}
	})

	t.Run("success", func(t *testing.T) {
		if err := uc.Execute(ctx, id, "teacher"); err != nil {
			t.Fatal(err)
		}
		u, err := users.FindByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if u.Role != value_objects.RoleTeacher {
			t.Fatalf("role = %q, want teacher", u.Role)
		}
	})
}
