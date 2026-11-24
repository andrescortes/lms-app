package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"lms-app/backend/lms-core/internal/model"
	"lms-app/backend/lms-core/internal/store"
)

// ErrorResponse represents the error response structure.
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse represents the health check response structure.
type HealthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	Database string `json:"database"`
	Uptime   string `json:"uptime"`
}

// DecodeJSON decodes the response body into the provided target.
func DecodeJSON(rr *httptest.ResponseRecorder, target interface{}) error {
	return json.NewDecoder(rr.Body).Decode(target)
}

// AssertStatus checks that the response has the expected status code.
func AssertStatus(t interface{ Errorf(string, ...interface{}) }, rr *httptest.ResponseRecorder, expected int) {
	if rr.Code != expected {
		t.Errorf("expected status %d, got %d", expected, rr.Code)
	}
}

// AssertContentType checks that the response has the expected Content-Type.
func AssertContentType(t interface{ Errorf(string, ...interface{}) }, rr *httptest.ResponseRecorder, expected string) {
	contentType := rr.Header().Get("Content-Type")
	if contentType != expected {
		t.Errorf("expected Content-Type %q, got %q", expected, contentType)
	}
}

// NewTestRequest creates a new test request with the given headers.
func NewTestRequest(method, path string, headers map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

// AuthHeaders returns the standard auth headers for testing.
func AuthHeaders(userID, role, level, email string) map[string]string {
	h := map[string]string{
		"X-User-ID": userID,
	}
	if role != "" {
		h["X-User-Role"] = role
	}
	if level != "" {
		h["X-User-Level"] = level
	}
	if email != "" {
		h["X-User-Email"] = email
	}
	return h
}

// MockLevelStore is a mock implementation of the LevelStoreInterface for testing.
type MockLevelStore struct {
	ListLevelsFunc func(ctx context.Context) ([]model.CEFRLevel, error)
}

func (m *MockLevelStore) ListLevels(ctx context.Context) ([]model.CEFRLevel, error) {
	if m.ListLevelsFunc != nil {
		return m.ListLevelsFunc(ctx)
	}
	return nil, nil
}

// Ensure MockLevelStore implements LevelStoreInterface
var _ store.LevelStoreInterface = (*MockLevelStore)(nil)

// testLevels returns a standard set of test levels.
func testLevels() []model.CEFRLevel {
	return []model.CEFRLevel{
		{Code: "A1", Name: "Beginner", Description: "Basic", RankOrder: 1},
		{Code: "A2", Name: "Elementary", Description: "Elementary", RankOrder: 2},
		{Code: "B1", Name: "Intermediate", Description: "Intermediate", RankOrder: 3},
		{Code: "B2", Name: "Upper Intermediate", Description: "Upper Intermediate", RankOrder: 4},
		{Code: "C1", Name: "Advanced", Description: "Advanced", RankOrder: 5},
		{Code: "C2", Name: "Proficiency", Description: "Proficiency", RankOrder: 6},
	}
}