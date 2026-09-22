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

func TestLogin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	hasher := PasswordHasher{}
	hash, err := hasher.Hash("password1")
	if err != nil {
		t.Fatal(err)
	}

	user := models.User{
		ID:           uuid.New(),
		Login:        value_objects.Login("alice"),
		PasswordHash: hash,
		Role:         value_objects.RoleStudent,
		CreatedAt:    time.Now().UTC(),
	}
	blockedAt := time.Now().UTC()
	blocked := user
	blocked.ID = uuid.New()
	blocked.Login = value_objects.Login("blocked")
	blocked.BlockedAt = &blockedAt

	users := newFakeUsers(user, blocked)
	sessions := newFakeSessions()
	uc := Login{
		Users:      users,
		Sessions:   sessions,
		Hasher:     hasher,
		SessionTTL: time.Hour,
	}

	t.Run("invalid login format", func(t *testing.T) {
		_, err := uc.Execute(ctx, LoginInput{Login: "ab", Password: "password1"})
		if err != errs.ErrInvalidCreds {
			t.Fatalf("err = %v, want %v", err, errs.ErrInvalidCreds)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		_, err := uc.Execute(ctx, LoginInput{Login: "nobody", Password: "password1"})
		if err != errs.ErrInvalidCreds {
			t.Fatalf("err = %v, want %v", err, errs.ErrInvalidCreds)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := uc.Execute(ctx, LoginInput{Login: "alice", Password: "wrongpass"})
		if err != errs.ErrInvalidCreds {
			t.Fatalf("err = %v, want %v", err, errs.ErrInvalidCreds)
		}
	})

	t.Run("blocked user", func(t *testing.T) {
		_, err := uc.Execute(ctx, LoginInput{Login: "blocked", Password: "password1"})
		if err != errs.ErrUserBlocked {
			t.Fatalf("err = %v, want %v", err, errs.ErrUserBlocked)
		}
	})

	t.Run("success", func(t *testing.T) {
		res, err := uc.Execute(ctx, LoginInput{Login: "alice", Password: "password1"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Token == "" {
			t.Fatal("expected token")
		}
		if res.User.ID != user.ID {
			t.Fatalf("user id = %s, want %s", res.User.ID, user.ID)
		}
		if len(sessions.created) == 0 {
			t.Fatal("expected session created")
		}
		created := sessions.created[len(sessions.created)-1]
		if created.UserID != user.ID {
			t.Fatalf("session user = %s, want %s", created.UserID, user.ID)
		}
		if !created.ExpiresAt.After(time.Now().UTC()) {
			t.Fatal("expected future expiry")
		}
		if created.TokenHash == res.Token {
			t.Fatal("token hash must not equal raw token")
		}
	})
}
