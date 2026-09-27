package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"traineebox/internal/audio"
	"traineebox/internal/platform/config"
	"traineebox/internal/testkit"
	ticketinfra "traineebox/internal/tickets/infrastructure"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedAudioTicket(t *testing.T, pool *pgxpool.Pool, snapshot string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	user := testkit.SeedUser(t, pool, "audio-seed-user", "password", "teacher")
	var topicID, moduleID, lessonID, variantID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO topics (title, created_by) VALUES ('Topic', $1) RETURNING id`, user.ID).Scan(&topicID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO modules (title, created_by) VALUES ('Module', $1) RETURNING id`, user.ID).Scan(&moduleID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO lessons (module_id, title) VALUES ($1, 'Lesson') RETURNING id`, moduleID).Scan(&lessonID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO variants (lesson_id, title) VALUES ($1, 'Variant') RETURNING id`, lessonID).Scan(&variantID); err != nil {
		t.Fatal(err)
	}
	var ticketID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO tickets (variant_id, topic_id, title, created_by, scenario) VALUES ($1, $2, 'Ticket', $3, $4::jsonb) RETURNING id`, variantID, topicID, user.ID, snapshot).Scan(&ticketID); err != nil {
		t.Fatal(err)
	}
	return ticketID
}

func TestPrerenderReusedReadyAndSpokenUrges(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM slot_questions`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM dialog_slots`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO dialog_slots (id, label) VALUES ('addr.street', 'Street'), ('addr.house', 'House')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO slot_questions (slot_id, question, source) VALUES ('addr.street', 'Say street now', 'urge')`); err != nil {
		t.Fatal(err)
	}
	ticketID := seedAudioTicket(t, pool, `{"id":"s","opening":"hello","facts":[{"key":"street","slot":"addr.street","answers":{"plain":"Main"},"numbers":[916,126]}]}`)
	requests := make(chan audio.EnsureRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req audio.EnsureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode ensure request: %v", err)
		}
		requests <- req
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}))
	defer server.Close()
	if err := prerenderTicket(ctx, pool, config.Config{AudioURL: server.URL}, ticketID); err != nil {
		t.Fatal(err)
	}
	req := <-requests
	if req.Slots["addr.street"] != "Street" || req.UrgeSlots["addr.street"] != "Say street now" || req.UrgeSlots["addr.house"] != "House" {
		t.Fatalf("ensure slot mappings = labels %v, urges %v", req.Slots, req.UrgeSlots)
	}
	var status, digest string
	var scenario []byte
	if err := pool.QueryRow(ctx, `SELECT audio_status, audio_digest, scenario FROM tickets WHERE id = $1`, ticketID).Scan(&status, &digest, &scenario); err != nil {
		t.Fatal(err)
	}
	if status != "ready" || digest != req.ScenarioDigest || string(scenario) == "" {
		t.Fatalf("stored audio status = %q, digest = %q, request digest = %q", status, digest, req.ScenarioDigest)
	}
	var saved struct {
		Facts []struct {
			Numbers []int64 `json:"numbers"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(scenario, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Facts) != 1 || len(saved.Facts[0].Numbers) != 2 || saved.Facts[0].Numbers[0] != 916 || saved.Facts[0].Numbers[1] != 126 {
		t.Fatalf("stored fact numbers = %+v", saved.Facts)
	}
	updated, err := ticketinfra.NewTicketRepository(pool).UpdateAudioStatus(ctx, ticketID, digest, "stale")
	if err != nil || updated {
		t.Fatalf("delayed stale update = %v, %v; want ignored", updated, err)
	}
	if err := pool.QueryRow(ctx, `SELECT audio_status FROM tickets WHERE id = $1`, ticketID).Scan(&status); err != nil || status != "ready" {
		t.Fatalf("status after delayed stale = %q, %v", status, err)
	}
}

func TestPrerenderDelayedErrorCannotDowngradeReady(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM slot_questions`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM dialog_slots`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO dialog_slots (id, label) VALUES ('addr.street', 'Street')`); err != nil {
		t.Fatal(err)
	}
	ticketID := seedAudioTicket(t, pool, `{"id":"s","opening":"hello","facts":[{"key":"street","slot":"addr.street","answers":{"plain":"Main"}}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := pool.Exec(ctx, `UPDATE tickets SET audio_status = 'ready' WHERE id = $1`, ticketID); err != nil {
			t.Errorf("mark ticket ready: %v", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	if err := prerenderTicket(ctx, pool, config.Config{AudioURL: server.URL}, ticketID); err == nil {
		t.Fatal("failed ensure unexpectedly succeeded")
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT audio_status FROM tickets WHERE id = $1`, ticketID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "ready" {
		t.Fatalf("delayed failure overwrote ready status: %q", status)
	}
}
