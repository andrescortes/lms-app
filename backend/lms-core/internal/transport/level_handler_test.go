package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"lms-app/backend/lms-core/internal/model"
)

func TestLevelHandler_GetLevels_Success(t *testing.T) {
	mockStore := &MockLevelStore{
		ListLevelsFunc: func(ctx context.Context) ([]model.CEFRLevel, error) {
			return testLevels(), nil
		},
	}

	handler := NewLevelHandler(mockStore)

	req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
	// Set the trusted user in context (simulating middleware)
	user := &TrustedUser{
		ID:        "user-123",
		Role:      "student",
		CEFRLevel: "B1",
		Email:     "student@example.com",
	}
	req = req.WithContext(context.WithValue(req.Context(), trustedUserContextKey, user))

	rr := httptest.NewRecorder()
	handler.GetLevels(rr, req)

	AssertStatus(t, rr, http.StatusOK)
	AssertContentType(t, rr, "application/json")

	var response model.LevelsResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got '%s'", response.UserID)
	}
	if response.CurrentLevel != "B1" {
		t.Errorf("expected CurrentLevel 'B1', got '%s'", response.CurrentLevel)
	}
	if len(response.Levels) != 6 {
		t.Errorf("expected 6 levels, got %d", len(response.Levels))
	}

	// Verify level enrichment
	// B1 has rank_order 3, so A1, A2, B1 should be unlocked
	expectedUnlocked := map[string]bool{
		"A1": true, "A2": true, "B1": true, "B2": false, "C1": false, "C2": false,
	}
	for _, level := range response.Levels {
		if level.Unlocked != expectedUnlocked[level.Code] {
			t.Errorf("level %s unlocked=%v, expected %v", level.Code, level.Unlocked, expectedUnlocked[level.Code])
		}
		if level.IsCurrent != (level.Code == "B1") {
			t.Errorf("level %s IsCurrent=%v, expected %v", level.Code, level.IsCurrent, level.Code == "B1")
		}
	}
}

func TestLevelHandler_GetLevels_DefaultLevel(t *testing.T) {
	mockStore := &MockLevelStore{
		ListLevelsFunc: func(ctx context.Context) ([]model.CEFRLevel, error) {
			return testLevels(), nil
		},
	}

	handler := NewLevelHandler(mockStore)

	// Request with user having no level (should default to A1 in middleware)
	user := &TrustedUser{
		ID:        "user-456",
		Role:      "student",
		CEFRLevel: "A1", // Middleware defaults to A1
		Email:     "student2@example.com",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
	req = req.WithContext(context.WithValue(req.Context(), trustedUserContextKey, user))

	rr := httptest.NewRecorder()
	handler.GetLevels(rr, req)

	AssertStatus(t, rr, http.StatusOK)

	var response model.LevelsResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.CurrentLevel != "A1" {
		t.Errorf("expected default CurrentLevel 'A1', got '%s'", response.CurrentLevel)
	}

	// Only A1 should be unlocked and current (rank_order 1)
	for _, level := range response.Levels {
		if level.Code == "A1" {
			if !level.Unlocked || !level.IsCurrent {
				t.Errorf("A1 should be unlocked and current")
			}
		} else {
			if level.Unlocked || level.IsCurrent {
				t.Errorf("level %s should not be unlocked or current", level.Code)
			}
		}
	}
}

func TestLevelHandler_GetLevels_UnknownLevel(t *testing.T) {
	mockStore := &MockLevelStore{
		ListLevelsFunc: func(ctx context.Context) ([]model.CEFRLevel, error) {
			return testLevels(), nil
		},
	}

	handler := NewLevelHandler(mockStore)

	// Request with unknown level
	user := &TrustedUser{
		ID:        "user-789",
		Role:      "student",
		CEFRLevel: "XX", // Unknown level
		Email:     "student3@example.com",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
	req = req.WithContext(context.WithValue(req.Context(), trustedUserContextKey, user))

	rr := httptest.NewRecorder()
	handler.GetLevels(rr, req)

	AssertStatus(t, rr, http.StatusOK)

	var response model.LevelsResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Unknown level should not match any, so currentRank stays at 1 (A1)
	if response.CurrentLevel != "XX" {
		t.Errorf("expected CurrentLevel 'XX', got '%s'", response.CurrentLevel)
	}

	// But only A1 should be unlocked (rank_order <= 1)
	for _, level := range response.Levels {
		if level.Code == "A1" {
			if !level.Unlocked {
				t.Errorf("A1 should be unlocked")
			}
		} else {
			if level.Unlocked {
				t.Errorf("level %s should not be unlocked for unknown level", level.Code)
			}
		}
	}
}

func TestLevelHandler_GetLevels_StoreError(t *testing.T) {
	mockStore := &MockLevelStore{
		ListLevelsFunc: func(ctx context.Context) ([]model.CEFRLevel, error) {
			return nil, ErrTestStoreError
		},
	}

	handler := NewLevelHandler(mockStore)

	user := &TrustedUser{
		ID:        "user-123",
		Role:      "student",
		CEFRLevel: "A1",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
	req = req.WithContext(context.WithValue(req.Context(), trustedUserContextKey, user))

	rr := httptest.NewRecorder()
	handler.GetLevels(rr, req)

	AssertStatus(t, rr, http.StatusInternalServerError)

	var response ErrorResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if response.Error != "internal server error" {
		t.Errorf("expected error 'internal server error', got '%s'", response.Error)
	}
}

func TestLevelHandler_GetLevels_MissingUserID(t *testing.T) {
	mockStore := &MockLevelStore{}
	handler := NewLevelHandler(mockStore)

	// Request WITHOUT user in context (simulating missing middleware)
	req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)

	rr := httptest.NewRecorder()
	handler.GetLevels(rr, req)

	AssertStatus(t, rr, http.StatusUnauthorized)

	var response ErrorResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if response.Error != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got '%s'", response.Error)
	}
}

func TestLevelHandler_GetLevels_EmptyLevels(t *testing.T) {
	mockStore := &MockLevelStore{
		ListLevelsFunc: func(ctx context.Context) ([]model.CEFRLevel, error) {
			return []model.CEFRLevel{}, nil
		},
	}

	handler := NewLevelHandler(mockStore)

	user := &TrustedUser{
		ID:        "user-123",
		Role:      "student",
		CEFRLevel: "A1",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
	req = req.WithContext(context.WithValue(req.Context(), trustedUserContextKey, user))

	rr := httptest.NewRecorder()
	handler.GetLevels(rr, req)

	AssertStatus(t, rr, http.StatusOK)

	var response model.LevelsResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Levels) != 0 {
		t.Errorf("expected 0 levels, got %d", len(response.Levels))
	}
}

// ErrTestStoreError is a test error constant.
const ErrTestStoreError = storeErr("test store error")

type storeErr string

func (e storeErr) Error() string {
	return string(e)
}