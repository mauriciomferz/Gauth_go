package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	delegation "github.com/mauriciomferz/Gauth_go/pkg/delegation"
)

// TestRevocationMerkleProofVerification performs end-to-end proof verification by
// requesting a proof for a specific revocation event and recomputing the root
// using the library VerifyProof + LeafDigestForEventHash helpers.
func TestRevocationMerkleProofVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := NewBetaServer("")
	t.Cleanup(func() { s.Shutdown() })
	if s.revocationChain == nil {
		s.revocationChain = delegation.NewRevocationChain()
	}
	// Append deterministic revocation events
	for i := 0; i < 6; i++ {
		ev := delegation.RevocationEvent{ID: string(rune('a' + i)), DelegationID: string(rune('d' + i)), Reason: string(delegation.RevocationReasonUserRequest)}
		if _, err := s.revocationChain.Append(ev); err != nil {
			t.Fatalf("append revocation: %v", err)
		}
		// Small sleep to vary timestamps (not required for Merkle, but keeps chain metadata distinct)
		time.Sleep(2 * time.Millisecond)
	}
	// Choose middle event to exercise both left/right sibling positions
	target := s.revocationChain.Events()[3]
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
	// Compute leaf domain digest from event hash and verify proof recomputes root.
	leafDigest := delegation.LeafDigestForEventHash(target.Hash)
	if !delegation.VerifyProof(leafDigest, resp.Proof, resp.MerkleRoot) {
		t.Fatalf("proof verification failed for target %s", target.ID)
	}
}
