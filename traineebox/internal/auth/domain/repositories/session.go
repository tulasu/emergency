package repositories

import (
	"context"
	"time"

	"traineebox/internal/auth/domain/models"
)

type SessionRepository interface {
	Create(ctx context.Context, session models.Session) error
	FindByTokenHash(ctx context.Context, tokenHash string) (models.Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteExpired(ctx context.Context, now time.Time) error
}
