package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr      string
	PostgresDSN   string
	SessionSecret string
	SessionTTL    time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:      envOr("TRAINEEBOX_HTTP_ADDR", ":8080"),
		PostgresDSN:   envOr("TRAINEEBOX_POSTGRES_DSN", "postgres://emergency:emergency@127.0.0.1:5432/emergency?sslmode=disable"),
		SessionSecret: envOr("TRAINEEBOX_SESSION_SECRET", "dev-session-secret-change-me"),
		SessionTTL:    24 * time.Hour,
	}
	if v := os.Getenv("TRAINEEBOX_SESSION_TTL_HOURS"); v != "" {
		h, err := strconv.Atoi(v)
		if err != nil || h <= 0 {
			return Config{}, fmt.Errorf("TRAINEEBOX_SESSION_TTL_HOURS: %w", err)
		}
		cfg.SessionTTL = time.Duration(h) * time.Hour
	}
	if cfg.SessionSecret == "" {
		return Config{}, fmt.Errorf("TRAINEEBOX_SESSION_SECRET is required")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
