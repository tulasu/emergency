package infrastructure_test

import (
	"context"
	"testing"
	"time"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/value_objects"
	"traineebox/internal/auth/infrastructure"
	"traineebox/internal/testkit"

	"github.com/google/uuid"
)

func TestUserRepository(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	repo := infrastructure.NewUserRepository(pool)
	ctx := context.Background()

	user := models.User{
		ID:           uuid.New(),
		Login:        value_objects.Login("alice"),
		PasswordHash: value_objects.PasswordHash("hashvalue"),
		Role:         value_objects.RoleStudent,
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatal(err)
	}

	byLogin, err := repo.FindByLogin(ctx, user.Login)
	if err != nil {
		t.Fatal(err)
	}
	if byLogin.ID != user.ID || byLogin.Role != user.Role {
		t.Fatalf("by login = %+v", byLogin)
	}

	byID, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if byID.Login != user.Login {
		t.Fatalf("by id login = %q", byID.Login)
	}

	err = repo.Create(ctx, models.User{
		ID:           uuid.New(),
		Login:        user.Login,
		PasswordHash: value_objects.PasswordHash("other"),
		Role:         value_objects.RoleTeacher,
		CreatedAt:    time.Now().UTC(),
	})
	if err != errs.ErrConflict {
		t.Fatalf("conflict err = %v, want %v", err, errs.ErrConflict)
	}

	_, err = repo.FindByID(ctx, uuid.New())
	if err != errs.ErrNotFound {
		t.Fatalf("not found err = %v", err)
	}

	if err := repo.SetBlocked(ctx, user.ID, true); err != nil {
		t.Fatal(err)
	}
	blocked, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !blocked.IsBlocked() {
		t.Fatal("expected blocked")
	}

	if err := repo.SetBlocked(ctx, user.ID, false); err != nil {
		t.Fatal(err)
	}
	unblocked, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unblocked.IsBlocked() {
		t.Fatal("expected unblocked")
	}

	if err := repo.SetRole(ctx, user.ID, value_objects.RoleTeacher); err != nil {
		t.Fatal(err)
	}
	changed, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Role != value_objects.RoleTeacher {
		t.Fatalf("role = %q", changed.Role)
	}
}
