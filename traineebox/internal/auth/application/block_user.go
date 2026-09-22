package application

import (
	"context"

	"traineebox/internal/auth/domain/repositories"

	"github.com/google/uuid"
)

type BlockUser struct {
	Users repositories.UserRepository
}

func (uc BlockUser) Execute(ctx context.Context, id uuid.UUID, blocked bool) error {
	return uc.Users.SetBlocked(ctx, id, blocked)
}
