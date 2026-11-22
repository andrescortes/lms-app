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
	Status   string `json:"status"`
	Service  string `json:"service"`
	Database string `json:"database,omitempty"`
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
		httpStatus = http.StatusServiceUnavailable
		serviceStatus = "error"
	}

	resp := healthResponse{
		Status:   serviceStatus,
		Service:  "gateway-auth",
		Database: dbStatus,
		Uptime:   time.Since(h.startTime).Round(time.Second).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(resp)
}
