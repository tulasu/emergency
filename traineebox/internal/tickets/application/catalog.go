package application

import (
	"context"

	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/repositories"

	"github.com/google/uuid"
)

type ListIncidentTypes struct {
	Catalog repositories.CatalogRepository
}

func (uc ListIncidentTypes) Execute(ctx context.Context) ([]models.IncidentType, error) {
	return uc.Catalog.ListIncidentTypes(ctx)
}

type ListTagsByType struct {
	Catalog repositories.CatalogRepository
}

func (uc ListTagsByType) Execute(ctx context.Context, typeID uuid.UUID) ([]models.IncidentTagGroup, error) {
	if _, err := uc.Catalog.FindIncidentTypeByID(ctx, typeID); err != nil {
		return nil, err
	}
	return uc.Catalog.ListTagGroupsByType(ctx, typeID)
}

type ListServices struct {
	Catalog repositories.CatalogRepository
}

func (uc ListServices) Execute(ctx context.Context) ([]models.EmergencyService, error) {
	return uc.Catalog.ListServices(ctx)
}
