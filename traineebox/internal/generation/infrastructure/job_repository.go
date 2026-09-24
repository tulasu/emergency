package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/value_objects"
	"traineebox/internal/generation/infrastructure/generationsql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRepository struct {
	q    *generationsql.Queries
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{q: generationsql.New(pool), pool: pool}
}

// ClaimNext atomically claims one queued (or lease-expired in-progress) job
// for building_dialog/checking_dialog driving. SKIP LOCKED lets N workers
// share the queue. Nothing else drives these statuses (spec K).
func (r *JobRepository) ClaimNext(ctx context.Context, workerID string, leaseSeconds int) (models.Job, bool, error) {
	if leaseSeconds <= 0 {
		leaseSeconds = 120
	}
	var row generationsql.TicketGenerationJob
	// pgx maps the row via the generated scanner: reuse GetGenerationJobByID shape.
	err := r.pool.QueryRow(ctx,
		`UPDATE ticket_generation_jobs AS j SET
			status = 'building_dialog',
			version = j.version + 1,
			claimed_by = $1,
			claimed_at = now(),
			lease_until = now() + make_interval(secs => $2),
			updated_at = now(),
			attempts = j.attempts + 1,
			error_message = ''
		WHERE j.id = (
			SELECT id FROM ticket_generation_jobs
			WHERE status = 'queued' OR (status IN (
				'enriching','filling_pii','picking_type','tagging_type','tagging_common',
				'building_services','building_dialog','checking_dialog','building_ref')
				AND lease_until IS NOT NULL AND lease_until < now())
			ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1)
		RETURNING id, group_id, created_by, prompt, status, version, scenario_text,
			draft_title, draft_reference, error_message, attempts, published_ticket_id,
			claimed_by, claimed_at, lease_until, created_at, updated_at`,
		workerID, leaseSeconds).Scan(
		&row.ID, &row.GroupID, &row.CreatedBy, &row.Prompt, &row.Status, &row.Version,
		&row.ScenarioText, &row.DraftTitle, &row.DraftReference, &row.ErrorMessage,
		&row.Attempts, &row.PublishedTicketID, &row.ClaimedBy, &row.ClaimedAt,
		&row.LeaseUntil, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Job{}, false, nil
		}
		return models.Job{}, false, err
	}
	job, err := mapJob(row)
	if err != nil {
		return models.Job{}, false, err
	}
	return job, true, nil
}

func (r *JobRepository) Create(ctx context.Context, job models.Job) error {
	ref, err := json.Marshal(job.DraftReference.Normalize())
	if err != nil {
		return err
	}
	return r.q.CreateGenerationJob(ctx, generationsql.CreateGenerationJobParams{
		ID:                job.ID,
		GroupID:           job.GroupID,
		CreatedBy:         job.CreatedBy,
		Prompt:            job.Prompt,
		Status:            job.Status.String(),
		Version:           int32(job.Version),
		ScenarioText:      job.ScenarioText,
		DraftTitle:        job.DraftTitle,
		DraftReference:    ref,
		ErrorMessage:      job.ErrorMessage,
		Attempts:          int32(job.Attempts),
		PublishedTicketID: job.PublishedTicketID,
		ClaimedBy:         job.ClaimedBy,
		ClaimedAt:         job.ClaimedAt,
		LeaseUntil:        job.LeaseUntil,
		CreatedAt:         job.CreatedAt,
		UpdatedAt:         job.UpdatedAt,
	})
}

func (r *JobRepository) FindByID(ctx context.Context, id uuid.UUID) (models.Job, error) {
	row, err := r.q.GetGenerationJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Job{}, errs.ErrNotFound
		}
		return models.Job{}, err
	}
	return mapJob(row)
}

func (r *JobRepository) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]models.Job, error) {
	rows, err := r.q.ListGenerationJobsByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Job, 0, len(rows))
	for _, row := range rows {
		job, err := mapJob(row)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, nil
}

func (r *JobRepository) SaveCAS(ctx context.Context, job models.Job, expectedStatus string, expectedVersion int) error {
	ref, err := json.Marshal(job.DraftReference.Normalize())
	if err != nil {
		return err
	}
	n, err := r.q.UpdateGenerationJobCAS(ctx, generationsql.UpdateGenerationJobCASParams{
		Status:            job.Status.String(),
		ScenarioText:      job.ScenarioText,
		DraftTitle:        job.DraftTitle,
		DraftReference:    ref,
		ErrorMessage:      job.ErrorMessage,
		Attempts:          int32(job.Attempts),
		PublishedTicketID: job.PublishedTicketID,
		ClaimedBy:         job.ClaimedBy,
		ClaimedAt:         job.ClaimedAt,
		LeaseUntil:        job.LeaseUntil,
		UpdatedAt:         job.UpdatedAt,
		ID:                job.ID,
		Status_2:          expectedStatus,
		Version:           int32(expectedVersion),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return errs.ErrConflict
	}
	return nil
}

func (r *JobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteGenerationJob(ctx, id)
}

func mapJob(row generationsql.TicketGenerationJob) (models.Job, error) {
	status, err := value_objects.ParseJobStatus(row.Status)
	if err != nil {
		return models.Job{}, err
	}
	var ref models.DraftReference
	if len(row.DraftReference) > 0 {
		if err := json.Unmarshal(row.DraftReference, &ref); err != nil {
			return models.Job{}, err
		}
	}
	ref = ref.Normalize()
	return models.Job{
		ID:                row.ID,
		GroupID:           row.GroupID,
		CreatedBy:         row.CreatedBy,
		Prompt:            row.Prompt,
		Status:            status,
		Version:           int(row.Version),
		ScenarioText:      row.ScenarioText,
		DraftTitle:        row.DraftTitle,
		DraftReference:    ref,
		ErrorMessage:      row.ErrorMessage,
		Attempts:          int(row.Attempts),
		PublishedTicketID: row.PublishedTicketID,
		ClaimedBy:         row.ClaimedBy,
		ClaimedAt:         row.ClaimedAt,
		LeaseUntil:        row.LeaseUntil,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}
