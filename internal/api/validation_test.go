package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidUUID(t *testing.T) {
	for _, value := range []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550E8400-E29B-41D4-A716-446655440000",
	} {
		if !validUUID(value) {
			t.Errorf("validUUID(%q) = false", value)
		}
	}
	for _, value := range []string{"", "not-a-uuid", "550e8400-e29b-41d4-a716-44665544000z", "550e8400e29b41d4a716446655440000"} {
		if validUUID(value) {
			t.Errorf("validUUID(%q) = true", value)
		}
	}
}

func TestDecodeRejectsTrailingJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"first"} {"name":"second"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	var body struct { Name string `json:"name"` }
	if decode(rec, req, &body) {
		t.Fatal("decode accepted multiple JSON values")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"demo","unexpected":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	var body struct { Name string `json:"name"` }
	if decode(rec, req, &body) {
		t.Fatal("decode accepted an unknown field")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
