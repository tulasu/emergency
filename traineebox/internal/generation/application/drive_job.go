package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"traineebox/internal/generation/domain/models"
	"traineebox/internal/generation/domain/repositories"
	genvo "traineebox/internal/generation/domain/value_objects"
)

type DraftClient interface {
	Draft(ctx context.Context, prompt string) (title, scenario string, ref models.DraftReference, err error)
}

type ScenarioLinter interface {
	Lint(ctx context.Context, snapshot string) (unreachable []string, skipped bool, err error)
}

type DriveJob struct {
	Jobs         repositories.JobRepository
	Catalog      repositories.Catalog
	Drafts       DraftClient
	Linter       ScenarioLinter
	WorkerID     string
	LeaseSeconds int
}

func (uc DriveJob) Execute(ctx context.Context) (bool, error) {
	job, claimed, err := uc.Jobs.ClaimNext(ctx, uc.WorkerID, uc.LeaseSeconds)
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
		if err := uc.Jobs.SaveCAS(ctx, job, expectedStatus, expectedVersion); err != nil {
			return err
		}
		expectedStatus, expectedVersion = job.Status.String(), expectedVersion+1
		return nil
	}
	draftTitle, scenario, ref, err := uc.Drafts.Draft(ctx, job.Prompt)
	if err != nil {
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			_ = cas(genvo.JobStatusFailed, trunc(redact(err.Error())), true)
			return true, ctx.Err()
		case <-timer.C:
		}
		draftTitle, scenario, ref, err = uc.Drafts.Draft(ctx, job.Prompt)
	}
	if err != nil {
		_ = cas(genvo.JobStatusFailed, trunc(redact(err.Error())), true)
		return true, nil
	}
	job.DraftTitle, job.ScenarioText, job.DraftReference = draftTitle, scenario, ref.Normalize()
	if err := cas(genvo.JobStatusCheckingDialog, "", false); err != nil {
		return true, nil
	}
	if err := uc.check(ctx, job); err != nil {
		_ = cas(genvo.JobStatusFailed, trunc(redact(err.Error())), true)
		return true, nil
	}
	if err := cas(genvo.JobStatusReady, "", true); err != nil {
		return true, nil
	}
	return true, nil
}

func (uc DriveJob) check(ctx context.Context, job models.Job) error {
	draft := job.DraftReference.Normalize()
	if draft.IncidentTypeCode == "" {
		return fmt.Errorf("empty incident_type_code")
	}
	if err := uc.Catalog.IncidentTypeExists(ctx, draft.IncidentTypeCode); err != nil {
		return fmt.Errorf("unknown type %q", draft.IncidentTypeCode)
	}
	if err := uc.Catalog.ValidateTags(ctx, draft.IncidentTypeCode, draft.TagCodes); err != nil {
		return fmt.Errorf("tag selection: %w", err)
	}
	if len(draft.ServiceCodes) > 0 {
		ok, err := uc.Catalog.ServicesExist(ctx, draft.ServiceCodes)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("unknown service code")
		}
	}
	if draft.ApplicantLastName == "" || draft.ApplicantFirstName == "" ||
		draft.CallerNumber == "" || draft.DictatedNumber == "" {
		return fmt.Errorf("incomplete PII on card")
	}
	return uc.lintDraft(ctx, job)
}

func (uc DriveJob) lintDraft(ctx context.Context, job models.Job) error {
	raw := strings.TrimSpace(job.ScenarioText)
	var probe struct {
		Facts []json.RawMessage `json:"facts"`
	}
	if !strings.HasPrefix(raw, "{") ||
		json.Unmarshal([]byte(raw), &probe) != nil || len(probe.Facts) == 0 {
		log.Printf("generation worker %s job %s: dialog lint skipped (no scenario facts)",
			uc.WorkerID, job.ID)
		return nil
	}
	if uc.Linter == nil {
		log.Printf("generation worker %s job %s: dialog lint skipped (no dialog workers)",
			uc.WorkerID, job.ID)
		return nil
	}
	unreachable, skipped, err := uc.Linter.Lint(ctx, raw)
	if skipped {
		log.Printf("generation worker %s job %s: dialog lint skipped (no dialog workers)",
			uc.WorkerID, job.ID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("dialog lint unavailable: %v", err)
	}
	if len(unreachable) > 0 {
		return fmt.Errorf("dialog lint: unreachable critical %v", unreachable)
	}
	return nil
}

func trunc(s string) string {
	if r := []rune(s); len(r) > 1000 {
		return string(r[:1000])
	}
	return s
}

var piiDigits = regexp.MustCompile(`\d{7,}`)

func redact(s string) string {
	return piiDigits.ReplaceAllString(s, "***")
}
