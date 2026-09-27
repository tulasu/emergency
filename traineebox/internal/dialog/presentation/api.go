package presentation

import (
	"context"

	"traineebox/internal/dialog/application"

	"github.com/google/uuid"
)

type AudioStatusStore interface {
	UpdateAudioStatus(ctx context.Context, ticketID uuid.UUID, digest, status string) (bool, error)
}

type API struct {
	put          application.PutScenario
	reload       application.ReloadBank
	version      application.BankVersion
	lint         application.LintScenario
	audioStatus  AudioStatusStore
	token        string
	ResolveActor func(ctx context.Context, header string) (uuid.UUID, error)
}

type Deps struct {
	Put          application.PutScenario
	Reload       application.ReloadBank
	Version      application.BankVersion
	Lint         application.LintScenario
	AudioStatus  AudioStatusStore
	ServiceToken string
	ResolveActor func(ctx context.Context, header string) (uuid.UUID, error)
}

func NewAPI(deps Deps) *API {
	return &API{
		put:          deps.Put,
		reload:       deps.Reload,
		version:      deps.Version,
		lint:         deps.Lint,
		audioStatus:  deps.AudioStatus,
		token:        deps.ServiceToken,
		ResolveActor: deps.ResolveActor,
	}
}
