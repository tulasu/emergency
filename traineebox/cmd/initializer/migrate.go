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
				if err := withGoose(cmd.Context(), cfg.PostgresDSN, goose.UpContext); err != nil {
					return err
				}
				return ensureAsteriskGrants(cmd.Context(), cfg.PostgresDSN)
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

// ensureAsteriskGrants grants realtime-VIEW SELECT to the asterisk role when
// it exists; fresh/test DBs without the role still migrate. Plain Go, not a
// .sql migration: goose's splitter breaks on the $$ block, while the server
// parses it fine as one string.
func ensureAsteriskGrants(ctx context.Context, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	_, err = db.ExecContext(ctx, `DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'asterisk') THEN
        GRANT SELECT ON ps_endpoints TO asterisk;
        GRANT SELECT ON ps_auths TO asterisk;
        GRANT SELECT ON ps_aors TO asterisk;
    END IF;
END $$;`)
	return err
}
