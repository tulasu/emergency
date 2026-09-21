package presentation

import "context"

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

func (a *API) health(_ context.Context, _ *struct{}) (*healthOutput, error) {
	out := &healthOutput{}
	out.Body.Status = "ok"
	return out, nil
}
