package rfc0111

import "testing"

func TestValidateMultiSignature(t *testing.T) {
	poa := &PowerOfAttorney{ID: "m1", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, Signers: []string{"alice", "agent"}, Threshold: 2}
	if err := ValidateMultiSignature(poa); err != nil {
		t.Fatalf("valid multi-signature failed: %v", err)
	}
	// duplicate
	poa2 := &PowerOfAttorney{ID: "m2", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, Signers: []string{"alice", "alice"}, Threshold: 2}
	if err := ValidateMultiSignature(poa2); err == nil {
		t.Fatalf("expected duplicate signer error")
	}
	// insufficient signers
	poa3 := &PowerOfAttorney{ID: "m3", Grantor: "alice", Grantee: "agent", Scope: []string{"read"}, Signers: []string{"alice"}, Threshold: 2}
	if err := ValidateMultiSignature(poa3); err == nil {
		t.Fatalf("expected insufficient signers error")
	}
}
