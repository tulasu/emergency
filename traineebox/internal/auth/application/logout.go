package application

import (
	"context"

	"traineebox/internal/auth/domain/repositories"
)

type Logout struct {
	Sessions repositories.SessionRepository
}

func (uc Logout) Execute(ctx context.Context, token string) error {
	return uc.Sessions.DeleteByTokenHash(ctx, hashToken(token))
}
