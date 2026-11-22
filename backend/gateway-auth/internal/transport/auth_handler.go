package transport

import (
	"encoding/json"
	"errors"
	"lms-app/backend/gateway-auth/internal/model"
	"lms-app/backend/gateway-auth/internal/service"
	"log/slog"
	"net/http"
	"time"
)

const refreshCookieName = "lms_refresh_token"

type AuthHandler struct {
	authService  *service.AuthService
	refreshTTL   time.Duration
	cookieSecure bool
}

func NewAuthHandler(
	authService *service.AuthService,
	refreshTTL time.Duration,
	cookieSecure bool,
) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		refreshTTL:   refreshTTL,
		cookieSecure: cookieSecure,
	}
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

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	authResp, rawRefreshToken, err := h.authService.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, model.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid email or password")
		default:
			slog.Error("unexpected error during login", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	h.setRefreshCookie(w, rawRefreshToken, int(h.refreshTTL.Seconds()))
	slog.Info("user logged in", "user_id", authResp.User.ID, "email", authResp.User.Email)
	writeJSON(w, http.StatusOK, authResp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "missing refresh token cookie")
		return
	}

	authResp, newRawRefreshToken, err := h.authService.Refresh(r.Context(), cookie.Value)
	if err != nil {
		// Delete cookie invalid or revoked in the browser
		h.setRefreshCookie(w, "", -1)

		switch {
		case errors.Is(err, model.ErrInvalidRefreshToken):
			writeError(w, http.StatusUnauthorized, err.Error())
		default:
			slog.Error("unexpected error refreshing token", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	h.setRefreshCookie(w, newRawRefreshToken, int(h.refreshTTL.Seconds()))
	writeJSON(w, http.StatusOK, authResp)
}

// setRefreshCookie adds HttpOnly cookie restricted to /api/auth
func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    value,
		Path:     "/api/auth",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
