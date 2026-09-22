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

func TestAuthenticate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	userID := uuid.New()
	user := models.User{
		ID:           userID,
		Login:        value_objects.Login("alice"),
		PasswordHash: value_objects.PasswordHash("hash"),
		Role:         value_objects.RoleStudent,
		CreatedAt:    time.Now().UTC(),
	}
	blockedAt := time.Now().UTC()
	blockedID := uuid.New()
	blocked := user
	blocked.ID = blockedID
	blocked.Login = value_objects.Login("blocked")
	blocked.BlockedAt = &blockedAt

	token := "raw-session-token"
	tokenHash := hashToken(token)
	expiredToken := "expired-token"
	expiredHash := hashToken(expiredToken)

	now := time.Now().UTC()
	sessions := newFakeSessions(
		models.Session{
			ID:        uuid.New(),
			UserID:    userID,
			TokenHash: tokenHash,
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		},
		models.Session{
			ID:        uuid.New(),
			UserID:    userID,
			TokenHash: expiredHash,
			ExpiresAt: now.Add(-time.Minute),
			CreatedAt: now.Add(-time.Hour),
		},
		models.Session{
			ID:        uuid.New(),
			UserID:    blockedID,
			TokenHash: hashToken("blocked-token"),
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		},
	)
	users := newFakeUsers(user, blocked)
	uc := Authenticate{Users: users, Sessions: sessions}

	t.Run("empty token", func(t *testing.T) {
		_, err := uc.Execute(ctx, "")
		if err != errs.ErrUnauthorized {
			t.Fatalf("err = %v, want %v", err, errs.ErrUnauthorized)
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		_, err := uc.Execute(ctx, "missing")
		if err != errs.ErrUnauthorized {
			t.Fatalf("err = %v, want %v", err, errs.ErrUnauthorized)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		_, err := uc.Execute(ctx, expiredToken)
		if err != errs.ErrUnauthorized {
			t.Fatalf("err = %v, want %v", err, errs.ErrUnauthorized)
		}
		if len(sessions.deleted) == 0 || sessions.deleted[len(sessions.deleted)-1] != expiredHash {
			t.Fatal("expected expired session deleted")
		}
	})

	t.Run("blocked user", func(t *testing.T) {
		_, err := uc.Execute(ctx, "blocked-token")
		if err != errs.ErrUserBlocked {
			t.Fatalf("err = %v, want %v", err, errs.ErrUserBlocked)
		}
	})

	t.Run("success", func(t *testing.T) {
		got, err := uc.Execute(ctx, token)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != userID {
			t.Fatalf("id = %s, want %s", got.ID, userID)
		}
	})
}
