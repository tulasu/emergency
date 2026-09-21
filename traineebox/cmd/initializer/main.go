package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"traineebox/internal/auth/application"
	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/value_objects"
	authinfra "traineebox/internal/auth/infrastructure"
	"traineebox/internal/platform/config"
	"traineebox/internal/platform/postgres"
	"traineebox/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	root := &cobra.Command{
		Use:           "initializer",
		Short:         "TraineeBox host operations",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newMigrateCmd(cfg), newSeedAdminCmd(cfg))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newMigrateCmd(cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migrations",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "up",
			Short: "Apply all pending migrations",
			RunE: func(cmd *cobra.Command, _ []string) error {
				return withGoose(cmd.Context(), cfg.PostgresDSN, goose.UpContext)
			},
		},
		&cobra.Command{
			Use:   "down",
			Short: "Roll back the last migration",
			RunE: func(cmd *cobra.Command, _ []string) error {
				return withGoose(cmd.Context(), cfg.PostgresDSN, goose.DownContext)
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show migration status",
			RunE: func(cmd *cobra.Command, _ []string) error {
				return withGoose(cmd.Context(), cfg.PostgresDSN, goose.StatusContext)
			},
		},
	)
	return cmd
}

func newSeedAdminCmd(cfg config.Config) *cobra.Command {
	var login, password string
	cmd := &cobra.Command{
		Use:   "seed-admin",
		Short: "Create the initial admin user if missing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := seedAdmin(cmd.Context(), cfg, login, password); err != nil {
				return err
			}
			fmt.Printf("admin user %q ready\n", login)
			return nil
		},
	}
	cmd.Flags().StringVar(&login, "login", envOr("TRAINEEBOX_ADMIN_LOGIN", "admin"), "admin login")
	cmd.Flags().StringVar(&password, "password", envOr("TRAINEEBOX_ADMIN_PASSWORD", "adminadmin"), "admin password")
	return cmd
}

type gooseFn func(context.Context, *sql.DB, string, ...goose.OptionsFunc) error

func withGoose(ctx context.Context, dsn string, fn gooseFn) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return fn(ctx, db, ".")
}

func seedAdmin(ctx context.Context, cfg config.Config, login, password string) error {
	pool, err := postgres.NewPool(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer pool.Close()
	users := authinfra.NewUserRepository(pool)
	uc := application.CreateUser{Users: users, Hasher: application.PasswordHasher{}}
	_, err = uc.Execute(ctx, application.CreateUserInput{
		Login:    login,
		Password: password,
		Role:     string(value_objects.RoleAdmin),
	})
	if err == errs.ErrConflict {
		return nil
	}
	return err
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
