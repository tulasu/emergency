package presentation

import (
	"context"
	"errors"
	"strings"

	"traineebox/internal/groups/application"
	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"

	"github.com/danielgtaylor/huma/v2"
)

func (a *API) currentUser(ctx context.Context, header string) (application.SessionUser, error) {
	user, err := a.authenticate.CurrentUser(ctx, bearerToken(header))
	if err != nil {
		return application.SessionUser{}, mapError(err)
	}
	return user, nil
}

func (a *API) requireSignedIn(ctx context.Context, header string) (application.SessionUser, error) {
	return a.currentUser(ctx, header)
}

func isAdmin(u application.SessionUser) bool {
	return u.Role == value_objects.AccountRoleAdmin
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return strings.TrimSpace(header)
}

func mapError(err error) error {
	switch {
	case errors.Is(err, errs.ErrNotFound):
		return huma.Error404NotFound("not found")
	case errors.Is(err, errs.ErrConflict):
		return huma.Error409Conflict("conflict")
	case errors.Is(err, errs.ErrInvalidInput):
		return huma.Error400BadRequest("invalid input")
	case errors.Is(err, errs.ErrUnauthorized):
		return huma.Error401Unauthorized("unauthorized")
	case errors.Is(err, errs.ErrForbidden):
		return huma.Error403Forbidden("forbidden")
	case errors.Is(err, errs.ErrUserBlocked):
		return huma.Error403Forbidden("user blocked")
	default:
		return err
	}
}
