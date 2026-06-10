package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_ReturnsOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	HealthHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("expected body %q, got %q", "ok", rec.Body.String())
	}
}

func TestHealthHandler_AnyMethod(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
		req := httptest.NewRequest(method, "/healthz", nil)
		rec := httptest.NewRecorder()

		HealthHandler()(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("method %s: expected status 200, got %d", method, rec.Code)
		}
	}
}
