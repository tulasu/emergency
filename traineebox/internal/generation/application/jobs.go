package application

import (
	"context"

	"traineebox/internal/generation/domain/abilities"
	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/repositories"
	"traineebox/internal/generation/domain/value_objects"

	"github.com/google/uuid"
)

type SessionUser struct {
	ID   uuid.UUID
	Role value_objects.AccountRole
}

type Authenticator interface {
	CurrentUser(ctx context.Context, token string) (SessionUser, error)
}

func requireManage(role value_objects.AccountRole) error {
	return abilities.ManageJob(role)
}

type CreateJob struct {
	Jobs       repositories.JobRepository
	Curriculum repositories.CurriculumExists
}

type CreateJobInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	VariantID uuid.UUID
	TopicID   uuid.UUID
	Prompt    string
}

func (uc CreateJob) Execute(ctx context.Context, in CreateJobInput) (models.Job, error) {
	if err := requireManage(in.Role); err != nil {
		return models.Job{}, err
	}
	if err := uc.Curriculum.VariantExists(ctx, in.VariantID); err != nil {
		return models.Job{}, err
	}
	if err := uc.Curriculum.TopicExists(ctx, in.TopicID); err != nil {
		return models.Job{}, err
	}
	job, err := models.NewJob(in.VariantID, in.TopicID, in.ActorID, in.Prompt)
	if err != nil {
		return models.Job{}, err
	}
	if err := uc.Jobs.Create(ctx, job); err != nil {
		return models.Job{}, err
	}
	return job, nil
}

type ListJobs struct {
	Jobs repositories.JobRepository
}

type ListJobsInput struct {
	Role      value_objects.AccountRole
	VariantID uuid.UUID
}

func (uc ListJobs) Execute(ctx context.Context, in ListJobsInput) ([]models.Job, error) {
	if err := requireManage(in.Role); err != nil {
		return nil, err
	}
	return uc.Jobs.ListByVariant(ctx, in.VariantID)
}

type GetJob struct {
	Jobs repositories.JobRepository
}

type GetJobInput struct {
	Role  value_objects.AccountRole
	JobID uuid.UUID
}

func (uc GetJob) Execute(ctx context.Context, in GetJobInput) (models.Job, error) {
	if err := requireManage(in.Role); err != nil {
		return models.Job{}, err
	}
	return uc.Jobs.FindByID(ctx, in.JobID)
}

type PatchJob struct {
	Jobs repositories.JobRepository
}

type PatchJobInput struct {
	Role         value_objects.AccountRole
	JobID        uuid.UUID
	DraftTitle   string
	ScenarioText string
	Reference    models.DraftReference
}

func (uc PatchJob) Execute(ctx context.Context, in PatchJobInput) (models.Job, error) {
	if err := requireManage(in.Role); err != nil {
		return models.Job{}, err
	}
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
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
	Jobs repositories.JobRepository
}

type RetryJobInput struct {
	Role  value_objects.AccountRole
	JobID uuid.UUID
}

func (uc RetryJob) Execute(ctx context.Context, in RetryJobInput) (models.Job, error) {
	if err := requireManage(in.Role); err != nil {
		return models.Job{}, err
	}
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
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
	Jobs repositories.JobRepository
}

type CancelJobInput struct {
	Role  value_objects.AccountRole
	JobID uuid.UUID
}

func (uc CancelJob) Execute(ctx context.Context, in CancelJobInput) (models.Job, error) {
	if err := requireManage(in.Role); err != nil {
		return models.Job{}, err
	}
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
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
	Jobs repositories.JobRepository
}

type DeleteJobInput struct {
	Role  value_objects.AccountRole
	JobID uuid.UUID
}

func (uc DeleteJob) Execute(ctx context.Context, in DeleteJobInput) error {
	if err := requireManage(in.Role); err != nil {
		return err
	}
	job, err := uc.Jobs.FindByID(ctx, in.JobID)
	if err != nil {
		return err
	}
	if !job.Status.CanDelete() {
		return errs.ErrInvalidState
	}
	return uc.Jobs.Delete(ctx, job.ID)
}
