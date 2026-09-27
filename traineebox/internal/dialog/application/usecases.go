package application

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"traineebox/internal/audio"
	"traineebox/internal/dialog/domain/errs"
	"traineebox/internal/dialog/domain/models"
	"traineebox/internal/dialog/domain/repositories"

	"github.com/google/uuid"
)

type ScenarioStore interface {
	Save(ctx context.Context, ticketID uuid.UUID, scenarioJSON, version, digest, status string) error
	UpdateAudioStatus(ctx context.Context, ticketID uuid.UUID, digest, status string) (bool, error)
}

type AudioEnsurer interface {
	Configured() bool
	EnsureWithRetry(ctx context.Context, body audio.EnsureRequest) (audio.EnsureResult, error)
}

type WorkerClient interface {
	Reload(base, version, digest string, slots map[string]string, questions map[string][]string, token string) error
	Lint(base string, scenario []byte, token string) (map[string]any, error)
}

type PutScenario struct {
	Bank  repositories.BankRepository
	Store ScenarioStore
	Audio AudioEnsurer
}

type PutScenarioResult struct {
	AudioStatus string
}

func (uc PutScenario) Execute(ctx context.Context, ticketID uuid.UUID, scenario []byte, version string, prerender bool) (PutScenarioResult, error) {
	if uc.Store == nil {
		return PutScenarioResult{}, errs.ErrNotFound
	}
	labels, err := uc.Bank.ListSlotLabels(ctx)
	if err != nil {
		return PutScenarioResult{}, err
	}
	knownSlots := make(map[string]bool, len(labels))
	for slotID := range labels {
		knownSlots[slotID] = true
	}
	canonical, err := audio.Canonicalize(scenario, knownSlots)
	if err != nil {
		return PutScenarioResult{}, err
	}
	if !prerender || canonical.Facts == 0 || uc.Audio == nil || !uc.Audio.Configured() {
		if canonical.Facts == 0 {
			log.Printf("audio prerender skipped for ticket %s: scenario has no facts", ticketID)
		}
		if err := uc.Store.Save(ctx, ticketID, string(canonical.JSON), version, "", "none"); err != nil {
			return PutScenarioResult{}, err
		}
		return PutScenarioResult{AudioStatus: "none"}, nil
	}
	urges, err := uc.Bank.ListSlotUrges(ctx)
	if err != nil {
		return PutScenarioResult{}, err
	}
	urgeSlots := make(map[string]string, len(labels))
	for id, label := range labels {
		urgeSlots[id] = label
		if urges[id] != "" {
			urgeSlots[id] = urges[id]
		}
	}
	if err := uc.Store.Save(ctx, ticketID, string(canonical.JSON), version, canonical.Digest, "pending"); err != nil {
		return PutScenarioResult{}, err
	}
	go uc.ensure(ticketID, canonical, labels, urgeSlots)
	return PutScenarioResult{AudioStatus: "pending"}, nil
}

func (uc PutScenario) ensure(ticketID uuid.UUID, canonical audio.CanonicalScenario, labels, urgeSlots map[string]string) {
	result, err := uc.Audio.EnsureWithRetry(context.Background(), audio.EnsureRequest{
		TicketID: ticketID, ScenarioDigest: canonical.Digest, Scenario: canonical.JSON, Slots: labels, UrgeSlots: urgeSlots,
	})
	if err != nil {
		if _, updateErr := uc.Store.UpdateAudioStatus(context.Background(), ticketID, canonical.Digest, "stale"); updateErr != nil {
			log.Printf("audio ensure failed for ticket %s: %v (status update: %v)", ticketID, err, updateErr)
			return
		}
		log.Printf("audio ensure failed for ticket %s: %v", ticketID, err)
		return
	}
	if result.StatusCode == http.StatusAccepted && result.Status == "pending" {
		return
	}
	status := "stale"
	if result.Status == "error" {
		status = "error"
	}
	if (result.StatusCode == http.StatusOK || result.StatusCode == http.StatusAccepted) && result.Status == "ready" {
		status = "ready"
	}
	if _, err := uc.Store.UpdateAudioStatus(context.Background(), ticketID, canonical.Digest, status); err != nil {
		log.Printf("audio status update failed for ticket %s: %v", ticketID, err)
	}
}

type ReloadBank struct {
	Bank    repositories.BankRepository
	Workers WorkerClient
	URLs    []string
	Token   string
}

type ReloadResult struct {
	Version string
	Digest  string
	Stale   []string
}

func (uc ReloadBank) Execute(ctx context.Context, version string) (ReloadResult, error) {
	if version == "" {
		return ReloadResult{}, errs.ErrInvalidInput
	}
	if len(uc.URLs) == 0 {
		return ReloadResult{}, errs.ErrNoWorkers
	}
	slots, questions, err := uc.Bank.BankSnapshot(ctx)
	if err != nil {
		return ReloadResult{}, err
	}
	if len(slots) == 0 {
		return ReloadResult{}, errs.ErrEmptyBank
	}
	digest := models.BankDigest(slots, questions)
	if err := uc.Bank.ReplaceBank(ctx, version, slots, questions); err != nil {
		return ReloadResult{}, err
	}
	stale := []string{}
	for _, base := range uc.URLs {
		if err := uc.Workers.Reload(base, version, digest, slots, questions, uc.Token); err != nil {
			stale = append(stale, base)
		}
	}
	return ReloadResult{Version: version, Digest: digest, Stale: stale}, nil
}

type BankVersion struct {
	Bank repositories.BankRepository
}

func (uc BankVersion) Execute(ctx context.Context) (version, digest string, err error) {
	version, digest, err = uc.Bank.BankVersion(ctx)
	if err != nil {
		return "", "", errs.ErrNotFound
	}
	return version, digest, nil
}

type LintScenario struct {
	Bank    repositories.BankRepository
	Workers WorkerClient
	URLs    []string
	Token   string
}

func (uc LintScenario) Execute(ctx context.Context, scenario []byte) (map[string]any, error) {
	if !json.Valid(scenario) {
		return nil, errs.ErrInvalidInput
	}
	for _, base := range uc.URLs {
		if out, err := uc.Workers.Lint(base, scenario, uc.Token); err == nil {
			return out, nil
		}
	}
	slots, err := uc.Bank.ListSlotIDs(ctx)
	if err != nil {
		return nil, err
	}
	sc, err := models.Validate(scenario, slots)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id": sc.ID, "facts": len(sc.Facts), "critical": sc.Critical, "unreachable": []string{},
		"lint_degraded": true,
	}, nil
}
