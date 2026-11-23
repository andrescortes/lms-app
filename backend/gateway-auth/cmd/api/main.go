package main

import (
	"context"
	"errors"
	"lms-app/backend/gateway-auth/internal/config"
	"lms-app/backend/gateway-auth/internal/service"
	"lms-app/backend/gateway-auth/internal/store"
	"lms-app/backend/gateway-auth/internal/transport"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()

	// Validate production configuration
	if err := cfg.ValidateLMSCoreURL(); err != nil {
		slog.Error("invalid LMS_CORE_URL", "error", err)
		os.Exit(1)
	}

	db, err := store.NewSQLiteDB(cfg.DBPath)
	if err != nil {
		slog.Error("error creating SQLite DB", "error", err)
		os.Exit(1)
	}

	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("error closing SQLite DB", "error", err)
		}
	}()

	migCtx, migCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := store.RunMigrations(migCtx, db); err != nil {
		migCancel()
		slog.Error("error running migrations", "error", err)
		os.Exit(1)
	}

	migCancel()

	//injection dependencies
	// stores
	userStore := store.NewUserStore(db)

	// services

	jwtService := service.NewJWTService(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := service.NewAuthService(userStore, jwtService, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	// handlers
	healthHandler := transport.NewHealthHandler(db)
	authHandler := transport.NewAuthHandler(authService, cfg.RefreshTokenTTL, cfg.CookieSecure, cfg.CookieDomain)
	lmsProxy, err := transport.NewMSReverseProxy(cfg.LMSCoreURL)
	if err != nil {
		slog.Error("error initializing lms-core reverse proxy", "error", err)
		os.Exit(1)
	}
	// router
	router := transport.NewRouter(healthHandler, authHandler, jwtService, lmsProxy)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Background job: cleanup expired refresh tokens daily
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				deleted, err := userStore.CleanupExpiredRefreshTokens(cleanupCtx, 30*24*time.Hour) // older than 30 days
				cancel()
				if err != nil {
					slog.Error("failed to cleanup expired refresh tokens", "error", err)
				} else if deleted > 0 {
					slog.Info("cleaned up expired refresh tokens", "count", deleted)
				}
			}
		}
	}()

	go func() {
		slog.Info("Starting gateway-auth", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("error fatal in Http Server", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("signal shutdown received, closing connections actives ...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("error during graceful shutdown", "error", err)
	}

	slog.Info("gateway-auth stopped cleanly")
}
