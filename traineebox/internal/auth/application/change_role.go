package application

import (
	"context"

	"traineebox/internal/auth/domain/repositories"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

type ChangeRole struct {
	Users repositories.UserRepository
}

func (uc ChangeRole) Execute(ctx context.Context, id uuid.UUID, role string) error {
	r, err := value_objects.ParseRole(role)
	if err != nil {
		return err
	}
	return uc.Users.SetRole(ctx, id, r)
}
