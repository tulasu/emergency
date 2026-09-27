package infrastructure

import (
	"context"
	"encoding/json"
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
	report, err := json.Marshal(attempt.Report)
	if err != nil {
		return err
	}
	if err := r.q.CreateAttempt(ctx, ticketssql.CreateAttemptParams{
		ID:            attempt.ID,
		VariantID:     attempt.VariantID,
		UserID:        attempt.UserID,
		GrantedBy:     attempt.GrantedBy,
		AttemptNo:     int32(attempt.AttemptNo),
		Status:        attempt.Status.String(),
		AvailableFrom: attempt.AvailableFrom,
		StartedAt:     attempt.StartedAt,
		DeadlineAt:    attempt.DeadlineAt,
		FinishedAt:    attempt.FinishedAt,
		Score:         intPtrToInt16(attempt.Score),
		Report:        report,
	}); err != nil {
		if isUniqueViolation(err) {
			return errs.ErrConflict
		}
		return err
	}
	return nil
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

func (r *AttemptRepository) FindOpen(ctx context.Context, variantID, userID uuid.UUID) (models.Attempt, error) {
	row, err := r.q.FindOpenAttempt(ctx, ticketssql.FindOpenAttemptParams{VariantID: variantID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Attempt{}, errs.ErrNotFound
		}
		return models.Attempt{}, err
	}
	return r.loadAttempt(ctx, row)
}

func (r *AttemptRepository) ListByVariantUser(ctx context.Context, variantID, userID uuid.UUID) ([]models.Attempt, error) {
	rows, err := r.q.ListAttemptsByVariantUser(ctx, ticketssql.ListAttemptsByVariantUserParams{
		VariantID: variantID, UserID: userID,
	})
	if err != nil {
		return nil, err
	}
	return r.loadAttempts(ctx, rows)
}

func (r *AttemptRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Attempt, error) {
	rows, err := r.q.ListAttemptsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return r.loadAttempts(ctx, rows)
}

func (r *AttemptRepository) HasAny(ctx context.Context, variantID, userID uuid.UUID) (bool, error) {
	return r.q.HasAnyAttempt(ctx, ticketssql.HasAnyAttemptParams{VariantID: variantID, UserID: userID})
}

func (r *AttemptRepository) NextAttemptNo(ctx context.Context, variantID, userID uuid.UUID) (int, error) {
	n, err := r.q.MaxAttemptNo(ctx, ticketssql.MaxAttemptNoParams{VariantID: variantID, UserID: userID})
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
	report, err := json.Marshal(attempt.Report)
	if err != nil {
		return err
	}
	if err := q.UpdateAttempt(ctx, ticketssql.UpdateAttemptParams{
		ID:         attempt.ID,
		Status:     attempt.Status.String(),
		StartedAt:  attempt.StartedAt,
		DeadlineAt: attempt.DeadlineAt,
		FinishedAt: attempt.FinishedAt,
		Score:      intPtrToInt16(attempt.Score),
		Report:     report,
	}); err != nil {
		return err
	}
	now := time.Now().UTC()
	for ticketID, answer := range attempt.Answers {
		if err := q.UpsertAttemptAnswer(ctx, ticketssql.UpsertAttemptAnswerParams{
			AttemptID:          attempt.ID,
			TicketID:           ticketID,
			IncidentTypeCode:   answer.IncidentTypeCode,
			ApplicantLastName:  answer.ApplicantLastName,
			ApplicantFirstName: answer.ApplicantFirstName,
			CallerNumber:       answer.CallerNumber,
			DictatedNumber:     answer.DictatedNumber,
			Notes:              answer.Notes.String(),
			UpdatedAt:          now,
		}); err != nil {
			return err
		}
		if err := q.DeleteAttemptAnswerTags(ctx, ticketssql.DeleteAttemptAnswerTagsParams{
			AttemptID: attempt.ID, TicketID: ticketID,
		}); err != nil {
			return err
		}
		if err := q.DeleteAttemptAnswerServices(ctx, ticketssql.DeleteAttemptAnswerServicesParams{
			AttemptID: attempt.ID, TicketID: ticketID,
		}); err != nil {
			return err
		}
		for _, tagCode := range answer.TagCodes {
			if err := q.InsertAttemptAnswerTag(ctx, ticketssql.InsertAttemptAnswerTagParams{
				AttemptID: attempt.ID, TicketID: ticketID, TagCode: tagCode,
			}); err != nil {
				return err
			}
		}
		for _, serviceCode := range answer.ServiceCodes {
			if err := q.InsertAttemptAnswerService(ctx, ticketssql.InsertAttemptAnswerServiceParams{
				AttemptID: attempt.ID, TicketID: ticketID, ServiceCode: serviceCode,
			}); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (r *AttemptRepository) MarkTimedOut(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ticket_attempts SET status = 'timed_out', finished_at = now()
		 WHERE id = $1 AND status = 'in_progress'`, id)
	return err
}

func (r *AttemptRepository) loadAttempts(ctx context.Context, rows []ticketssql.TicketAttempt) ([]models.Attempt, error) {
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

func (r *AttemptRepository) loadAttempt(ctx context.Context, row ticketssql.TicketAttempt) (models.Attempt, error) {
	status, err := value_objects.ParseAttemptStatus(row.Status)
	if err != nil {
		return models.Attempt{}, err
	}
	report := models.EmptyReport()
	if len(row.Report) > 0 {
		_ = json.Unmarshal(row.Report, &report)
	}
	answers := map[uuid.UUID]models.Answer{}
	ansRows, err := r.q.ListAttemptAnswers(ctx, row.ID)
	if err != nil {
		return models.Attempt{}, err
	}
	for _, ans := range ansRows {
		tags, err := r.q.ListAttemptAnswerTags(ctx, ticketssql.ListAttemptAnswerTagsParams{
			AttemptID: row.ID, TicketID: ans.TicketID,
		})
		if err != nil {
			return models.Attempt{}, err
		}
		services, err := r.q.ListAttemptAnswerServices(ctx, ticketssql.ListAttemptAnswerServicesParams{
			AttemptID: row.ID, TicketID: ans.TicketID,
		})
		if err != nil {
			return models.Attempt{}, err
		}
		notes, err := value_objects.NewNotes(ans.Notes)
		if err != nil {
			return models.Attempt{}, err
		}
		answers[ans.TicketID] = models.NewAnswer(
			ans.IncidentTypeCode, tags, services,
			ans.ApplicantLastName, ans.ApplicantFirstName,
			ans.CallerNumber, ans.DictatedNumber, notes,
		)
	}
	return models.Attempt{
		ID:            row.ID,
		VariantID:     row.VariantID,
		UserID:        row.UserID,
		GrantedBy:     row.GrantedBy,
		AttemptNo:     int(row.AttemptNo),
		Status:        status,
		AvailableFrom: row.AvailableFrom,
		StartedAt:     row.StartedAt,
		DeadlineAt:    row.DeadlineAt,
		FinishedAt:    row.FinishedAt,
		Score:         int16PtrToInt(row.Score),
		Answers:       answers,
		Report:        report,
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
