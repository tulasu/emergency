package presentation

import "context"

type meOutput struct {
	Body userDTO
}

func (a *API) meHandler(ctx context.Context, in *authHeaderInput) (*meOutput, error) {
	user, err := a.me.Execute(ctx, bearerToken(in.Authorization))
	if err != nil {
		return nil, mapError(err)
	}
	out := &meOutput{}
	out.Body = toUserDTO(user)
	return out, nil
}
