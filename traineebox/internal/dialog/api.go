package dialog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	ticketserrs "traineebox/internal/tickets/domain/errs"
)

// dialogHTTP bounds worker fan-out; DefaultClient would hang reload/lint forever.
var dialogHTTP = &http.Client{Timeout: 10 * time.Second}

// API fronts the canon: PUT /scenario, reload fan-out, lint proxy.
// Single-writer: persistent state only in traineebox/Postgres.
type API struct {
	repo         *Repository
	dialogURLs   []string
	serviceToken string
	// SaveScenario persists the validated snapshot (tickets repo in main).
	SaveScenario func(ctx context.Context, ticketID uuid.UUID, scenarioJSON, version string) error
	// ResolveActor maps Authorization header → user id (session auth in main).
	ResolveActor func(ctx context.Context, header string) (uuid.UUID, error)
}

func NewAPI(repo *Repository, dialogURLs []string, serviceToken string) *API {
	return &API{repo: repo, dialogURLs: dialogURLs, serviceToken: serviceToken}
}

func Register(api huma.API, a *API) {
	huma.Register(api, huma.Operation{
		OperationID: "put-ticket-scenario",
		Method:      http.MethodPut,
		Path:        "/tickets/{ticketId}/scenario",
		Summary:     "Set ticket dialog scenario snapshot",
		Tags:        []string{"Dialog"},
	}, a.putScenario)
	huma.Register(api, huma.Operation{
		OperationID: "bank-reload",
		Method:      http.MethodPost,
		Path:        "/bank/reload",
		Summary:     "Fan-out bank reload to dialog workers (service token)",
		Tags:        []string{"Dialog"},
	}, a.reload)
	huma.Register(api, huma.Operation{
		OperationID: "bank-version",
		Method:      http.MethodGet,
		Path:        "/bank/version",
		Summary:     "Current bank digest",
		Tags:        []string{"Dialog"},
	}, a.version)
	huma.Register(api, huma.Operation{
		OperationID: "lint-scenario",
		Method:      http.MethodPost,
		Path:        "/scenarios/lint",
		Summary:     "Lint scenario reachability via dialog",
		Tags:        []string{"Dialog"},
	}, a.lint)
}

type putScenarioIn struct {
	TicketID      string `path:"ticketId"`
	Authorization string `header:"Authorization"`
	Body          struct {
		Scenario json.RawMessage `json:"scenario"`
		Version  string          `json:"version"`
	} `json:"body"`
}

func (a *API) putScenario(ctx context.Context, in *putScenarioIn) (*struct{}, error) {
	if a.ResolveActor == nil {
		return nil, huma.Error401Unauthorized("unauthorized")
	}
	if _, err := a.ResolveActor(ctx, in.Authorization); err != nil {
		return nil, huma.Error401Unauthorized("unauthorized")
	}
	if !json.Valid(in.Body.Scenario) {
		return nil, huma.Error400BadRequest("scenario must be JSON")
	}
	slots, err := a.repo.ListSlotIDs(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := Validate(in.Body.Scenario, slots); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	ticketID, err := uuid.Parse(in.TicketID)
	if err != nil {
		return nil, huma.Error400BadRequest("bad ticket id")
	}
	if a.SaveScenario == nil {
		return nil, huma.Error500InternalServerError("scenario store not wired")
	}
	if err := a.SaveScenario(ctx, ticketID, string(in.Body.Scenario), in.Body.Version); err != nil {
		return nil, mapSaveError(err)
	}
	return &struct{}{}, nil
}

type reloadIn struct {
	ServiceToken string `header:"X-Service-Token"`
	Body         struct {
		Version string `json:"version"`
	} `json:"body"`
}

func (a *API) reload(ctx context.Context, in *reloadIn) (*struct{ Body map[string]any }, error) {
	if in.ServiceToken != a.serviceToken {
		return nil, huma.Error401Unauthorized("bad service token")
	}
	version := in.Body.Version
	if version == "" {
		return nil, huma.Error400BadRequest("version required")
	}
	if len(a.dialogURLs) == 0 {
		return nil, huma.Error503ServiceUnavailable("no dialog workers configured")
	}
	// DB first: recompute digest from the edited canon, never trust the caller (spec J).
	slots, questions, err := a.repo.BankSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	if len(slots) == 0 {
		return nil, huma.Error400BadRequest("empty slots would wipe bank canon")
	}
	digest := BankDigest(slots, questions)
	if err := a.repo.ReplaceBank(ctx, version, slots, questions); err != nil {
		return nil, err
	}
	// Fan-out carries the full snapshot: workers swap without a fetch round-trip.
	// 3 retries per worker, then bank_stale in the response (spec I/O matrix).
	stale := []string{}
	for _, base := range a.dialogURLs {
		if err := postReload(base, version, digest, slots, questions, a.serviceToken); err != nil {
			stale = append(stale, base)
		}
	}
	out := map[string]any{"version": version, "digest": digest}
	if len(stale) > 0 {
		out["bank_stale"] = stale
	}
	return &struct{ Body map[string]any }{Body: out}, nil
}

func postReload(base, version, digest string, slots map[string]string, questions map[string][]string, token string) error {
	if strings.TrimSpace(base) == "" {
		return fmt.Errorf("empty dialog worker URL")
	}
	if v := os.Getenv("DIALOG_URL"); v != "" && base == "env" {
		base = v
	}
	var last error = fmt.Errorf("no attempt")
	for i := 0; i < 3; i++ {
		body, _ := json.Marshal(map[string]any{
			"version": version, "digest": digest, "slots": slots, "questions": questions,
		})
		req, err := http.NewRequest("POST", strings.TrimSuffix(base, "/")+"/bank/reload", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("X-Service-Token", token)
		}
		resp, err := dialogHTTP.Do(req)
		if err != nil {
			last = err
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 300 {
			return nil
		}
		last = fmt.Errorf("dialog reload: %s", resp.Status)
	}
	return last
}

type versionIn struct {
	ServiceToken string `header:"X-Service-Token"`
}

type versionOut struct {
	Body struct {
		Version string `json:"version"`
		Digest  string `json:"digest"`
	} `json:"body"`
}

func (a *API) version(ctx context.Context, in *versionIn) (*versionOut, error) {
	if in.ServiceToken != a.serviceToken {
		return nil, huma.Error401Unauthorized("bad service token")
	}
	version, digest, err := a.repo.BankVersion(ctx)
	if err != nil {
		return nil, huma.Error404NotFound("no bank")
	}
	out := &versionOut{}
	out.Body.Version = version
	out.Body.Digest = digest
	return out, nil
}

type lintIn struct {
	Body struct {
		Scenario json.RawMessage `json:"scenario"`
	} `json:"body"`
}

func (a *API) lint(ctx context.Context, in *lintIn) (*struct{ Body map[string]any }, error) {
	if !json.Valid(in.Body.Scenario) {
		return nil, huma.Error400BadRequest("scenario must be JSON")
	}
	// Proxy the dialog worker: its lint runs the frozen lexical bank, so the
	// gate actually gates (a local hardcoded unreachable:[] never would — spec B).
	// Every worker is tried before the degraded fallback.
	for _, base := range a.dialogURLs {
		if out, err := postLint(base, in.Body.Scenario, a.serviceToken); err == nil {
			return &struct{ Body map[string]any }{Body: out}, nil
		}
	}
	slots, err := a.repo.ListSlotIDs(ctx)
	if err != nil {
		return nil, err
	}
	sc, err := Validate(in.Body.Scenario, slots)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	// Fallback without a worker: critical linkage only, unreachable unknown.
	return &struct{ Body map[string]any }{Body: map[string]any{
		"id": sc.ID, "facts": len(sc.Facts), "critical": sc.Critical, "unreachable": []string{},
		"lint_degraded": true,
	}}, nil
}

// postLint forwards the snapshot to dialog /scenarios/lint (service token).
func postLint(base string, scenario json.RawMessage, token string) (map[string]any, error) {
	body, _ := json.Marshal(map[string]any{"scenario": json.RawMessage(scenario)})
	req, err := http.NewRequest("POST", strings.TrimSuffix(base, "/")+"/scenarios/lint", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Service-Token", token)
	}
	resp, err := dialogHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dialog lint: %s", resp.Status)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func mapSaveError(err error) error {
	if errors.Is(err, ticketserrs.ErrNotFound) {
		return huma.Error404NotFound("no ticket")
	}
	return err
}
