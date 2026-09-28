package application

import (
	"context"

	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/repositories"
)

type ListUsers struct {
	Users repositories.UserRepository
}

func (uc ListUsers) Execute(ctx context.Context) ([]models.User, error) {
	return uc.Users.List(ctx)
}
