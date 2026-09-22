package infrastructure

import (
	"context"
	"errors"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/infrastructure/ticketssql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CatalogRepository struct {
	q *ticketssql.Queries
}

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{q: ticketssql.New(pool)}
}

func (r *CatalogRepository) ListIncidentTypes(ctx context.Context) ([]models.IncidentType, error) {
	rows, err := r.q.ListIncidentTypes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.IncidentType, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.IncidentType{ID: row.ID, Code: row.Code, Title: row.Title})
	}
	return out, nil
}

func (r *CatalogRepository) ListTagsByType(ctx context.Context, typeID uuid.UUID) ([]models.IncidentTag, error) {
	rows, err := r.q.ListTagsByType(ctx, typeID)
	if err != nil {
		return nil, err
	}
	out := make([]models.IncidentTag, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.IncidentTag{
			ID: row.ID, IncidentTypeID: row.IncidentTypeID, Code: row.Code, Title: row.Title,
		})
	}
	return out, nil
}

func (r *CatalogRepository) ListServices(ctx context.Context) ([]models.EmergencyService, error) {
	rows, err := r.q.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.EmergencyService, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.EmergencyService{ID: row.ID, Code: row.Code, Title: row.Title})
	}
	return out, nil
}

func (r *CatalogRepository) UpsertIncidentType(ctx context.Context, code, title string) (models.IncidentType, error) {
	row, err := r.q.UpsertIncidentType(ctx, ticketssql.UpsertIncidentTypeParams{
		ID: uuid.New(), Code: code, Title: title,
	})
	if err != nil {
		return models.IncidentType{}, err
	}
	return models.IncidentType{ID: row.ID, Code: row.Code, Title: row.Title}, nil
}

func (r *CatalogRepository) UpsertIncidentTag(ctx context.Context, typeID uuid.UUID, code, title string) (models.IncidentTag, error) {
	row, err := r.q.UpsertIncidentTag(ctx, ticketssql.UpsertIncidentTagParams{
		ID: uuid.New(), IncidentTypeID: typeID, Code: code, Title: title,
	})
	if err != nil {
		return models.IncidentTag{}, err
	}
	return models.IncidentTag{
		ID: row.ID, IncidentTypeID: row.IncidentTypeID, Code: row.Code, Title: row.Title,
	}, nil
}

func (r *CatalogRepository) UpsertService(ctx context.Context, code, title string) (models.EmergencyService, error) {
	row, err := r.q.UpsertService(ctx, ticketssql.UpsertServiceParams{
		ID: uuid.New(), Code: code, Title: title,
	})
	if err != nil {
		return models.EmergencyService{}, err
	}
	return models.EmergencyService{ID: row.ID, Code: row.Code, Title: row.Title}, nil
}

func (r *CatalogRepository) FindIncidentTypeByID(ctx context.Context, id uuid.UUID) (models.IncidentType, error) {
	row, err := r.q.GetIncidentTypeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.IncidentType{}, errs.ErrNotFound
		}
		return models.IncidentType{}, err
	}
	return models.IncidentType{ID: row.ID, Code: row.Code, Title: row.Title}, nil
}

func (r *CatalogRepository) FindTagIDsForType(ctx context.Context, typeID uuid.UUID) (map[uuid.UUID]struct{}, error) {
	ids, err := r.q.ListTagIDsByType(ctx, typeID)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out, nil
}

func (r *CatalogRepository) ServiceExists(ctx context.Context, ids []uuid.UUID) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	unique := uniqueUUIDs(ids)
	n, err := r.q.CountServicesByIDs(ctx, unique)
	if err != nil {
		return false, err
	}
	return int(n) == len(unique), nil
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
