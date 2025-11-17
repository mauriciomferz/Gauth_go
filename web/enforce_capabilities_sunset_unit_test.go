package web

import (
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/capability"
)

// TestEnforceCapabilitiesSunsetUnit directly exercises enforceCapabilities to ensure
// a capability past its SunsetAfter is treated as missing when GAUTH_CAP_LIFECYCLE_SUNSET_ENFORCE=1.
func TestEnforceCapabilitiesSunsetUnit(t *testing.T) {
	t.Setenv("GAUTH_CAPABILITY_ENFORCE", "1")
	t.Setenv("GAUTH_CAP_LIFECYCLE_SUNSET_ENFORCE", "1")
	capability.Reset([]capability.Capability{})
	// Sunset time in past relative to test date
	capability.Register(capability.Capability{ID: "cap.unit.sunset", Version: "1.0", Stable: true, SunsetAfter: "2025-01-01T00:00:00Z"})
	srv := NewBetaServer("")
	t.Cleanup(func() { srv.Shutdown() })
	// Override required action mapping directly
	srv.requiredActionCaps["delegation:create"] = []string{"cap.unit.sunset"}
	// Claims provide the capability (which should be denied due to sunset)
	claims := map[string]any{"cap": []string{"cap.unit.sunset"}}
	allowed, missing := srv.enforceCapabilities("delegation:create", claims)
	if allowed {
		t.Fatalf("expected enforcement denial after sunset; got allowed=true")
	}
	if len(missing) == 0 {
		t.Fatalf("expected missing slice to include sunset marker")
	}
	// Expect first missing entry has suffix (sunset)
	found := false
	for _, m := range missing {
		if m == "cap.unit.sunset(sunset)" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing entry with sunset marker; got %v", missing)
	}
}
