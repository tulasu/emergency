package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"traineebox/internal/auth/application"
	dialoginfra "traineebox/internal/dialog/infrastructure"
	"traineebox/internal/platform/config"
	"traineebox/internal/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

// demoOptions carries every seed-demo flag. All defaults are idempotent on re-run.
type demoOptions struct {
	userLogin        string
	userPassword     string
	endpoint         string
	endpointPassword string
	scenarioID       string
	attemptFile      string
	deadlineHours    int
	scenarioSource   string
	prerender        bool
}

func newSeedDemoCmd(cfg config.Config) *cobra.Command {
	var opts demoOptions
	cmd := &cobra.Command{
		Use:   "seed-demo",
		Short: "Create demo student + SIP endpoint + ticket + in_progress attempt for `just call`",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return seedDemo(cmd.Context(), cfg, opts)
		},
	}
	cmd.Flags().StringVar(&opts.userLogin, "user-login", "demo", "demo student login")
	cmd.Flags().StringVar(&opts.userPassword, "user-password", "demo1234", "demo student password (≥8 chars: PasswordHasher minimum)")
	cmd.Flags().StringVar(&opts.endpoint, "endpoint", "demo", "SIP endpoint name (default = user-login: MicroSIP registers as the same name you log into traineebox with)")
	cmd.Flags().StringVar(&opts.endpointPassword, "endpoint-password", "demo1234", "SIP auth password stored in user_sip_endpoints.password_hash (default = user-password, same creds for HTTP login and SIP register)")
	cmd.Flags().StringVar(&opts.scenarioID, "scenario-id", "demo_call", "scenario id baked into the demo ticket (marker for find-or-create)")
	cmd.Flags().StringVar(&opts.attemptFile, "attempt-file", "/tmp/seed-demo-attempt-id", "ephemeral path inside the run-container; printed in summary so `just call` can take it as $2")
	cmd.Flags().IntVar(&opts.deadlineHours, "deadline-hours", 24, "attempt deadline in hours from now")
	cmd.Flags().StringVar(&opts.scenarioSource, "scenario-source", "data/scenarios/bilet01_call01.json", "corpus scenario to base the demo on (relative to embed root)")
	cmd.Flags().BoolVar(&opts.prerender, "prerender", true, "ensure audio for the persisted demo ticket snapshot")
	return cmd
}

func seedDemo(ctx context.Context, cfg config.Config, opts demoOptions) error {
	// Canonical snapshot built before the tx: pure function of the embedded bytes.
	raw, err := embeddedScenarios.ReadFile(opts.scenarioSource)
	if err != nil {
		return fmt.Errorf("read scenario %q: %w", opts.scenarioSource, err)
	}
	snapshot, err := buildDemoSnapshot(raw, opts.scenarioID)
	if err != nil {
		return err
	}
	hash, err := (application.PasswordHasher{}).Hash(opts.userPassword)
	if err != nil {
		return err
	}
	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 2. Upsert student user (password refresh on re-run).
	var userID uuid.UUID
	if err := tx.QueryRow(ctx,
		`INSERT INTO users (login, password_hash, role) VALUES ($1, $2, 'student')
		 ON CONFLICT (login) DO UPDATE SET password_hash = EXCLUDED.password_hash
		 RETURNING id`,
		opts.userLogin, hash.String()).Scan(&userID); err != nil {
		return err
	}
	// 3. Upsert SIP endpoint (raw password: asterisk reads it via ps_auths).
	if _, err := tx.Exec(ctx,
		`INSERT INTO user_sip_endpoints (user_id, endpoint, password_hash) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE SET endpoint = EXCLUDED.endpoint,
		   password_hash = EXCLUDED.password_hash, enabled = true, updated_at = now()`,
		userID, opts.endpoint, opts.endpointPassword); err != nil {
		return err
	}
	var ticketID, variantID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id, variant_id FROM tickets WHERE scenario->>'id' = $1 LIMIT 1`, opts.scenarioID).Scan(&ticketID, &variantID)
	switch {
	case err == nil:
	case err == pgx.ErrNoRows:
		topicID, moduleID, lessonID, err := ensureDemoCurriculum(ctx, tx, userID)
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO variants (lesson_id, title, position) VALUES ($1, 'Demo variant', 0) RETURNING id`,
			lessonID).Scan(&variantID); err != nil {
			return err
		}
		_ = moduleID
		if err := tx.QueryRow(ctx,
			`INSERT INTO tickets (variant_id, topic_id, title, body, created_by, scenario, scenario_version, mode)
			 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, 'voice') RETURNING id`,
			variantID, topicID, "Demo call", "", userID, string(snapshot), opts.scenarioID).Scan(&ticketID); err != nil {
			return err
		}
	default:
		return err
	}
	var attemptID uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT id FROM ticket_attempts
		 WHERE variant_id = $1 AND user_id = $2 AND status = 'in_progress'
		   AND (deadline_at IS NULL OR deadline_at > now())
		 LIMIT 1`,
		variantID, userID).Scan(&attemptID)
	switch {
	case err == nil:
	case err == pgx.ErrNoRows:
		if err := tx.QueryRow(ctx,
			`INSERT INTO ticket_attempts (variant_id, user_id, granted_by, attempt_no, status, started_at, deadline_at)
			 VALUES ($1, $2, $2, COALESCE((SELECT MAX(attempt_no) FROM ticket_attempts WHERE variant_id = $1 AND user_id = $2), 0) + 1,
			         'in_progress', now(), now() + $3 * interval '1 hour')
			 RETURNING id`,
			variantID, userID, opts.deadlineHours).Scan(&attemptID); err != nil {
			return err
		}
	default:
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	// 4. Bank reload is a side effect: outside the tx, best-effort.
	// ponytail: single 2-attempt call, no partial fan-out retry.
	if err := bankReload(ctx, cfg.HTTPAddr, os.Getenv("INTERNAL_SERVICE_TOKEN")); err != nil {
		fmt.Fprintf(os.Stderr, "warn: bank reload failed (attempt may use stale digest): %v\n", err)
	}
	if opts.prerender {
		if err := prerenderTicket(ctx, pool, cfg, ticketID); err != nil {
			fmt.Fprintf(os.Stderr, "warn: audio prerender ticket %s: %v\n", ticketID, err)
		}
	}
	// 5. Current canon digest (recomputed by reload's ReplaceBank).
	_, digest, err := dialoginfra.NewBankRepository(pool).BankVersion(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: bank version: %v\n", err)
		digest = ""
	}
	// 8. Write attempt id locally (ephemeral — gone when this run-container exits).
	// Primary path: `just call <attempt_id>` takes it from the summary line below.
	if err := os.WriteFile(opts.attemptFile, []byte(attemptID.String()), 0o644); err != nil {
		// non-fatal: summary line still carries the id
		fmt.Fprintf(os.Stderr, "warn: write %s: %v\n", opts.attemptFile, err)
	}

	// 9. Summary.
	fmt.Printf("demo: user=%s endpoint=%s ticket=%s attempt=%s scenario=%s bank_digest=%s\n",
		opts.userLogin, opts.endpoint, ticketID, attemptID, opts.scenarioID, digest)
	fmt.Printf("wrote %s\n", opts.attemptFile)
	return nil
}

// ensureDemoGroup finds or creates the demo group (groups.name has no unique
// constraint, so find-then-insert; sequential re-runs are idempotent).
// ponytail: no owner_id column — group ownership lives in group_members, which
// the call path doesn't need, so we only create the bare group.
func ensureDemoCurriculum(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (topicID, moduleID, lessonID uuid.UUID, err error) {
	err = tx.QueryRow(ctx, `SELECT id FROM topics WHERE title = 'Demo topic' LIMIT 1`).Scan(&topicID)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `INSERT INTO topics (title, created_by) VALUES ('Demo topic', $1) RETURNING id`, userID).Scan(&topicID)
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	err = tx.QueryRow(ctx, `SELECT id FROM modules WHERE title = 'Demo module' LIMIT 1`).Scan(&moduleID)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `INSERT INTO modules (title, description, created_by) VALUES ('Demo module', '', $1) RETURNING id`, userID).Scan(&moduleID)
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	err = tx.QueryRow(ctx, `SELECT id FROM lessons WHERE module_id = $1 AND title = 'Demo lesson' LIMIT 1`, moduleID).Scan(&lessonID)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx,
			`INSERT INTO lessons (module_id, title, position, duration_seconds) VALUES ($1, 'Demo lesson', 0, NULL) RETURNING id`,
			moduleID).Scan(&lessonID)
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	return topicID, moduleID, lessonID, nil
}

// bankReload POSTs /bank/reload with the internal service token (2 attempts).
// Адрес — compose service name «traineebox»: initializer бежит в одноразовом
// контейнере через `docker compose run --rm`, и localhost внутри него — сам
// initializer-контейнер, а не traineebox-app.
func bankReload(ctx context.Context, httpAddr, token string) error {
	port := strings.TrimPrefix(httpAddr, ":")
	if port == "" {
		port = "8080"
	}
	url := "http://traineebox:" + port + "/bank/reload"
	client := &http.Client{Timeout: 5 * time.Second}
	body := []byte(`{"version":"demo-v1"}`)

	var lastErr error
	for i := 0; i < 2; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("X-Service-Token", token)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("bank reload: %s", resp.Status)
	}
	return lastErr
}

// ---- scenario canonicalization (mirrors dialog/validator.py) ----

type biletScenario struct {
	Persona struct {
		Opening string `json:"opening"`
	} `json:"persona"`
	Critical []string `json:"critical"`
	Facts    []struct {
		ID         string            `json:"id"`
		Slot       string            `json:"slot"`
		Answers    map[string]string `json:"answers"`
		Numbers    []json.Number     `json:"numbers"`
		Requires   []string          `json:"requires"`
		Disclosure string            `json:"disclosure"`
	} `json:"facts"`
}

type canonFact struct {
	Key        string            `json:"key"`
	Slot       string            `json:"slot"`
	Answers    map[string]string `json:"answers"`
	Numbers    []json.Number     `json:"numbers"`
	Requires   []string          `json:"requires"`
	Disclosure string            `json:"disclosure"`
}

type canonScenario struct {
	ID       string      `json:"id"`
	Opening  string      `json:"opening"`
	Mode     string      `json:"mode"`
	Critical []string    `json:"critical"`
	Facts    []canonFact `json:"facts"`
}

// buildDemoSnapshot strips a corpus scenario down to the AD-7 snapshot canon:
// id (overridden to the demo marker), opening, mode, critical, facts with
// key/slot/answers.plain + optional numbers/requires/disclosure.
func buildDemoSnapshot(raw []byte, scenarioID string) (json.RawMessage, error) {
	var b biletScenario
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, fmt.Errorf("parse demo scenario: %w", err)
	}
	facts := make([]canonFact, 0, len(b.Facts))
	for _, f := range b.Facts {
		disc := f.Disclosure
		if disc == "" {
			disc = "volunteered"
		}
		numbers := f.Numbers
		if numbers == nil {
			numbers = []json.Number{}
		}
		requires := f.Requires
		if requires == nil {
			requires = []string{}
		}
		facts = append(facts, canonFact{
			Key:        f.ID,
			Slot:       f.Slot,
			Answers:    f.Answers,
			Numbers:    numbers,
			Requires:   requires,
			Disclosure: disc,
		})
	}
	critical := b.Critical
	if critical == nil {
		critical = []string{}
	}
	return json.Marshal(canonScenario{
		ID:       scenarioID,
		Opening:  b.Persona.Opening,
		Mode:     "voice",
		Critical: critical,
		Facts:    facts,
	})
}
