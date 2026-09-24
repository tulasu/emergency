package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"math"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/value_objects"
	"traineebox/internal/tickets/infrastructure/ticketssql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	// Create + dialog snapshot in ONE transaction: no snapshot-less ticket.
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
	if err := updateDialogSnapshotTx(ctx, tx, ticket); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// saveDialogSnapshot persists the dialog snapshot columns outside a caller
// transaction (empty snapshot is a no-op).
func (r *TicketRepository) saveDialogSnapshot(ctx context.Context, ticket models.Ticket) error {
	return updateDialogSnapshotTx(ctx, r.pool, ticket)
}

// updateDialogSnapshotTx persists the dialog snapshot columns added in 00008.
// Raw SQL (no sqlc regen) so legacy queries keep working; empty snapshot is a no-op.
// Non-JSON can never reach the jsonb cast: fallback '{}' (spec P).
// db is *pgxpool.Pool or pgx.Tx — create and snapshot stay one statement set.
func updateDialogSnapshotTx(ctx context.Context, db interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}, ticket models.Ticket) error {
	if ticket.ScenarioJSON == "" && ticket.ScenarioVersion == "" && ticket.Mode == "" && ticket.Briefing == "" {
		return nil
	}
	mode := ticket.Mode
	if mode == "" {
		mode = "voice"
	}
	scenario := validOrEmptyJSON(ticket.ScenarioJSON)
	_, err := db.Exec(ctx,
		`UPDATE tickets SET scenario = $2::jsonb, scenario_version = $3, mode = $4, briefing = $5 WHERE id = $1`,
		ticket.ID, scenario, ticket.ScenarioVersion, mode, ticket.Briefing)
	return err
}

func validOrEmptyJSON(s string) string {
	if s == "" || !json.Valid([]byte(s)) {
		return "{}"
	}
	return s
}

// UpdateDialogSnapshot implements PUT /tickets/{id}/scenario (canon lives here).
// 0 rows → 404, never silent success (spec P).
func (r *TicketRepository) UpdateDialogSnapshot(ctx context.Context, ticketID uuid.UUID, scenarioJSON, version string) error {
	scenario := validOrEmptyJSON(scenarioJSON)
	if scenarioJSON != "" && scenario == "{}" {
		return errs.ErrInvalidInput
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE tickets SET scenario = $2::jsonb, scenario_version = $3 WHERE id = $1`,
		ticketID, scenario, version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *TicketRepository) FindByID(ctx context.Context, id uuid.UUID) (models.Ticket, error) {
	row, err := r.q.GetTicketByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Ticket{}, errs.ErrNotFound
		}
		return models.Ticket{}, err
	}
	ticket := mapTicket(row)
	// sqlc queries predate 00008: overlay the dialog snapshot columns so every
	// dialog open gets the stored snapshot, never the "{}" fallback (spec D).
	if err := r.hydrateDialogSnapshot(ctx, &ticket); err != nil {
		return models.Ticket{}, err
	}
	return ticket, nil
}

func (r *TicketRepository) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]models.Ticket, error) {
	rows, err := r.q.ListTicketsByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Ticket, 0, len(rows))
	for _, row := range rows {
		t := mapTicket(row)
		if err := r.hydrateDialogSnapshot(ctx, &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// hydrateDialogSnapshot overlays scenario/scenario_version/reference/mode/briefing.
func (r *TicketRepository) hydrateDialogSnapshot(ctx context.Context, t *models.Ticket) error {
	var scenario []byte
	var version, mode, briefing string
	var reference []byte
	err := r.pool.QueryRow(ctx,
		`SELECT scenario, scenario_version, mode, briefing,
		 COALESCE((SELECT row_to_json(ra) FROM ticket_reference_answers ra WHERE ra.ticket_id = tickets.id), '{}')
		 FROM tickets WHERE id = $1`, t.ID).
		Scan(&scenario, &version, &mode, &briefing, &reference)
	if err != nil {
		return err
	}
	t.ScenarioJSON = string(scenario)
	t.ScenarioVersion = version
	t.Mode = mode
	t.Briefing = briefing
	t.Reference = string(reference)
	return nil
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
	if err := updateDialogSnapshotTx(ctx, tx, ticket); err != nil {
		return err
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
	if *v > math.MaxInt32 { // int→int32 never truncates silently (spec P)
		n := int32(math.MaxInt32)
		return &n
	}
	if *v < math.MinInt32 {
		n := int32(math.MinInt32)
		return &n
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
