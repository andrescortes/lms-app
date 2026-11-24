package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON_Success(t *testing.T) {
	data := map[string]string{"key": "value"}

	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, data)

	AssertStatus(t, rr, http.StatusOK)
	AssertContentType(t, rr, "application/json")

	var response map[string]string
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["key"] != "value" {
		t.Errorf("expected value 'value', got '%s'", response["key"])
	}
}

func TestWriteJSON_Struct(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	data := TestStruct{Name: "test", Value: 42}

	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusCreated, data)

	AssertStatus(t, rr, http.StatusCreated)

	var response TestStruct
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "test" {
		t.Errorf("expected Name 'test', got '%s'", response.Name)
	}
	if response.Value != 42 {
		t.Errorf("expected Value 42, got %d", response.Value)
	}
}

func TestWriteJSON_EmptyStruct(t *testing.T) {
	data := struct{}{}

	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, data)

	AssertStatus(t, rr, http.StatusOK)

	var response map[string]interface{}
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("expected empty object, got %v", response)
	}
}

func TestWriteJSON_Nil(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, nil)

	AssertStatus(t, rr, http.StatusOK)

	var response interface{}
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response != nil {
		t.Errorf("expected null, got %v", response)
	}
}

func TestWriteJSON_Slice(t *testing.T) {
	data := []string{"a", "b", "c"}

	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, data)

	AssertStatus(t, rr, http.StatusOK)

	var response []string
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 3 {
		t.Errorf("expected 3 elements, got %d", len(response))
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, http.StatusNotFound, "resource not found")

	AssertStatus(t, rr, http.StatusNotFound)
	AssertContentType(t, rr, "application/json")

	var response ErrorResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error != "resource not found" {
		t.Errorf("expected error 'resource not found', got '%s'", response.Error)
	}
}

func TestWriteError_EmptyMessage(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, http.StatusBadRequest, "")

	AssertStatus(t, rr, http.StatusBadRequest)

	var response ErrorResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error != "" {
		t.Errorf("expected empty error, got '%s'", response.Error)
	}
}

func TestWriteError_SpecialCharacters(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, http.StatusInternalServerError, `error with "quotes" and \backslashes\`)

	AssertStatus(t, rr, http.StatusInternalServerError)

	var response ErrorResponse
	if err := DecodeJSON(rr, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expected := `error with "quotes" and \backslashes\`
	if response.Error != expected {
		t.Errorf("expected error %q, got %q", expected, response.Error)
	}
}