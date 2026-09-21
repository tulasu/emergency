package presentation

import (
	"context"

	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

type changeRoleInput struct {
	Authorization string    `header:"Authorization"`
	UserID        uuid.UUID `path:"userId"`
	Body          struct {
		Role string `json:"role" enum:"admin,teacher,student"`
	}
}

func (a *API) changeRoleHandler(ctx context.Context, in *changeRoleInput) (*emptyOutput, error) {
	if _, err := a.requireRole(ctx, bearerToken(in.Authorization), value_objects.RoleAdmin); err != nil {
		return nil, err
	}
	if err := a.changeRole.Execute(ctx, in.UserID, in.Body.Role); err != nil {
		return nil, mapError(err)
	}
	return &emptyOutput{}, nil
}
