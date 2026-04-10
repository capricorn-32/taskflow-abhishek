package httpapi

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
)

type decodeBodyFixture struct {
	Name string `json:"name"`
}

func TestDecodeJSONBodyRejectsUnknownField(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"ok","extra":"nope"}`))
	w := httptest.NewRecorder()

	var dst decodeBodyFixture
	err := decodeJSONBody(w, req, &dst)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestDecodeJSONBodyRejectsMultipleObjects(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"ok"}{"name":"again"}`))
	w := httptest.NewRecorder()

	var dst decodeBodyFixture
	err := decodeJSONBody(w, req, &dst)
	if err == nil {
		t.Fatal("expected error for multiple JSON objects")
	}
	if !strings.Contains(err.Error(), "single JSON object") {
		t.Fatalf("expected single object error, got %v", err)
	}
}

func TestDecodeJSONBodyRejectsTooLargeBody(t *testing.T) {
	large := bytes.Repeat([]byte("a"), int(maxRequestBodyBytes)+10)
	body := []byte(`{"name":"`)
	body = append(body, large...)
	body = append(body, []byte(`"}`)...)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	w := httptest.NewRecorder()

	var dst decodeBodyFixture
	err := decodeJSONBody(w, req, &dst)
	if err == nil {
		t.Fatal("expected error for oversized request body")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected too large error, got %v", err)
	}
}
