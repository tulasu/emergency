package main

import (
	"context"
	"errors"
	"os"
	"strconv"

	genapp "traineebox/internal/generation/application"
	generrs "traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/repositories"
	geninfra "traineebox/internal/generation/infrastructure"
	ticketserrs "traineebox/internal/tickets/domain/errs"
	ticketsmodels "traineebox/internal/tickets/domain/models"
	ticketsrepos "traineebox/internal/tickets/domain/repositories"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type generationCurriculum struct {
	inner interface {
		VariantExists(ctx context.Context, variantID uuid.UUID) error
		TopicExists(ctx context.Context, topicID uuid.UUID) error
	}
}

func (c generationCurriculum) VariantExists(ctx context.Context, variantID uuid.UUID) error {
	return mapTicketsToGeneration(c.inner.VariantExists(ctx, variantID))
}

func (c generationCurriculum) TopicExists(ctx context.Context, topicID uuid.UUID) error {
	return mapTicketsToGeneration(c.inner.TopicExists(ctx, topicID))
}

type generationCatalog struct {
	tickets ticketsrepos.CatalogRepository
}

func (c generationCatalog) IncidentTypeExists(ctx context.Context, code string) error {
	_, err := c.tickets.FindIncidentTypeByCode(ctx, code)
	return mapTicketsToGeneration(err)
}

func (c generationCatalog) ValidateTags(ctx context.Context, incidentType string, tags []string) error {
	groups, err := c.tickets.ListTagGroupsByType(ctx, incidentType)
	if err != nil {
		return mapTicketsToGeneration(err)
	}
	ref, err := ticketsmodels.NewReferenceAnswer(uuid.New(), incidentType, tags, nil, "", "", "", "")
	if err != nil {
		return mapTicketsToGeneration(err)
	}
	return mapTicketsToGeneration(ref.ValidateTagSelection(groups))
}

func (c generationCatalog) ServicesExist(ctx context.Context, codes []string) (bool, error) {
	ok, err := c.tickets.ServiceExists(ctx, codes)
	return ok, mapTicketsToGeneration(err)
}

type generationPublisher struct {
	pool *pgxpool.Pool
}

func (p generationPublisher) Publish(ctx context.Context, draft repositories.PublishDraft) (uuid.UUID, error) {
	if err := geninfra.ApproveAtomically(ctx, p.pool, draft); err != nil {
		return uuid.Nil, err
	}
	return draft.TicketID, nil
}

func mapTicketsToGeneration(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ticketserrs.ErrForbidden):
		return generrs.ErrForbidden
	case errors.Is(err, ticketserrs.ErrNotFound):
		return generrs.ErrNotFound
	case errors.Is(err, ticketserrs.ErrUnauthorized):
		return generrs.ErrUnauthorized
	case errors.Is(err, ticketserrs.ErrUserBlocked):
		return generrs.ErrUserBlocked
	case errors.Is(err, ticketserrs.ErrInvalidInput), errors.Is(err, ticketserrs.ErrInvalidTags), errors.Is(err, ticketserrs.ErrInvalidTagSelection):
		return generrs.ErrInvalidInput
	case errors.Is(err, ticketserrs.ErrConflict):
		return generrs.ErrConflict
	default:
		return err
	}
}

func newGenerationDrive(jobs *geninfra.JobRepository, catalog repositories.Catalog, dialogURLs []string, serviceToken string) genapp.DriveJob {
	lease := 120
	if v, err := strconv.Atoi(os.Getenv("TICKETGEN_LEASE_SECONDS")); err == nil && v > 0 {
		lease = v
	}
	workerID := os.Getenv("TICKETGEN_WORKER_ID")
	if workerID == "" {
		workerID = "traineebox-" + uuid.New().String()[:8]
	}
	return genapp.DriveJob{
		Jobs:         jobs,
		Catalog:      catalog,
		Drafts:       geninfra.NewTicketgenClient(os.Getenv("TICKETGEN_URL")),
		Linter:       geninfra.NewDialogLinter(dialogURLs, serviceToken),
		WorkerID:     workerID,
		LeaseSeconds: lease,
	}
}
