package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"traineebox/internal/dialog/domain/errs"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

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
	}, a.reloadHandler)
	huma.Register(api, huma.Operation{
		OperationID: "bank-version",
		Method:      http.MethodGet,
		Path:        "/bank/version",
		Summary:     "Current bank digest",
		Tags:        []string{"Dialog"},
	}, a.versionHandler)
	huma.Register(api, huma.Operation{
		OperationID: "lint-scenario",
		Method:      http.MethodPost,
		Path:        "/scenarios/lint",
		Summary:     "Lint scenario reachability via dialog",
		Tags:        []string{"Dialog"},
	}, a.lintHandler)
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
	ticketID, err := uuid.Parse(in.TicketID)
	if err != nil {
		return nil, huma.Error400BadRequest("bad ticket id")
	}
	if err := a.put.Execute(ctx, ticketID, in.Body.Scenario, in.Body.Version); err != nil {
		return nil, mapDialogError(err)
	}
	return &struct{}{}, nil
}

type reloadIn struct {
	ServiceToken string `header:"X-Service-Token"`
	Body         struct {
		Version string `json:"version"`
	} `json:"body"`
}

func (a *API) reloadHandler(ctx context.Context, in *reloadIn) (*struct{ Body map[string]any }, error) {
	if in.ServiceToken != a.token {
		return nil, huma.Error401Unauthorized("bad service token")
	}
	res, err := a.reload.Execute(ctx, in.Body.Version)
	if err != nil {
		return nil, mapDialogError(err)
	}
	out := map[string]any{"version": res.Version, "digest": res.Digest}
	if len(res.Stale) > 0 {
		out["bank_stale"] = res.Stale
	}
	return &struct{ Body map[string]any }{Body: out}, nil
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

func (a *API) versionHandler(ctx context.Context, in *versionIn) (*versionOut, error) {
	if in.ServiceToken != a.token {
		return nil, huma.Error401Unauthorized("bad service token")
	}
	version, digest, err := a.version.Execute(ctx)
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

func (a *API) lintHandler(ctx context.Context, in *lintIn) (*struct{ Body map[string]any }, error) {
	out, err := a.lint.Execute(ctx, in.Body.Scenario)
	if err != nil {
		return nil, mapDialogError(err)
	}
	return &struct{ Body map[string]any }{Body: out}, nil
}

func mapDialogError(err error) error {
	switch {
	case errors.Is(err, errs.ErrNotFound):
		return huma.Error404NotFound("no ticket")
	case errors.Is(err, errs.ErrNoWorkers):
		return huma.Error503ServiceUnavailable("no dialog workers configured")
	case errors.Is(err, errs.ErrEmptyBank):
		return huma.Error400BadRequest("empty slots would wipe bank canon")
	case errors.Is(err, errs.ErrInvalidInput):
		return huma.Error400BadRequest(err.Error())
	default:
		return err
	}
}
