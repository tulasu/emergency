package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr      string
	PostgresDSN   string
	SessionSecret string
	SessionTTL    time.Duration
	CatalogPath   string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:      envOr("TRAINEEBOX_HTTP_ADDR", ":8080"),
		PostgresDSN:   envOr("TRAINEEBOX_POSTGRES_DSN", "postgres://emergency:emergency@127.0.0.1:5432/emergency?sslmode=disable"),
		SessionSecret: envOr("TRAINEEBOX_SESSION_SECRET", "dev-session-secret-change-me"),
		SessionTTL:    24 * time.Hour,
		CatalogPath:   envOr("TRAINEEBOX_CATALOG_PATH", defaultCatalogPath()),
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

func defaultCatalogPath() string {
	if p := findCatalogNearCWD(); p != "" {
		return p
	}
	if p := findCatalogFromExecutable(); p != "" {
		return p
	}
	return filepath.Clean("../artifacts/etc/traineebox/catalog")
}

func catalogMarker(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "services.yaml"))
	return err == nil && !st.IsDir()
}

func findCatalogNearCWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "artifacts", "etc", "traineebox", "catalog")
		if catalogMarker(candidate) {
			return candidate
		}
		candidate = filepath.Join(dir, "..", "artifacts", "etc", "traineebox", "catalog")
		if catalogMarker(candidate) {
			return filepath.Clean(candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func findCatalogFromExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "artifacts", "etc", "traineebox", "catalog")
		if catalogMarker(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
