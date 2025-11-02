package poa

import (
	"testing"
	"time"
)

// TestCanonicalDigestStability verifies deterministic digest unaffected by metadata order or evidence map changes.
func TestCanonicalDigestStability(t *testing.T) {
	svc := NewMemoryService()
	poa, err := svc.Issue(nil, &Request{Subject: "alice", Resource: "vault", Action: "read", Scope: []string{"vault.read"}, Context: map[string]interface{}{"z": 1, "a": 2}})
	if err != nil { t.Fatalf("issue failed: %v", err) }
	if poa.Digest == "" { t.Fatalf("digest missing") }
	orig := poa.Digest
	// Mutate Metadata (excluded) should not change digest.
	poa.Metadata["new_field"] = "value"
	if CanonicalDigest(poa) != orig { t.Fatalf("digest changed after metadata mutation") }
	// Mutate Attestation evidence map (excluded) should not change digest.
	if poa.Attestation == nil { t.Fatalf("attestation missing") }
	poa.Attestation.Evidence["extra"] = 42
	if CanonicalDigest(poa) != orig { t.Fatalf("digest changed after attestation evidence mutation") }
}

// TestCanonicalDigestTamper ensures tampering with core fields changes digest and VerifyDigest detects mismatch.
func TestCanonicalDigestTamper(t *testing.T) {
	svc := NewMemoryService()
	poa, err := svc.Issue(nil, &Request{Subject: "bob", Resource: "ledger", Action: "append", Scope: []string{"ledger.append"}})
	if err != nil { t.Fatalf("issue failed: %v", err) }
	if !VerifyDigest(poa) { t.Fatalf("initial digest verify failed") }
	orig := poa.Digest
	// Tamper core field
	poa.Action = "delete"
	if VerifyDigest(poa) { t.Fatalf("verify should fail after tamper") }
	if CanonicalDigest(poa) == orig { t.Fatalf("digest did not change after tamper") }
}

// TestCanonicalDigestDelegationIncluded verifies delegation fields impact digest.
func TestCanonicalDigestDelegationIncluded(t *testing.T) {
	svc := NewMemoryService()
	req := &Request{Subject: "carol", Resource: "db", Action: "query", Scope: []string{"db.query"}, Delegation: &DelegationRequest{DelegatedBy: "root", Scope: []string{"db.query"}, Duration: time.Minute}}
	poa, err := svc.Issue(nil, req)
	if err != nil { t.Fatalf("issue failed: %v", err) }
	d1 := poa.Digest
	// Modify delegation scope -> digest must change
	poa.Delegation.Scope = append(poa.Delegation.Scope, "db.export")
	if CanonicalDigest(poa) == d1 { t.Fatalf("digest did not change after delegation scope modification") }
}
