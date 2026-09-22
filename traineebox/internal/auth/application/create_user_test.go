package application

import (
	"context"
	"testing"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/value_objects"
)

func TestCreateUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	users := newFakeUsers()
	uc := CreateUser{Users: users, Hasher: PasswordHasher{}}

	t.Run("invalid login", func(t *testing.T) {
		_, err := uc.Execute(ctx, CreateUserInput{Login: "ab", Password: "password1", Role: "student"})
		if err != errs.ErrInvalidInput {
			t.Fatalf("err = %v, want %v", err, errs.ErrInvalidInput)
		}
	})

	t.Run("invalid role", func(t *testing.T) {
		_, err := uc.Execute(ctx, CreateUserInput{Login: "alice", Password: "password1", Role: "guest"})
		if err != errs.ErrInvalidInput {
			t.Fatalf("err = %v, want %v", err, errs.ErrInvalidInput)
		}
	})

	t.Run("short password", func(t *testing.T) {
		_, err := uc.Execute(ctx, CreateUserInput{Login: "alice", Password: "short", Role: "student"})
		if err != errs.ErrInvalidInput {
			t.Fatalf("err = %v, want %v", err, errs.ErrInvalidInput)
		}
	})

	t.Run("success and conflict", func(t *testing.T) {
		user, err := uc.Execute(ctx, CreateUserInput{Login: "alice", Password: "password1", Role: "teacher"})
		if err != nil {
			t.Fatal(err)
		}
		if user.Role != value_objects.RoleTeacher {
			t.Fatalf("role = %q", user.Role)
		}
		_, err = uc.Execute(ctx, CreateUserInput{Login: "alice", Password: "password1", Role: "student"})
		if err != errs.ErrConflict {
			t.Fatalf("err = %v, want %v", err, errs.ErrConflict)
		}
	})
}
