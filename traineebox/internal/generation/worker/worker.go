package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"traineebox/internal/generation/domain/models"
	genvo "traineebox/internal/generation/domain/value_objects"
	geninfra "traineebox/internal/generation/infrastructure"
	ticketsmodels "traineebox/internal/tickets/domain/models"
	ticketsrepos "traineebox/internal/tickets/domain/repositories"

	"github.com/google/uuid"
)

// Worker drives building_dialog/checking_dialog jobs to ready (spec K).
// Transport pinned: ticketgen is a pure function over HTTP POST /draft
// (spine bet); ticketgen never polls jobs. Dialog lint runs wherever a
// snapshot exists; approve stays a teacher action via ApproveJob+AtomicPublish.
type Worker struct {
	Jobs         *geninfra.JobRepository
	Catalog      ticketsrepos.CatalogRepository
	TicketgenURL string
	DialogURLs   []string
	ServiceToken string
	WorkerID     string
	LeaseSeconds int
	PollInterval time.Duration
}

func FromEnv(jobs *geninfra.JobRepository, catalog ticketsrepos.CatalogRepository) *Worker {
	urls := []string{}
	if u := strings.TrimSpace(os.Getenv("DIALOG_URL")); u != "" {
		urls = append(urls, u)
	}
	if extra := os.Getenv("DIALOG_URLS"); extra != "" {
		for _, u := range strings.Split(extra, ",") {
			if s := strings.TrimSpace(u); s != "" {
				urls = append(urls, s)
			}
		}
	}
	lease := 120
	if v, err := strconv.Atoi(os.Getenv("TICKETGEN_LEASE_SECONDS")); err == nil && v > 0 {
		lease = v
	}
	poll := 2 * time.Second
	if v, err := strconv.Atoi(os.Getenv("TICKETGEN_POLL_SECONDS")); err == nil && v > 0 {
		poll = time.Duration(v) * time.Second
	}
	return &Worker{
		Jobs:         jobs,
		Catalog:      catalog,
		TicketgenURL: strings.TrimSpace(os.Getenv("TICKETGEN_URL")),
		DialogURLs:   urls,
		ServiceToken: os.Getenv("INTERNAL_SERVICE_TOKEN"),
		WorkerID:     workerID(),
		LeaseSeconds: lease,
		PollInterval: poll,
	}
}

func workerID() string {
	if v := os.Getenv("TICKETGEN_WORKER_ID"); v != "" {
		return v
	}
	return "traineebox-" + uuid.New().String()[:8]
}

// Loop claims and drives jobs until ctx stops. No ticketgen URL → disabled.
func (w *Worker) Loop(ctx context.Context) {
	if w.TicketgenURL == "" {
		log.Print("generation worker disabled (TICKETGEN_URL unset)")
		return
	}
	log.Printf("generation worker %s driving via %s", w.WorkerID, w.TicketgenURL)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		drove, err := w.DriveOnce(ctx)
		if err != nil {
			log.Printf("generation worker: %v", redact(err.Error()))
		}
		if !drove {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.PollInterval):
			}
		}
	}
}

// DriveOnce claims one job and runs draft → check → ready/failed.
func (w *Worker) DriveOnce(ctx context.Context) (bool, error) {
	job, claimed, err := w.Jobs.ClaimNext(ctx, w.WorkerID, w.LeaseSeconds)
	if err != nil {
		return false, err
	}
	if !claimed {
		return false, nil
	}
	expectedStatus, expectedVersion := job.Status.String(), job.Version
	cas := func(status genvo.JobStatus, msg string, clear bool) error {
		job.Status = status
		job.ErrorMessage = msg
		if clear {
			job.ClaimedBy = ""
			job.ClaimedAt = nil
			job.LeaseUntil = nil
		}
		if err := w.Jobs.SaveCAS(ctx, job, expectedStatus, expectedVersion); err != nil {
			return err
		}
		expectedStatus, expectedVersion = job.Status.String(), expectedVersion+1
		return nil
	}
	// building_dialog: pure draft over HTTP (no DB in ticketgen).
	// One retry with backoff: a ticketgen restart must not fail the job.
	draft, err := postDraft(ctx, w.TicketgenURL, job.Prompt)
	if err != nil {
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			_ = cas(genvo.JobStatusFailed, trunc(redact(err.Error())), true)
			return true, ctx.Err()
		case <-timer.C:
		}
		draft, err = postDraft(ctx, w.TicketgenURL, job.Prompt)
	}
	if err != nil {
		_ = cas(genvo.JobStatusFailed, trunc(redact(err.Error())), true)
		return true, nil
	}
	job.DraftTitle, job.ScenarioText, job.DraftReference = draft.Title, draft.Scenario, draft.Reference.Normalize()
	if err := cas(genvo.JobStatusCheckingDialog, "", false); err != nil {
		return true, nil // lost claim; another worker owns it now
	}
	// checking_dialog: catalog + PII gate, dialog lint where a snapshot exists.
	if err := w.check(ctx, job); err != nil {
		_ = cas(genvo.JobStatusFailed, trunc(redact(err.Error())), true)
		return true, nil
	}
	if err := cas(genvo.JobStatusReady, "", true); err != nil {
		return true, nil
	}
	return true, nil
}

type draftOut struct {
	Title     string
	Scenario  string
	Reference models.DraftReference
}

func postDraft(ctx context.Context, base, prompt string) (draftOut, error) {
	body, _ := json.Marshal(map[string]string{"prompt": prompt})
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimSuffix(base, "/")+"/draft", bytes.NewReader(body))
	if err != nil {
		return draftOut{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return draftOut{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return draftOut{}, fmt.Errorf("ticketgen draft: %s", resp.Status)
	}
	var out struct {
		DraftTitle     string                `json:"draft_title"`
		ScenarioText   string                `json:"scenario_text"`
		DraftReference models.DraftReference `json:"draft_reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return draftOut{}, err
	}
	if strings.TrimSpace(out.DraftTitle) == "" || strings.TrimSpace(out.ScenarioText) == "" {
		return draftOut{}, fmt.Errorf("ticketgen returned empty draft")
	}
	return draftOut{Title: out.DraftTitle, Scenario: out.ScenarioText, Reference: out.DraftReference}, nil
}

// check runs the checking_dialog gate: reference must validate against the
// catalog and PII must match the card. Dialog snapshot lint runs at
// PUT /scenario (proxied to the worker) — generated drafts carry prose,
// not slot facts, so there is nothing to lint yet.
func (w *Worker) check(ctx context.Context, job models.Job) error {
	draft := job.DraftReference.Normalize()
	if draft.IncidentTypeCode == "" {
		return fmt.Errorf("empty incident_type_code")
	}
	if _, err := w.Catalog.FindIncidentTypeByCode(ctx, draft.IncidentTypeCode); err != nil {
		return fmt.Errorf("unknown type %q", draft.IncidentTypeCode)
	}
	groups, err := w.Catalog.ListTagGroupsByType(ctx, draft.IncidentTypeCode)
	if err != nil {
		return err
	}
	ref, err := ticketsmodels.NewReferenceAnswer(
		uuid.New(), draft.IncidentTypeCode, draft.TagCodes, draft.ServiceCodes,
		draft.ApplicantLastName, draft.ApplicantFirstName, draft.CallerNumber, draft.DictatedNumber)
	if err != nil {
		return err
	}
	if err := ref.ValidateTagSelection(groups); err != nil {
		return fmt.Errorf("tag selection: %w", err)
	}
	if len(draft.ServiceCodes) > 0 {
		ok, err := w.Catalog.ServiceExists(ctx, draft.ServiceCodes)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("unknown service code")
		}
	}
	// PII matches card: every card field must be filled.
	if draft.ApplicantLastName == "" || draft.ApplicantFirstName == "" ||
		draft.CallerNumber == "" || draft.DictatedNumber == "" {
		return fmt.Errorf("incomplete PII on card")
	}
	return w.lintDraft(ctx, job)
}

// lintDraft runs dialog lint when the draft carries scenario facts (a JSON
// snapshot). Prose drafts have nothing to lint — the skip is recorded in the
// log and the gate passes (slot-fact authoring stays teacher-side).
func (w *Worker) lintDraft(ctx context.Context, job models.Job) error {
	raw := strings.TrimSpace(job.ScenarioText)
	var probe struct {
		Facts []json.RawMessage `json:"facts"`
	}
	if !strings.HasPrefix(raw, "{") ||
		json.Unmarshal([]byte(raw), &probe) != nil || len(probe.Facts) == 0 {
		log.Printf("generation worker %s job %s: dialog lint skipped (no scenario facts)",
			w.WorkerID, job.ID)
		return nil
	}
	if len(w.DialogURLs) == 0 {
		log.Printf("generation worker %s job %s: dialog lint skipped (no dialog workers)",
			w.WorkerID, job.ID)
		return nil
	}
	var last error
	for _, base := range w.DialogURLs {
		out, err := postLintURL(ctx, base, raw, w.ServiceToken)
		if err != nil {
			last = err
			continue
		}
		if bad, _ := out["unreachable"].([]any); len(bad) > 0 {
			return fmt.Errorf("dialog lint: unreachable critical %v", bad)
		}
		return nil
	}
	return fmt.Errorf("dialog lint unavailable: %v", last)
}

// lintHTTP bounds the lint call; the shared ticketgen postDraft path keeps
// its own 120s ctx timeout plus one retry.
var lintHTTP = &http.Client{Timeout: 10 * time.Second}

func postLintURL(ctx context.Context, base, snapshot string, token string) (map[string]any, error) {
	body, _ := json.Marshal(map[string]any{"scenario": json.RawMessage(snapshot)})
	req, err := http.NewRequestWithContext(ctx, "POST",
		strings.TrimSuffix(base, "/")+"/scenarios/lint", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Service-Token", token)
	}
	resp, err := lintHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dialog lint: %s", resp.Status)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func trunc(s string) string {
	// Rune-safe: byte slicing would split multi-byte runes (Cyrillic).
	if r := []rune(s); len(r) > 1000 {
		return string(r[:1000])
	}
	return s
}

// piiDigits masks phone-like runs so prompts/drafts never land in logs or
// job error_message unredacted.
var piiDigits = regexp.MustCompile(`\d{7,}`)

func redact(s string) string {
	return piiDigits.ReplaceAllString(s, "***")
}
