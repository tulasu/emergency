package repositories

import (
	"context"

	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type CatalogRepository interface {
	ListIncidentTypes(ctx context.Context) ([]models.IncidentType, error)
	ListTagGroupsByType(ctx context.Context, typeCode string) ([]models.IncidentTagGroup, error)
	ListServices(ctx context.Context) ([]models.EmergencyService, error)
	FindIncidentTypeByCode(ctx context.Context, code string) (models.IncidentType, error)
	ServiceExists(ctx context.Context, codes []string) (bool, error)
	RecommendServices(ctx context.Context, typeCode string, tagCodes []string) ([]string, error)
}

type TicketRepository interface {
	Create(ctx context.Context, ticket models.Ticket) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Ticket, error)
	ListByGroup(ctx context.Context, groupID uuid.UUID) ([]models.Ticket, error)
	SaveReference(ctx context.Context, ref models.ReferenceAnswer) error
	FindReference(ctx context.Context, ticketID uuid.UUID) (models.ReferenceAnswer, error)
}

type AttemptRepository interface {
	Create(ctx context.Context, attempt models.Attempt) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Attempt, error)
	FindInProgress(ctx context.Context, ticketID, userID uuid.UUID) (models.Attempt, error)
	ListByTicketUser(ctx context.Context, ticketID, userID uuid.UUID) ([]models.Attempt, error)
	CountFinished(ctx context.Context, ticketID, userID uuid.UUID) (int, error)
	NextAttemptNo(ctx context.Context, ticketID, userID uuid.UUID) (int, error)
	Save(ctx context.Context, attempt models.Attempt) error
}

type GroupMembership interface {
	RoleOf(ctx context.Context, groupID, userID uuid.UUID) (value_objects.MemberRole, error)
}
