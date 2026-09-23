package application

import (
	"context"

	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/repositories"
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

func (uc ListTagsByType) Execute(ctx context.Context, typeCode string) ([]models.IncidentTagGroup, error) {
	if _, err := uc.Catalog.FindIncidentTypeByCode(ctx, typeCode); err != nil {
		return nil, err
	}
	return uc.Catalog.ListTagGroupsByType(ctx, typeCode)
}

type ListServices struct {
	Catalog repositories.CatalogRepository
}

func (uc ListServices) Execute(ctx context.Context) ([]models.EmergencyService, error) {
	return uc.Catalog.ListServices(ctx)
}

type RecommendServices struct {
	Catalog repositories.CatalogRepository
}

type RecommendServicesInput struct {
	TypeCode string
	TagCodes []string
}

func (uc RecommendServices) Execute(ctx context.Context, in RecommendServicesInput) ([]string, error) {
	if _, err := uc.Catalog.FindIncidentTypeByCode(ctx, in.TypeCode); err != nil {
		return nil, err
	}
	return uc.Catalog.RecommendServices(ctx, in.TypeCode, in.TagCodes)
}
