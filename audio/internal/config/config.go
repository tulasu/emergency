// Package config loads audio service configuration from the environment.
// Every field has a sane dev default except INTERNAL_SERVICE_TOKEN,
// which must be set (fail-closed, same as traineebox calls_dialog.go).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime settings for the audio service.
type Config struct {
	HTTPAddr           string
	DatabaseURL        string
	S3Endpoint         string
	S3Bucket           string
	S3Key              string
	S3Secret           string
	Voice              string
	Rate               int
	SynthURL           string
	InternalToken      string
	SweepInterval      time.Duration
	SweepLimit         int
	WorkerPollInterval time.Duration
	WorkerSynthTimeout time.Duration
	// PauseWorker stops the synth queue worker (offline/CPU mode per spec 06).
	// Sweeps and HTTP keep running.
	PauseWorker bool
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getint(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func getdur(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	if n, err := strconv.Atoi(v); err == nil {
		return time.Duration(n) * time.Second
	}
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		return d
	}
	return fallback
}

// Load reads configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:           getenv("AUDIO_HTTP_ADDR", ":8002"),
		DatabaseURL:        getenv("AUDIO_DATABASE_URL", "postgres://emergency:emergency@audio-postgres:5432/emergency_audio?sslmode=disable"),
		S3Endpoint:         strings.TrimSuffix(getenv("AUDIO_S3_ENDPOINT", "http://rustfs:9000"), "/"),
		S3Bucket:           getenv("AUDIO_S3_BUCKET", "emergency-audio"),
		S3Key:              os.Getenv("AUDIO_S3_KEY"),
		S3Secret:           os.Getenv("AUDIO_S3_SECRET"),
		Voice:              getenv("AUDIO_VOICE", "kseniya"),
		Rate:               getint("AUDIO_RATE", 8000),
		SynthURL:           strings.TrimSuffix(getenv("AUDIO_SYNTH_URL", "http://audio-synth:8003"), "/"),
		InternalToken:      os.Getenv("INTERNAL_SERVICE_TOKEN"),
		SweepInterval:      getdur("AUDIO_SWEEP_MINUTES", 10*time.Minute),
		SweepLimit:         getint("AUDIO_SWEEP_LIMIT", 500),
		WorkerPollInterval: getdur("AUDIO_WORKER_POLL_SECONDS", 2*time.Second),
		WorkerSynthTimeout: getdur("AUDIO_WORKER_SYNTH_TIMEOUT_SECONDS", 60*time.Second),
		PauseWorker:        strings.TrimSpace(os.Getenv("AUDIO_WORKER_PAUSED")) == "1",
	}
	if cfg.InternalToken == "" {
		return Config{}, fmt.Errorf("INTERNAL_SERVICE_TOKEN must be set")
	}
	if cfg.SweepInterval <= 0 {
		cfg.SweepInterval = 10 * time.Minute
	}
	return cfg, nil
}
