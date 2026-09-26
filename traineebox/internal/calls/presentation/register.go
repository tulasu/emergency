package presentation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"traineebox/internal/calls/application"
	"traineebox/internal/calls/domain/errs"
	"traineebox/internal/calls/domain/models"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

func Register(api huma.API, a *API) {
	huma.Register(api, huma.Operation{
		OperationID: "request-call",
		Method:      http.MethodPost,
		Path:        "/attempts/{attemptId}/call",
		Summary:     "Originate student call (system only)",
		Tags:        []string{"Calls"},
	}, a.requestCall)
	huma.Register(api, huma.Operation{
		OperationID: "list-attempt-calls",
		Method:      http.MethodGet,
		Path:        "/attempts/{attemptId}/calls",
		Summary:     "List calls for attempt",
		Tags:        []string{"Calls"},
	}, a.listCalls)
	huma.Register(api, huma.Operation{
		OperationID: "hangup-call",
		Method:      http.MethodDelete,
		Path:        "/calls/{callId}",
		Summary:     "Hangup call leg",
		Tags:        []string{"Calls"},
	}, a.hangup)
	huma.Register(api, huma.Operation{
		OperationID: "internal-call-event",
		Method:      http.MethodPost,
		Path:        "/internal/calls/{callId}/events",
		Summary:     "ARI outcome event (service token)",
		Tags:        []string{"Internal"},
	}, a.event)
	huma.Register(api, huma.Operation{
		OperationID: "internal-call-turns",
		Method:      http.MethodPost,
		Path:        "/internal/calls/{callId}/turns",
		Summary:     "Dialog turns on close (service token)",
		Tags:        []string{"Internal"},
	}, a.turns)
}

type requestCallIn struct {
	AttemptID     uuid.UUID `path:"attemptId"`
	Authorization string    `header:"Authorization"`
	Body          struct {
		To string `json:"to"`
	} `json:"body"`
}

type callDTO struct {
	ID         string `json:"id"`
	AttemptID  string `json:"attempt_id"`
	Status     string `json:"status"`
	ScenarioID string `json:"scenario_id"`
}

func toDTO(c models.Call) callDTO {
	return callDTO{ID: c.ID.String(), AttemptID: c.AttemptID.String(), Status: c.Status, ScenarioID: c.ScenarioID}
}

func (a *API) requestCall(ctx context.Context, in *requestCallIn) (*struct{ Body callDTO }, error) {
	actor, err := a.actorOf(ctx, in.Authorization)
	if err != nil {
		return nil, err
	}
	call, err := a.svc.RequestCall(ctx, application.RequestCallInput{AttemptID: in.AttemptID, ActorID: actor, To: in.Body.To})
	if err != nil {
		fmt.Fprintf(os.Stderr, "requestCall: attempt=%s actor=%s err=%v\n", in.AttemptID, actor, err)
		return nil, mapCallError(err)
	}
	return &struct{ Body callDTO }{Body: toDTO(call)}, nil
}

func (a *API) actorOf(ctx context.Context, header string) (uuid.UUID, error) {
	if a.ResolveActor == nil {
		return uuid.Nil, huma.Error401Unauthorized("unauthorized")
	}
	id, err := a.ResolveActor(ctx, header)
	if err != nil {
		return uuid.Nil, huma.Error401Unauthorized("unauthorized")
	}
	return id, nil
}

type listCallsIn struct {
	AttemptID     uuid.UUID `path:"attemptId"`
	Authorization string    `header:"Authorization"`
}

func (a *API) listCalls(ctx context.Context, in *listCallsIn) (*struct{ Body []callDTO }, error) {
	if _, err := a.actorOf(ctx, in.Authorization); err != nil {
		return nil, err
	}
	items, err := a.svc.ListByAttempt(ctx, in.AttemptID)
	if err != nil {
		return nil, err
	}
	out := make([]callDTO, 0, len(items))
	for _, c := range items {
		out = append(out, toDTO(c))
	}
	return &struct{ Body []callDTO }{Body: out}, nil
}

type hangupIn struct {
	CallID        uuid.UUID `path:"callId"`
	Authorization string    `header:"Authorization"`
}

func (a *API) hangup(ctx context.Context, in *hangupIn) (*struct{}, error) {
	if _, err := a.actorOf(ctx, in.Authorization); err != nil {
		return nil, err
	}
	if err := a.svc.Hangup(ctx, in.CallID); err != nil {
		return nil, mapCallError(err)
	}
	return &struct{}{}, nil
}

type eventIn struct {
	CallID       uuid.UUID `path:"callId"`
	ServiceToken string    `header:"X-Service-Token"`
	Body         struct {
		Type string `json:"type"`
	} `json:"body"`
}

func (a *API) event(ctx context.Context, in *eventIn) (*struct{ Body callDTO }, error) {
	if !a.authorized(in.ServiceToken) {
		return nil, huma.Error401Unauthorized("bad service token")
	}
	c, err := a.svc.OnEvent(ctx, in.CallID, in.Body.Type)
	if err != nil {
		return nil, mapCallError(err)
	}
	return &struct{ Body callDTO }{Body: toDTO(c)}, nil
}

type turnsIn struct {
	CallID       uuid.UUID `path:"callId"`
	ServiceToken string    `header:"X-Service-Token"`
	Body         struct {
		Turns []models.Turn `json:"turns"`
	} `json:"body"`
}

func (a *API) turns(ctx context.Context, in *turnsIn) (*struct{}, error) {
	if !a.authorized(in.ServiceToken) {
		return nil, huma.Error401Unauthorized("bad service token")
	}
	for _, t := range in.Body.Turns {
		if t.N <= 0 {
			return nil, huma.Error400BadRequest("turn n must be > 0")
		}
	}
	if _, err := a.svc.FindByID(ctx, in.CallID); err != nil {
		return nil, mapCallError(err)
	}
	if err := a.svc.SaveTurns(ctx, in.CallID, in.Body.Turns); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

func (a *API) authorized(tok string) bool {
	if a.serviceToken == "" {
		return false
	}
	return tok == a.serviceToken
}

func mapCallError(err error) error {
	switch {
	case errors.Is(err, errs.ErrConflict):
		return huma.Error409Conflict("call already active")
	case errors.Is(err, errs.ErrGone):
		return huma.Error410Gone("attempt deadline passed")
	case errors.Is(err, errs.ErrBadSnapshot):
		return huma.Error400BadRequest("scenario drift")
	case errors.Is(err, errs.ErrNotFound):
		return huma.Error404NotFound("no call")
	case errors.Is(err, errs.ErrInvalidInput):
		return huma.Error400BadRequest("invalid input")
	case errors.Is(err, errs.ErrForbidden):
		return huma.Error403Forbidden("forbidden")
	default:
		return err
	}
}
