package transport

import (
	"context"
	"database/sql"
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
	Status   string `json:"status"`
	Service  string `json:"service"`
	Database string `json:"database"`
	Uptime   string `json:"uptime"`
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "connected"
	httpStatus := http.StatusOK
	serviceStatus := "ok"

	if err := h.db.PingContext(ctx); err != nil {
		dbStatus = "disconnected"
		serviceStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, healthResponse{
		Status:   serviceStatus,
		Service:  "lms-core",
		Database: dbStatus,
		Uptime:   time.Since(h.startTime).Round(time.Second).String(),
	})
}
