package web

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/delegation"
	"github.com/mauriciomferz/Gauth_go/pkg/verification"
)

// httpClientAdapter implements verification.HTTPClient
// (no adapter needed; httptest.Client implements http.Get method and fits interface)

func TestVerificationPackageEndToEnd(t *testing.T) {
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	s := NewBetaServer("")
	t.Cleanup(func() { s.Shutdown() })
	km, err := crypto.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	crypto.GlobalEdDSARegistry = km
	os.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "2")
	rc := delegation.NewRevocationChain()
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
	if err := verification.VerifyAll(client, ts.URL, ""); err != nil {
		t.Fatalf("VerifyAll error: %v", err)
	}
}
