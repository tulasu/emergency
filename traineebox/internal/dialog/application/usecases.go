package application

import (
	"context"
	"encoding/json"

	"traineebox/internal/dialog/domain/errs"
	"traineebox/internal/dialog/domain/models"
	"traineebox/internal/dialog/domain/repositories"

	"github.com/google/uuid"
)

type ScenarioStore interface {
	Save(ctx context.Context, ticketID uuid.UUID, scenarioJSON, version string) error
}

type WorkerClient interface {
	Reload(base, version, digest string, slots map[string]string, questions map[string][]string, token string) error
	Lint(base string, scenario []byte, token string) (map[string]any, error)
}

type PutScenario struct {
	Bank  repositories.BankRepository
	Store ScenarioStore
}

func (uc PutScenario) Execute(ctx context.Context, ticketID uuid.UUID, scenario []byte, version string) error {
	if !json.Valid(scenario) {
		return errs.ErrInvalidInput
	}
	slots, err := uc.Bank.ListSlotIDs(ctx)
	if err != nil {
		return err
	}
	if _, err := models.Validate(scenario, slots); err != nil {
		return err
	}
	if uc.Store == nil {
		return errs.ErrNotFound
	}
	return uc.Store.Save(ctx, ticketID, string(scenario), version)
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
