package infrastructure_test

import (
	"context"
	"strings"
	"testing"
	"time"

	genmodels "traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/repositories"
	genvo "traineebox/internal/generation/domain/value_objects"
	geninfra "traineebox/internal/generation/infrastructure"
	"traineebox/internal/testkit"

	"github.com/google/uuid"
)

func TestApproveAtomically(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	ctx := context.Background()

	teacher := testkit.SeedUser(t, pool, "atomowner", "password1", "teacher")
	topicID := uuid.New()
	moduleID := uuid.New()
	lessonID := uuid.New()
	variantID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO topics (id, title, created_by) VALUES ($1, 'Atom topic', $2)`, topicID, teacher.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO modules (id, title, description, created_by) VALUES ($1, 'Atom module', '', $2)`, moduleID, teacher.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO lessons (id, module_id, title, position) VALUES ($1, $2, 'Atom lesson', 0)`, lessonID, moduleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO variants (id, lesson_id, title, position) VALUES ($1, $2, 'Atom variant', 0)`, variantID, lessonID); err != nil {
		t.Fatal(err)
	}

	jobsRepo := geninfra.NewJobRepository(pool)
	job, err := genmodels.NewJob(variantID, topicID, teacher.ID, "пожар на складе")
	if err != nil {
		t.Fatal(err)
	}
	if err := jobsRepo.Create(ctx, job); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE ticket_generation_jobs SET
			status = 'ready', version = version + 1,
			draft_title = 'Пожар', scenario_text = 'Горит бак',
			draft_reference = $2::jsonb, error_message = '', updated_at = now()
		WHERE id = $1`, job.ID, `{"incident_type_code":"101"}`); err != nil {
		t.Fatal(err)
	}
	ready, err := jobsRepo.FindByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}

	ticketID := uuid.New()
	draft := repositories.PublishDraft{
		TicketID:        ticketID,
		VariantID:       variantID,
		TopicID:         topicID,
		Title:           "Пожар на складе",
		Body:            "b",
		CreatedBy:       teacher.ID,
		CreatedAt:       time.Now().UTC(),
		ScenarioJSON:    `{"id":"s1","opening":"Алло","facts":[]}`,
		ScenarioVersion: "v3",
		Mode:            "text",
		Briefing:        "briefing line",
		Reference: genmodels.DraftReference{
			IncidentTypeCode:   "101",
			TagCodes:           []string{"where_street"},
			ServiceCodes:       []string{"sluzhba_101"},
			ApplicantLastName:  "Петров",
			ApplicantFirstName: "Петр",
			CallerNumber:       "79001112233",
			DictatedNumber:     "79001112233",
		},
		JobID:           job.ID,
		ExpectedStatus:  ready.Status.String(),
		ExpectedVersion: ready.Version,
	}

	if err := geninfra.ApproveAtomically(ctx, pool, draft); err != nil {
		t.Fatal(err)
	}

	var scenarioVersion, mode, briefing, scenarioJSON string
	if err := pool.QueryRow(ctx,
		`SELECT scenario_version, mode, briefing, scenario::text FROM tickets WHERE id = $1`,
		ticketID).Scan(&scenarioVersion, &mode, &briefing, &scenarioJSON); err != nil {
		t.Fatal(err)
	}
	if scenarioVersion != "v3" || mode != "text" || briefing != "briefing line" {
		t.Fatalf("snapshot = %s/%s/%s, want v3/text/briefing", scenarioVersion, mode, briefing)
	}
	if !strings.Contains(scenarioJSON, "s1") {
		t.Fatalf("scenario = %q, want stored snapshot", scenarioJSON)
	}

	var typeCode string
	var tagCount, svcCount int
	if err := pool.QueryRow(ctx,
		`SELECT incident_type_code FROM ticket_reference_answers WHERE ticket_id = $1`, ticketID).Scan(&typeCode); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM reference_answer_tags WHERE ticket_id = $1`, ticketID).Scan(&tagCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM reference_answer_services WHERE ticket_id = $1`, ticketID).Scan(&svcCount); err != nil {
		t.Fatal(err)
	}
	if typeCode != "101" || tagCount != 1 || svcCount != 1 {
		t.Fatalf("reference = type=%s tags=%d svcs=%d", typeCode, tagCount, svcCount)
	}
	done, err := jobsRepo.FindByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != genvo.JobStatusPublished {
		t.Fatalf("job status = %s, want published", done.Status)
	}
	if done.PublishedTicketID == nil || *done.PublishedTicketID != ticketID {
		t.Fatalf("published_ticket_id = %v, want %s", done.PublishedTicketID, ticketID)
	}
}

