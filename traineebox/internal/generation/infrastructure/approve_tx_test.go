package infrastructure_test

import (
	"context"
	"strings"
	"testing"
	"time"

	authvo "traineebox/internal/auth/domain/value_objects"
	genmodels "traineebox/internal/generation/domain/models"
	genvo "traineebox/internal/generation/domain/value_objects"
	geninfra "traineebox/internal/generation/infrastructure"
	"traineebox/internal/testkit"
	ticketsmodels "traineebox/internal/tickets/domain/models"
	ticketsvo "traineebox/internal/tickets/domain/value_objects"
	ticketsinfra "traineebox/internal/tickets/infrastructure"

	"github.com/google/uuid"
)

// ApproveAtomically publishes ticket + reference + scenario/mode/briefing and
// the job row in ONE transaction: no ticket without its reference, no
// published ticket with a job stuck in ready.
func TestApproveAtomically(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	ctx := context.Background()

	teacher := testkit.SeedUser(t, pool, "atomowner", "password1", authvo.RoleTeacher)
	groupID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO groups (id, name) VALUES ($1, 'Atom')`, groupID); err != nil {
		t.Fatal(err)
	}

	jobsRepo := geninfra.NewJobRepository(pool)
	job, err := genmodels.NewJob(groupID, teacher.ID, "пожар на складе")
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

	title, err := ticketsvo.NewTicketTitle("Пожар на складе")
	if err != nil {
		t.Fatal(err)
	}
	ticket := ticketsmodels.Ticket{
		ID: ticketID(), GroupID: groupID, Title: title, Body: "b",
		CreatedBy: teacher.ID, CreatedAt: time.Now().UTC(),
		ScenarioJSON:    `{"id":"s1","opening":"Алло","facts":[]}`,
		ScenarioVersion: "v3", Mode: "text", Briefing: "briefing line",
	}
	ref, err := ticketsmodels.NewReferenceAnswer(ticket.ID, "101",
		[]string{"where_street"}, []string{"sluzhba_101"},
		"Петров", "Петр", "79001112233", "79001112233")
	if err != nil {
		t.Fatal(err)
	}

	if err := geninfra.ApproveAtomically(ctx, pool, ticket, ref,
		job.ID, ready.Status.String(), ready.Version); err != nil {
		t.Fatal(err)
	}

	ticketsRepo := ticketsinfra.NewTicketRepository(pool)
	got, err := ticketsRepo.FindByID(ctx, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ScenarioVersion != "v3" || got.Mode != "text" || got.Briefing != "briefing line" {
		t.Fatalf("snapshot = %+v, want v3/text/briefing", got)
	}
	if got.ScenarioJSON != ticket.ScenarioJSON {
		t.Fatalf("scenario = %q, want %q", got.ScenarioJSON, ticket.ScenarioJSON)
	}
	if !strings.Contains(got.Reference, "101") {
		t.Fatalf("reference overlay = %q, want the stored reference answer", got.Reference)
	}
	gotRef, err := ticketsRepo.FindReference(ctx, ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotRef.IncidentTypeCode != "101" || len(gotRef.TagCodes) != 1 || len(gotRef.ServiceCodes) != 1 {
		t.Fatalf("reference = %+v", gotRef)
	}
	done, err := jobsRepo.FindByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != genvo.JobStatusPublished {
		t.Fatalf("job status = %s, want published", done.Status)
	}
	if done.PublishedTicketID == nil || *done.PublishedTicketID != ticket.ID {
		t.Fatalf("published_ticket_id = %v, want %s", done.PublishedTicketID, ticket.ID)
	}
}

func ticketID() uuid.UUID { return uuid.New() }
