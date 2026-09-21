package presentation

import (
	"context"

	"traineebox/internal/auth/application"
	"traineebox/internal/auth/domain/value_objects"
)

type createUserInput struct {
	Authorization string `header:"Authorization"`
	Body          struct {
		Login    string `json:"login" minLength:"3" maxLength:"64"`
		Password string `json:"password" minLength:"8" maxLength:"128"`
		Role     string `json:"role" enum:"admin,teacher,student"`
	}
}

type createUserOutput struct {
	Body userDTO
}

func (a *API) createUserHandler(ctx context.Context, in *createUserInput) (*createUserOutput, error) {
	if _, err := a.requireRole(ctx, bearerToken(in.Authorization), value_objects.RoleAdmin); err != nil {
		return nil, err
	}
	user, err := a.createUser.Execute(ctx, application.CreateUserInput{
		Login:    in.Body.Login,
		Password: in.Body.Password,
		Role:     in.Body.Role,
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := &createUserOutput{}
	out.Body = toUserDTO(user)
	return out, nil
}
