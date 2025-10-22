package notary

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"time"
)

// RotationVerifier encapsulates dependencies required for verifying rotations via HTTP.
// For now, it expects caller to supply descriptors, receipt hashes and key sets.
// Future: integrate with persistent stores.
type RotationVerifier struct {
	Descriptors   []*KeyRotationDescriptor
	ReceiptHashes []string
	OldKeys       []ed25519.PublicKey
	NewKeys       []ed25519.PublicKey
}

// Handler returns an http.HandlerFunc serving GET /api/rotations/verification
// Response JSON: {"generated_at":"RFC3339Nano","summary":RotationVerificationSummary}
func (rv *RotationVerifier) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		start := time.Now()
		summary := VerifyAllRotations(rv.Descriptors, rv.ReceiptHashes, rv.OldKeys, rv.NewKeys)
		recordRotationVerification(start, summary)
		resp := struct {
			GeneratedAt string                      `json:"generated_at"`
			Summary     RotationVerificationSummary `json:"summary"`
		}{GeneratedAt: start.UTC().Format(time.RFC3339Nano), Summary: summary}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		if err := enc.Encode(resp); err != nil {
			// best-effort error response
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
