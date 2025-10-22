package web

import (
	"net/http/httptest"
	"testing"
)

// NewTestServerNoSeed provides a BetaServer with policy seeding disabled so tests
// can deterministically control bundle counts.
func NewTestServerNoSeed(t *testing.T) *BetaServer {
	// Ensure we don't auto-seed demo bundle; tests should add precisely what they need.
	// Use Setenv to keep setting local to this test.
	t.Setenv("GAUTH_SEED_POLICY", "0")
	return NewBetaServer("")
}

// PerformRequest performs an HTTP request against the beta server returning recorder.
// Added helper to allow external test package to exercise endpoints without accessing unexported router field.
func PerformRequest(s *BetaServer, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	s.router.ServeHTTP(w, req)
	return w
}
