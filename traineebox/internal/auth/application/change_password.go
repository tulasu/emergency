package application

import (
	"context"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/repositories"

	"github.com/google/uuid"
)

type ChangePassword struct {
	Users  repositories.UserRepository
	Hasher PasswordHasher
}

func (uc ChangePassword) Execute(ctx context.Context, userID uuid.UUID, current, next string) error {
	user, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !uc.Hasher.Verify(current, user.PasswordHash) {
		return errs.ErrInvalidCreds
	}
	hash, err := uc.Hasher.Hash(next)
	if err != nil {
		return err
	}
	return uc.Users.SetPasswordHash(ctx, userID, hash)
}
