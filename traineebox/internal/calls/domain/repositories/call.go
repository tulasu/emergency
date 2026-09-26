package repositories

import (
	"context"
	"time"

	"traineebox/internal/calls/domain/models"

	"github.com/google/uuid"
)

type CallRepository interface {
	ActiveForAttempt(ctx context.Context, attemptID uuid.UUID) (models.Call, bool, error)
	Create(ctx context.Context, c models.Call) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Call, error)
	ListByAttempt(ctx context.Context, attemptID uuid.UUID) ([]models.Call, error)
	ListExpired(ctx context.Context, now time.Time) ([]models.Call, error)
	SetChannelID(ctx context.Context, id uuid.UUID, channelID string) error
	SetStatus(ctx context.Context, id uuid.UUID, status string) error
	SaveTurns(ctx context.Context, callID uuid.UUID, turns []models.Turn) error
}
