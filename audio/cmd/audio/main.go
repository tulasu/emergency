// Command audio runs the standalone audio service (spec 06):
// ensure → synth queue → S3 + audio DB, blobs/manifests for dialog.
// Subcommands: serve (default), migrate {up|down|status} on AUDIO_DATABASE_URL.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"audio/internal/config"
	httpapi "audio/internal/handlers/http"
	"audio/internal/repositories"
	"audio/internal/services"
	"audio/migrations"
	"audio/pkg/s3"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := runMigrate(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := runServe(); err != nil {
		log.Fatal(err)
	}
}

func runServe() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := repositories.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	st := repositories.New(pool)
	s3c := s3.New(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3Key, cfg.S3Secret)
	if err := s3c.EnsureBucket(ctx); err != nil {
		log.Printf("audio: ensure bucket: %v", err)
	}
	stt := services.NewStats()
	api := httpapi.New(cfg, st, s3c, stt)
	mux := http.NewServeMux()
	api.Register(mux)

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	sc := services.NewSynthClient(cfg.SynthURL, cfg.WorkerSynthTimeout)
	go services.NewWorker(cfg, st, s3c, sc, stt).Loop(runCtx)
	go services.NewSweeper(cfg, st, s3c).Loop(runCtx)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("audio listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	<-runCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func runMigrate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: audio migrate {up|down|status}")
	}
	dsn := os.Getenv("AUDIO_DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("AUDIO_DATABASE_URL must be set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		return err
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	ctx := context.Background()
	switch args[0] {
	case "up":
		return goose.UpContext(ctx, db, ".")
	case "down":
		return goose.DownContext(ctx, db, ".")
	case "status":
		return goose.StatusContext(ctx, db, ".")
	default:
		return fmt.Errorf("unknown migrate command %q", args[0])
	}
}
