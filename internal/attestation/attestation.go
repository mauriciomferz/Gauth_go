// Package attestation provides cryptographic proof and evidence for compliance attestations (RFC 0111 sec4.item2).
package attestation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Attestation represents a compliance attestation with cryptographic proof.
type Attestation struct {
	ID            string                 `json:"id"`
	Subject       string                 `json:"subject"` // Who is attesting
	Claim         string                 `json:"claim"`   // What is being attested
	Evidence      []Evidence             `json:"evidence"`
	Timestamp     time.Time              `json:"timestamp"`
	ExpiresAt     *time.Time             `json:"expires_at,omitempty"`
	Signature     string                 `json:"signature,omitempty"`
	ProofHash     string                 `json:"proof_hash"` // SHA-256 of evidence
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Verified      bool                   `json:"verified"`
	VerifiedAt    *time.Time             `json:"verified_at,omitempty"`
}

// Evidence represents a piece of supporting evidence for an attestation.
type Evidence struct {
	Type        string                 `json:"type"` // "document", "audit_log", "certificate", "measurement"
	Source      string                 `json:"source"`
	Content     string                 `json:"content,omitempty"`
	ContentHash string                 `json:"content_hash"` // SHA-256 of content
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Proof represents cryptographic proof of an attestation.
type Proof struct {
	AttestationID string    `json:"attestation_id"`
	Algorithm     string    `json:"algorithm"` // "SHA-256", "SHA-512", etc.
	Hash          string    `json:"hash"`
	Timestamp     time.Time `json:"timestamp"`
	ChainHash     string    `json:"chain_hash,omitempty"` // Hash linking to previous proof
}

// Store manages attestations and their proofs.
type Store interface {
	// CreateAttestation stores a new attestation with cryptographic proof.
	CreateAttestation(attestation *Attestation) (*Proof, error)

	// GetAttestation retrieves an attestation by ID.
	GetAttestation(attestationID string) (*Attestation, error)

	// VerifyAttestation checks the cryptographic proof of an attestation.
	VerifyAttestation(attestationID string) (bool, error)

	// ListAttestations retrieves attestations with optional filtering.
	ListAttestations(filter AttestationFilter) ([]*Attestation, error)

	// AddEvidence adds new evidence to an existing attestation.
	AddEvidence(attestationID string, evidence Evidence) error
}

// AttestationFilter specifies filtering criteria.
type AttestationFilter struct {
	Subject    string
	Claim      string
	ValidOnly  bool // Only return non-expired attestations
	VerifiedOnly bool
	Since      *time.Time
	Until      *time.Time
}

// InMemoryStore provides a simple in-memory attestation store.
type InMemoryStore struct {
	attestations map[string]*Attestation
	proofs       map[string]*Proof
	chainHead    string // Hash of the most recent proof (for chain linking)
}

// NewInMemoryStore creates a new in-memory attestation store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		attestations: make(map[string]*Attestation),
		proofs:       make(map[string]*Proof),
	}
}

// CreateAttestation stores an attestation and generates cryptographic proof.
func (s *InMemoryStore) CreateAttestation(attestation *Attestation) (*Proof, error) {
	// Generate proof hash from evidence
	proofHash, err := generateProofHash(attestation)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof hash: %w", err)
	}

	attestation.ProofHash = proofHash
	attestation.Verified = false

	// Create proof record
	proof := &Proof{
		AttestationID: attestation.ID,
		Algorithm:     "SHA-256",
		Hash:          proofHash,
		Timestamp:     time.Now(),
		ChainHash:     s.chainHead, // Link to previous proof
	}

	// Update chain head
	s.chainHead = proofHash

	// Store
	s.attestations[attestation.ID] = attestation
	s.proofs[attestation.ID] = proof

	return proof, nil
}

// GetAttestation retrieves an attestation by ID.
func (s *InMemoryStore) GetAttestation(attestationID string) (*Attestation, error) {
	attestation, exists := s.attestations[attestationID]
	if !exists {
		return nil, fmt.Errorf("attestation not found: %s", attestationID)
	}
	return attestation, nil
}

// VerifyAttestation checks the cryptographic integrity of an attestation.
func (s *InMemoryStore) VerifyAttestation(attestationID string) (bool, error) {
	attestation, err := s.GetAttestation(attestationID)
	if err != nil {
		return false, err
	}

	// Recompute proof hash
	computedHash, err := generateProofHash(attestation)
	if err != nil {
		return false, fmt.Errorf("failed to recompute hash: %w", err)
	}

	// Compare with stored hash
	verified := computedHash == attestation.ProofHash

	if verified {
		now := time.Now()
		attestation.Verified = true
		attestation.VerifiedAt = &now
	}

	return verified, nil
}

// ListAttestations retrieves attestations matching the filter.
func (s *InMemoryStore) ListAttestations(filter AttestationFilter) ([]*Attestation, error) {
	var results []*Attestation

	for _, attestation := range s.attestations {
		// Apply filters
		if filter.Subject != "" && attestation.Subject != filter.Subject {
			continue
		}
		if filter.Claim != "" && attestation.Claim != filter.Claim {
			continue
		}
		if filter.ValidOnly && attestation.ExpiresAt != nil && attestation.ExpiresAt.Before(time.Now()) {
			continue
		}
		if filter.VerifiedOnly && !attestation.Verified {
			continue
		}
		if filter.Since != nil && attestation.Timestamp.Before(*filter.Since) {
			continue
		}
		if filter.Until != nil && attestation.Timestamp.After(*filter.Until) {
			continue
		}

		results = append(results, attestation)
	}

	return results, nil
}

// AddEvidence adds new evidence to an existing attestation.
func (s *InMemoryStore) AddEvidence(attestationID string, evidence Evidence) error {
	attestation, err := s.GetAttestation(attestationID)
	if err != nil {
		return err
	}

	// Add evidence
	attestation.Evidence = append(attestation.Evidence, evidence)

	// Recalculate proof hash
	newHash, err := generateProofHash(attestation)
	if err != nil {
		return fmt.Errorf("failed to regenerate proof hash: %w", err)
	}

	attestation.ProofHash = newHash
	attestation.Verified = false // Must be re-verified after modification

	return nil
}

// generateProofHash creates a SHA-256 hash of the attestation evidence.
func generateProofHash(attestation *Attestation) (string, error) {
	// Serialize evidence to JSON
	evidenceJSON, err := json.Marshal(attestation.Evidence)
	if err != nil {
		return "", fmt.Errorf("failed to serialize evidence: %w", err)
	}

	// Compute SHA-256
	hash := sha256.Sum256(evidenceJSON)
	return hex.EncodeToString(hash[:]), nil
}

// HashEvidence computes SHA-256 hash of evidence content.
func HashEvidence(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}
