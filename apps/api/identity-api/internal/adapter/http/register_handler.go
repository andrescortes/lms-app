package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"lms-english/identity-service/internal/usecase"
)

// maxBodyBytes bounds the request body so a large payload cannot exhaust memory.
const maxBodyBytes = 1 << 20 // 1 MiB

// registerRequest is the JSON request DTO. Unknown fields are rejected.
type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

// registerResponse is the success DTO: only the new user id is returned.
type registerResponse struct {
	UserID string `json:"userId"`
}

// registerHandler serves POST /api/identity/auth/register.
type registerHandler struct {
	uc *usecase.RegisterUserUseCase
}

// ServeHTTP decodes the request, delegates to the use case, and maps the result.
func (h *registerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req registerRequest
	if err := dec.Decode(&req); err != nil {
		respondBadRequest(w, r, "request body is malformed or contains unknown fields")
		return
	}
	// Reject trailing content after the first JSON value.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		respondBadRequest(w, r, "request body must contain a single JSON object")
		return
	}

	result, err := h.uc.Execute(r.Context(), usecase.RegisterUserCommand{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/identity/users/"+result.UserID)
	writeJSON(w, http.StatusCreated, registerResponse{UserID: result.UserID})
}
