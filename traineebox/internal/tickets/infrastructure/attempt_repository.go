package infrastructure

import (
	"context"
	"errors"
	"time"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/value_objects"
	"traineebox/internal/tickets/infrastructure/ticketssql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttemptRepository struct {
	pool *pgxpool.Pool
	q    *ticketssql.Queries
}

func NewAttemptRepository(pool *pgxpool.Pool) *AttemptRepository {
	return &AttemptRepository{pool: pool, q: ticketssql.New(pool)}
}

func (r *AttemptRepository) Create(ctx context.Context, attempt models.Attempt) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)
	if err := q.CreateAttempt(ctx, ticketssql.CreateAttemptParams{
		ID:         attempt.ID,
		TicketID:   attempt.TicketID,
		UserID:     attempt.UserID,
		AttemptNo:  int32(attempt.AttemptNo),
		Status:     attempt.Status.String(),
		StartedAt:  attempt.StartedAt,
		DeadlineAt: attempt.DeadlineAt,
		FinishedAt: attempt.FinishedAt,
		Score:      intPtrToInt16(attempt.Score),
	}); err != nil {
		if isUniqueViolation(err) {
			return errs.ErrConflict
		}
		return err
	}
	now := time.Now().UTC()
	if err := q.CreateAttemptAnswer(ctx, ticketssql.CreateAttemptAnswerParams{
		AttemptID:          attempt.ID,
		IncidentTypeID:     attempt.Answer.IncidentTypeID,
		ApplicantLastName:  attempt.Answer.ApplicantLastName,
		ApplicantFirstName: attempt.Answer.ApplicantFirstName,
		CallerNumber:       attempt.Answer.CallerNumber,
		DictatedNumber:     attempt.Answer.DictatedNumber,
		Notes:              attempt.Answer.Notes.String(),
		UpdatedAt:          now,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AttemptRepository) FindByID(ctx context.Context, id uuid.UUID) (models.Attempt, error) {
	row, err := r.q.GetAttemptByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Attempt{}, errs.ErrNotFound
		}
		return models.Attempt{}, err
	}
	return r.loadAttempt(ctx, row)
}

func (r *AttemptRepository) FindInProgress(ctx context.Context, ticketID, userID uuid.UUID) (models.Attempt, error) {
	row, err := r.q.FindInProgressAttempt(ctx, ticketssql.FindInProgressAttemptParams{
		TicketID: ticketID, UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Attempt{}, errs.ErrNotFound
		}
		return models.Attempt{}, err
	}
	return r.loadAttempt(ctx, row)
}

func (r *AttemptRepository) ListByTicketUser(ctx context.Context, ticketID, userID uuid.UUID) ([]models.Attempt, error) {
	rows, err := r.q.ListAttemptsByTicketUser(ctx, ticketssql.ListAttemptsByTicketUserParams{
		TicketID: ticketID, UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]models.Attempt, 0, len(rows))
	for _, row := range rows {
		a, err := r.loadAttempt(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *AttemptRepository) CountFinished(ctx context.Context, ticketID, userID uuid.UUID) (int, error) {
	n, err := r.q.CountFinishedAttempts(ctx, ticketssql.CountFinishedAttemptsParams{
		TicketID: ticketID, UserID: userID,
	})
	return int(n), err
}

func (r *AttemptRepository) NextAttemptNo(ctx context.Context, ticketID, userID uuid.UUID) (int, error) {
	n, err := r.q.MaxAttemptNo(ctx, ticketssql.MaxAttemptNoParams{
		TicketID: ticketID, UserID: userID,
	})
	if err != nil {
		return 0, err
	}
	return int(n) + 1, nil
}

func (r *AttemptRepository) Save(ctx context.Context, attempt models.Attempt) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)
	if err := q.UpdateAttempt(ctx, ticketssql.UpdateAttemptParams{
		ID:         attempt.ID,
		Status:     attempt.Status.String(),
		FinishedAt: attempt.FinishedAt,
		Score:      intPtrToInt16(attempt.Score),
	}); err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := q.UpdateAttemptAnswer(ctx, ticketssql.UpdateAttemptAnswerParams{
		AttemptID:          attempt.ID,
		IncidentTypeID:     attempt.Answer.IncidentTypeID,
		ApplicantLastName:  attempt.Answer.ApplicantLastName,
		ApplicantFirstName: attempt.Answer.ApplicantFirstName,
		CallerNumber:       attempt.Answer.CallerNumber,
		DictatedNumber:     attempt.Answer.DictatedNumber,
		Notes:              attempt.Answer.Notes.String(),
		UpdatedAt:          now,
	}); err != nil {
		return err
	}
	if err := q.DeleteAttemptAnswerTags(ctx, attempt.ID); err != nil {
		return err
	}
	if err := q.DeleteAttemptAnswerServices(ctx, attempt.ID); err != nil {
		return err
	}
	for _, tagID := range attempt.Answer.TagIDs {
		if err := q.InsertAttemptAnswerTag(ctx, ticketssql.InsertAttemptAnswerTagParams{
			AttemptID: attempt.ID, TagID: tagID,
		}); err != nil {
			return err
		}
	}
	for _, serviceID := range attempt.Answer.ServiceIDs {
		if err := q.InsertAttemptAnswerService(ctx, ticketssql.InsertAttemptAnswerServiceParams{
			AttemptID: attempt.ID, ServiceID: serviceID,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *AttemptRepository) loadAttempt(ctx context.Context, row ticketssql.TicketAttempt) (models.Attempt, error) {
	ans, err := r.q.GetAttemptAnswer(ctx, row.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Attempt{}, errs.ErrNotFound
		}
		return models.Attempt{}, err
	}
	tags, err := r.q.ListAttemptAnswerTags(ctx, row.ID)
	if err != nil {
		return models.Attempt{}, err
	}
	services, err := r.q.ListAttemptAnswerServices(ctx, row.ID)
	if err != nil {
		return models.Attempt{}, err
	}
	status, err := value_objects.ParseAttemptStatus(row.Status)
	if err != nil {
		return models.Attempt{}, err
	}
	notes, err := value_objects.NewNotes(ans.Notes)
	if err != nil {
		return models.Attempt{}, err
	}
	return models.Attempt{
		ID:         row.ID,
		TicketID:   row.TicketID,
		UserID:     row.UserID,
		AttemptNo:  int(row.AttemptNo),
		Status:     status,
		StartedAt:  row.StartedAt,
		DeadlineAt: row.DeadlineAt,
		FinishedAt: row.FinishedAt,
		Score:      int16PtrToInt(row.Score),
		Answer: models.NewAnswer(
			ans.IncidentTypeID, tags, services,
			ans.ApplicantLastName, ans.ApplicantFirstName,
			ans.CallerNumber, ans.DictatedNumber, notes,
		),
	}, nil
}

func intPtrToInt16(v *int) *int16 {
	if v == nil {
		return nil
	}
	n := int16(*v)
	return &n
}

func int16PtrToInt(v *int16) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
