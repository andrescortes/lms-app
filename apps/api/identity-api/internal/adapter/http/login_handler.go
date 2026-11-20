package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"lms-english/identity-service/internal/usecase"
)

// loginRequest is the JSON request DTO. Unknown fields are rejected.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse is the success DTO: the access token and the refresh token.
type loginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// loginHandler serves POST /api/identity/auth/login.
type loginHandler struct {
	uc *usecase.LoginUserUseCase
}

// ServeHTTP decodes the request, delegates to the use case, and maps the
// outcome to the single error envelope or the token response.
func (h *loginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req loginRequest
	if err := dec.Decode(&req); err != nil {
		respondBadRequest(w, r, "request body is malformed or contains unknown fields")
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		respondBadRequest(w, r, "request body must contain a single JSON object")
		return
	}

	result, err := h.uc.Execute(r.Context(), usecase.LoginUserCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}
