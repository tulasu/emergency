package repository

import (
	"context"

	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user models.User) error
	FindByID(ctx context.Context, id uuid.UUID) (models.User, error)
	FindByLogin(ctx context.Context, login value_objects.Login) (models.User, error)
	SetBlocked(ctx context.Context, id uuid.UUID, blocked bool) error
	SetRole(ctx context.Context, id uuid.UUID, role value_objects.Role) error
}
