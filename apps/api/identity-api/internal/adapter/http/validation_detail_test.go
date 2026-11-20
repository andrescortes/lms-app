package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"lms-english/identity-service/internal/adapter/security"
	"lms-english/identity-service/internal/usecase"
)

// registerBody builds a valid registration JSON body without manual quoting.
func registerBody(t *testing.T, email, password string) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return string(body)
}

// TestRegisterNamesSpecificPasswordRule verifies the 422 envelope names the
// precise password rule for each failure (FR-003, SC-004).
func TestRegisterNamesSpecificPasswordRule(t *testing.T) {
	cases := []struct {
		name     string
		password string
		want     string
	}{
		{"too short", "Ab1", "at least 8 characters"},
		{"too long", strings.Repeat("aB3", 24) + "x", "at most 72 characters"},
		{"no uppercase", "abcdefg1", "uppercase letter"},
		{"no digit", "Abcdefgh", "digit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t, stubStore{})
			rec := post(h, "10.20.0.1:1234", registerBody(t, "a@b.com", tc.password))
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
			}
			assertEnvelopeMessage(t, rec.Body.Bytes(), codeValidation, tc.want)
			if strings.Contains(rec.Body.String(), tc.password) {
				t.Fatal("response must not echo the password")
			}
		})
	}
}

// TestRegisterNamesSpecificEmailRule verifies the 422 envelope names the precise
// email format rule for each failure (FR-002, SC-004).
func TestRegisterNamesSpecificEmailRule(t *testing.T) {
	cases := []struct {
		name  string
		email string
		want  string
	}{
		{"missing at", "student.example.com", "contain an @"},
		{"missing dot", "student@example", "domain must contain a dot"},
		{"trailing dot", "student@example.", "must not end with a dot"},
		{"empty local part", "@example.com", "local part"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler(t, stubStore{})
			rec := post(h, "10.20.0.2:1234", registerBody(t, tc.email, "Learn123"))
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422; body=%s", rec.Code, rec.Body.String())
			}
			assertEnvelopeMessage(t, rec.Body.Bytes(), codeValidation, tc.want)
		})
	}
}

// TestRegisterAccepts72CharacterMultibytePassword is the end-to-end check for
// FR-003: a policy-maximum password of accented characters exceeds bcrypt's
// 72-byte limit but must still register successfully.
func TestRegisterAccepts72CharacterMultibytePassword(t *testing.T) {
	uc := usecase.NewRegisterUserUseCase(
		stubStore{},
		security.NewBcryptHasher(bcrypt.MinCost),
		stubID{},
		stubClock{},
		stubPublisher{},
	)
	loginUc := usecase.NewLoginUserUseCase(nil, nil, nil, nil, nil, nil)
	h := NewRouter(uc, loginUc, nil, NewVisitorLimiter(), nil)

	password := strings.Repeat("Á", 71) + "1"
	if len([]rune(password)) != 72 || len(password) <= 72 {
		t.Fatalf("test password has %d runes and %d bytes, want 72 runes and >72 bytes", len([]rune(password)), len(password))
	}

	rec := post(h, "10.20.0.3:1234", registerBody(t, "a@b.com", password))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
}
