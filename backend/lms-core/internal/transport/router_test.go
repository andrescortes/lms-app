package transport

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"lms-app/backend/lms-core/internal/model"

	_ "modernc.org/sqlite"
)

// TestRouter_HealthEndpoint tests the health endpoint routing.
func TestRouter_HealthEndpoint(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	_, err = db.ExecContext(ctx, "CREATE TABLE test (id INTEGER)")
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	healthHandler := NewHealthHandler(db)
	levelHandler := NewLevelHandler(nil) // Not used for health check

	router := NewRouter(healthHandler, levelHandler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusOK)

	var response HealthResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response.Status)
	}
}

// TestRouter_LevelsEndpoint tests the levels endpoint routing with middleware.
func TestRouter_LevelsEndpoint(t *testing.T) {
	mockStore := &MockLevelStore{
		ListLevelsFunc: func(ctx context.Context) ([]model.CEFRLevel, error) {
			return testLevels(), nil
		},
	}

	healthHandler := NewHealthHandler(nil) // Not used
	levelHandler := NewLevelHandler(mockStore)

	router := NewRouter(healthHandler, levelHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
	req.Header.Set("X-User-ID", "user-123")
	req.Header.Set("X-User-Role", "student")
	req.Header.Set("X-User-Level", "B1")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusOK)

	var response model.LevelsResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got '%s'", response.UserID)
	}
	if len(response.Levels) != 6 {
		t.Errorf("expected 6 levels, got %d", len(response.Levels))
	}
}

// TestRouter_LevelsEndpoint_WithoutAuth tests that levels endpoint requires auth.
func TestRouter_LevelsEndpoint_WithoutAuth(t *testing.T) {
	mockStore := &MockLevelStore{}
	healthHandler := NewHealthHandler(nil)
	levelHandler := NewLevelHandler(mockStore)

	router := NewRouter(healthHandler, levelHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
	// No auth headers

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusUnauthorized)
}

// TestRouter_NotFound tests 404 for unknown routes.
func TestRouter_NotFound(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	_, err = db.ExecContext(ctx, "CREATE TABLE test (id INTEGER)")
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	healthHandler := NewHealthHandler(db)
	levelHandler := NewLevelHandler(nil)

	router := NewRouter(healthHandler, levelHandler)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusNotFound)
}

// TestRouter_MethodNotAllowed tests 405 for wrong method.
func TestRouter_MethodNotAllowed(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	_, err = db.ExecContext(ctx, "CREATE TABLE test (id INTEGER)")
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	healthHandler := NewHealthHandler(db)
	levelHandler := NewLevelHandler(nil)

	router := NewRouter(healthHandler, levelHandler)

	// Health only accepts GET
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusMethodNotAllowed)
}

// TestRouter_RecoveryMiddleware tests that panic in handler is recovered.
func TestRouter_RecoveryMiddleware(t *testing.T) {
	panicHandler := NewLevelHandler(&MockLevelStore{
		ListLevelsFunc: func(ctx context.Context) ([]model.CEFRLevel, error) {
			panic("test panic in handler")
		},
	})

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	_, err = db.ExecContext(ctx, "CREATE TABLE test (id INTEGER)")
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	healthHandler := NewHealthHandler(db)

	router := NewRouter(healthHandler, panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
	req.Header.Set("X-User-ID", "user-123")
	req.Header.Set("X-User-Level", "A1")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should recover and return 500
	AssertStatus(t, rr, http.StatusInternalServerError)

	var response ErrorResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if response.Error != "internal server error" {
		t.Errorf("expected error 'internal server error', got '%s'", response.Error)
	}
}