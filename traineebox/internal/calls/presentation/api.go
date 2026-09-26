package presentation

import (
	"context"

	"traineebox/internal/calls/application"

	"github.com/google/uuid"
)

type API struct {
	svc          *application.Service
	serviceToken string
	ResolveActor func(ctx context.Context, header string) (uuid.UUID, error)
}

func NewAPI(svc *application.Service, serviceToken string) *API {
	return &API{svc: svc, serviceToken: serviceToken}
}
