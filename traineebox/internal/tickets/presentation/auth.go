package presentation

import (
	"context"
	"errors"
	"strings"

	"traineebox/internal/tickets/application"
	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/danielgtaylor/huma/v2"
)

func (a *API) requireSignedIn(ctx context.Context, header string) (application.SessionUser, error) {
	user, err := a.authenticate.CurrentUser(ctx, bearerToken(header))
	if err != nil {
		return application.SessionUser{}, mapError(err)
	}
	return user, nil
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
	case errors.Is(err, errs.ErrConflict), errors.Is(err, errs.ErrAttemptInProgress):
		return huma.Error409Conflict("conflict")
	case errors.Is(err, errs.ErrInvalidInput), errors.Is(err, errs.ErrInvalidTags), errors.Is(err, errs.ErrInvalidTagSelection):
		return huma.Error400BadRequest("invalid input")
	case errors.Is(err, errs.ErrUnauthorized):
		return huma.Error401Unauthorized("unauthorized")
	case errors.Is(err, errs.ErrForbidden), errors.Is(err, errs.ErrUserBlocked):
		return huma.Error403Forbidden("forbidden")
	case errors.Is(err, errs.ErrUnavailable), errors.Is(err, errs.ErrAttemptsExhausted):
		return huma.Error403Forbidden("forbidden")
	case errors.Is(err, errs.ErrAttemptNotActive):
		return huma.Error409Conflict("attempt not active")
	case errors.Is(err, errs.ErrNoReferenceAnswer):
		return huma.Error409Conflict("no reference answer")
	default:
		return err
	}
}
