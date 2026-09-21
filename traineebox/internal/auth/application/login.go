package application

import (
	"context"
	"time"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/repository"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

type Login struct {
	Users      repository.UserRepository
	Sessions   repository.SessionRepository
	Hasher     PasswordHasher
	SessionTTL time.Duration
}

type LoginInput struct {
	Login    string
	Password string
}

type LoginResult struct {
	Token string
	User  models.User
}

func (uc Login) Execute(ctx context.Context, in LoginInput) (LoginResult, error) {
	login, err := value_objects.NewLogin(in.Login)
	if err != nil {
		return LoginResult{}, errs.ErrInvalidCreds
	}
	user, err := uc.Users.FindByLogin(ctx, login)
	if err != nil {
		if err == errs.ErrNotFound {
			return LoginResult{}, errs.ErrInvalidCreds
		}
		return LoginResult{}, err
	}
	if user.IsBlocked() {
		return LoginResult{}, errs.ErrUserBlocked
	}
	if !uc.Hasher.Verify(in.Password, user.PasswordHash) {
		return LoginResult{}, errs.ErrInvalidCreds
	}
	token, tokenHash, err := newSessionToken()
	if err != nil {
		return LoginResult{}, err
	}
	now := time.Now().UTC()
	sess := models.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(uc.SessionTTL),
		CreatedAt: now,
	}
	if err := uc.Sessions.Create(ctx, sess); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, User: user}, nil
}
