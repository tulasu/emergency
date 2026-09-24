package application

import (
	"context"
	"errors"

	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/repositories"
	"traineebox/internal/tickets/domain/abilities"
	ticketserrs "traineebox/internal/tickets/domain/errs"
	ticketsvo "traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type SessionUser struct {
	ID   uuid.UUID
	Role ticketsvo.AccountRole
}

type Authenticator interface {
	CurrentUser(ctx context.Context, token string) (SessionUser, error)
}

func requireManage(ctx context.Context, membership repositories.GroupMembership, groupID, actorID uuid.UUID, admin bool) error {
	if admin {
		return mapAbility(abilities.ManageTicket("", true))
	}
	role, err := membership.RoleOf(ctx, groupID, actorID)
	if err != nil {
		return mapTicketsErr(err)
	}
	return mapAbility(abilities.ManageTicket(role, false))
}

func mapAbility(err error) error {
	if err == nil {
		return nil
	}
	return mapTicketsErr(err)
}

func mapTicketsErr(err error) error {
	switch {
	case errors.Is(err, ticketserrs.ErrForbidden):
		return errs.ErrForbidden
	case errors.Is(err, ticketserrs.ErrNotFound):
		return errs.ErrNotFound
	case errors.Is(err, ticketserrs.ErrUnauthorized):
		return errs.ErrUnauthorized
	case errors.Is(err, ticketserrs.ErrUserBlocked):
		return errs.ErrUserBlocked
	case errors.Is(err, ticketserrs.ErrInvalidInput), errors.Is(err, ticketserrs.ErrInvalidTags), errors.Is(err, ticketserrs.ErrInvalidTagSelection):
		return errs.ErrInvalidInput
	case errors.Is(err, ticketserrs.ErrConflict):
		return errs.ErrConflict
	default:
		return err
	}
}

type CreateJob struct {
	Jobs       repositories.JobRepository
	Membership repositories.GroupMembership
}

type CreateJobInput struct {
	ActorID uuid.UUID
	Admin   bool
	GroupID uuid.UUID
	Prompt  string
}

func (uc CreateJob) Execute(ctx context.Context, in CreateJobInput) (models.Job, error) {
	if err := requireManage(ctx, uc.Membership, in.GroupID, in.ActorID, in.Admin); err != nil {
		return models.Job{}, err
	}
	job, err := models.NewJob(in.GroupID, in.ActorID, in.Prompt)
	if err != nil {
		return models.Job{}, err
	}
	if err := uc.Jobs.Create(ctx, job); err != nil {
		return models.Job{}, err
	}
	return job, nil
}

type ListJobs struct {
	Jobs       repositories.JobRepository
	Membership repositories.GroupMembership
}

type ListJobsInput struct {
	ActorID uuid.UUID
	Admin   bool
	GroupID uuid.UUID
}

func (uc ListJobs) Execute(ctx context.Context, in ListJobsInput) ([]models.Job, error) {
	if err := requireManage(ctx, uc.Membership, in.GroupID, in.ActorID, in.Admin); err != nil {
		return nil, err
	}
	return uc.Jobs.ListByGroup(ctx, in.GroupID)
}

type GetJob struct {
	Jobs       repositories.JobRepository
	Membership repositories.GroupMembership
}

type GetJobInput struct {
	ActorID uuid.UUID
	Admin   bool
	JobID   uuid.UUID
}

func (uc GetJob) Execute(ctx context.Context, in GetJobInput) (models.Job, error) {
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
		return models.Job{}, err
	}
	if err := requireManage(ctx, uc.Membership, job.GroupID, in.ActorID, in.Admin); err != nil {
		return models.Job{}, err
	}
	return job, nil
}

type PatchJob struct {
	Jobs       repositories.JobRepository
	Membership repositories.GroupMembership
}

type PatchJobInput struct {
	ActorID     uuid.UUID
	Admin       bool
	JobID       uuid.UUID
	DraftTitle  string
	ScenarioText string
	Reference   models.DraftReference
}

func (uc PatchJob) Execute(ctx context.Context, in PatchJobInput) (models.Job, error) {
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
		return models.Job{}, err
	}
	if err := requireManage(ctx, uc.Membership, job.GroupID, in.ActorID, in.Admin); err != nil {
		return models.Job{}, err
	}
	expectedStatus := job.Status.String()
	expectedVersion := job.Version
	if err := job.ApplyDraft(in.DraftTitle, in.ScenarioText, in.Reference); err != nil {
		return models.Job{}, err
	}
	if err := uc.Jobs.SaveCAS(ctx, job, expectedStatus, expectedVersion); err != nil {
		return models.Job{}, err
	}
	job.Version = expectedVersion + 1
	return job, nil
}

type RetryJob struct {
	Jobs       repositories.JobRepository
	Membership repositories.GroupMembership
}

type RetryJobInput struct {
	ActorID uuid.UUID
	Admin   bool
	JobID   uuid.UUID
}

func (uc RetryJob) Execute(ctx context.Context, in RetryJobInput) (models.Job, error) {
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
		return models.Job{}, err
	}
	if err := requireManage(ctx, uc.Membership, job.GroupID, in.ActorID, in.Admin); err != nil {
		return models.Job{}, err
	}
	expectedStatus := job.Status.String()
	expectedVersion := job.Version
	if err := job.MarkRetry(); err != nil {
		return models.Job{}, err
	}
	if err := uc.Jobs.SaveCAS(ctx, job, expectedStatus, expectedVersion); err != nil {
		return models.Job{}, err
	}
	job.Version = expectedVersion + 1
	return job, nil
}

type CancelJob struct {
	Jobs       repositories.JobRepository
	Membership repositories.GroupMembership
}

type CancelJobInput struct {
	ActorID uuid.UUID
	Admin   bool
	JobID   uuid.UUID
}

func (uc CancelJob) Execute(ctx context.Context, in CancelJobInput) (models.Job, error) {
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
		return models.Job{}, err
	}
	if err := requireManage(ctx, uc.Membership, job.GroupID, in.ActorID, in.Admin); err != nil {
		return models.Job{}, err
	}
	expectedStatus := job.Status.String()
	expectedVersion := job.Version
	if err := job.Cancel(); err != nil {
		return models.Job{}, err
	}
	if err := uc.Jobs.SaveCAS(ctx, job, expectedStatus, expectedVersion); err != nil {
		return models.Job{}, err
	}
	job.Version = expectedVersion + 1
	return job, nil
}

type DeleteJob struct {
	Jobs       repositories.JobRepository
	Membership repositories.GroupMembership
}

type DeleteJobInput struct {
	ActorID uuid.UUID
	Admin   bool
	JobID   uuid.UUID
}

func (uc DeleteJob) Execute(ctx context.Context, in DeleteJobInput) error {
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
		return err
	}
	if err := requireManage(ctx, uc.Membership, job.GroupID, in.ActorID, in.Admin); err != nil {
		return err
	}
	if !job.Status.CanDelete() {
		return errs.ErrInvalidState
	}
	return uc.Jobs.Delete(ctx, job.ID)
}
