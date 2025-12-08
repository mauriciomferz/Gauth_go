package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// helper to build an engine with the corsMiddleware only and a simple endpoint
func newTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
	return r
}

func corsRequest(t *testing.T, r http.Handler, method, path, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func withEnv(t *testing.T, key, val string, fn func()) {
	old := os.Getenv(key)
	if err := os.Setenv(key, val); err != nil {
		t.Fatalf("set env failed: %v", err)
	}
	defer func() { _ = os.Setenv(key, old) }()
	fn()
}

// TestCORSAllowAll validates that wildcard ("*") configuration reflects any origin.
func TestCORSAllowAll(t *testing.T) {
	withEnv(t, "GAUTH_CORS_ALLOW", "*", func() {
		engine := newTestEngine()
		origin := "http://localhost:5173"
		w := corsRequest(t, engine, http.MethodGet, "/ping", origin)
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("expected allow-origin header to echo origin, got=%q", got)
		}
		if vary := w.Header().Get("Vary"); vary != "Origin" {
			t.Fatalf("expected Vary=Origin, got=%q", vary)
		}
	})
}

// TestCORSAllowList verifies only listed origins are reflected.
func TestCORSAllowList(t *testing.T) {
	withEnv(t, "GAUTH_CORS_ALLOW", "https://app.example.com, https://admin.example.com", func() {
		engine := newTestEngine()
		allowed := "https://admin.example.com"
		disallowed := "https://evil.example.com"

		// Allowed origin
		w1 := corsRequest(t, engine, http.MethodGet, "/ping", allowed)
		if got := w1.Header().Get("Access-Control-Allow-Origin"); got != allowed {
			t.Fatalf("expected allowed origin header, got=%q", got)
		}

		// Disallowed origin should not get header
		w2 := corsRequest(t, engine, http.MethodGet, "/ping", disallowed)
		if got := w2.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("expected no allow-origin header for disallowed origin, got=%q", got)
		}
	})
}

// TestCORSPreflight ensures OPTIONS request short-circuits with 204 and proper headers when allowed.
func TestCORSPreflight(t *testing.T) {
	withEnv(t, "GAUTH_CORS_ALLOW", "https://app.example.com", func() {
		engine := newTestEngine()
		origin := "https://app.example.com"
		w := corsRequest(t, engine, http.MethodOptions, "/ping", origin)
		if w.Code != 204 {
			t.Fatalf("expected preflight status 204, got=%d", w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("expected allow-origin header for preflight, got=%q", got)
		}
		// Methods header should include OPTIONS
		if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" || !strings.Contains(methods, "OPTIONS") {
			t.Fatalf("expected Access-Control-Allow-Methods to include OPTIONS, got=%q", methods)
		}
	})
}
