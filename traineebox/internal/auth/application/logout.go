package application

import (
	"context"

	"traineebox/internal/auth/domain/repository"
)

type Logout struct {
	Sessions repository.SessionRepository
}

func (uc Logout) Execute(ctx context.Context, token string) error {
	return uc.Sessions.DeleteByTokenHash(ctx, hashToken(token))
}
