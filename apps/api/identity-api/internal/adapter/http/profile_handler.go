package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"lms-english/identity-service/internal/usecase"
)

// profileResponse is the profile DTO. It carries no password material.
type profileResponse struct {
	ID          string            `json:"id"`
	Email       string            `json:"email"`
	DisplayName string            `json:"displayName"`
	Role        string            `json:"role"`
	CEFRLevel   *string           `json:"cefrLevel"`
	CreatedAt   string            `json:"createdAt"`
	Profile     profileFieldsJSON `json:"profile"`
}

// profileFieldsJSON is the profile sub-object. An unset field is null.
type profileFieldsJSON struct {
	NativeLanguage *string `json:"nativeLanguage"`
	Goals          *string `json:"goals"`
	Timezone       *string `json:"timezone"`
	AvatarURL      *string `json:"avatarUrl"`
}

// updateProfileRequest mirrors the JSON body. Values are decoded separately so
// the handler can tell which fields the request actually named.
type updateProfileRequest struct {
	DisplayName     *string `json:"displayName"`
	NativeLanguage  *string `json:"nativeLanguage"`
	Goals           *string `json:"goals"`
	Timezone        *string `json:"timezone"`
	AvatarURL       *string `json:"avatarUrl"`
	CurrentPassword *string `json:"currentPassword"`
	NewPassword     *string `json:"newPassword"`
}

// getProfileHandler serves GET /api/identity/users/me.
type getProfileHandler struct {
	uc *usecase.GetProfileUseCase
}

// updateProfileHandler serves PATCH /api/identity/users/me.
type updateProfileHandler struct {
	uc *usecase.UpdateProfileUseCase
}

// namedFields is the set of top-level keys a request actually contained.
type namedFields map[string]bool

// decodeNamedFields reports which top-level keys the raw body names, so an
// explicit null differs from an absent field. The body has already been read, so
// this runs on the buffered copy.
func decodeNamedFields(raw []byte) namedFields {
	var probe map[string]json.RawMessage
	named := namedFields{}
	if json.Unmarshal(raw, &probe) != nil {
		return named
	}
	for key := range probe {
		named[key] = true
	}
	return named
}

// ServeHTTP returns the caller's own account and profile.
func (h *getProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		respondError(w, r, errMissingIdentity)
		return
	}
	view, err := h.uc.Execute(r.Context(), claims.UserID)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProfileResponse(view))
}

// ServeHTTP applies the partial update to the caller's own account and profile.
func (h *updateProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		respondError(w, r, errMissingIdentity)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		respondBadRequest(w, r, "request body is malformed or contains unknown fields")
		return
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var req updateProfileRequest
	if err := dec.Decode(&req); err != nil {
		respondBadRequest(w, r, "request body is malformed or contains unknown fields")
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		respondBadRequest(w, r, "request body must contain a single JSON object")
		return
	}

	named := decodeNamedFields(raw)
	cmd := usecase.UpdateProfileCommand{
		UserID:         claims.UserID,
		DisplayName:    req.DisplayName,
		NativeLanguage: profileField(named, "nativeLanguage", req.NativeLanguage),
		Goals:          profileField(named, "goals", req.Goals),
		Timezone:       profileField(named, "timezone", req.Timezone),
		AvatarURL:      profileField(named, "avatarUrl", req.AvatarURL),
	}
	if req.NewPassword != nil {
		cmd.NewPassword = *req.NewPassword
	}
	if req.CurrentPassword != nil {
		cmd.CurrentPassword = *req.CurrentPassword
	}

	view, err := h.uc.Execute(r.Context(), cmd)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProfileResponse(view))
}

// profileField tells the "leave alone", "clear", and "set" cases apart. A field the
// request did not name is nil. A field sent as null or as an empty string is an
// empty string, which clears it. Anything else carries its value.
func profileField(named namedFields, key string, value *string) *string {
	if !named[key] {
		return nil
	}
	if value == nil {
		empty := ""
		return &empty
	}
	return value
}

// toProfileResponse maps the read model onto the wire shape, converting a nil
// learning level to null.
func toProfileResponse(view usecase.ProfileView) profileResponse {
	var cefr *string
	if view.CEFRLevel != nil {
		level := string(*view.CEFRLevel)
		cefr = &level
	}
	return profileResponse{
		ID:          view.ID,
		Email:       view.Email,
		DisplayName: view.DisplayName,
		Role:        view.Role.String(),
		CEFRLevel:   cefr,
		CreatedAt:   view.CreatedAt.UTC().Format(time.RFC3339),
		Profile: profileFieldsJSON{
			NativeLanguage: view.NativeLanguage,
			Goals:          view.Goals,
			Timezone:       view.Timezone,
			AvatarURL:      view.AvatarURL,
		},
	}
}
