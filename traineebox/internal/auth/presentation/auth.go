package presentation

import (
	"context"
	"errors"
	"slices"
	"strings"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/danielgtaylor/huma/v2"
)

func (a *API) requireRole(ctx context.Context, token string, roles ...value_objects.Role) (models.User, error) {
	user, err := a.authenticate.Execute(ctx, token)
	if err != nil {
		return models.User{}, mapError(err)
	}
	if slices.Contains(roles, user.Role) {
		return user, nil
	}
	return models.User{}, huma.Error403Forbidden(codeForbidden)
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return strings.TrimSpace(header)
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, errs.ErrNotFound):
		return codeNotFound
	case errors.Is(err, errs.ErrConflict):
		return codeConflict
	case errors.Is(err, errs.ErrInvalidInput):
		return codeInvalidInput
	case errors.Is(err, errs.ErrInvalidCreds):
		return codeInvalidCredentials
	case errors.Is(err, errs.ErrUnauthorized):
		return codeUnauthorized
	case errors.Is(err, errs.ErrForbidden):
		return codeForbidden
	case errors.Is(err, errs.ErrUserBlocked):
		return codeUserBlocked
	default:
		return codeFailed
	}
}

func mapError(err error) error {
	code := errorCode(err)
	switch {
	case errors.Is(err, errs.ErrNotFound):
		return huma.Error404NotFound(code)
	case errors.Is(err, errs.ErrConflict):
		return huma.Error409Conflict(code)
	case errors.Is(err, errs.ErrInvalidInput):
		return huma.Error400BadRequest(code)
	case errors.Is(err, errs.ErrInvalidCreds):
		return huma.Error401Unauthorized(code)
	case errors.Is(err, errs.ErrUnauthorized):
		return huma.Error401Unauthorized(code)
	case errors.Is(err, errs.ErrForbidden):
		return huma.Error403Forbidden(code)
	case errors.Is(err, errs.ErrUserBlocked):
		return huma.Error403Forbidden(code)
	default:
		return err
	}
}
