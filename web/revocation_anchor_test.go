package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/delegation"
)

// TestRevocationAnchoring ensures audit meta includes anchor fields when enabled.
func TestRevocationAnchoring(t *testing.T) {
	os.Setenv("GAUTH_ANCHOR_PROVIDER", "memory")
	os.Setenv("GAUTH_ANCHOR_REVOCATIONS", "1")
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	defer func() {
		os.Unsetenv("GAUTH_ANCHOR_PROVIDER")
		os.Unsetenv("GAUTH_ANCHOR_REVOCATIONS")
		os.Unsetenv("GAUTH_TOKEN_SIG_MODE")
	}()
	s := NewBetaServer("")
	// Key manager should be initialized already in server init; if not, force it.
	if crypto.GlobalEdDSARegistry == nil {
		km, _ := crypto.NewManager(time.Hour)
		crypto.GlobalEdDSARegistry = km
	}
	// Append a revocation event via chain directly (simpler than hitting revoke endpoint)
	if s.revocationChain == nil {
		t.Fatalf("revocation chain expected")
	}
	_, err := s.revocationChain.Append(delegation.RevocationEvent{ID: "rev-anchor-1", DelegationID: "d-1"})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	// Query audit logs endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit/logs", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	body := w.Body.String()
	if !(containsString(body, "anchor_hash") && containsString(body, "revocation_hash")) {
		t.Fatalf("expected anchor_hash & revocation_hash in audit log body: %s", body)
	}
}

// containsString is a small helper to avoid importing strings repeatedly.
func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool { return indexOf(s, sub) >= 0 })()
}

// indexOf naive search (avoid strings.Contains to keep minimal imports for this test file).
func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
