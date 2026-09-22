package testkit

import (
	"net/http"
	"testing"
	"time"

	"traineebox/internal/auth/application"
	authinfra "traineebox/internal/auth/infrastructure"
	"traineebox/internal/auth/presentation"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewAPI(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	users := authinfra.NewUserRepository(pool)
	sessions := authinfra.NewSessionRepository(pool)
	hasher := application.PasswordHasher{}
	authenticate := application.Authenticate{Users: users, Sessions: sessions}

	apiHandlers := presentation.NewAPI(presentation.Deps{
		Version:      "test",
		CreateUser:   application.CreateUser{Users: users, Hasher: hasher},
		Login:        application.Login{Users: users, Sessions: sessions, Hasher: hasher, SessionTTL: time.Hour},
		Logout:       application.Logout{Sessions: sessions},
		Me:           application.Me{Authenticate: authenticate},
		BlockUser:    application.BlockUser{Users: users},
		ChangeRole:   application.ChangeRole{Users: users},
		Authenticate: authenticate,
	})

	router := chi.NewMux()
	apiCfg := huma.DefaultConfig("TraineeBox API", "test")
	apiCfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "session-token",
		},
	}
	api := humachi.New(router, apiCfg)
	presentation.Register(api, apiHandlers)
	return router
}
