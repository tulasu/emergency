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

func TestSessionRepository(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	users := infrastructure.NewUserRepository(pool)
	sessions := infrastructure.NewSessionRepository(pool)
	ctx := context.Background()

	user := models.User{
		ID:           uuid.New(),
		Login:        value_objects.Login("bob"),
		PasswordHash: value_objects.PasswordHash("hash"),
		Role:         value_objects.RoleAdmin,
		CreatedAt:    time.Now().UTC(),
	}
	if err := users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	sess := models.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: "token-hash-1",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	if err := sessions.Create(ctx, sess); err != nil {
		t.Fatal(err)
	}

	got, err := sessions.FindByTokenHash(ctx, sess.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != sess.ID || got.UserID != user.ID {
		t.Fatalf("got = %+v", got)
	}

	_, err = sessions.FindByTokenHash(ctx, "missing")
	if err != errs.ErrNotFound {
		t.Fatalf("err = %v", err)
	}

	expired := models.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: "token-hash-expired",
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now.Add(-time.Hour),
	}
	if err := sessions.Create(ctx, expired); err != nil {
		t.Fatal(err)
	}
	if err := sessions.DeleteExpired(ctx, now); err != nil {
		t.Fatal(err)
	}
	_, err = sessions.FindByTokenHash(ctx, expired.TokenHash)
	if err != errs.ErrNotFound {
		t.Fatalf("expired should be gone: %v", err)
	}
	if _, err := sessions.FindByTokenHash(ctx, sess.TokenHash); err != nil {
		t.Fatalf("active session should remain: %v", err)
	}

	if err := sessions.DeleteByTokenHash(ctx, sess.TokenHash); err != nil {
		t.Fatal(err)
	}
	_, err = sessions.FindByTokenHash(ctx, sess.TokenHash)
	if err != errs.ErrNotFound {
		t.Fatalf("deleted err = %v", err)
	}
}
