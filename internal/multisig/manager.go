// Package multisig provides multi-signature Power of Attorney (PoA) orchestration.
//
// This implements RFC0115 Section B (Authorization Type) joint/collective signature
// enforcement (GAP_MATRIX sec3.item3) with M-of-N threshold policies, signature
// collection workflow, and weighted voting support.
//
// Features:
// - M-of-N threshold signature collection
// - Weighted signatures (optional)
// - Concurrent signature submission
// - Automatic threshold completion detection
// - Comprehensive audit trail
// - Status tracking and querying
//
// Thread-safety: All public methods are safe for concurrent use.
package multisig

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
)

// SignatureStatus represents the current state of signature collection.
type SignatureStatus string

const (
	StatusPending   SignatureStatus = "pending"   // Awaiting signatures
	StatusCompleted SignatureStatus = "completed" // Threshold met
	StatusActive    SignatureStatus = "active"    // PoA activated
	StatusExpired   SignatureStatus = "expired"   // Collection window expired
	StatusRejected  SignatureStatus = "rejected"  // Explicitly rejected
)

// SignatureRecord tracks an individual signature submission.
type SignatureRecord struct {
	SignerID  string    `json:"signer_id"`
	KeyID     string    `json:"key_id"`
	Signature string    `json:"signature"` // base64-encoded
	SignedAt  time.Time `json:"signed_at"`
	Weight    int       `json:"weight"`     // For weighted voting
	IPAddress string    `json:"ip_address"` // Optional audit info
	UserAgent string    `json:"user_agent"` // Optional audit info
}

// PoASignatureState tracks the multi-signature collection state for a PoA.
type PoASignatureState struct {
	PoAID             string                      `json:"poa_id"`
	Threshold         int                         `json:"threshold"`
	RequiredSigners   []string                    `json:"required_signers"`
	UseWeightedVoting bool                        `json:"use_weighted_voting"`
	Weights           map[string]int              `json:"weights,omitempty"`
	Signatures        map[string]*SignatureRecord `json:"signatures"`
	Status            SignatureStatus             `json:"status"`
	CreatedAt         time.Time                   `json:"created_at"`
	CompletedAt       *time.Time                  `json:"completed_at,omitempty"`
	ActivatedAt       *time.Time                  `json:"activated_at,omitempty"`
	ExpiresAt         *time.Time                  `json:"expires_at,omitempty"`
	CanonicalDigest   string                      `json:"canonical_digest"` // hex
	TotalWeight       int                         `json:"total_weight"`     // Sum of all possible weights
	CollectedWeight   int                         `json:"collected_weight"` // Sum of collected weights
}

// SignatureManager orchestrates multi-signature PoA collection and verification.
type SignatureManager struct {
	mu       sync.RWMutex
	states   map[string]*PoASignatureState // keyed by PoA ID
	verifier VerificationProvider
}

// VerificationProvider defines signature verification capabilities.
type VerificationProvider interface {
	// PublicKey retrieves the public key for a key ID
	PublicKey(keyID string) ([]byte, string, error)
	// VerifySignature verifies a signature over canonical digest
	VerifySignature(digest []byte, signature []byte, publicKey []byte) error
}

// NewSignatureManager creates a new multi-signature orchestration manager.
func NewSignatureManager(verifier VerificationProvider) *SignatureManager {
	return &SignatureManager{
		states:   make(map[string]*PoASignatureState),
		verifier: verifier,
	}
}

// InitiateCollection starts signature collection for a PoA.
func (sm *SignatureManager) InitiateCollection(
	ctx context.Context,
	poa *rfc0111.PowerOfAttorney,
	expiresIn time.Duration,
) error {
	if poa == nil {
		return errors.New("nil PoA")
	}
	if poa.Threshold <= 1 {
		return errors.New("threshold must be > 1 for multi-signature collection")
	}
	if len(poa.Signers) < poa.Threshold {
		return fmt.Errorf("insufficient signers: have %d need %d", len(poa.Signers), poa.Threshold)
	}

	// Compute canonical digest
	digestHex, _, err := rfc0111.CanonicalPOADigest(poa)
	if err != nil {
		return fmt.Errorf("canonical digest failed: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[poa.ID]; exists {
		return fmt.Errorf("collection already initiated for PoA %s", poa.ID)
	}

	state := &PoASignatureState{
		PoAID:           poa.ID,
		Threshold:       poa.Threshold,
		RequiredSigners: poa.Signers,
		Signatures:      make(map[string]*SignatureRecord),
		Status:          StatusPending,
		CreatedAt:       time.Now().UTC(),
		CanonicalDigest: digestHex,
	}

	if expiresIn > 0 {
		expiresAt := time.Now().UTC().Add(expiresIn)
		state.ExpiresAt = &expiresAt
	}

	// Weighted voting is configured via GAUTH_MULTI_SIG_WEIGHTS env var in the verifier
	// The manager stores weights if provided explicitly during initialization
	// For now, assume equal weight (weight=1) per signer unless overridden

	sm.states[poa.ID] = state
	return nil
}

// SubmitSignature adds a signature to the collection.
func (sm *SignatureManager) SubmitSignature(
	ctx context.Context,
	poaID string,
	signerID string,
	keyID string,
	signatureB64 string,
	metadata map[string]string,
) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[poaID]
	if !exists {
		return fmt.Errorf("no signature collection for PoA %s", poaID)
	}

	// Check if already completed
	if state.Status == StatusCompleted || state.Status == StatusActive {
		return fmt.Errorf("signature collection already completed for PoA %s", poaID)
	}

	// Check expiration
	if state.ExpiresAt != nil && time.Now().UTC().After(*state.ExpiresAt) {
		state.Status = StatusExpired
		return fmt.Errorf("signature collection expired for PoA %s", poaID)
	}

	// Validate signer is in required list
	found := false
	for _, reqSigner := range state.RequiredSigners {
		if reqSigner == signerID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("signer %s not in required signers list", signerID)
	}

	// Check for duplicate signature
	if _, exists := state.Signatures[signerID]; exists {
		return fmt.Errorf("signature already submitted by %s", signerID)
	}

	// Verify signature
	digestBytes, err := hex2bytes(state.CanonicalDigest)
	if err != nil {
		return fmt.Errorf("invalid canonical digest: %w", err)
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	publicKey, _, err := sm.verifier.PublicKey(keyID)
	if err != nil {
		return fmt.Errorf("failed to retrieve public key: %w", err)
	}

	// Verify Ed25519 signature
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: %d", len(publicKey))
	}
	if len(signatureBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature size: %d", len(signatureBytes))
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), digestBytes, signatureBytes) {
		return errors.New("signature verification failed")
	}

	// Determine weight
	weight := 1
	if state.UseWeightedVoting {
		if w, ok := state.Weights[signerID]; ok {
			weight = w
		}
	}

	// Record signature
	record := &SignatureRecord{
		SignerID:  signerID,
		KeyID:     keyID,
		Signature: signatureB64,
		SignedAt:  time.Now().UTC(),
		Weight:    weight,
	}
	if metadata != nil {
		record.IPAddress = metadata["ip_address"]
		record.UserAgent = metadata["user_agent"]
	}

	state.Signatures[signerID] = record
	state.CollectedWeight += weight

	// Check if threshold met
	if sm.isThresholdMetLocked(state) {
		state.Status = StatusCompleted
		now := time.Now().UTC()
		state.CompletedAt = &now
	}

	return nil
}

// GetStatus retrieves the current signature collection status.
func (sm *SignatureManager) GetStatus(ctx context.Context, poaID string) (*PoASignatureState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[poaID]
	if !exists {
		return nil, fmt.Errorf("no signature collection for PoA %s", poaID)
	}

	// Return a copy to prevent external mutation
	stateCopy := *state
	stateCopy.Signatures = make(map[string]*SignatureRecord, len(state.Signatures))
	for k, v := range state.Signatures {
		recordCopy := *v
		stateCopy.Signatures[k] = &recordCopy
	}

	return &stateCopy, nil
}

// ActivatePoA marks the PoA as active after threshold completion.
func (sm *SignatureManager) ActivatePoA(ctx context.Context, poaID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[poaID]
	if !exists {
		return fmt.Errorf("no signature collection for PoA %s", poaID)
	}

	if state.Status != StatusCompleted {
		return fmt.Errorf("cannot activate PoA %s in status %s", poaID, state.Status)
	}

	state.Status = StatusActive
	now := time.Now().UTC()
	state.ActivatedAt = &now

	return nil
}

// ListPending returns all PoAs with pending signature collection.
func (sm *SignatureManager) ListPending(ctx context.Context) []*PoASignatureState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var pending []*PoASignatureState
	for _, state := range sm.states {
		if state.Status == StatusPending {
			stateCopy := *state
			pending = append(pending, &stateCopy)
		}
	}

	return pending
}

// isThresholdMetLocked checks if the threshold is met (must hold lock).
func (sm *SignatureManager) isThresholdMetLocked(state *PoASignatureState) bool {
	if state.UseWeightedVoting {
		return state.CollectedWeight >= state.Threshold
	}
	return len(state.Signatures) >= state.Threshold
}

// GetSignatures returns all collected signatures for a PoA.
func (sm *SignatureManager) GetSignatures(ctx context.Context, poaID string) (map[string]*rfc0111.POASignature, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[poaID]
	if !exists {
		return nil, fmt.Errorf("no signature collection for PoA %s", poaID)
	}

	// Convert to RFC0111 format
	result := make(map[string]*rfc0111.POASignature, len(state.Signatures))
	for signerID, record := range state.Signatures {
		result[signerID] = &rfc0111.POASignature{
			Algorithm: "ed25519",
			KeyID:     record.KeyID,
			SigBase64: record.Signature,
		}
	}

	return result, nil
}

// RejectCollection rejects a signature collection (e.g., PoA withdrawn).
func (sm *SignatureManager) RejectCollection(ctx context.Context, poaID string, reason string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[poaID]
	if !exists {
		return fmt.Errorf("no signature collection for PoA %s", poaID)
	}

	if state.Status == StatusActive {
		return fmt.Errorf("cannot reject active PoA %s", poaID)
	}

	state.Status = StatusRejected
	return nil
}

// hex2bytes converts hex string to bytes.
func hex2bytes(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		return nil, errors.New("hex string has odd length")
	}
	result := make([]byte, len(hexStr)/2)
	for i := 0; i < len(result); i++ {
		high := hexCharValue(hexStr[2*i])
		low := hexCharValue(hexStr[2*i+1])
		if high == 255 || low == 255 {
			return nil, errors.New("invalid hex character")
		}
		result[i] = (high << 4) | low
	}
	return result, nil
}

func hexCharValue(c byte) byte {
	if c >= '0' && c <= '9' {
		return c - '0'
	}
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 10
	}
	if c >= 'A' && c <= 'F' {
		return c - 'A' + 10
	}
	return 255
}
