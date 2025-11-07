package compliance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/attest"
)

// AttestationStore provides persistent storage and retrieval of compliance attestations.
// Implements evidence ingestion for sec4.item2 (P2 gap).
type AttestationStore interface {
	// Store saves an attestation with verification status
	Store(ctx context.Context, proof *attest.AttestationProof, verified bool) error

	// Get retrieves an attestation by nonce (unique identifier)
	Get(ctx context.Context, nonce string) (*StoredAttestation, error)

	// Query finds attestations matching criteria
	Query(ctx context.Context, filter AttestationFilter) ([]*StoredAttestation, error)

	// Delete removes an attestation (admin operation)
	Delete(ctx context.Context, nonce string) error

	// Count returns total stored attestations
	Count(ctx context.Context) (int, error)

	// Close gracefully shuts down the store
	Close() error
}

// StoredAttestation wraps an attestation proof with storage metadata.
type StoredAttestation struct {
	Proof        *attest.AttestationProof `json:"proof"`
	Verified     bool                      `json:"verified"`
	StoredAt     time.Time                 `json:"stored_at"`
	VerifiedAt   *time.Time                `json:"verified_at,omitempty"`
	Jurisdiction string                    `json:"jurisdiction,omitempty"`
	Notes        string                    `json:"notes,omitempty"`
}

// AttestationFilter defines query criteria for attestations.
type AttestationFilter struct {
	Subject      string     // Match by subject
	Issuer       string     // Match by issuer
	Jurisdiction string     // Match by jurisdiction
	VerifiedOnly bool       // Only return verified attestations
	Since        *time.Time // Only return attestations issued after this time
	Until        *time.Time // Only return attestations issued before this time
	Limit        int        // Max results (0 = unlimited)
}

// InMemoryAttestationStore implements AttestationStore with in-memory map storage.
// Suitable for testing and development; production should use persistent storage (DB, KV store).
type InMemoryAttestationStore struct {
	attestations map[string]*StoredAttestation // keyed by nonce
	mu           sync.RWMutex
}

// NewInMemoryAttestationStore creates a new in-memory attestation store.
func NewInMemoryAttestationStore() *InMemoryAttestationStore {
	return &InMemoryAttestationStore{
		attestations: make(map[string]*StoredAttestation),
	}
}

// Store saves an attestation with verification status.
func (s *InMemoryAttestationStore) Store(ctx context.Context, proof *attest.AttestationProof, verified bool) error {
	if proof == nil {
		return errors.New("nil attestation proof")
	}
	if proof.Nonce == "" {
		return errors.New("attestation proof must have nonce for storage")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	stored := &StoredAttestation{
		Proof:    proof,
		Verified: verified,
		StoredAt: time.Now(),
	}

	if verified {
		now := time.Now()
		stored.VerifiedAt = &now
	}

	s.attestations[proof.Nonce] = stored
	return nil
}

// Get retrieves an attestation by nonce.
func (s *InMemoryAttestationStore) Get(ctx context.Context, nonce string) (*StoredAttestation, error) {
	if nonce == "" {
		return nil, errors.New("empty nonce")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, exists := s.attestations[nonce]
	if !exists {
		return nil, fmt.Errorf("attestation not found: %s", nonce)
	}

	return stored, nil
}

// Query finds attestations matching filter criteria.
func (s *InMemoryAttestationStore) Query(ctx context.Context, filter AttestationFilter) ([]*StoredAttestation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*StoredAttestation, 0)

	for _, stored := range s.attestations {
		if !s.matchesFilter(stored, filter) {
			continue
		}
		results = append(results, stored)

		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}

	return results, nil
}

// matchesFilter checks if a stored attestation matches filter criteria.
func (s *InMemoryAttestationStore) matchesFilter(stored *StoredAttestation, filter AttestationFilter) bool {
	if filter.Subject != "" && stored.Proof.Subject != filter.Subject {
		return false
	}
	if filter.Issuer != "" && stored.Proof.Issuer != filter.Issuer {
		return false
	}
	if filter.Jurisdiction != "" && stored.Jurisdiction != filter.Jurisdiction {
		return false
	}
	if filter.VerifiedOnly && !stored.Verified {
		return false
	}
	if filter.Since != nil && stored.Proof.IssuedAt.Before(*filter.Since) {
		return false
	}
	if filter.Until != nil && stored.Proof.IssuedAt.After(*filter.Until) {
		return false
	}
	return true
}

// Delete removes an attestation by nonce.
func (s *InMemoryAttestationStore) Delete(ctx context.Context, nonce string) error {
	if nonce == "" {
		return errors.New("empty nonce")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.attestations[nonce]; !exists {
		return fmt.Errorf("attestation not found: %s", nonce)
	}

	delete(s.attestations, nonce)
	return nil
}

// Count returns total stored attestations.
func (s *InMemoryAttestationStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.attestations), nil
}

// Close is a no-op for in-memory store.
func (s *InMemoryAttestationStore) Close() error {
	return nil
}

// JSONLAttestationStore implements AttestationStore with append-only JSONL file persistence.
// Provides durability and crash recovery. Line-based format enables streaming reads and writes.
type JSONLAttestationStore struct {
	filePath     string
	attestations map[string]*StoredAttestation // in-memory index
	mu           sync.RWMutex
}

// NewJSONLAttestationStore creates a JSONL-backed attestation store.
// Loads existing attestations from file on initialization.
func NewJSONLAttestationStore(filePath string) (*JSONLAttestationStore, error) {
	store := &JSONLAttestationStore{
		filePath:     filePath,
		attestations: make(map[string]*StoredAttestation),
	}

	// Load existing attestations if file exists
	if err := store.loadFromFile(); err != nil {
		return nil, fmt.Errorf("load attestations: %w", err)
	}

	return store, nil
}

// loadFromFile reads all attestations from JSONL file into memory index.
func (s *JSONLAttestationStore) loadFromFile() error {
	file, err := os.Open(s.filePath)
	if os.IsNotExist(err) {
		// File doesn't exist yet, start with empty store
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for decoder.More() {
		var stored StoredAttestation
		if err := decoder.Decode(&stored); err != nil {
			// Skip corrupted lines
			continue
		}
		if stored.Proof != nil && stored.Proof.Nonce != "" {
			s.attestations[stored.Proof.Nonce] = &stored
		}
	}

	return nil
}

// Store saves an attestation with verification status and appends to JSONL file.
func (s *JSONLAttestationStore) Store(ctx context.Context, proof *attest.AttestationProof, verified bool) error {
	if proof == nil {
		return errors.New("nil attestation proof")
	}
	if proof.Nonce == "" {
		return errors.New("attestation proof must have nonce for storage")
	}

	stored := &StoredAttestation{
		Proof:    proof,
		Verified: verified,
		StoredAt: time.Now(),
	}

	if verified {
		now := time.Now()
		stored.VerifiedAt = &now
	}

	// Append to file
	if err := s.appendToFile(stored); err != nil {
		return fmt.Errorf("append to file: %w", err)
	}

	// Update in-memory index
	s.mu.Lock()
	s.attestations[proof.Nonce] = stored
	s.mu.Unlock()

	return nil
}

// appendToFile appends a stored attestation to the JSONL file.
func (s *JSONLAttestationStore) appendToFile(stored *StoredAttestation) error {
	file, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}

	return nil
}

// Get retrieves an attestation by nonce.
func (s *JSONLAttestationStore) Get(ctx context.Context, nonce string) (*StoredAttestation, error) {
	if nonce == "" {
		return nil, errors.New("empty nonce")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, exists := s.attestations[nonce]
	if !exists {
		return nil, fmt.Errorf("attestation not found: %s", nonce)
	}

	return stored, nil
}

// Query finds attestations matching filter criteria.
func (s *JSONLAttestationStore) Query(ctx context.Context, filter AttestationFilter) ([]*StoredAttestation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*StoredAttestation, 0)

	for _, stored := range s.attestations {
		if !s.matchesFilter(stored, filter) {
			continue
		}
		results = append(results, stored)

		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}

	return results, nil
}

// matchesFilter checks if a stored attestation matches filter criteria.
func (s *JSONLAttestationStore) matchesFilter(stored *StoredAttestation, filter AttestationFilter) bool {
	if filter.Subject != "" && stored.Proof.Subject != filter.Subject {
		return false
	}
	if filter.Issuer != "" && stored.Proof.Issuer != filter.Issuer {
		return false
	}
	if filter.Jurisdiction != "" && stored.Jurisdiction != filter.Jurisdiction {
		return false
	}
	if filter.VerifiedOnly && !stored.Verified {
		return false
	}
	if filter.Since != nil && stored.Proof.IssuedAt.Before(*filter.Since) {
		return false
	}
	if filter.Until != nil && stored.Proof.IssuedAt.After(*filter.Until) {
		return false
	}
	return true
}

// Delete removes an attestation by nonce.
func (s *JSONLAttestationStore) Delete(ctx context.Context, nonce string) error {
	if nonce == "" {
		return errors.New("empty nonce")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.attestations[nonce]; !exists {
		return fmt.Errorf("attestation not found: %s", nonce)
	}

	delete(s.attestations, nonce)

	// Note: JSONL store doesn't physically delete from file (append-only)
	// Deleted records are only removed from in-memory index
	// Full compaction could be implemented as a separate maintenance operation

	return nil
}

// Count returns total stored attestations.
func (s *JSONLAttestationStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.attestations), nil
}

// Close is a no-op for JSONL store (file handle managed per-operation).
func (s *JSONLAttestationStore) Close() error {
	return nil
}
