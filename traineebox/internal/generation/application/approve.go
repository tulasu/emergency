package application

import (
	"context"

	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/repositories"
	ticketsmodels "traineebox/internal/tickets/domain/models"
	ticketsrepos "traineebox/internal/tickets/domain/repositories"
	ticketsvo "traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type ApproveJob struct {
	Jobs       repositories.JobRepository
	Membership repositories.GroupMembership
	Tickets    ticketsrepos.TicketRepository
	Catalog    ticketsrepos.CatalogRepository
	// Atomically publishes ticket+reference+job in one txn when set.
	// Nil keeps the legacy two-step path (tests/fakes).
	AtomicPublish func(ctx context.Context, ticket ticketsmodels.Ticket, ref ticketsmodels.ReferenceAnswer, jobID uuid.UUID, expectedStatus string, expectedVersion int) error
}

type ApproveJobInput struct {
	ActorID uuid.UUID
	Admin   bool
	JobID   uuid.UUID
}

func (uc ApproveJob) Execute(ctx context.Context, in ApproveJobInput) (models.Job, error) {
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
		return models.Job{}, err
	}
	if err := requireManage(ctx, uc.Membership, job.GroupID, in.ActorID, in.Admin); err != nil {
		return models.Job{}, err
	}
	expectedStatus := job.Status.String()
	expectedVersion := job.Version

	draft := job.DraftReference.Normalize()
	title, err := ticketsvo.NewTicketTitle(job.DraftTitle)
	if err != nil {
		return models.Job{}, mapTicketsErr(err)
	}
	if _, err := uc.Catalog.FindIncidentTypeByCode(ctx, draft.IncidentTypeCode); err != nil {
		return models.Job{}, mapTicketsErr(err)
	}
	groups, err := uc.Catalog.ListTagGroupsByType(ctx, draft.IncidentTypeCode)
	if err != nil {
		return models.Job{}, mapTicketsErr(err)
	}
	ticket, err := ticketsmodels.NewTicket(
		job.GroupID, title, job.ScenarioText, in.ActorID, nil, nil, nil, nil,
	)
	if err != nil {
		return models.Job{}, mapTicketsErr(err)
	}
	// Dialog snapshot travels with the ticket; mode defaults to voice (AD-9).
	// Generated tickets carry no teacher snapshot yet: '{}' keeps the jsonb
	// cast valid until PUT /scenario authors the real one (spec C).
	ticket.Mode = "voice"
	ticket.Briefing = job.ScenarioText
	if ticket.ScenarioJSON == "" {
		ticket.ScenarioJSON = "{}"
	}
	ref, err := ticketsmodels.NewReferenceAnswer(
		ticket.ID, draft.IncidentTypeCode, draft.TagCodes, draft.ServiceCodes,
		draft.ApplicantLastName, draft.ApplicantFirstName, draft.CallerNumber, draft.DictatedNumber,
	)
	if err != nil {
		return models.Job{}, mapTicketsErr(err)
	}
	if err := ref.ValidateTagSelection(groups); err != nil {
		return models.Job{}, mapTicketsErr(err)
	}
	if len(draft.ServiceCodes) > 0 {
		ok, err := uc.Catalog.ServiceExists(ctx, draft.ServiceCodes)
		if err != nil {
			return models.Job{}, mapTicketsErr(err)
		}
		if !ok {
			return models.Job{}, errs.ErrInvalidInput
		}
	}
	if uc.AtomicPublish != nil {
		if err := uc.AtomicPublish(ctx, ticket, ref, job.ID, expectedStatus, expectedVersion); err != nil {
			return models.Job{}, err
		}
		if err := job.MarkPublished(ticket.ID); err != nil {
			return models.Job{}, err
		}
		job.Version = expectedVersion + 1
		return job, nil
	}
	if err := uc.Tickets.CreateWithReference(ctx, ticket, ref); err != nil {
		return models.Job{}, err
	}
	if err := job.MarkPublished(ticket.ID); err != nil {
		return models.Job{}, err
	}
	if err := uc.Jobs.SaveCAS(ctx, job, expectedStatus, expectedVersion); err != nil {
		return models.Job{}, err
	}
	job.Version = expectedVersion + 1
	return job, nil
}
