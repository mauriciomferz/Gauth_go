package gauth_rfc_001

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/attest"
	cr "github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/rfc"
	"github.com/google/uuid"
)

// IssueAttestationProof creates and signs a new AttestationProof.
// Fail-closed semantics: any canonicalization or signing error results in integrity_failure.
func (s *Service) IssueAttestationProof(ctx context.Context, statement, subject string, duration time.Duration) (*attest.AttestationProof, error) {
	if statement == "" || subject == "" {
		return nil, rfc.New(rfc.ErrInvalidRequest, "statement and subject required")
	}
	if len(statement) > 1024 {
		return nil, rfc.New(rfc.ErrInvalidRequest, "statement too large (>1024 bytes)")
	}
	if duration < 0 {
		duration = 0
	}
	now := s.nowFn().UTC()
	proof := &attest.AttestationProof{
		Version:   "att/v1",
		Statement: statement,
		Subject:   subject,
		Issuer:    subject, // default issuer=subject; caller may override with wrapper later
		IssuedAt:  now,
	}
	if duration > 0 {
		proof.ExpiresAt = now.Add(duration)
	}
	// Nonce for replay uniqueness
	proof.Nonce = uuid.New().String()
	// Optional chain binding: if issuance chain exists record current tip hash + algo if available.
	if s.issChain != nil {
		proof.RawPOAChainHash = chainTip(s.issChain)
		// Chain hash algorithm negotiation already recorded on envelopes; reuse env var logic.
		alg := lookupRawPOAHashAlgo()
		if alg != "" {
			proof.RawPOAChainAlgo = alg
		}
	}
	// Signer required
	if s.signerProvider == nil {
		if s.metrics != nil {
			s.metrics.IncAttestationProofIssueFailures()
		}
		return nil, rfc.New(rfc.ErrIntegrityFailure, "signer unavailable")
	}
	signer, err := s.signerProvider()
	if err != nil || signer == nil {
		if s.metrics != nil {
			s.metrics.IncAttestationProofIssueFailures()
		}
		return nil, rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("signer error: %v", err))
	}
	dig, canon, derr := attest.CanonicalAttestationDigest(proof)
	if derr != nil {
		if s.metrics != nil {
			s.metrics.IncAttestationProofIssueFailures()
		}
		return nil, rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("canonicalization failed: %v", derr))
	}
	sigBytes, serr := signer.Sign(canon)
	if serr != nil {
		if s.metrics != nil {
			s.metrics.IncAttestationProofIssueFailures()
		}
		return nil, rfc.New(rfc.ErrIntegrityFailure, fmt.Sprintf("sign failed: %v", serr))
	}
	proof.DigestHex = dig
	proof.CanonicalDigest = dig
	proof.Algorithm = signer.Algorithm()
	proof.KeyID = signer.KeyID()
	proof.Signature = base64.StdEncoding.EncodeToString(sigBytes)
	if s.metrics != nil {
		s.metrics.IncAttestationProofIssued()
	}
	return proof, nil
}

// VerifyAttestationProof validates signature & digest of an AttestationProof.
// Returns nil on success or RFC integrity/expired errors. Metrics are recorded for success & failure.
//
//nolint:gocyclo // Attestation proof verification with signature checks
func (s *Service) VerifyAttestationProof(ctx context.Context, proof *attest.AttestationProof) error {
	if proof == nil {
		return rfc.New(rfc.ErrInvalidRequest, "nil proof")
	}
	start := time.Now()
	if !proof.ExpiresAt.IsZero() && time.Now().UTC().After(proof.ExpiresAt.UTC()) {
		return rfc.New(rfc.ErrExpired, "attestation proof expired")
	}
	// Optional trust anchor enforcement (strict issuer binding). Enabled when GAUTH_ATTEST_REQUIRE_TRUST_ANCHOR=1.
	requireAnchor := parseBoolEnv("GAUTH_ATTEST_REQUIRE_TRUST_ANCHOR")
	if requireAnchor && s.attestAnchors != nil {
		anchor, ok := s.attestAnchors.Get(proof.Issuer)
		if !ok {
			if s.metrics != nil {
				s.metrics.IncAttestationProofVerificationFailures()
				s.metrics.IncAttestationProofTrustAnchorMissing()
			}
			return rfc.New(rfc.ErrIntegrityFailure, "trust anchor missing for issuer")
		}
		// Enforce algorithm and key binding match.
		if anchor.Algorithm != "" && anchor.Algorithm != proof.Algorithm {
			if s.metrics != nil {
				s.metrics.IncAttestationProofVerificationFailures()
				s.metrics.IncAttestationProofTrustAnchorAlgorithmMismatch()
			}
			return rfc.New(rfc.ErrIntegrityFailure, "attestation algorithm mismatch with trust anchor")
		}
		if anchor.KeyID != "" && anchor.KeyID != proof.KeyID {
			if s.metrics != nil {
				s.metrics.IncAttestationProofVerificationFailures()
				s.metrics.IncAttestationProofTrustAnchorKeyMismatch()
			}
			return rfc.New(rfc.ErrIntegrityFailure, "attestation key mismatch with trust anchor")
		}
	}
	dig, canon, derr := attest.CanonicalAttestationDigest(proof)
	if derr != nil {
		if s.metrics != nil {
			s.metrics.IncAttestationProofVerificationFailures()
		}
		return rfc.New(rfc.ErrIntegrityFailure, "canonicalization failed")
	}
	if dig != proof.DigestHex {
		if s.metrics != nil {
			s.metrics.IncAttestationProofDigestMismatch()
			s.metrics.IncAttestationProofVerificationFailures()
		}
		return rfc.New(rfc.ErrIntegrityFailure, "digest mismatch")
	}
	// Dispatch verification through registered algorithm
	algo := proof.Algorithm
	if algo == "" {
		algo = cr.AlgoEd25519
	}
	if cr.GetAlgorithm(algo) == nil {
		if s.metrics != nil {
			s.metrics.IncAttestationProofVerificationFailures()
		}
		return rfc.New(rfc.ErrIntegrityFailure, "unsupported algorithm")
	}
	if s.keyProvider == nil {
		if s.metrics != nil {
			s.metrics.IncAttestationProofVerificationFailures()
		}
		return rfc.New(rfc.ErrIntegrityFailure, "key provider unavailable")
	}
	err := cr.VerifyAlgorithm(algo, canon, proof.Signature, proof.KeyID, s.keyProvider)
	if err != nil {
		if errors.Is(err, cr.ErrUnknownKey) {
			// Unknown key treated as integrity_failure (strict posture for attestation)
			if s.metrics != nil {
				s.metrics.IncAttestationProofVerificationFailures()
			}
			return rfc.New(rfc.ErrIntegrityFailure, "public key missing")
		}
		if s.metrics != nil {
			s.metrics.IncAttestationProofVerificationFailures()
		}
		return rfc.New(rfc.ErrIntegrityFailure, "signature verification failed")
	}
	if s.metrics != nil {
		s.metrics.IncAttestationProofVerifications()
		s.metrics.ObserveAttestationProofVerificationLatency(time.Since(start))
	}
	return nil
}

// lookupRawPOAHashAlgo returns negotiated hash algo from env (shared with RawPOA chain hashing posture).
// Fallback is sha256. Only allowed values are sha256, blake2b256, sha3_256.
func lookupRawPOAHashAlgo() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GAUTH_RAW_POA_CHAIN_HASH_ALGO")))
	if v == "blake2b256" || v == "sha3_256" || v == "sha256" {
		return v
	}
	return "sha256"
}

// parseBoolEnv reused for local attestation enforcement settings.
func parseBoolEnv(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
