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
	groupsAuth := sessionAuthenticator{auth: authenticate}
	groupsHandlers := groupspresentation.NewAPI(groupspresentation.Deps{
		CreateGroup:  groupsapp.CreateGroup{Groups: groupsRepo, Directory: directory},
		RenameGroup:  groupsapp.RenameGroup{Groups: groupsRepo},
		DeleteGroup:  groupsapp.DeleteGroup{Groups: groupsRepo},
		ListGroups:   groupsapp.ListGroups{Groups: groupsRepo},
		GetGroup:     groupsapp.GetGroup{Groups: groupsRepo},
		AddMember:    groupsapp.AddMember{Groups: groupsRepo, Directory: directory},
		RemoveMember: groupsapp.RemoveMember{Groups: groupsRepo},
		Authenticate: groupsAuth,
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
