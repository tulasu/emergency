package infrastructure

import (
	"context"
	"encoding/json"
	"strings"

	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ApproveAtomically writes ticket + reference + scenario and marks the job
// published in ONE transaction. The old two-step path (CreateWithReference
// then SaveCAS) could publish a ticket while the job stayed ready.
func ApproveAtomically(
	ctx context.Context,
	pool *pgxpool.Pool,
	draft repositories.PublishDraft,
) error {
	scenarioJSON := validScenarioJSON(draft.ScenarioJSON)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`INSERT INTO tickets (id, variant_id, topic_id, title, body, created_by, created_at,
			scenario, scenario_version, mode, briefing)
		 VALUES ($1,$2,$3,$4,$5,$6,$7, $8::jsonb, $9, $10, $11)`,
		draft.TicketID, draft.VariantID, draft.TopicID, draft.Title, draft.Body,
		draft.CreatedBy, draft.CreatedAt,
		scenarioJSON, draft.ScenarioVersion, firstNonEmpty(draft.Mode, "voice"), draft.Briefing,
	); err != nil {
		return err
	}
	ref := draft.Reference.Normalize()
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
		draft.TicketID, ref.IncidentTypeCode, ref.ApplicantLastName, ref.ApplicantFirstName,
		ref.CallerNumber, ref.DictatedNumber,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM reference_answer_tags WHERE ticket_id = $1`, draft.TicketID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM reference_answer_services WHERE ticket_id = $1`, draft.TicketID); err != nil {
		return err
	}
	for _, tag := range ref.TagCodes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO reference_answer_tags (ticket_id, tag_code) VALUES ($1,$2)`, draft.TicketID, tag); err != nil {
			return err
		}
	}
	for _, svc := range ref.ServiceCodes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO reference_answer_services (ticket_id, service_code) VALUES ($1,$2)`, draft.TicketID, svc); err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx,
		`UPDATE ticket_generation_jobs SET status = 'published', version = version + 1,
			published_ticket_id = $1, error_message = '', claimed_by = '', claimed_at = NULL,
			lease_until = NULL, updated_at = now()
		 WHERE id = $2 AND status = $3 AND version = $4`,
		draft.TicketID, draft.JobID, draft.ExpectedStatus, draft.ExpectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrConflict
	}
	return tx.Commit(ctx)
}

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
