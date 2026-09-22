package testkit

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"traineebox/internal/platform/postgres"
	"traineebox/migrations"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var (
	pgOnce   sync.Once
	pgPool   *pgxpool.Pool
	pgErr    error
	pgBinary *embeddedpostgres.EmbeddedPostgres
)

func StartPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pgOnce.Do(func() {
		pgErr = startEmbedded()
	})
	if pgErr != nil {
		t.Fatalf("embedded postgres: %v", pgErr)
	}
	return pgPool
}

func startEmbedded() error {
	unlock, err := acquireCacheLock()
	if err != nil {
		return err
	}
	defer unlock()

	port, err := freePort()
	if err != nil {
		return err
	}

	runtimePath, err := os.MkdirTemp("", "traineebox-pg-runtime-*")
	if err != nil {
		return err
	}

	cfg := embeddedpostgres.DefaultConfig().
		Username("traineebox").
		Password("traineebox").
		Database("traineebox").
		Version(embeddedpostgres.V16).
		Port(uint32(port)).
		RuntimePath(runtimePath).
		StartTimeout(90 * time.Second)

	pgBinary = embeddedpostgres.NewDatabase(cfg)
	if err := pgBinary.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	dsn := fmt.Sprintf(
		"postgres://traineebox:traineebox@127.0.0.1:%d/traineebox?sslmode=disable",
		port,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrate(ctx, dsn); err != nil {
		_ = pgBinary.Stop()
		return fmt.Errorf("migrate: %w", err)
	}

	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		_ = pgBinary.Stop()
		return fmt.Errorf("pool: %w", err)
	}
	pgPool = pool
	return nil
}

func acquireCacheLock() (unlock func(), err error) {
	lockPath := filepath.Join(os.TempDir(), "traineebox-embedded-postgres.lock")
	deadline := time.Now().Add(3 * time.Minute)
	for {
		f, openErr := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if openErr == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			return func() {
				_ = f.Close()
				_ = os.Remove(lockPath)
			}, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for embedded postgres cache lock: %w", openErr)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func migrate(ctx context.Context, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	for i := 0; i < 30; i++ {
		if err = db.PingContext(ctx); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return err
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, db, ".")
}

func Truncate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, `TRUNCATE sessions, users CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}
