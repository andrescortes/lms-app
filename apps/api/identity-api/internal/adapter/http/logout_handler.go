package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"lms-english/identity-service/internal/usecase"
)

// logoutRequest is the JSON request DTO. Unknown fields are rejected.
type logoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// logoutHandler serves POST /api/identity/auth/logout behind Authenticate.
type logoutHandler struct {
	uc *usecase.LogoutUseCase
}

// ServeHTTP ends the caller's session. The caller id comes from the identity the
// Authenticate middleware put in the context, never from the request body.
func (h *logoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		// Authenticate always sets the identity; reaching here means the route was
		// mounted without it, which is a server mistake rather than a bad caller.
		respondError(w, r, errMissingIdentity)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req logoutRequest
	if err := dec.Decode(&req); err != nil {
		respondBadRequest(w, r, "request body is malformed or contains unknown fields")
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		respondBadRequest(w, r, "request body must contain a single JSON object")
		return
	}
	if req.RefreshToken == "" {
		respondBadRequest(w, r, "refreshToken is required")
		return
	}

	if err := h.uc.Execute(r.Context(), usecase.LogoutCommand{
		UserID:       claims.UserID,
		RefreshToken: req.RefreshToken,
	}); err != nil {
		respondError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
