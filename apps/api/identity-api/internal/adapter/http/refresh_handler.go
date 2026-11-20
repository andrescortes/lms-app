package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"lms-english/identity-service/internal/usecase"
)

// refreshRequest is the JSON request DTO. Unknown fields are rejected.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// refreshResponse is the success DTO: the rotated token pair.
type refreshResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// refreshHandler serves POST /api/identity/auth/refresh.
type refreshHandler struct {
	uc *usecase.RefreshTokenUseCase
}

// ServeHTTP decodes the presented refresh token, delegates to the use case, and
// maps the outcome to the token pair or the single error envelope.
func (h *refreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req refreshRequest
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

	result, err := h.uc.Execute(r.Context(), usecase.RefreshCommand{RefreshToken: req.RefreshToken})
	if err != nil {
		respondError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, refreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}
