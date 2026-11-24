package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lms-app/backend/lms-core/internal/config"
	"lms-app/backend/lms-core/internal/store"
	"lms-app/backend/lms-core/internal/transport"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()

	db, err := store.NewSQLiteDB(cfg.DBPath)
	if err != nil {
		slog.Error("error initializing database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("error closing database", "error", err)
		}
	}()

	migCtx, migCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := store.RunMigrations(migCtx, db); err != nil {
		migCancel()
		slog.Error("error running migrations", "error", err)
		os.Exit(1)
	}
	migCancel()

	// initializing layers
	//store
	levelStore := store.NewLevelStore(db)
	//service
	//transport
	healthHandler := transport.NewHealthHandler(db)
	levelHandler := transport.NewLevelHandler(levelStore)
	//router
	router := transport.NewRouter(healthHandler, levelHandler)

	addr := cfg.Host + ":" + cfg.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("starting lms-core", "addr", addr, "env", cfg.Env, "db_path", cfg.DBPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("fatal error in HTTP server", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, closing active connections...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("error during graceful shutdown", "error", err)
	}

	slog.Info("lms-core stopped cleanly")
}
