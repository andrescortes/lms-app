package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"lms-english/identity-service/internal/usecase"
)

// userListResponse is the page DTO.
type userListResponse struct {
	Items []userSummaryJSON `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

// userSummaryJSON is one account in the list. It carries no password material.
type userSummaryJSON struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"displayName"`
	Role        string  `json:"role"`
	CEFRLevel   *string `json:"cefrLevel"`
	CreatedAt   string  `json:"createdAt"`
}

// userListHandler serves GET /api/identity/users behind Authenticate and
// RequireRole("ADMIN").
type userListHandler struct {
	uc *usecase.ListUsersUseCase
}

// ServeHTTP parses the query parameters and returns one page.
func (h *userListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	page, hasPage, err := intParam(query, "page")
	if err != nil {
		respondBadRequest(w, r, "page must be a whole number")
		return
	}
	limit, hasLimit, err := intParam(query, "limit")
	if err != nil {
		respondBadRequest(w, r, "limit must be a whole number")
		return
	}

	result, err := h.uc.Execute(r.Context(), usecase.ListUsersQuery{
		Role:      strings.TrimSpace(query.Get("role")),
		CEFRLevel: strings.TrimSpace(query.Get("cefrLevel")),
		Search:    query.Get("q"),
		Page:      page,
		Limit:     limit,
		HasPage:   hasPage,
		HasLimit:  hasLimit,
	})
	if err != nil {
		respondError(w, r, err)
		return
	}

	items := make([]userSummaryJSON, 0, len(result.Items))
	for _, item := range result.Items {
		var cefr *string
		if item.CEFRLevel != nil {
			level := string(*item.CEFRLevel)
			cefr = &level
		}
		items = append(items, userSummaryJSON{
			ID:          item.ID,
			Email:       item.Email,
			DisplayName: item.DisplayName,
			Role:        item.Role.String(),
			CEFRLevel:   cefr,
			CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, userListResponse{
		Items: items,
		Total: result.Total,
		Page:  result.Page,
		Limit: result.Limit,
	})
}

// intParam parses an optional whole-number query parameter. An absent parameter is
// reported as present=false, which the use case reads as "use the default". A
// parameter supplied as zero is reported as present so validation refuses it rather
// than silently taking the default.
func intParam(values map[string][]string, name string) (value int, present bool, err error) {
	raw, ok := values[name]
	if !ok || len(raw) == 0 {
		return 0, false, nil
	}
	parsed, err := strconv.Atoi(raw[0])
	if err != nil {
		return 0, true, err
	}
	return parsed, true, nil
}
