package presentation

import "context"

func (a *API) logoutHandler(ctx context.Context, in *authHeaderInput) (*emptyOutput, error) {
	if err := a.logout.Execute(ctx, bearerToken(in.Authorization)); err != nil {
		return nil, mapError(err)
	}
	return &emptyOutput{}, nil
}
