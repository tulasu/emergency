package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"traineebox/internal/auth/application"
	authinfra "traineebox/internal/auth/infrastructure"
	authpresentation "traineebox/internal/auth/presentation"
	groupsapp "traineebox/internal/groups/application"
	groupsinfra "traineebox/internal/groups/infrastructure"
	groupspresentation "traineebox/internal/groups/presentation"
	"traineebox/internal/platform/config"
	"traineebox/internal/platform/postgres"
	ticketsapp "traineebox/internal/tickets/application"
	ticketsinfra "traineebox/internal/tickets/infrastructure"
	ticketspresentation "traineebox/internal/tickets/presentation"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	users := authinfra.NewUserRepository(pool)
	sessions := authinfra.NewSessionRepository(pool)
	hasher := application.PasswordHasher{}
	authenticate := application.Authenticate{Users: users, Sessions: sessions}

	authHandlers := authpresentation.NewAPI(authpresentation.Deps{
		Version:      version,
		CreateUser:   application.CreateUser{Users: users, Hasher: hasher},
		Login:        application.Login{Users: users, Sessions: sessions, Hasher: hasher, SessionTTL: cfg.SessionTTL},
		Logout:       application.Logout{Sessions: sessions},
		Me:           application.Me{Authenticate: authenticate},
		BlockUser:    application.BlockUser{Users: users},
		ChangeRole:   application.ChangeRole{Users: users},
		Authenticate: authenticate,
	})

	groupsRepo := groupsinfra.NewGroupRepository(pool)
	directory := groupsinfra.NewUserDirectory(pool)
	groupsHandlers := groupspresentation.NewAPI(groupspresentation.Deps{
		CreateGroup:  groupsapp.CreateGroup{Groups: groupsRepo, Directory: directory},
		RenameGroup:  groupsapp.RenameGroup{Groups: groupsRepo},
		DeleteGroup:  groupsapp.DeleteGroup{Groups: groupsRepo},
		ListGroups:   groupsapp.ListGroups{Groups: groupsRepo},
		GetGroup:     groupsapp.GetGroup{Groups: groupsRepo},
		AddMember:    groupsapp.AddMember{Groups: groupsRepo, Directory: directory},
		RemoveMember: groupsapp.RemoveMember{Groups: groupsRepo},
		Authenticate: groupsSessionAuthenticator{auth: authenticate},
	})

	cat, err := ticketsinfra.LoadCatalog(cfg.CatalogPath)
	if err != nil {
		log.Fatalf("catalog: %v", err)
	}
	catalogRepo := ticketsinfra.NewCatalogRepository(cat)
	ticketsRepo := ticketsinfra.NewTicketRepository(pool)
	attemptsRepo := ticketsinfra.NewAttemptRepository(pool)
	membership := ticketsinfra.NewGroupMembership(pool)
	ticketsHandlers := ticketspresentation.NewAPI(ticketspresentation.Deps{
		ListIncidentTypes:  ticketsapp.ListIncidentTypes{Catalog: catalogRepo},
		ListTagsByType:     ticketsapp.ListTagsByType{Catalog: catalogRepo},
		ListServices:       ticketsapp.ListServices{Catalog: catalogRepo},
		RecommendServices:  ticketsapp.RecommendServices{Catalog: catalogRepo},
		CreateTicket:       ticketsapp.CreateTicket{Tickets: ticketsRepo, Membership: membership},
		ListTicketsByGroup: ticketsapp.ListTicketsByGroup{Tickets: ticketsRepo, Membership: membership},
		GetTicket:          ticketsapp.GetTicket{Tickets: ticketsRepo, Membership: membership},
		SetReferenceAnswer: ticketsapp.SetReferenceAnswer{Tickets: ticketsRepo, Catalog: catalogRepo, Membership: membership},
		StartAttempt:       ticketsapp.StartAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo, Membership: membership},
		SaveAttemptAnswer:  ticketsapp.SaveAttemptAnswer{Tickets: ticketsRepo, Attempts: attemptsRepo, Catalog: catalogRepo, Membership: membership},
		SubmitAttempt:      ticketsapp.SubmitAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo, Catalog: catalogRepo, Membership: membership},
		GetMyAttempt:       ticketsapp.GetMyAttempt{Tickets: ticketsRepo, Attempts: attemptsRepo, Membership: membership},
		ListMyAttempts:     ticketsapp.ListMyAttempts{Tickets: ticketsRepo, Attempts: attemptsRepo, Membership: membership},
		Authenticate:       ticketsSessionAuthenticator{auth: authenticate},
	})

	router := chi.NewMux()
	apiCfg := huma.DefaultConfig("TraineeBox API", version)
	apiCfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "session-token",
		},
	}
	api := humachi.New(router, apiCfg)
	authpresentation.Register(api, authHandlers)
	groupspresentation.Register(api, groupsHandlers)
	ticketspresentation.Register(api, ticketsHandlers)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s version=%s", cfg.HTTPAddr, version)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
