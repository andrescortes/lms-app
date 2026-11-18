// Command api is the Identity service entrypoint. It loads configuration from
// the environment, initializes the SQLite database, wires dependencies with
// explicit constructor injection, and serves HTTP until a shutdown signal.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"

	"golang.org/x/crypto/bcrypt"
	adapterevent "lms-english/identity-service/internal/adapter/event"
	adapterhttp "lms-english/identity-service/internal/adapter/http"

	"lms-english/identity-service/internal/adapter/security"
	adaptersqlite "lms-english/identity-service/internal/adapter/sqlite"
	"lms-english/identity-service/internal/usecase"
)

// config holds runtime configuration loaded from the environment.
type config struct {
	env      string
	httpPort string
	dbPath   string
}

// loadConfig reads configuration from the environment. Values without a
// deployment override fall back to documented local examples, never hardcoded
// absolute paths or secrets.
func loadConfig() (config, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		return config{}, errors.New("DB_PATH is required")
	}
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8081"
	}
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}
	return config{env: env, httpPort: port, dbPath: dbPath}, nil
}

func main() {
	if err := run(); err != nil {
		slog.Error("identity service exited with error", "error", err)
		os.Exit(1)
	}
}

// run wires and runs the service, returning when the process is asked to stop.
func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := adaptersqlite.Open(ctx, cfg.dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := migrateUp(ctx, db); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	// Manual dependency injection; no reflection container.
	store := adaptersqlite.NewRegistrationStore(db)
	hasher := security.NewBcryptHasher(bcrypt.DefaultCost)
	ids := newUUIDGenerator()
	clock := systemClock{}
	publisher := adapterevent.NewPublisher(db, []adapterevent.Handler{logRegistrationHook(logger)},
		adapterevent.WithLogger(logger))

	// Recover any events left pending by a previous run, then continue in the
	// background while serving.
	if err := publisher.Dispatch(ctx); err != nil {
		logger.Warn("initial event dispatch failed", "error", err)
	}
	go runEventWorker(ctx, publisher, logger)

	uc := usecase.NewRegisterUserUseCase(store, hasher, ids, clock, publisher)

	limiter := adapterhttp.NewVisitorLimiter()
	router := adapterhttp.NewRouter(uc, limiter, logger)

	srv := &http.Server{
		Addr:              ":" + cfg.httpPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("identity service listening", "addr", srv.Addr, "env", cfg.env)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("identity service stopped")
	return nil
}

// migrateUp applies the goose migrations in db/migrations.
func migrateUp(ctx context.Context, db *sql.DB) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	return goose.UpContext(ctx, db, "db/migrations")
}

// runEventWorker periodically dispatches pending outbox events until ctx ends.
func runEventWorker(ctx context.Context, p *adapterevent.Publisher, logger *slog.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.Dispatch(ctx); err != nil {
				logger.Warn("event worker dispatch failed", "error", err)
			}
		}
	}
}

// logRegistrationHook is the default downstream consumer; it logs a non-sensitive
// summary so the event flow is observable without exposing account details.
func logRegistrationHook(logger *slog.Logger) adapterevent.Handler {
	return func(_ context.Context, eventID, eventType string, payload []byte) error {
		logger.Info("user registered event delivered",
			"event_id", eventID, "event_type", eventType, "bytes", len(payload))
		return nil
	}
}

// systemClock supplies UTC wall-clock time.
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }
