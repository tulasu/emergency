package application

import (
	"context"

	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type SessionUser struct {
	ID   uuid.UUID
	Role value_objects.AccountRole
}

type Authenticator interface {
	CurrentUser(ctx context.Context, token string) (SessionUser, error)
}
