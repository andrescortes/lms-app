package transport

import (
	"encoding/json"
	"errors"
	"lms-app/backend/gateway-auth/internal/model"
	"lms-app/backend/gateway-auth/internal/service"
	"log/slog"
	"net/http"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	user, err := h.authService.RegisterUser(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, model.ErrEmailAlreadyExists):
			writeError(w, http.StatusConflict, "an account with this email already exist")
		default:
			slog.Error("unexpected error registering user", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	slog.Info("new user registered", "user_id", user.ID, "email", user.Email, "cefr_level", user.CEFRLevel)
	writeJSON(w, http.StatusCreated, user)
}
