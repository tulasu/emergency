package application

import (
	"context"
	"time"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/repositories"
)

type Authenticate struct {
	Users    repositories.UserRepository
	Sessions repositories.SessionRepository
}

func (uc Authenticate) Execute(ctx context.Context, token string) (models.User, error) {
	if token == "" {
		return models.User{}, errs.ErrUnauthorized
	}
	sess, err := uc.Sessions.FindByTokenHash(ctx, hashToken(token))
	if err != nil {
		if err == errs.ErrNotFound {
			return models.User{}, errs.ErrUnauthorized
		}
		return models.User{}, err
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		_ = uc.Sessions.DeleteByTokenHash(ctx, sess.TokenHash)
		return models.User{}, errs.ErrUnauthorized
	}
	user, err := uc.Users.FindByID(ctx, sess.UserID)
	if err != nil {
		return models.User{}, err
	}
	if user.IsBlocked() {
		return models.User{}, errs.ErrUserBlocked
	}
	return user, nil
}
