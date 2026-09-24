package repositories

import (
	"context"

	"traineebox/internal/generation/domain/models"
	ticketsvo "traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type JobRepository interface {
	Create(ctx context.Context, job models.Job) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Job, error)
	ListByGroup(ctx context.Context, groupID uuid.UUID) ([]models.Job, error)
	// SaveCAS updates job only if status and version match expected; bumps version.
	SaveCAS(ctx context.Context, job models.Job, expectedStatus string, expectedVersion int) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type GroupMembership interface {
	RoleOf(ctx context.Context, groupID, userID uuid.UUID) (ticketsvo.MemberRole, error)
}
