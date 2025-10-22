package verification

import (
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	delegation "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/delegation"
)

// TestVerifyAllSuccess constructs a minimal chain + STH and ensures VerifyAll returns nil.
func TestVerifyAllSuccess(t *testing.T) {
	km, _ := crypto.NewManager(time.Hour)
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	os.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "1")
	rc := delegation.NewRevocationChain()
	for i := 0; i < 3; i++ {
		_, _ = rc.Append(delegation.RevocationEvent{ID: "succ-" + time.Now().Format("150405") + string(rune('a'+i)), DelegationID: "del"})
	}
	if _, err := rc.SignTreeHead(); err != nil {
		t.Fatalf("sign: %v", err)
	}
	ts := buildTestServer(rc, true)
	defer ts.Close()
	if err := VerifyAll(ts.Client(), ts.URL, ""); err != nil {
		t.Fatalf("VerifyAll returned error: %v", err)
	}
}
