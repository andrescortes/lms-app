package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustedGatewayMiddleware_Success(t *testing.T) {
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
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Level", "B2")
	req.Header.Set("X-User-Email", "admin@example.com")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusOK)

	if receivedUser == nil {
		t.Fatal("receivedUser is nil")
	}
	if receivedUser.ID != "user-123" {
		t.Errorf("expected ID 'user-123', got '%s'", receivedUser.ID)
	}
	if receivedUser.Role != "admin" {
		t.Errorf("expected Role 'admin', got '%s'", receivedUser.Role)
	}
	if receivedUser.CEFRLevel != "B2" {
		t.Errorf("expected CEFRLevel 'B2', got '%s'", receivedUser.CEFRLevel)
	}
	if receivedUser.Email != "admin@example.com" {
		t.Errorf("expected Email 'admin@example.com', got '%s'", receivedUser.Email)
	}
}

func TestTrustedGatewayMiddleware_Defaults(t *testing.T) {
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

	// Only X-User-ID is required
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "user-456")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusOK)

	if receivedUser == nil {
		t.Fatal("receivedUser is nil")
	}
	if receivedUser.ID != "user-456" {
		t.Errorf("expected ID 'user-456', got '%s'", receivedUser.ID)
	}
	if receivedUser.Role != "student" {
		t.Errorf("expected default Role 'student', got '%s'", receivedUser.Role)
	}
	if receivedUser.CEFRLevel != "A1" {
		t.Errorf("expected default CEFRLevel 'A1', got '%s'", receivedUser.CEFRLevel)
	}
	if receivedUser.Email != "" {
		t.Errorf("expected empty Email, got '%s'", receivedUser.Email)
	}
}

func TestTrustedGatewayMiddleware_TrimsWhitespace(t *testing.T) {
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
	req.Header.Set("X-User-ID", "  user-789  ")
	req.Header.Set("X-User-Role", "  teacher  ")
	req.Header.Set("X-User-Level", "  c1  ")
	req.Header.Set("X-User-Email", "  teacher@example.com  ")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusOK)

	if receivedUser == nil {
		t.Fatal("receivedUser is nil")
	}
	if receivedUser.ID != "user-789" {
		t.Errorf("expected trimmed ID 'user-789', got '%s'", receivedUser.ID)
	}
	if receivedUser.Role != "teacher" {
		t.Errorf("expected trimmed Role 'teacher', got '%s'", receivedUser.Role)
	}
	if receivedUser.CEFRLevel != "C1" {
		t.Errorf("expected trimmed and uppercased CEFRLevel 'C1', got '%s'", receivedUser.CEFRLevel)
	}
	if receivedUser.Email != "teacher@example.com" {
		t.Errorf("expected trimmed Email 'teacher@example.com', got '%s'", receivedUser.Email)
	}
}

func TestTrustedGatewayMiddleware_MissingUserID(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	middleware := TrustedGatewayMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No X-User-ID header

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusUnauthorized)

	// Verify error response
	var response ErrorResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if response.Error != "missing trusted gateway identity headers" {
		t.Errorf("expected error 'missing trusted gateway identity headers', got '%s'", response.Error)
	}
}

func TestTrustedGatewayMiddleware_EmptyUserID(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	middleware := TrustedGatewayMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "") // Empty

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusUnauthorized)
}

func TestTrustedGatewayMiddleware_CaseInsensitiveLevel(t *testing.T) {
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
		{"A1", "A1"},
		{"b2", "B2"},
		{"B2", "B2"},
		{"c1", "C1"},
		{"C2", "C2"},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-User-ID", "user-123")
		req.Header.Set("X-User-Level", tc.input)

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		AssertStatus(t, rr, http.StatusOK)

		if receivedUser.CEFRLevel != tc.expected {
			t.Errorf("input %q: expected CEFRLevel %q, got %q", tc.input, tc.expected, receivedUser.CEFRLevel)
		}
	}
}

func TestGetTrustedUser_NotFound(t *testing.T) {
	ctx := context.Background()
	user, ok := GetTrustedUser(ctx)
	if ok {
		t.Errorf("expected ok=false, got ok=true with user=%v", user)
	}
	if user != nil {
		t.Errorf("expected user=nil, got %v", user)
	}
}

func TestGetTrustedUser_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), trustedUserContextKey, "not a user")
	user, ok := GetTrustedUser(ctx)
	if ok {
		t.Errorf("expected ok=false, got ok=true with user=%v", user)
	}
	if user != nil {
		t.Errorf("expected user=nil, got %v", user)
	}
}

func TestGetTrustedUser_NilValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), trustedUserContextKey, (*TrustedUser)(nil))
	user, ok := GetTrustedUser(ctx)
	if ok {
		t.Errorf("expected ok=false, got ok=true with user=%v", user)
	}
	if user != nil {
		t.Errorf("expected user=nil, got %v", user)
	}
}

func TestRecoveryMiddleware_RecoversFromPanic(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := recoveryMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusInternalServerError)

	var response ErrorResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if response.Error != "internal server error" {
		t.Errorf("expected error 'internal server error', got '%s'", response.Error)
	}
}

func TestRecoveryMiddleware_PassesThroughNormalRequests(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := recoveryMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	AssertStatus(t, rr, http.StatusOK)
	if rr.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got '%s'", rr.Body.String())
	}
}