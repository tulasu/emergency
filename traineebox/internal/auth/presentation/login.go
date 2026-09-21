package presentation

import (
	"context"

	"traineebox/internal/auth/application"
)

type loginInput struct {
	Body struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
}

type loginOutput struct {
	Body struct {
		Token string  `json:"token"`
		User  userDTO `json:"user"`
	}
}

func (a *API) loginHandler(ctx context.Context, in *loginInput) (*loginOutput, error) {
	res, err := a.login.Execute(ctx, application.LoginInput{
		Login:    in.Body.Login,
		Password: in.Body.Password,
	})
	if err != nil {
		return nil, mapError(err)
	}
	out := &loginOutput{}
	out.Body.Token = res.Token
	out.Body.User = toUserDTO(res.User)
	return out, nil
}
