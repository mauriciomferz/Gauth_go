package web

import (
	"net/http/httptest"
	"testing"
)

// NewTestServer wraps NewBetaServer and registers cleanup to prevent goroutine leaks.
// All tests should use this instead of calling NewBetaServer directly.
func NewTestServer(t *testing.T, port string) *BetaServer {
	srv := NewBetaServer(port)
	t.Cleanup(func() {
		srv.Shutdown()
	})
	return srv
}

// NewTestServerNoSeed provides a BetaServer with policy seeding disabled so tests
// can deterministically control bundle counts.
func NewTestServerNoSeed(t *testing.T) *BetaServer {
	// Ensure we don't auto-seed demo bundle; tests should add precisely what they need.
	// Use Setenv to keep setting local to this test.
	t.Setenv("GAUTH_SEED_POLICY", "0")
	srv := NewBetaServer("")
	t.Cleanup(func() {
		srv.Shutdown()
	})
	return srv
}

// PerformRequest performs an HTTP request against the beta server returning recorder.
// Added helper to allow external test package to exercise endpoints without accessing unexported router field.
func PerformRequest(s *BetaServer, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	s.router.ServeHTTP(w, req)
	return w
}
