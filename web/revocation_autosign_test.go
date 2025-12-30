package web

import (
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

// TestRevocationAutoSignDuplicateSuppression ensures that rotating keys twice without
// adding new revocation events does not emit a second identical SignedTreeHead.
func TestRevocationAutoSignDuplicateSuppression(t *testing.T) {
	t.Setenv("AGENTAUTH_TOKEN_SIG_MODE", sigModeEdDSA)
	defer os.Unsetenv("AGENTAUTH_TOKEN_SIG_MODE")

	km, _ := crypto.NewManager(24 * time.Hour)
	s := NewBetaServer("", WithKeyProvider(km))
	t.Cleanup(func() { s.Shutdown() })
	// Append a single revocation event so auto-sign logic is eligible.
	ev := delegation.RevocationEvent{ID: "rev-test-1", DelegationID: "del1", Reason: string(delegation.RevocationReasonUserRequest)}
	if _, err := s.revocationChain.Append(ev); err != nil {
		t.Fatalf("append revocation: %v", err)
	}

	// Force a manual rotation (first rotation already happened inside NewManager during server init).
	firstHeads := s.revocationChain.TreeHeads()
	initialHeadCount := len(firstHeads)

	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate 1: %v", err)
	}
	// Allow async callback to run (OnKeyRotated executes inline within Rotate, so minimal sleep)
	time.Sleep(10 * time.Millisecond)
	headsAfterFirst := s.revocationChain.TreeHeads()
	if len(headsAfterFirst) != initialHeadCount+1 {
		t.Fatalf("expected one new tree head after first manual rotation; got %d (initial %d)", len(headsAfterFirst), initialHeadCount)
	}

	// Second rotation without new revocation events should be suppressed by duplicate detection.
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate 2: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	headsAfterSecond := s.revocationChain.TreeHeads()
	if len(headsAfterSecond) != len(headsAfterFirst) {
		t.Fatalf("expected no additional tree head (duplicate suppressed); got %d want %d", len(headsAfterSecond), len(headsAfterFirst))
	}
}
