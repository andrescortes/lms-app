package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustedGatewayMiddleware_InvalidCEFRLevel(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for invalid level")
	})

	middleware := TrustedGatewayMiddleware(next)

	testCases := []struct {
		name   string
		level  string
		expect string
	}{
		{"unknown level", "XX", "invalid CEFR level: XX"},
		{"lowercase unknown", "xx", "invalid CEFR level: XX"},
		{"numeric", "123", "invalid CEFR level: 123"},
		{"invalid format", "A3", "invalid CEFR level: A3"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-User-ID", "user-123")
			req.Header.Set("X-User-Level", tc.level)

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			AssertStatus(t, rr, http.StatusBadRequest)

			var response ErrorResponse
			if err := DecodeJSON(rr, &response); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if response.Error != tc.expect {
				t.Errorf("expected error %q, got %q", tc.expect, response.Error)
			}
		})
	}
}

// TestTrustedGatewayMiddleware_EmptyLevelDefaultsToA1 tests that empty level defaults to A1
func TestTrustedGatewayMiddleware_EmptyLevelDefaultsToA1(t *testing.T) {
	var receivedUser *TrustedUser
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetTrustedUser(r.Context())
		if !ok {
			t.Error("TrustedUser not found in context")
			return
		}
		receivedUser = user
		w.WriteHeader(http.StatusOK)
	})

	middleware := TrustedGatewayMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "user-123")
	// No X-User-Level header

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusOK)

	if receivedUser == nil {
		t.Fatal("receivedUser is nil")
	}
	if receivedUser.CEFRLevel != "A1" {
		t.Errorf("expected default CEFRLevel 'A1', got %q", receivedUser.CEFRLevel)
	}
}

func TestTrustedGatewayMiddleware_ValidCEFRLevels(t *testing.T) {
	var receivedUser *TrustedUser
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetTrustedUser(r.Context())
		if !ok {
			t.Error("TrustedUser not found in context")
			return
		}
		receivedUser = user
		w.WriteHeader(http.StatusOK)
	})

	middleware := TrustedGatewayMiddleware(next)

	validLevels := []string{"A1", "A2", "B1", "B2", "C1", "C2"}

	for _, level := range validLevels {
		t.Run("valid_"+level, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-User-ID", "user-123")
			req.Header.Set("X-User-Level", level)

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			AssertStatus(t, rr, http.StatusOK)

			if receivedUser == nil {
				t.Fatal("receivedUser is nil")
			}
			if receivedUser.CEFRLevel != level {
				t.Errorf("expected CEFRLevel %q, got %q", level, receivedUser.CEFRLevel)
			}
		})
	}
}

func TestTrustedGatewayMiddleware_CaseInsensitiveValidLevels(t *testing.T) {
	var receivedUser *TrustedUser
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetTrustedUser(r.Context())
		if !ok {
			t.Error("TrustedUser not found in context")
			return
		}
		receivedUser = user
		w.WriteHeader(http.StatusOK)
	})

	middleware := TrustedGatewayMiddleware(next)

	testCases := []struct {
		input    string
		expected string
	}{
		{"a1", "A1"},
		{"a2", "A2"},
		{"b1", "B1"},
		{"b2", "B2"},
		{"c1", "C1"},
		{"c2", "C2"},
		{"A1", "A1"},
		{"C2", "C2"},
	}

	for _, tc := range testCases {
		t.Run("case_"+tc.input, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-User-ID", "user-123")
			req.Header.Set("X-User-Level", tc.input)

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			AssertStatus(t, rr, http.StatusOK)

			if receivedUser == nil {
				t.Fatal("receivedUser is nil")
			}
			if receivedUser.CEFRLevel != tc.expected {
				t.Errorf("input %q: expected CEFRLevel %q, got %q", tc.input, tc.expected, receivedUser.CEFRLevel)
			}
		})
	}
}