package transport

import (
	"encoding/json"
	"net/http"
	"time"
)

type HealthHandler struct {
	startTime time.Time
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
	}
}

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Uptime  string `json:"uptime"`
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:  "ok",
		Service: "gateway-auth",
		Uptime:  time.Since(h.startTime).Round(time.Second).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
