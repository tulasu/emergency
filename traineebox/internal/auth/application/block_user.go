package application

import (
	"context"

	"traineebox/internal/auth/domain/repository"

	"github.com/google/uuid"
)

type BlockUser struct {
	Users repository.UserRepository
}

func (uc BlockUser) Execute(ctx context.Context, id uuid.UUID, blocked bool) error {
	return uc.Users.SetBlocked(ctx, id, blocked)
}
