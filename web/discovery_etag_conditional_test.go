package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiscoveryConditionalETag(t *testing.T) {
	bs := NewTestServerNoSeed(t)
	// Wait for async startup anchor to complete
	time.Sleep(100 * time.Millisecond)
	// First request to get ETag
	r1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	bs.router.ServeHTTP(r1, req1)
	if r1.Code != 200 {
		t.Fatalf("expected 200, got %d", r1.Code)
	}
	etag := r1.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("missing ETag")
	}
	// Second with If-None-Match
	r2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	req2.Header.Set("If-None-Match", etag)
	bs.router.ServeHTTP(r2, req2)
	if r2.Code != 304 {
		t.Fatalf("expected 304, got %d", r2.Code)
	}
}

func TestOpenAPIYamlConditionalETag(t *testing.T) {
	bs := NewTestServerNoSeed(t)
	r1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/openapi.yaml", nil)
	bs.router.ServeHTTP(r1, req1)
	if r1.Code != 200 {
		t.Fatalf("expected 200, got %d", r1.Code)
	}
	etag := r1.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("missing ETag")
	}
	// Conditional
	r2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/openapi.yaml", nil)
	req2.Header.Set("If-None-Match", etag)
	bs.router.ServeHTTP(r2, req2)
	if r2.Code != 304 {
		t.Fatalf("expected 304, got %d", r2.Code)
	}
}

func TestOpenAPIJsonConditionalETag(t *testing.T) {
	bs := NewTestServerNoSeed(t)
	r1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/api/v1/openapi", nil)
	bs.router.ServeHTTP(r1, req1)
	if r1.Code != 200 {
		t.Fatalf("expected 200, got %d", r1.Code)
	}
	etag := r1.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("missing ETag")
	}
	r2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/openapi", nil)
	req2.Header.Set("If-None-Match", etag)
	bs.router.ServeHTTP(r2, req2)
	if r2.Code != 304 {
		t.Fatalf("expected 304, got %d", r2.Code)
	}
}
