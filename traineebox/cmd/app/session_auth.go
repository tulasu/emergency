package main

import (
	"context"
	"errors"

	authapp "traineebox/internal/auth/application"
	autherrs "traineebox/internal/auth/domain/errs"
	groupsapp "traineebox/internal/groups/application"
	groupserrs "traineebox/internal/groups/domain/errs"
	groupsvo "traineebox/internal/groups/domain/value_objects"
	ticketsapp "traineebox/internal/tickets/application"
	ticketserrs "traineebox/internal/tickets/domain/errs"
	ticketsvo "traineebox/internal/tickets/domain/value_objects"
)

type groupsSessionAuthenticator struct {
	auth authapp.Authenticate
}

func (a groupsSessionAuthenticator) CurrentUser(ctx context.Context, token string) (groupsapp.SessionUser, error) {
	user, err := a.auth.Execute(ctx, token)
	if err != nil {
		return groupsapp.SessionUser{}, mapGroupsAuthError(err)
	}
	role, err := groupsvo.ParseAccountRole(string(user.Role))
	if err != nil {
		return groupsapp.SessionUser{}, err
	}
	return groupsapp.SessionUser{ID: user.ID, Role: role}, nil
}

type ticketsSessionAuthenticator struct {
	auth authapp.Authenticate
}

func (a ticketsSessionAuthenticator) CurrentUser(ctx context.Context, token string) (ticketsapp.SessionUser, error) {
	user, err := a.auth.Execute(ctx, token)
	if err != nil {
		return ticketsapp.SessionUser{}, mapTicketsAuthError(err)
	}
	role, err := ticketsvo.ParseAccountRole(string(user.Role))
	if err != nil {
		return ticketsapp.SessionUser{}, err
	}
	return ticketsapp.SessionUser{ID: user.ID, Role: role}, nil
}

func mapGroupsAuthError(err error) error {
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

func mapTicketsAuthError(err error) error {
	switch {
	case errors.Is(err, autherrs.ErrUnauthorized):
		return ticketserrs.ErrUnauthorized
	case errors.Is(err, autherrs.ErrUserBlocked):
		return ticketserrs.ErrUserBlocked
	case errors.Is(err, autherrs.ErrNotFound):
		return ticketserrs.ErrNotFound
	case errors.Is(err, autherrs.ErrForbidden):
		return ticketserrs.ErrForbidden
	default:
		return err
	}
}
