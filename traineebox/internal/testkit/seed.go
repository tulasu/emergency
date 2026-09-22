package testkit

import (
	"context"
	"testing"
	"time"

	"traineebox/internal/auth/application"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/value_objects"
	authinfra "traineebox/internal/auth/infrastructure"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SeedUser(t *testing.T, pool *pgxpool.Pool, login, password string, role value_objects.Role) models.User {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uc := application.CreateUser{
		Users:  authinfra.NewUserRepository(pool),
		Hasher: application.PasswordHasher{},
	}
	user, err := uc.Execute(ctx, application.CreateUserInput{
		Login:    login,
		Password: password,
		Role:     string(role),
	})
	if err != nil {
		t.Fatalf("seed user %q: %v", login, err)
	}
	return user
}

func SetBlocked(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, blocked bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := authinfra.NewUserRepository(pool).SetBlocked(ctx, userID, blocked); err != nil {
		t.Fatalf("set blocked: %v", err)
	}
}
