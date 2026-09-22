package main

import (
	"context"
	"fmt"
	"os"

	"traineebox/internal/auth/application"
	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/value_objects"
	authinfra "traineebox/internal/auth/infrastructure"
	"traineebox/internal/platform/config"
	"traineebox/internal/platform/postgres"

	"github.com/spf13/cobra"
)

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
