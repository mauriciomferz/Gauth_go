package verification

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
	delegation "github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

// TestVerifyAllSuccess constructs a minimal chain + STH and ensures VerifyAll returns nil.
func TestVerifyAllSuccess(t *testing.T) {
	km, _ := crypto.NewManager(time.Hour)
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	t.Setenv("AGENTAUTH_MULTI_SIG_THRESHOLD", "1")
	rc := delegation.NewRevocationChain(delegation.WithKeyProvider(km))
	for i := 0; i < 3; i++ {
		id := "succ-" + time.Now().Format("150405") + string(rune('a'+i))
		_, _ = rc.Append(delegation.RevocationEvent{
			ID:           id,
			DelegationID: "del",
		})
	}
	if _, err := rc.SignTreeHead(); err != nil {
		t.Fatalf("sign: %v", err)
	}
	ts := buildTestServer(rc, true, km)
	defer ts.Close()
	events := rc.Events()
	lastHash := events[len(events)-1].Hash
	if err := VerifyAll(ts.Client(), ts.URL, lastHash); err != nil {
		t.Fatalf("VerifyAll returned error: %v", err)
	}
}
