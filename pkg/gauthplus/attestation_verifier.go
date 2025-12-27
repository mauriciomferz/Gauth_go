package gauthplus

import (
	"context"
	"crypto/ed25519"
	"fmt"

	"github.com/mauriciomferz/Gauth_go/pkg/compliance"
)

// DefaultAttestationVerifier is a robust implementation using compliance core.
type DefaultAttestationVerifier struct {
	complianceVerif compliance.AttestationVerifier
}

// NewDefaultAttestationVerifier creates a new default attestation verifier
func NewDefaultAttestationVerifier() *DefaultAttestationVerifier {
	return &DefaultAttestationVerifier{
		complianceVerif: compliance.NewDefaultAttestationVerifier(),
	}
}

// Verify verifies a gauthplus.Attestation by bridging it to compliance model.
func (v *DefaultAttestationVerifier) Verify(ctx context.Context, att Attestation) (bool, error) {
	// Map to compliance.Attestation
	cAtt := compliance.Attestation{
		ID:        att.ID,
		FlowID:    att.Provider, // Using Provider as FlowID for bridging
		SignerID:  att.Provider,
		Timestamp: att.VerifiedAt.Unix(),
	}

	// Extract proof from Metadata if it exists
	if att.Metadata != nil {
		if proof, ok := att.Metadata["proof"].([]byte); ok {
			cAtt.Proof = proof
		} else if proofStr, ok := att.Metadata["proof"].(string); ok {
			cAtt.Proof = []byte(proofStr)
		}
	}

	// Verify using compliance verifier
	return v.complianceVerif.Verify(cAtt)
}

// AttestationSigner creates cryptographic proofs for attestations
type AttestationSigner interface {
	Sign(ctx context.Context, att Attestation) ([]byte, error)
	SignerID() string
}

// DefaultAttestationSigner implements AttestationSigner using ed25519
type DefaultAttestationSigner struct {
	signerID string
	privKey  ed25519.PrivateKey
}

// NewDefaultAttestationSigner creates a new default attestation signer
func NewDefaultAttestationSigner(signerID string, privKey ed25519.PrivateKey) *DefaultAttestationSigner {
	return &DefaultAttestationSigner{
		signerID: signerID,
		privKey:  privKey,
	}
}

// Sign signs an attestation and returns the proof
func (s *DefaultAttestationSigner) Sign(ctx context.Context, att Attestation) ([]byte, error) {
	msg := fmt.Sprintf("%s:%s:%d", att.ID, att.Provider, att.VerifiedAt.Unix())
	return ed25519.Sign(s.privKey, []byte(msg)), nil
}

// SignerID returns the signer identity
func (s *DefaultAttestationSigner) SignerID() string {
	return s.signerID
}

// RegisterKey allows registering a public key for verification (bridge to compliance)
func (v *DefaultAttestationVerifier) RegisterKey(signerID string, pubKey ed25519.PublicKey) {
	if dv, ok := v.complianceVerif.(*compliance.DefaultAttestationVerifier); ok {
		dv.RegisterKey(signerID, pubKey)
	}
}
