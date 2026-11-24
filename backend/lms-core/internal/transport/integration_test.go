package transport

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"lms-app/backend/lms-core/internal/model"
	"lms-app/backend/lms-core/internal/store"

	_ "modernc.org/sqlite"
)

// TestIntegration_FullStack tests the complete flow from HTTP request to database.
func TestIntegration_FullStack(t *testing.T) {
	// Setup real database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := store.NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := store.RunMigrations(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Setup all layers
	levelStore := store.NewLevelStore(db)
	healthHandler := NewHealthHandler(db)
	levelHandler := NewLevelHandler(levelStore)
	router := NewRouter(healthHandler, levelHandler)

	// Test health endpoint
	t.Run("HealthCheck", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		AssertStatus(t, rr, http.StatusOK)

		var response HealthResponse
		if err := DecodeJSON(rr, &response); err != nil {
			t.Fatalf("health check: failed to decode response: %v", err)
		}

		if response.Status != "ok" {
			t.Errorf("health check: expected status 'ok', got '%s'", response.Status)
		}
		if response.Database != "connected" {
			t.Errorf("health check: expected database 'connected', got '%s'", response.Database)
		}
	})

	// Test levels endpoint
	t.Run("GetLevels", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
		req.Header.Set("X-User-ID", "integration-user-123")
		req.Header.Set("X-User-Role", "student")
		req.Header.Set("X-User-Level", "B2")
		req.Header.Set("X-User-Email", "integration@example.com")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		AssertStatus(t, rr, http.StatusOK)

		var response model.LevelsResponse
		if err := DecodeJSON(rr, &response); err != nil {
			t.Fatalf("get levels: failed to decode response: %v", err)
		}

		if response.UserID != "integration-user-123" {
			t.Errorf("get levels: expected UserID 'integration-user-123', got '%s'", response.UserID)
		}
		if response.CurrentLevel != "B2" {
			t.Errorf("get levels: expected CurrentLevel 'B2', got '%s'", response.CurrentLevel)
		}
		if len(response.Levels) != 6 {
			t.Errorf("get levels: expected 6 levels, got %d", len(response.Levels))
		}

		// Verify level enrichment logic
		// B2 has rank_order 4, so A1, A2, B1, B2 should be unlocked
		expectedUnlocked := map[string]bool{
			"A1": true, "A2": true, "B1": true, "B2": true, "C1": false, "C2": false,
		}
		for _, level := range response.Levels {
			if level.Unlocked != expectedUnlocked[level.Code] {
				t.Errorf("get levels: level %s unlocked=%v, expected %v", level.Code, level.Unlocked, expectedUnlocked[level.Code])
			}
			if level.IsCurrent != (level.Code == "B2") {
				t.Errorf("get levels: level %s IsCurrent=%v, expected %v", level.Code, level.IsCurrent, level.Code == "B2")
			}
		}
	})

	// Test levels endpoint with default level
	t.Run("GetLevels_DefaultLevel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
		req.Header.Set("X-User-ID", "integration-user-456")
		// No X-User-Level header

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		AssertStatus(t, rr, http.StatusOK)

		var response model.LevelsResponse
		if err := DecodeJSON(rr, &response); err != nil {
			t.Fatalf("get levels default: failed to decode response: %v", err)
		}

		if response.CurrentLevel != "A1" {
			t.Errorf("get levels default: expected CurrentLevel 'A1', got '%s'", response.CurrentLevel)
		}
	})

	// Test unauthorized access
	t.Run("Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
		// No headers

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		AssertStatus(t, rr, http.StatusUnauthorized)
	})
}

// TestIntegration_ConcurrentRequests tests concurrent access to the levels endpoint.
func TestIntegration_ConcurrentRequests(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := store.NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := store.RunMigrations(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	levelStore := store.NewLevelStore(db)
	healthHandler := NewHealthHandler(db)
	levelHandler := NewLevelHandler(levelStore)
	router := NewRouter(healthHandler, levelHandler)

	const numRequests = 10
	done := make(chan struct {
		code int
		err  error
	}, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(userID string) {
			req := httptest.NewRequest(http.MethodGet, "/api/lms/levels", nil)
			req.Header.Set("X-User-ID", userID)
			req.Header.Set("X-User-Level", "A1")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			done <- struct {
				code int
				err  error
			}{code: rr.Code}
		}("user-" + string(rune('0'+i)))
	}

	for i := 0; i < numRequests; i++ {
		result := <-done
		if result.err != nil {
			t.Errorf("request %d failed: %v", i, result.err)
		}
		if result.code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, result.code)
		}
	}
}

// TestIntegration_DatabaseFailure tests behavior when database is unavailable.
func TestIntegration_DatabaseFailure(t *testing.T) {
	// Create a handler with a closed database
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	db.Close() // Close immediately to simulate failure

	healthHandler := NewHealthHandler(db)
	levelHandler := NewLevelHandler(nil)
	router := NewRouter(healthHandler, levelHandler)

	// Health check should return degraded
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusServiceUnavailable)

	var response HealthResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("health check: failed to decode response: %v", err)
	}

	if response.Status != "degraded" {
		t.Errorf("health check: expected status 'degraded', got '%s'", response.Status)
	}
	if response.Database != "disconnected" {
		t.Errorf("health check: expected database 'disconnected', got '%s'", response.Database)
	}
}