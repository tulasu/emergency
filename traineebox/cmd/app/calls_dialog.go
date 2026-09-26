package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"traineebox/internal/auth/application"
	callsapp "traineebox/internal/calls/application"
	callserrs "traineebox/internal/calls/domain/errs"
	callsinfra "traineebox/internal/calls/infrastructure"
	callspresentation "traineebox/internal/calls/presentation"
	dialogapp "traineebox/internal/dialog/application"
	dialogerrs "traineebox/internal/dialog/domain/errs"
	dialoginfra "traineebox/internal/dialog/infrastructure"
	dialogpresentation "traineebox/internal/dialog/presentation"
	"traineebox/internal/generation/domain/repositories"
	geninfra "traineebox/internal/generation/infrastructure"
	genworker "traineebox/internal/generation/worker"
	ticketserrs "traineebox/internal/tickets/domain/errs"
	ticketsinfra "traineebox/internal/tickets/infrastructure"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func wireCallsAndDialog(
	api huma.API,
	pool *pgxpool.Pool,
	authenticate application.Authenticate,
	ticketsRepo *ticketsinfra.TicketRepository,
	attemptsRepo *ticketsinfra.AttemptRepository,
	jobsRepo *geninfra.JobRepository,
	catalogRepo repositories.Catalog,
) {
	dialogRepo := dialoginfra.NewBankRepository(pool)
	callsRepo := callsinfra.NewCallRepository(pool)
	serviceToken := os.Getenv("INTERNAL_SERVICE_TOKEN")
	if serviceToken == "" {
		log.Fatal("INTERNAL_SERVICE_TOKEN must be set")
	}
	dialogURL := os.Getenv("DIALOG_URL")

	svc := &callsapp.Service{
		Calls:    callsRepo,
		Dialer:   callsinfra.ARIFromEnv(),
		Sessions: callsinfra.NewDialogClient(dialogURL),
		MarkAttemptTimedOut: func(ctx context.Context, attemptID uuid.UUID) error {
			return attemptsRepo.MarkTimedOut(ctx, attemptID)
		},
		ResolveEndpoint: func(ctx context.Context, userID uuid.UUID) (string, error) {
			var endpoint string
			err := pool.QueryRow(ctx,
				`SELECT endpoint FROM user_sip_endpoints WHERE user_id = $1 AND enabled`,
				userID).Scan(&endpoint)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return "", callserrs.ErrInvalidInput
				}
				return "", err
			}
			return endpoint, nil
		},
		LoadAttempt: func(ctx context.Context, attemptID, actorID uuid.UUID) (uuid.UUID, uuid.UUID, *time.Time, string, error) {
			attempt, err := attemptsRepo.FindByID(ctx, attemptID)
			if err != nil {
				return uuid.Nil, uuid.Nil, nil, "", mapTicketsToCalls(err)
			}
			if actorID != uuid.Nil && actorID != attempt.UserID {
				return uuid.Nil, uuid.Nil, nil, "", callserrs.ErrForbidden
			}
			tickets, err := ticketsRepo.ListByVariant(ctx, attempt.VariantID)
			if err != nil {
				return uuid.Nil, uuid.Nil, nil, "", mapTicketsToCalls(err)
			}
			if len(tickets) == 0 {
				return uuid.Nil, uuid.Nil, nil, "", callserrs.ErrNotFound
			}
			return tickets[0].ID, attempt.UserID, attempt.DeadlineAt, attempt.Status.String(), nil
		},
		LoadTicket: func(ctx context.Context, ticketID uuid.UUID) (string, string, string, error) {
			ticket, err := ticketsRepo.FindByID(ctx, ticketID)
			if err != nil {
				return "", "", "", mapTicketsToCalls(err)
			}
			scenarioID := ticket.ScenarioVersion
			scenarioJSON := ticket.ScenarioJSON
			if scenarioJSON == "" {
				scenarioJSON = "{}"
			} else {
				var probe struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal([]byte(scenarioJSON), &probe); err == nil && probe.ID != "" {
					scenarioID = probe.ID
				}
			}
			if scenarioID == "" {
				scenarioID = ticket.ID.String()
			}
			digest := ""
			if v, d, err := dialogRepo.BankVersion(ctx); err == nil {
				_ = v
				digest = d
			}
			return scenarioID, scenarioJSON, digest, nil
		},
	}

	callsAPI := callspresentation.NewAPI(svc, serviceToken)
	callsAPI.ResolveActor = func(ctx context.Context, header string) (uuid.UUID, error) {
		user, err := authenticate.Execute(ctx, bearerOf(header))
		if err != nil {
			return uuid.Nil, err
		}
		return user.ID, nil
	}
	callspresentation.Register(api, callsAPI)

	dialogURLs := []string{}
	if dialogURL != "" {
		dialogURLs = append(dialogURLs, dialogURL)
	}
	if extra := os.Getenv("DIALOG_URLS"); extra != "" {
		for _, u := range strings.Split(extra, ",") {
			if s := strings.TrimSpace(u); s != "" {
				dialogURLs = append(dialogURLs, s)
			}
		}
	}
	workers := dialoginfra.NewWorkerClient()
	dialogAPI := dialogpresentation.NewAPI(dialogpresentation.Deps{
		Put: dialogapp.PutScenario{
			Bank: dialogRepo,
			Store: dialogScenarioStore{save: func(ctx context.Context, ticketID uuid.UUID, scenarioJSON, version string) error {
				if err := ticketsRepo.UpdateDialogSnapshot(ctx, ticketID, scenarioJSON, version); err != nil {
					return mapTicketsToDialog(err)
				}
				return nil
			}},
		},
		Reload: dialogapp.ReloadBank{
			Bank: dialogRepo, Workers: workers, URLs: dialogURLs, Token: serviceToken,
		},
		Version:      dialogapp.BankVersion{Bank: dialogRepo},
		Lint:         dialogapp.LintScenario{Bank: dialogRepo, Workers: workers, URLs: dialogURLs, Token: serviceToken},
		ServiceToken: serviceToken,
		ResolveActor: callsAPI.ResolveActor,
	})
	dialogpresentation.Register(api, dialogAPI)

	go svc.WatchDeadlines(context.Background(), 15*time.Second, nil)
	if catalogRepo != nil {
		go genworker.FromEnv(newGenerationDrive(jobsRepo, catalogRepo, dialogURLs, serviceToken)).Loop(context.Background())
	} else {
		log.Print("generation worker disabled (no catalog)")
	}
}

type dialogScenarioStore struct {
	save func(ctx context.Context, ticketID uuid.UUID, scenarioJSON, version string) error
}

func (s dialogScenarioStore) Save(ctx context.Context, ticketID uuid.UUID, scenarioJSON, version string) error {
	return s.save(ctx, ticketID, scenarioJSON, version)
}

func mapTicketsToCalls(err error) error {
	switch {
	case errors.Is(err, ticketserrs.ErrNotFound):
		return callserrs.ErrNotFound
	case errors.Is(err, ticketserrs.ErrForbidden):
		return callserrs.ErrForbidden
	default:
		return err
	}
}

func mapTicketsToDialog(err error) error {
	if errors.Is(err, ticketserrs.ErrNotFound) {
		return dialogerrs.ErrNotFound
	}
	return err
}

func bearerOf(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return strings.TrimSpace(header)
}
