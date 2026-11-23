package transport

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lms-app/backend/gateway-auth/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&_pragma=foreign_keys(ON)")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Apply migrations
	if err := store.RunMigrations(ctx, db); err != nil {
		db.Close()
		require.NoError(t, err)
	}

	return db
}

func TestHealthHandler_Check(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewHealthHandler(db)

	t.Run("healthy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		handler.Check(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var resp healthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, "gateway-auth", resp.Service)
		assert.Equal(t, "connected", resp.Database)
		assert.NotEmpty(t, resp.Uptime)
	})

	t.Run("database disconnected", func(t *testing.T) {
		// Close the database to simulate disconnection
		db.Close()

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		handler.Check(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var resp healthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "error", resp.Status)
		assert.Equal(t, "gateway-auth", resp.Service)
		assert.Equal(t, "disconnected", resp.Database)
	})
}

func TestHealthHandler_ResponseStructure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewHealthHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Check(w, req)

	// Verify all required fields are present
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Contains(t, resp, "status")
	assert.Contains(t, resp, "service")
	assert.Contains(t, resp, "database")
	assert.Contains(t, resp, "uptime")

	// Verify types
	assert.IsType(t, "", resp["status"])
	assert.IsType(t, "", resp["service"])
	assert.IsType(t, "", resp["database"])
	assert.IsType(t, "", resp["uptime"])
}