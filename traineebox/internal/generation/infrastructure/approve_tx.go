package infrastructure

import (
	"context"
	"encoding/json"
	"math"
	"strings"

	"traineebox/internal/generation/domain/errs"
	ticketsmodels "traineebox/internal/tickets/domain/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApproveAtomically writes ticket + reference + scenario and marks the job
// published in ONE transaction. The old two-step path (CreateWithReference
// then SaveCAS) could publish a ticket while the job stayed ready.
// ponytail: raw SQL here to avoid regenerating sqlc for the new columns;
// move to sqlc queries when the schema settles.
func ApproveAtomically(
	ctx context.Context,
	pool *pgxpool.Pool,
	ticket ticketsmodels.Ticket,
	ref ticketsmodels.ReferenceAnswer,
	jobID uuid.UUID,
	expectedStatus string,
	expectedVersion int,
) error {
	draftRef := validScenarioJSON(ticket.ScenarioJSON)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`INSERT INTO tickets (id, group_id, title, body, max_attempts, available_from, available_until,
			duration_seconds, created_by, created_at, scenario, scenario_version, mode, briefing)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, $11::jsonb, $12, $13, $14)`,
		ticket.ID, ticket.GroupID, ticket.Title.String(), ticket.Body,
		toInt32(ticket.MaxAttempts), ticket.AvailableFrom, ticket.AvailableUntil,
		toInt32(ticket.DurationSeconds), ticket.CreatedBy, ticket.CreatedAt,
		draftRef, ticket.ScenarioVersion, firstNonEmpty(ticket.Mode, "voice"), ticket.Briefing,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO ticket_reference_answers
		 (ticket_id, incident_type_code, applicant_last_name, applicant_first_name, caller_number, dictated_number)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (ticket_id) DO UPDATE SET
			incident_type_code = EXCLUDED.incident_type_code,
			applicant_last_name = EXCLUDED.applicant_last_name,
			applicant_first_name = EXCLUDED.applicant_first_name,
			caller_number = EXCLUDED.caller_number,
			dictated_number = EXCLUDED.dictated_number`,
		ref.TicketID, ref.IncidentTypeCode, ref.ApplicantLastName, ref.ApplicantFirstName,
		ref.CallerNumber, ref.DictatedNumber,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM reference_answer_tags WHERE ticket_id = $1`, ref.TicketID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM reference_answer_services WHERE ticket_id = $1`, ref.TicketID); err != nil {
		return err
	}
	for _, tag := range ref.TagCodes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO reference_answer_tags (ticket_id, tag_code) VALUES ($1,$2)`, ref.TicketID, tag); err != nil {
			return err
		}
	}
	for _, svc := range ref.ServiceCodes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO reference_answer_services (ticket_id, service_code) VALUES ($1,$2)`, ref.TicketID, svc); err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx,
		`UPDATE ticket_generation_jobs SET status = 'published', version = version + 1,
			published_ticket_id = $1, error_message = '', claimed_by = '', claimed_at = NULL,
			lease_until = NULL, updated_at = now()
		 WHERE id = $2 AND status = $3 AND version = $4`,
		ticket.ID, jobID, expectedStatus, expectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrConflict
	}
	return tx.Commit(ctx)
}

// validScenarioJSON defaults empty/garbage to '{}' so the jsonb cast never
// trips (spec C/P); real snapshots arrive via PUT /scenario before approve.
func validScenarioJSON(s string) string {
	if strings.TrimSpace(s) == "" || !json.Valid([]byte(s)) {
		return "{}"
	}
	return s
}

func firstNonEmpty(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

func toInt32(v *int) *int32 {
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
