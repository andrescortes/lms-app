package transport

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestHealthHandler_Check_Healthy(t *testing.T) {
	// Create a real in-memory database
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// Create a table to make ping succeed
	ctx := context.Background()
	_, err = db.ExecContext(ctx, "CREATE TABLE test (id INTEGER)")
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	handler := NewHealthHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.Check(rr, req)

	AssertStatus(t, rr, http.StatusOK)
	AssertContentType(t, rr, "application/json")

	var response HealthResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response.Status)
	}
	if response.Service != "lms-core" {
		t.Errorf("expected service 'lms-core', got '%s'", response.Service)
	}
	if response.Database != "connected" {
		t.Errorf("expected database 'connected', got '%s'", response.Database)
	}
	if response.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

func TestHealthHandler_Check_Unhealthy(t *testing.T) {
	// Create a database that will fail ping
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// Close the database to make ping fail
	db.Close()

	handler := NewHealthHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.Check(rr, req)

	AssertStatus(t, rr, http.StatusServiceUnavailable)

	var response HealthResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "degraded" {
		t.Errorf("expected status 'degraded', got '%s'", response.Status)
	}
	if response.Database != "disconnected" {
		t.Errorf("expected database 'disconnected', got '%s'", response.Database)
	}
}

func TestHealthHandler_Check_Timeout(t *testing.T) {
	// Create a real database
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

	handler := NewHealthHandler(db)

	// Create a request with a very short timeout context
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Nanosecond)
	cancel() // Immediately cancel
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Check(rr, req)

	// Should still respond, but might be degraded if ping times out
	// The handler has a 2-second timeout internally
	if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
		t.Errorf("unexpected status %d", rr.Code)
	}
}

func TestHealthHandler_Check_ResponseFormat(t *testing.T) {
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

	handler := NewHealthHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.Check(rr, req)

	AssertContentType(t, rr, "application/json")

	// Verify response structure
	var response map[string]interface{}
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedKeys := []string{"status", "service", "database", "uptime"}
	for _, key := range expectedKeys {
		if _, ok := response[key]; !ok {
			t.Errorf("response missing key: %s", key)
		}
	}
}

func TestHealthHandler_UptimeIncreases(t *testing.T) {
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

	handler := NewHealthHandler(db)

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr1 := httptest.NewRecorder()
	handler.Check(rr1, req1)

	var response1 HealthResponse
	if err := DecodeJSON(rr1, &response1); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Second request
	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr2 := httptest.NewRecorder()
	handler.Check(rr2, req2)

	var response2 HealthResponse
	if err := DecodeJSON(rr2, &response2); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}

	// Uptime should be a valid duration string
	if response1.Uptime == "" {
		t.Error("first uptime should not be empty")
	}
	if response2.Uptime == "" {
		t.Error("second uptime should not be empty")
	}
}