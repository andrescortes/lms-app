package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"lms-english/identity-service/internal/usecase"
)

// changeRoleRequest is the request DTO. Unknown fields are rejected.
type changeRoleRequest struct {
	Role string `json:"role"`
}

// userRoleHandler serves PATCH /api/identity/users/{id}/role behind Authenticate and
// RequireRole("ADMIN").
type userRoleHandler struct {
	uc *usecase.ChangeUserRoleUseCase
}

// ServeHTTP reads the target from the path and the actor from the token, then
// applies the role change.
func (h *userRoleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		respondError(w, r, errMissingIdentity)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req changeRoleRequest
	if err := dec.Decode(&req); err != nil {
		respondBadRequest(w, r, "request body is malformed or contains unknown fields")
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		respondBadRequest(w, r, "request body must contain a single JSON object")
		return
	}

	user, err := h.uc.Execute(r.Context(), usecase.ChangeUserRoleCommand{
		ActorUserID:  claims.UserID,
		TargetUserID: chi.URLParam(r, "id"),
		NewRole:      req.Role,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}

	var cefr *string
	if user.CEFRLevel != nil {
		level := string(*user.CEFRLevel)
		cefr = &level
	}
	writeJSON(w, http.StatusOK, userSummaryJSON{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role.String(),
		CEFRLevel:   cefr,
		CreatedAt:   user.CreatedAt.UTC().Format(time.RFC3339),
	})
}
