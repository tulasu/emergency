package presentation

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"traineebox/internal/auth/application"
	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

func TestBearerToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "bearer", input: "Bearer abc.def", want: "abc.def"},
		{name: "raw", input: "raw-token", want: "raw-token"},
		{name: "empty", input: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := bearerToken(tt.input); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		code int
	}{
		{name: "not found", err: errs.ErrNotFound, code: 404},
		{name: "conflict", err: errs.ErrConflict, code: 409},
		{name: "invalid input", err: errs.ErrInvalidInput, code: 400},
		{name: "invalid creds", err: errs.ErrInvalidCreds, code: 401},
		{name: "unauthorized", err: errs.ErrUnauthorized, code: 401},
		{name: "forbidden", err: errs.ErrForbidden, code: 403},
		{name: "blocked", err: errs.ErrUserBlocked, code: 403},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mapped := mapError(tt.err)
			var se huma.StatusError
			if !errors.As(mapped, &se) {
				t.Fatalf("expected StatusError, got %T %v", mapped, mapped)
			}
			if se.GetStatus() != tt.code {
				t.Fatalf("status = %d, want %d", se.GetStatus(), tt.code)
			}
		})
	}

	t.Run("unknown passthrough", func(t *testing.T) {
		t.Parallel()
		orig := errors.New("boom")
		if mapError(orig) != orig {
			t.Fatal("expected passthrough")
		}
	})
}

func TestRequireRole(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	adminID := uuid.New()
	teacherID := uuid.New()
	now := time.Now().UTC()

	users := &memUsers{byID: map[uuid.UUID]models.User{
		adminID: {
			ID:    adminID,
			Login: value_objects.Login("admin"),
			Role:  value_objects.RoleAdmin,
		},
		teacherID: {
			ID:    teacherID,
			Login: value_objects.Login("teacher"),
			Role:  value_objects.RoleTeacher,
		},
	}}
	sessions := &memSessions{byHash: map[string]models.Session{
		tokenHash("admin-tok"): {
			ID: uuid.New(), UserID: adminID, TokenHash: tokenHash("admin-tok"),
			ExpiresAt: now.Add(time.Hour), CreatedAt: now,
		},
		tokenHash("teacher-tok"): {
			ID: uuid.New(), UserID: teacherID, TokenHash: tokenHash("teacher-tok"),
			ExpiresAt: now.Add(time.Hour), CreatedAt: now,
		},
	}}

	api := NewAPI(Deps{
		Authenticate: application.Authenticate{Users: users, Sessions: sessions},
	})

	t.Run("admin ok", func(t *testing.T) {
		u, err := api.requireRole(ctx, "admin-tok", value_objects.RoleAdmin)
		if err != nil {
			t.Fatal(err)
		}
		if u.ID != adminID {
			t.Fatalf("id = %s", u.ID)
		}
	})

	t.Run("teacher forbidden", func(t *testing.T) {
		_, err := api.requireRole(ctx, "teacher-tok", value_objects.RoleAdmin)
		var se huma.StatusError
		if !errors.As(err, &se) || se.GetStatus() != 403 {
			t.Fatalf("err = %v, want 403", err)
		}
	})
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

type memUsers struct {
	byID map[uuid.UUID]models.User
}

func (m *memUsers) Create(context.Context, models.User) error { return nil }
func (m *memUsers) FindByID(_ context.Context, id uuid.UUID) (models.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return models.User{}, errs.ErrNotFound
	}
	return u, nil
}
func (m *memUsers) FindByLogin(context.Context, value_objects.Login) (models.User, error) {
	return models.User{}, errs.ErrNotFound
}
func (m *memUsers) SetBlocked(context.Context, uuid.UUID, bool) error { return nil }
func (m *memUsers) SetRole(context.Context, uuid.UUID, value_objects.Role) error {
	return nil
}

type memSessions struct {
	byHash map[string]models.Session
}

func (m *memSessions) Create(context.Context, models.Session) error { return nil }
func (m *memSessions) FindByTokenHash(_ context.Context, tokenHash string) (models.Session, error) {
	s, ok := m.byHash[tokenHash]
	if !ok {
		return models.Session{}, errs.ErrNotFound
	}
	return s, nil
}
func (m *memSessions) DeleteByTokenHash(context.Context, string) error { return nil }
func (m *memSessions) DeleteExpired(context.Context, time.Time) error  { return nil }
