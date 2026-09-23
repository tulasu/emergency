package infrastructure

import (
	"context"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
)

// CatalogRepository is an in-memory CatalogRepository backed by YAML.
type CatalogRepository struct {
	c *yamlCatalog
}

func NewCatalogRepository(c *yamlCatalog) *CatalogRepository {
	return &CatalogRepository{c: c}
}

func (r *CatalogRepository) ListIncidentTypes(context.Context) ([]models.IncidentType, error) {
	out := make([]models.IncidentType, len(r.c.types))
	copy(out, r.c.types)
	return out, nil
}

func (r *CatalogRepository) ListTagGroupsByType(_ context.Context, typeCode string) ([]models.IncidentTagGroup, error) {
	lt, ok := r.c.typeBy[typeCode]
	if !ok {
		return nil, errs.ErrNotFound
	}
	out := make([]models.IncidentTagGroup, len(lt.Groups))
	copy(out, lt.Groups)
	return out, nil
}

func (r *CatalogRepository) ListServices(context.Context) ([]models.EmergencyService, error) {
	out := make([]models.EmergencyService, len(r.c.services))
	copy(out, r.c.services)
	return out, nil
}

func (r *CatalogRepository) FindIncidentTypeByCode(_ context.Context, code string) (models.IncidentType, error) {
	lt, ok := r.c.typeBy[code]
	if !ok {
		return models.IncidentType{}, errs.ErrNotFound
	}
	return lt.IncidentType, nil
}

func (r *CatalogRepository) ServiceExists(_ context.Context, codes []string) (bool, error) {
	for _, code := range codes {
		if _, ok := r.c.svcBy[code]; !ok {
			return false, nil
		}
	}
	return true, nil
}

func (r *CatalogRepository) RecommendServices(_ context.Context, typeCode string, tagCodes []string) ([]string, error) {
	return r.c.recommendServices(typeCode, tagCodes), nil
}
