package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeOpenAPI(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()
	serveOpenAPI(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/yaml" {
		t.Fatalf("content-type %q, want application/yaml", ct)
	}
	if !strings.HasPrefix(rec.Body.String(), "openapi:") {
		t.Fatal("expected openapi spec body")
	}
}

func TestServeIndex(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	serveIndex(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "swagger-ui") {
		t.Fatal("expected swagger ui html")
	}
}
