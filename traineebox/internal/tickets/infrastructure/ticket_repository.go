package infrastructure

import (
	"context"
	"encoding/json"
	"errors"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/value_objects"
	"traineebox/internal/tickets/infrastructure/ticketssql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	mode := ticket.Mode
	if mode == "" {
		mode = "voice"
	}
	scenario := validOrEmptyJSON(ticket.ScenarioJSON)
	return r.q.CreateTicket(ctx, ticketssql.CreateTicketParams{
		ID:              ticket.ID,
		VariantID:       ticket.VariantID,
		TopicID:         ticket.TopicID,
		Title:           ticket.Title.String(),
		Body:            ticket.Body,
		CreatedBy:       ticket.CreatedBy,
		CreatedAt:       ticket.CreatedAt,
		Scenario:        []byte(scenario),
		ScenarioVersion: ticket.ScenarioVersion,
		Mode:            mode,
		Briefing:        ticket.Briefing,
	})
}

func (r *TicketRepository) FindByID(ctx context.Context, id uuid.UUID) (models.Ticket, error) {
	ticket, err := scanTicket(r.pool.QueryRow(ctx, `SELECT id, variant_id, topic_id, title, body, created_by, created_at,
		scenario, scenario_version, audio_digest, audio_status, reference, mode, briefing
		FROM tickets WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, errs.ErrNotFound
		}
		return models.Ticket{}, err
	}
	return ticket, nil
}

type LibraryTicketExtras struct {
	IncidentTypeCode string
	IncidentType     string
	SlotsTotal       int
	SlotsRequired    int
	VariantUsage     int
}

func (r *TicketRepository) LibraryExtras(ctx context.Context, ticketID uuid.UUID) (LibraryTicketExtras, error) {
	var out LibraryTicketExtras
	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(r.incident_type_code, ''), COALESCE(it.title, '')
		FROM ticket_reference_answers r
		LEFT JOIN incident_types it ON it.code = r.incident_type_code
		WHERE r.ticket_id = $1
	`, ticketID).Scan(&out.IncidentTypeCode, &out.IncidentType)
	_ = r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM reference_answer_tags WHERE ticket_id = $1
	`, ticketID).Scan(&out.SlotsTotal)
	out.SlotsRequired = out.SlotsTotal
	if out.SlotsRequired > 2 {
		out.SlotsRequired = out.SlotsTotal - out.SlotsTotal/3
	}
	_ = r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM tickets
		WHERE variant_id IS NOT NULL AND title = (SELECT title FROM tickets WHERE id = $1)
	`, ticketID).Scan(&out.VariantUsage)
	return out, nil
}

func (r *TicketRepository) ListByVariant(ctx context.Context, variantID uuid.UUID) ([]models.Ticket, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, variant_id, topic_id, title, body, created_by, created_at,
		scenario, scenario_version, audio_digest, audio_status, reference, mode, briefing
		FROM tickets WHERE variant_id = $1 ORDER BY created_at`, variantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tickets []models.Ticket
	for rows.Next() {
		ticket, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	return tickets, rows.Err()
}

func (r *TicketRepository) ListLibrary(ctx context.Context, q string) ([]models.Ticket, error) {
	rows, err := r.q.ListLibraryTickets(ctx, textArg(q))
	if err != nil {
		return nil, err
	}
	out := make([]models.Ticket, 0, len(rows))
	for _, row := range rows {
		t := models.Ticket{
			ID: row.ID, VariantID: row.VariantID, TopicID: row.TopicID,
			Title: value_objects.TicketTitle(row.Title), Body: row.Body,
			CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt,
			ScenarioJSON: string(row.Scenario), ScenarioVersion: row.ScenarioVersion,
			AudioDigest: row.AudioDigest, AudioStatus: row.AudioStatus,
			Reference: string(row.Reference), Mode: row.Mode, Briefing: row.Briefing,
		}
		out = append(out, t)
	}
	return out, nil
}

func textArg(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func (r *TicketRepository) Update(ctx context.Context, ticket models.Ticket) error {
	return r.q.UpdateTicket(ctx, ticketssql.UpdateTicketParams{
		ID: ticket.ID, Title: ticket.Title.String(), Body: ticket.Body, TopicID: ticket.TopicID,
	})
}

func (r *TicketRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTicket(ctx, id)
}

func (r *TicketRepository) UpdateDialogSnapshot(ctx context.Context, ticketID uuid.UUID, scenarioJSON, version, digest, status string) error {
	scenario := validOrEmptyJSON(scenarioJSON)
	if scenarioJSON != "" && scenario == "{}" {
		return errs.ErrInvalidInput
	}
	result, err := r.pool.Exec(ctx, `UPDATE tickets
		SET scenario = $2::jsonb, scenario_version = $3, audio_digest = $4, audio_status = $5
		WHERE id = $1`, ticketID, []byte(scenario), version, digest, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *TicketRepository) UpdateAudioStatus(ctx context.Context, ticketID uuid.UUID, digest, status string) (bool, error) {
	result, err := r.pool.Exec(ctx, `UPDATE tickets SET audio_status = $3 WHERE id = $1 AND audio_digest = $2 AND (audio_status = 'pending' OR $3 = 'ready')`, ticketID, digest, status)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, nil
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
	if err := r.Create(ctx, ticket); err != nil {
		return err
	}
	return r.SaveReference(ctx, ref)
}

type ticketScanner interface {
	Scan(dest ...any) error
}

func scanTicket(row ticketScanner) (models.Ticket, error) {
	var ticket models.Ticket
	var title string
	var scenario, reference []byte
	err := row.Scan(
		&ticket.ID,
		&ticket.VariantID,
		&ticket.TopicID,
		&title,
		&ticket.Body,
		&ticket.CreatedBy,
		&ticket.CreatedAt,
		&scenario,
		&ticket.ScenarioVersion,
		&ticket.AudioDigest,
		&ticket.AudioStatus,
		&reference,
		&ticket.Mode,
		&ticket.Briefing,
	)
	if err != nil {
		return models.Ticket{}, err
	}
	ticket.Title = value_objects.TicketTitle(title)
	ticket.ScenarioJSON = string(scenario)
	ticket.Reference = string(reference)
	return ticket, nil
}

func validOrEmptyJSON(s string) string {
	if s == "" || !json.Valid([]byte(s)) {
		return "{}"
	}
	return s
}
