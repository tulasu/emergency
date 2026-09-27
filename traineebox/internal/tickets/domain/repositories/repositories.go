package repositories

import (
	"context"

	"traineebox/internal/tickets/domain/models"

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
	ListByVariant(ctx context.Context, variantID uuid.UUID) ([]models.Ticket, error)
	ListLibrary(ctx context.Context, q string) ([]models.Ticket, error)
	Update(ctx context.Context, ticket models.Ticket) error
	Delete(ctx context.Context, id uuid.UUID) error
	SaveReference(ctx context.Context, ref models.ReferenceAnswer) error
	FindReference(ctx context.Context, ticketID uuid.UUID) (models.ReferenceAnswer, error)
	CreateWithReference(ctx context.Context, ticket models.Ticket, ref models.ReferenceAnswer) error
}

type AttemptRepository interface {
	Create(ctx context.Context, attempt models.Attempt) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Attempt, error)
	FindOpen(ctx context.Context, variantID, userID uuid.UUID) (models.Attempt, error)
	ListByVariantUser(ctx context.Context, variantID, userID uuid.UUID) ([]models.Attempt, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Attempt, error)
	HasAny(ctx context.Context, variantID, userID uuid.UUID) (bool, error)
	NextAttemptNo(ctx context.Context, variantID, userID uuid.UUID) (int, error)
	Save(ctx context.Context, attempt models.Attempt) error
}

type LessonClock interface {
	DurationSecondsByVariant(ctx context.Context, variantID uuid.UUID) (*int, error)
}

type TopicExists interface {
	TopicExists(ctx context.Context, topicID uuid.UUID) error
	VariantExists(ctx context.Context, variantID uuid.UUID) error
}
