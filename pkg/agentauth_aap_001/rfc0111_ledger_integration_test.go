package agentauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	cr "github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/pkg/ledger"
)

func TestLedgerIssuanceAndRevocation(t *testing.T) {
	al := audit.NewMemoryLogger(nil)
	az := authz.NewMemoryAuthorizer()
	// Allow grantor to issue and revoke delegations
	az.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	az.AddPolicy(authz.Policy{ID: "allow-revoke", Subject: "alice", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	lstore := ledger.NewMemoryStore()
	kp, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("key provider: %v", err)
	}
	svc := NewService(al, az, WithLedger(lstore), WithMandatorySignatures(), WithSignerProvider(func() (cr.Signer, error) { return kp.ActiveSigner() })).WithClock(time.Now)

	// Issue delegation
	req := DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Duration: time.Hour}
	del, err := svc.CreateDelegationCtx(context.Background(), req)
	if err != nil {
		t.Fatalf("issuance failed: %v", err)
	}

	// Revoke delegation
	if err2 := svc.RevokeDelegation(del.POA.ID, "alice"); err2 != nil {
		t.Fatalf("revocation failed: %v", err2)
	}

	// Query ledger by subject (grantor)
	entries, err := svc.LedgerEntries("alice", "")
	if err != nil {
		t.Fatalf("ledger query failed: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected >=2 ledger entries for issuance + revocation, got %d", len(entries))
	}

	// Basic checks for presence of issuance and revocation types
	hasIss, hasRev := false, false
	for _, e := range entries {
		if e == nil {
			continue
		}
		switch e.Type {
		case "delegation_issuance":
			hasIss = true
		case "delegation_revocation":
			hasRev = true
		}
	}
	if !hasIss {
		t.Errorf("missing delegation_issuance entry")
	}
	if !hasRev {
		t.Errorf("missing delegation_revocation entry")
	}

	// Query by object (delegation ID) should yield both records
	objEntries, err := svc.LedgerEntries("", del.POA.ID)
	if err != nil {
		t.Fatalf("object ledger query failed: %v", err)
	}
	if len(objEntries) < 2 {
		t.Errorf("expected object query to return >=2 entries, got %d", len(objEntries))
	}

	// Verify chain integrity directly
	vr, err := lstore.VerifyChain(context.Background())
	if err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	if vr.Count < 2 {
		t.Errorf("expected at least 2 entries in chain, got %d", vr.Count)
	}
	if vr.Mismatches != 0 {
		t.Errorf("unexpected mismatches: %d", vr.Mismatches)
	}
}
