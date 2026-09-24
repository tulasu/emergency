package infrastructure

import (
	"context"
	"errors"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/value_objects"
	"traineebox/internal/tickets/infrastructure/ticketssql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TicketRepository struct {
	pool *pgxpool.Pool
	q    *ticketssql.Queries
}

func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{pool: pool, q: ticketssql.New(pool)}
}

func (r *TicketRepository) Create(ctx context.Context, ticket models.Ticket) error {
	return r.q.CreateTicket(ctx, ticketssql.CreateTicketParams{
		ID:              ticket.ID,
		GroupID:         ticket.GroupID,
		Title:           ticket.Title.String(),
		Body:            ticket.Body,
		MaxAttempts:     intPtrToInt32(ticket.MaxAttempts),
		AvailableFrom:   ticket.AvailableFrom,
		AvailableUntil:  ticket.AvailableUntil,
		DurationSeconds: intPtrToInt32(ticket.DurationSeconds),
		CreatedBy:       ticket.CreatedBy,
		CreatedAt:       ticket.CreatedAt,
	})
}

func (r *TicketRepository) FindByID(ctx context.Context, id uuid.UUID) (models.Ticket, error) {
	row, err := r.q.GetTicketByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, errs.ErrNotFound
		}
		return models.Ticket{}, err
	}
	return mapTicket(row), nil
}

func (r *TicketRepository) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]models.Ticket, error) {
	rows, err := r.q.ListTicketsByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Ticket, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTicket(row))
	}
	return out, nil
}

func (r *TicketRepository) SaveReference(ctx context.Context, ref models.ReferenceAnswer) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)
	if err := q.UpsertReferenceAnswer(ctx, ticketssql.UpsertReferenceAnswerParams{
		TicketID:           ref.TicketID,
		IncidentTypeCode:   ref.IncidentTypeCode,
		ApplicantLastName:  ref.ApplicantLastName,
		ApplicantFirstName: ref.ApplicantFirstName,
		CallerNumber:       ref.CallerNumber,
		DictatedNumber:     ref.DictatedNumber,
	}); err != nil {
		return err
	}
	if err := q.DeleteReferenceAnswerTags(ctx, ref.TicketID); err != nil {
		return err
	}
	if err := q.DeleteReferenceAnswerServices(ctx, ref.TicketID); err != nil {
		return err
	}
	for _, tagCode := range ref.TagCodes {
		if err := q.InsertReferenceAnswerTag(ctx, ticketssql.InsertReferenceAnswerTagParams{
			TicketID: ref.TicketID, TagCode: tagCode,
		}); err != nil {
			return err
		}
	}
	for _, serviceCode := range ref.ServiceCodes {
		if err := q.InsertReferenceAnswerService(ctx, ticketssql.InsertReferenceAnswerServiceParams{
			TicketID: ref.TicketID, ServiceCode: serviceCode,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *TicketRepository) FindReference(ctx context.Context, ticketID uuid.UUID) (models.ReferenceAnswer, error) {
	row, err := r.q.GetReferenceAnswer(ctx, ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ReferenceAnswer{}, errs.ErrNotFound
		}
		return models.ReferenceAnswer{}, err
	}
	tags, err := r.q.ListReferenceAnswerTags(ctx, ticketID)
	if err != nil {
		return models.ReferenceAnswer{}, err
	}
	services, err := r.q.ListReferenceAnswerServices(ctx, ticketID)
	if err != nil {
		return models.ReferenceAnswer{}, err
	}
	return models.ReferenceAnswer{
		TicketID:           row.TicketID,
		IncidentTypeCode:   row.IncidentTypeCode,
		TagCodes:           tags,
		ServiceCodes:       services,
		ApplicantLastName:  row.ApplicantLastName,
		ApplicantFirstName: row.ApplicantFirstName,
		CallerNumber:       row.CallerNumber,
		DictatedNumber:     row.DictatedNumber,
	}, nil
}

func (r *TicketRepository) CreateWithReference(ctx context.Context, ticket models.Ticket, ref models.ReferenceAnswer) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)
	if err := q.CreateTicket(ctx, ticketssql.CreateTicketParams{
		ID:              ticket.ID,
		GroupID:         ticket.GroupID,
		Title:           ticket.Title.String(),
		Body:            ticket.Body,
		MaxAttempts:     intPtrToInt32(ticket.MaxAttempts),
		AvailableFrom:   ticket.AvailableFrom,
		AvailableUntil:  ticket.AvailableUntil,
		DurationSeconds: intPtrToInt32(ticket.DurationSeconds),
		CreatedBy:       ticket.CreatedBy,
		CreatedAt:       ticket.CreatedAt,
	}); err != nil {
		return err
	}
	if err := q.UpsertReferenceAnswer(ctx, ticketssql.UpsertReferenceAnswerParams{
		TicketID:           ref.TicketID,
		IncidentTypeCode:   ref.IncidentTypeCode,
		ApplicantLastName:  ref.ApplicantLastName,
		ApplicantFirstName: ref.ApplicantFirstName,
		CallerNumber:       ref.CallerNumber,
		DictatedNumber:     ref.DictatedNumber,
	}); err != nil {
		return err
	}
	for _, tagCode := range ref.TagCodes {
		if err := q.InsertReferenceAnswerTag(ctx, ticketssql.InsertReferenceAnswerTagParams{
			TicketID: ref.TicketID, TagCode: tagCode,
		}); err != nil {
			return err
		}
	}
	for _, serviceCode := range ref.ServiceCodes {
		if err := q.InsertReferenceAnswerService(ctx, ticketssql.InsertReferenceAnswerServiceParams{
			TicketID: ref.TicketID, ServiceCode: serviceCode,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func mapTicket(row ticketssql.Ticket) models.Ticket {
	return models.Ticket{
		ID:              row.ID,
		GroupID:         row.GroupID,
		Title:           value_objects.TicketTitle(row.Title),
		Body:            row.Body,
		MaxAttempts:     int32PtrToInt(row.MaxAttempts),
		AvailableFrom:   row.AvailableFrom,
		AvailableUntil:  row.AvailableUntil,
		DurationSeconds: int32PtrToInt(row.DurationSeconds),
		CreatedBy:       row.CreatedBy,
		CreatedAt:       row.CreatedAt,
	}
}

func intPtrToInt32(v *int) *int32 {
	if v == nil {
		return nil
	}
	n := int32(*v)
	return &n
}

func int32PtrToInt(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}
