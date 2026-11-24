package transport

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// HealthCheckCallback is called when health check status changes.
type HealthCheckCallback func(healthy bool, err error)

type HealthHandler struct {
	startTime      time.Time
	db             *sql.DB
	mu             sync.RWMutex
	lastHealthy    *bool // nil means first check
	onStatusChange HealthCheckCallback
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{
		startTime:   time.Time{},
		db:          db,
		lastHealthy: nil, // First check will always trigger callback
	}
}

// WithStatusChangeCallback sets a callback for health status changes.
// The callback is called when the database transitions between healthy/unhealthy.
func (h *HealthHandler) WithStatusChangeCallback(cb HealthCheckCallback) *HealthHandler {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onStatusChange = cb
	return h
}

type healthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	Database string `json:"database"`
	Uptime   string `json:"uptime"`
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	// Initialize startTime on first check
	if h.startTime.IsZero() {
		h.startTime = time.Now()
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var err error
	// Try to ping with a short timeout first
	pingCtx, pingCancel := context.WithTimeout(ctx, 1*time.Second)
	err = h.db.PingContext(pingCtx)
	pingCancel()

	// If ping failed, try once more with remaining time
	if err != nil {
		slog.Warn("health check ping failed, retrying", "error", err)
		// Second attempt with full remaining timeout
		err = h.db.PingContext(ctx)
	}

	dbStatus := "connected"
	httpStatus := http.StatusOK
	serviceStatus := "ok"
	healthy := true

	if err != nil {
		dbStatus = "disconnected"
		serviceStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable
		healthy = false
	}

	// Notify callback if status changed
	h.notifyStatusChange(healthy, err)

	writeJSON(w, httpStatus, healthResponse{
		Status:   serviceStatus,
		Service:  "lms-core",
		Database: dbStatus,
		Uptime:   time.Since(h.startTime).Round(time.Second).String(),
	})
}

func (h *HealthHandler) notifyStatusChange(healthy bool, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.onStatusChange == nil {
		return
	}

	// First check always triggers callback (lastHealthy is nil)
	shouldNotify := h.lastHealthy == nil || healthy != *h.lastHealthy
	if shouldNotify {
		h.lastHealthy = &healthy
		// Call callback without holding lock to avoid deadlocks
		go h.onStatusChange(healthy, err)
	}
}
