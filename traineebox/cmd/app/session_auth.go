package main

import (
	"context"
	"errors"

	authapp "traineebox/internal/auth/application"
	autherrs "traineebox/internal/auth/domain/errs"
	groupsapp "traineebox/internal/groups/application"
	groupserrs "traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"
)

type sessionAuthenticator struct {
	auth authapp.Authenticate
}

func (a sessionAuthenticator) CurrentUser(ctx context.Context, token string) (groupsapp.SessionUser, error) {
	user, err := a.auth.Execute(ctx, token)
	if err != nil {
		return groupsapp.SessionUser{}, mapAuthError(err)
	}
	role, err := value_objects.ParseAccountRole(string(user.Role))
	if err != nil {
		return groupsapp.SessionUser{}, err
	}
	return groupsapp.SessionUser{ID: user.ID, Role: role}, nil
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, autherrs.ErrUnauthorized):
		return groupserrs.ErrUnauthorized
	case errors.Is(err, autherrs.ErrUserBlocked):
		return groupserrs.ErrUserBlocked
	case errors.Is(err, autherrs.ErrNotFound):
		return groupserrs.ErrNotFound
	case errors.Is(err, autherrs.ErrForbidden):
		return groupserrs.ErrForbidden
	default:
		return err
	}
}
