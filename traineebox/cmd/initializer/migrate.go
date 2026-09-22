package main

import (
	"context"
	"database/sql"

	"traineebox/internal/platform/config"
	"traineebox/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

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
