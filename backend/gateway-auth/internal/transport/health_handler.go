package transport

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type HealthHandler struct {
	startTime time.Time
	db        *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
		db:        db,
	}
}

type healthResponse struct {
	Status     string `json:"status"`
	Service    string `json:"service"`
	Database   string `json:"database,omitempty"`
	Migrations string `json:"migrations,omitempty"`
	Uptime     string `json:"uptime"`
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "connected"
	migrationsStatus := "applied"
	httpStatus := http.StatusOK
	serviceStatus := "ok"

	// Check database connectivity
	if err := h.db.PingContext(ctx); err != nil {
		dbStatus = "disconnected"
		httpStatus = http.StatusServiceUnavailable
		serviceStatus = "error"
	}

	// Verify migrations table exists and has entries
	if dbStatus == "connected" {
		var count int
		err := h.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations").Scan(&count)
		if err != nil {
			migrationsStatus = "not_applied"
			httpStatus = http.StatusServiceUnavailable
			serviceStatus = "degraded"
		} else if count == 0 {
			migrationsStatus = "pending"
			serviceStatus = "degraded"
		}
	}

	resp := healthResponse{
		Status:     serviceStatus,
		Service:    "gateway-auth",
		Database:   dbStatus,
		Migrations: migrationsStatus,
		Uptime:     time.Since(h.startTime).Round(time.Second).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}
