package presentation

import (
	"context"

	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

type blockUserInput struct {
	Authorization string    `header:"Authorization"`
	UserID        uuid.UUID `path:"userId"`
	Body          struct {
		Blocked bool `json:"blocked"`
	}
}

func (a *API) blockUserHandler(ctx context.Context, in *blockUserInput) (*emptyOutput, error) {
	if _, err := a.requireRole(ctx, bearerToken(in.Authorization), value_objects.RoleAdmin); err != nil {
		return nil, err
	}
	if err := a.blockUser.Execute(ctx, in.UserID, in.Body.Blocked); err != nil {
		return nil, mapError(err)
	}
	return &emptyOutput{}, nil
}
