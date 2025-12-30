package web

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
	"github.com/mauriciomferz/AgentAuth/pkg/verification"
)

// httpClientAdapter implements verification.HTTPClient
// (no adapter needed; httptest.Client implements http.Get method and fits interface)

func TestVerificationPackageEndToEnd(t *testing.T) {
	t.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	km, err := crypto.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	s := NewBetaServer("", WithKeyProvider(km))
	t.Cleanup(func() { s.Shutdown() })
	t.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "1")
	rc := delegation.NewRevocationChain(delegation.WithKeyProvider(km))
	for i := 0; i < 4; i++ {
		_, _ = rc.Append(delegation.RevocationEvent{ID: "rev-vpkg-int-" + time.Now().Format("150405") + string(rune('a'+i)), DelegationID: "del-vpkg"})
		time.Sleep(2 * time.Millisecond)
	}
	if _, err := rc.SignTreeHead(); err != nil {
		t.Fatalf("sign head: %v", err)
	}
	s.revocationChain = rc
	ts := httptest.NewServer(s.router)
	defer ts.Close()
	client := ts.Client()
	events := rc.Events()
	lastHash := events[len(events)-1].Hash
	if err := verification.VerifyAll(client, ts.URL, lastHash); err != nil {
		t.Fatalf("VerifyAll error: %v", err)
	}
}
