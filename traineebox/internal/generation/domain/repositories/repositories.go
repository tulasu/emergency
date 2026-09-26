package repositories

import (
	"context"
	"time"

	"traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/value_objects"

	"github.com/google/uuid"
)

type JobRepository interface {
	Create(ctx context.Context, job models.Job) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Job, error)
	ListByGroup(ctx context.Context, groupID uuid.UUID) ([]models.Job, error)
	// SaveCAS updates job only if status and version match expected; bumps version.
	SaveCAS(ctx context.Context, job models.Job, expectedStatus string, expectedVersion int) error
	Delete(ctx context.Context, id uuid.UUID) error
	ClaimNext(ctx context.Context, workerID string, leaseSeconds int) (models.Job, bool, error)
}

type GroupMembership interface {
	RoleOf(ctx context.Context, groupID, userID uuid.UUID) (value_objects.MemberRole, error)
}

type Catalog interface {
	IncidentTypeExists(ctx context.Context, code string) error
	ValidateTags(ctx context.Context, incidentType string, tags []string) error
	ServicesExist(ctx context.Context, codes []string) (bool, error)
}

type PublishDraft struct {
	TicketID        uuid.UUID
	GroupID         uuid.UUID
	Title           string
	Body            string
	CreatedBy       uuid.UUID
	CreatedAt       time.Time
	ScenarioJSON    string
	ScenarioVersion string
	Mode            string
	Briefing        string
	Reference       models.DraftReference
	JobID           uuid.UUID
	ExpectedStatus  string
	ExpectedVersion int
}

type TicketPublisher interface {
	Publish(ctx context.Context, draft PublishDraft) (uuid.UUID, error)
}
