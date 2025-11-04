package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	delegation "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/delegation"
	"github.com/gin-gonic/gin"
)

// TestRevocationMerkleProofTamperDetection ensures that altering any sibling digest
// in the returned proof causes verification failure. This guards against a class
// of tampering where an attacker attempts to swap a sibling to fabricate a root
// without recomputing all affected parent hashes.
func TestRevocationMerkleProofTamperDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := NewBetaServer("")
	if s.revocationChain == nil {
		s.revocationChain = delegation.NewRevocationChain()
	}
	// Append deterministic events
	for i := 0; i < 5; i++ {
		ev := delegation.RevocationEvent{ID: string(rune('x' + i)), DelegationID: string(rune('X' + i)), Reason: string(delegation.RevocationReasonUserRequest)}
		if _, err := s.revocationChain.Append(ev); err != nil {
			t.Fatalf("append revocation: %v", err)
		}
		time.Sleep(1 * time.Millisecond)
	}
	target := s.revocationChain.Events()[2] // middle element to exercise both sides
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/token/revocation/proof?id="+target.ID, nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool                         `json:"success"`
		Target     string                       `json:"target"`
		MerkleRoot string                       `json:"merkle_root"`
		Proof      []delegation.MerkleProofStep `json:"proof"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || resp.MerkleRoot == "" {
		t.Fatalf("malformed response: %+v", resp)
	}
	if len(resp.Proof) == 0 {
		t.Fatalf("expected non-empty proof for middle element")
	}
	// Verify original proof succeeds
	leafDigest := delegation.LeafDigestForEventHash(target.Hash)
	if !delegation.VerifyProof(leafDigest, resp.Proof, resp.MerkleRoot) {
		t.Fatalf("original proof verification failed")
	}
	// Tamper with first sibling digest (flip first hex char) and expect failure
	tampered := make([]delegation.MerkleProofStep, len(resp.Proof))
	copy(tampered, resp.Proof)
	// Simple mutation: invert first nibble (avoid introducing non-hex by mapping)
	if tampered[0].Sibling[0] == 'a' {
		tampered[0].Sibling = "b" + tampered[0].Sibling[1:]
	} else {
		tampered[0].Sibling = "a" + tampered[0].Sibling[1:]
	}
	if delegation.VerifyProof(leafDigest, tampered, resp.MerkleRoot) {
		t.Fatalf("tampered proof unexpectedly verified")
	}
}
