package application

import (
	"context"
	"time"

	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/repositories"
	"traineebox/internal/generation/domain/value_objects"

	"github.com/google/uuid"
)

type ApproveJob struct {
	Jobs      repositories.JobRepository
	Catalog   repositories.Catalog
	Publisher repositories.TicketPublisher
}

type ApproveJobInput struct {
	ActorID uuid.UUID
	Role    value_objects.AccountRole
	JobID   uuid.UUID
}

func (uc ApproveJob) Execute(ctx context.Context, in ApproveJobInput) (models.Job, error) {
	if err := requireManage(in.Role); err != nil {
		return models.Job{}, err
	}
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
		return models.Job{}, err
	}
	expectedStatus := job.Status.String()
	expectedVersion := job.Version

	draft := job.DraftReference.Normalize()
	title, err := value_objects.NewTicketTitle(job.DraftTitle)
	if err != nil {
		return models.Job{}, err
	}
	if err := uc.Catalog.IncidentTypeExists(ctx, draft.IncidentTypeCode); err != nil {
		return models.Job{}, err
	}
	if err := uc.Catalog.ValidateTags(ctx, draft.IncidentTypeCode, draft.TagCodes); err != nil {
		return models.Job{}, err
	}
	if len(draft.ServiceCodes) > 0 {
		ok, err := uc.Catalog.ServicesExist(ctx, draft.ServiceCodes)
		if err != nil {
			return models.Job{}, err
		}
		if !ok {
			return models.Job{}, errs.ErrInvalidInput
		}
	}

	scenarioJSON := job.ScenarioText
	if scenarioJSON == "" {
		scenarioJSON = "{}"
	}
	ticketID := uuid.New()
	publishedID, err := uc.Publisher.Publish(ctx, repositories.PublishDraft{
		TicketID:        ticketID,
		VariantID:       job.VariantID,
		TopicID:         job.TopicID,
		Title:           title.String(),
		Body:            job.ScenarioText,
		CreatedBy:       in.ActorID,
		CreatedAt:       time.Now().UTC(),
		ScenarioJSON:    scenarioJSON,
		Mode:            "voice",
		Briefing:        job.ScenarioText,
		Reference:       draft,
		JobID:           job.ID,
		ExpectedStatus:  expectedStatus,
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		return models.Job{}, err
	}
	if err := job.MarkPublished(publishedID); err != nil {
		return models.Job{}, err
	}
	job.Version = expectedVersion + 1
	return job, nil
}
