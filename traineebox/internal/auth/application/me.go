package application

import (
	"context"

	"traineebox/internal/auth/domain/models"
)

type Me struct {
	Authenticate Authenticate
}

func (uc Me) Execute(ctx context.Context, token string) (models.User, error) {
	return uc.Authenticate.Execute(ctx, token)
}
