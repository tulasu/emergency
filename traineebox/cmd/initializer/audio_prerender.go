package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"traineebox/internal/audio"
	"traineebox/internal/platform/config"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func prerenderTicket(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, ticketID uuid.UUID) error {
	if strings.TrimSpace(cfg.AudioURL) == "" {
		return nil
	}
	labels, err := slotLabels(ctx, pool)
	if err != nil {
		return err
	}
	urges, err := slotUrges(ctx, pool, labels)
	if err != nil {
		return err
	}
	return prerenderTicketWithLabels(ctx, pool, audio.Client{
		BaseURL: cfg.AudioURL, ServiceToken: strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_TOKEN")),
	}, ticketID, labels, urges)
}

func backfillTicketAudio(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) (attempted, failed int, err error) {
	if strings.TrimSpace(cfg.AudioURL) == "" {
		return 0, 0, nil
	}
	labels, err := slotLabels(ctx, pool)
	if err != nil {
		return 0, 0, err
	}
	urges, err := slotUrges(ctx, pool, labels)
	if err != nil {
		return 0, 0, err
	}
	rows, err := pool.Query(ctx, `SELECT id FROM tickets ORDER BY created_at`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	client := audio.Client{BaseURL: cfg.AudioURL, ServiceToken: strings.TrimSpace(os.Getenv("INTERNAL_SERVICE_TOKEN"))}
	for rows.Next() {
		var ticketID uuid.UUID
		if err := rows.Scan(&ticketID); err != nil {
			return attempted, failed, err
		}
		attempted++
		if err := prerenderTicketWithLabels(ctx, pool, client, ticketID, labels, urges); err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "warn: audio prerender ticket %s: %v\n", ticketID, err)
		}
	}
	return attempted, failed, rows.Err()
}

func slotLabels(ctx context.Context, pool *pgxpool.Pool) (map[string]string, error) {
	rows, err := pool.Query(ctx, `SELECT id, label FROM dialog_slots`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	labels := map[string]string{}
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, err
		}
		labels[id] = label
	}
	return labels, rows.Err()
}

func slotUrges(ctx context.Context, pool *pgxpool.Pool, labels map[string]string) (map[string]string, error) {
	urges := make(map[string]string, len(labels))
	for id, label := range labels {
		urges[id] = label
	}
	rows, err := pool.Query(ctx, `SELECT slot_id, question FROM slot_questions WHERE source = 'urge' ORDER BY slot_id, question`)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	defer rows.Close()
	for rows.Next() {
		var id, urge string
		if err := rows.Scan(&id, &urge); err != nil {
			return nil, err
		}
		if _, exists := urges[id]; exists && urge != "" && !seen[id] {
			urges[id] = urge
			seen[id] = true
		}
	}
	return urges, rows.Err()
}

func prerenderTicketWithLabels(ctx context.Context, pool *pgxpool.Pool, client audio.Client, ticketID uuid.UUID, labels, urges map[string]string) error {
	var snapshot string
	if err := pool.QueryRow(ctx, `SELECT scenario::text FROM tickets WHERE id = $1`, ticketID).Scan(&snapshot); err != nil {
		return err
	}
	if strings.TrimSpace(snapshot) == "" || strings.TrimSpace(snapshot) == "{}" || strings.TrimSpace(snapshot) == "null" {
		fmt.Fprintf(os.Stderr, "audio prerender skipped for ticket %s: empty snapshot\n", ticketID)
		_, err := pool.Exec(ctx, `UPDATE tickets SET audio_digest = '', audio_status = 'none' WHERE id = $1`, ticketID)
		return err
	}
	var probe struct {
		Facts []json.RawMessage `json:"facts"`
	}
	if err := json.Unmarshal([]byte(snapshot), &probe); err != nil {
		return err
	}
	if len(probe.Facts) == 0 {
		fmt.Fprintf(os.Stderr, "audio prerender skipped for ticket %s: snapshot has no facts\n", ticketID)
		_, err := pool.Exec(ctx, `UPDATE tickets SET audio_digest = '', audio_status = 'none' WHERE id = $1`, ticketID)
		return err
	}
	knownSlots := make(map[string]bool, len(labels))
	for slotID := range labels {
		knownSlots[slotID] = true
	}
	canonical, err := audio.Canonicalize([]byte(snapshot), knownSlots)
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `UPDATE tickets SET scenario = $2::jsonb, audio_digest = $3, audio_status = 'pending' WHERE id = $1`, ticketID, canonical.JSON, canonical.Digest); err != nil {
		return err
	}
	result, err := client.EnsureWithRetry(ctx, audio.EnsureRequest{
		TicketID: ticketID, ScenarioDigest: canonical.Digest, Scenario: canonical.JSON, Slots: labels, UrgeSlots: urges,
	})
	if err != nil {
		_, updateErr := pool.Exec(ctx, `UPDATE tickets SET audio_status = 'stale' WHERE id = $1 AND audio_digest = $2 AND audio_status = 'pending'`, ticketID, canonical.Digest)
		if updateErr != nil {
			return fmt.Errorf("ensure: %w (mark stale: %v)", err, updateErr)
		}
		return err
	}
	if result.StatusCode == http.StatusAccepted && result.Status == "pending" {
		return nil
	}
	status := "stale"
	if result.Status == "error" {
		status = "error"
	}
	if (result.StatusCode == http.StatusOK || result.StatusCode == http.StatusAccepted) && result.Status == "ready" {
		status = "ready"
	}
	_, err = pool.Exec(ctx, `UPDATE tickets SET audio_status = $3 WHERE id = $1 AND audio_digest = $2 AND (audio_status = 'pending' OR $3 = 'ready')`, ticketID, canonical.Digest, status)
	return err
}
