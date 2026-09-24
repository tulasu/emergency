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
	"traineebox/internal/calls"
	"traineebox/internal/dialog"
	geninfra "traineebox/internal/generation/infrastructure"
	genworker "traineebox/internal/generation/worker"
	ticketserrs "traineebox/internal/tickets/domain/errs"
	ticketsrepos "traineebox/internal/tickets/domain/repositories"
	ticketsinfra "traineebox/internal/tickets/infrastructure"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ticketsCatalog is the catalog surface the generation worker checks against.
type ticketsCatalog = ticketsrepos.CatalogRepository

// wireCallsAndDialog adds owner-executor voice wiring (AD-1/AD-2/AD-6/AD-9).
// Ticketgen never dials; dialog writes turns only via service-token internal
// endpoints on close; hot call path never hits DB (snapshot lives in dialog).
func wireCallsAndDialog(
	api huma.API,
	pool *pgxpool.Pool,
	authenticate application.Authenticate,
	ticketsRepo *ticketsinfra.TicketRepository,
	attemptsRepo *ticketsinfra.AttemptRepository,
	jobsRepo *geninfra.JobRepository,
	catalogRepo ticketsCatalog,
) {
	dialogRepo := dialog.NewRepository(pool)
	callsRepo := calls.NewRepository(pool)
	serviceToken := os.Getenv("INTERNAL_SERVICE_TOKEN")
	if serviceToken == "" {
		// Fail closed: internal endpoints must never accept an empty token.
		log.Fatal("INTERNAL_SERVICE_TOKEN must be set")
	}
	dialogURL := os.Getenv("DIALOG_URL")

	svc := &calls.Service{
		Calls:     callsRepo,
		ARI:       calls.ARIFromEnv(),
		DialogURL: dialogURL,
		MarkAttemptTimedOut: func(ctx context.Context, attemptID uuid.UUID) error {
			// Deadline race closes the attempt too; the update is conditional
			// on status still in_progress so a submitted attempt is never clobbered.
			return attemptsRepo.MarkTimedOut(ctx, attemptID)
		},
		ResolveEndpoint: func(ctx context.Context, userID uuid.UUID) (string, error) {
			// `to` resolves from user_sip_endpoints, never trusted blindly (spec H).
			var endpoint string
			err := pool.QueryRow(ctx,
				`SELECT endpoint FROM user_sip_endpoints WHERE user_id = $1 AND enabled`,
				userID).Scan(&endpoint)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return "", calls.ErrInvalidInput
				}
				return "", err
			}
			return endpoint, nil
		},
		LoadAttempt: func(ctx context.Context, attemptID, actorID uuid.UUID) (uuid.UUID, uuid.UUID, *time.Time, string, error) {
			attempt, err := attemptsRepo.FindByID(ctx, attemptID)
			if err != nil {
				return uuid.Nil, uuid.Nil, nil, "", err
			}
			if actorID != uuid.Nil && actorID != attempt.UserID {
				return uuid.Nil, uuid.Nil, nil, "", ticketserrs.ErrForbidden
			}
			return attempt.TicketID, attempt.UserID, attempt.DeadlineAt, attempt.Status.String(), nil
		},
		LoadTicket: func(ctx context.Context, ticketID uuid.UUID) (string, string, string, error) {
			ticket, err := ticketsRepo.FindByID(ctx, ticketID)
			if err != nil {
				return "", "", "", err
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

	callsAPI := calls.NewAPI(svc, serviceToken)
	callsAPI.ResolveActor = func(ctx context.Context, header string) (uuid.UUID, error) {
		user, err := authenticate.Execute(ctx, bearerOf(header))
		if err != nil {
			return uuid.Nil, err
		}
		return user.ID, nil
	}
	calls.Register(api, callsAPI)

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
	dialogAPI := dialog.NewAPI(dialogRepo, dialogURLs, serviceToken)
	dialogAPI.SaveScenario = ticketsRepo.UpdateDialogSnapshot
	dialogAPI.ResolveActor = callsAPI.ResolveActor
	dialog.Register(api, dialogAPI)

	// Deadline ticker: expired calls hang up + time out without an event (spec Q).
	go svc.WatchDeadlines(context.Background(), 15*time.Second, nil)
	// Generation driver: claims building_dialog/checking_dialog via ticketgen HTTP (spec K).
	if catalogRepo != nil {
		go genworker.FromEnv(jobsRepo, catalogRepo).Loop(context.Background())
	} else {
		log.Print("generation worker disabled (no catalog)")
	}
}

func bearerOf(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return strings.TrimSpace(header)
}
