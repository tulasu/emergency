package application

import (
	"context"

	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/repositories"

	"github.com/google/uuid"
)

type GetUser struct {
	Users repositories.UserRepository
}

func (uc GetUser) Execute(ctx context.Context, id uuid.UUID) (models.User, error) {
	return uc.Users.FindByID(ctx, id)
}
