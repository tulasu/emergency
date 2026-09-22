package application

import (
	"context"
	"time"

	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/repositories"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

type CreateUser struct {
	Users  repositories.UserRepository
	Hasher PasswordHasher
}

type CreateUserInput struct {
	Login    string
	Password string
	Role     string
}

func (uc CreateUser) Execute(ctx context.Context, in CreateUserInput) (models.User, error) {
	login, err := value_objects.NewLogin(in.Login)
	if err != nil {
		return models.User{}, err
	}
	role, err := value_objects.ParseRole(in.Role)
	if err != nil {
		return models.User{}, err
	}
	hash, err := uc.Hasher.Hash(in.Password)
	if err != nil {
		return models.User{}, err
	}
	user := models.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
	}
	if err := uc.Users.Create(ctx, user); err != nil {
		return models.User{}, err
	}
	return user, nil
}
