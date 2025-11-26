package compliance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/attest"
)

// TestInMemoryAttestationStore_StoreAndGet verifies basic store and retrieval.
func TestInMemoryAttestationStore_StoreAndGet(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()
	proof := &attest.AttestationProof{
		Version:   "att/v1",
		Subject:   "user:alice",
		Issuer:    "authority:gdpr",
		Statement: "User consented to data processing under GDPR Article 6(1)(a)",
		IssuedAt:  time.Now(),
		Nonce:     "test-nonce-001",
	}

	// Store attestation
	err := store.Store(ctx, proof, true)
	if err != nil {
		t.Fatalf("failed to store attestation: %v", err)
	}

	// Retrieve attestation
	stored, err := store.Get(ctx, "test-nonce-001")
	if err != nil {
		t.Fatalf("failed to get attestation: %v", err)
	}

	if stored.Proof.Subject != "user:alice" {
		t.Errorf("expected subject 'user:alice', got '%s'", stored.Proof.Subject)
	}
	if !stored.Verified {
		t.Error("expected verified=true")
	}
	if stored.VerifiedAt == nil {
		t.Error("expected VerifiedAt to be set")
	}
}

// TestInMemoryAttestationStore_StoreUnverified verifies storing unverified attestations.
func TestInMemoryAttestationStore_StoreUnverified(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()
	proof := &attest.AttestationProof{
		Version:   "att/v1",
		Subject:   "user:bob",
		Issuer:    "authority:hipaa",
		Statement: "User provided medical consent",
		IssuedAt:  time.Now(),
		Nonce:     "test-nonce-002",
	}

	// Store as unverified
	err := store.Store(ctx, proof, false)
	if err != nil {
		t.Fatalf("failed to store attestation: %v", err)
	}

	// Retrieve and check verification status
	stored, err := store.Get(ctx, "test-nonce-002")
	if err != nil {
		t.Fatalf("failed to get attestation: %v", err)
	}

	if stored.Verified {
		t.Error("expected verified=false")
	}
	if stored.VerifiedAt != nil {
		t.Error("expected VerifiedAt to be nil for unverified attestation")
	}
}

// TestInMemoryAttestationStore_QueryBySubject verifies querying by subject.
func TestInMemoryAttestationStore_QueryBySubject(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()

	// Store multiple attestations
	proofs := []*attest.AttestationProof{
		{
			Version:   "att/v1",
			Subject:   "user:alice",
			Issuer:    "authority:gdpr",
			Statement: "GDPR consent",
			IssuedAt:  time.Now(),
			Nonce:     "nonce-001",
		},
		{
			Version:   "att/v1",
			Subject:   "user:alice",
			Issuer:    "authority:ccpa",
			Statement: "CCPA consent",
			IssuedAt:  time.Now(),
			Nonce:     "nonce-002",
		},
		{
			Version:   "att/v1",
			Subject:   "user:bob",
			Issuer:    "authority:gdpr",
			Statement: "GDPR consent",
			IssuedAt:  time.Now(),
			Nonce:     "nonce-003",
		},
	}

	for _, proof := range proofs {
		if err := store.Store(ctx, proof, true); err != nil {
			t.Fatalf("failed to store attestation: %v", err)
		}
	}

	// Query by subject
	filter := AttestationFilter{Subject: "user:alice"}
	results, err := store.Query(ctx, filter)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results for user:alice, got %d", len(results))
	}

	for _, result := range results {
		if result.Proof.Subject != "user:alice" {
			t.Errorf("expected subject 'user:alice', got '%s'", result.Proof.Subject)
		}
	}
}

// TestInMemoryAttestationStore_QueryByIssuer verifies querying by issuer.
func TestInMemoryAttestationStore_QueryByIssuer(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()

	// Store attestations
	proofs := []*attest.AttestationProof{
		{
			Version:   "att/v1",
			Subject:   "user:alice",
			Issuer:    "authority:gdpr",
			Statement: "GDPR consent",
			IssuedAt:  time.Now(),
			Nonce:     "nonce-004",
		},
		{
			Version:   "att/v1",
			Subject:   "user:bob",
			Issuer:    "authority:gdpr",
			Statement: "GDPR consent",
			IssuedAt:  time.Now(),
			Nonce:     "nonce-005",
		},
	}

	for _, proof := range proofs {
		if err := store.Store(ctx, proof, true); err != nil {
			t.Fatalf("failed to store attestation: %v", err)
		}
	}

	// Query by issuer
	filter := AttestationFilter{Issuer: "authority:gdpr"}
	results, err := store.Query(ctx, filter)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results for authority:gdpr, got %d", len(results))
	}
}

// TestInMemoryAttestationStore_QueryVerifiedOnly verifies filtering by verification status.
func TestInMemoryAttestationStore_QueryVerifiedOnly(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()

	// Store mixed verified/unverified attestations
	proofs := []struct {
		proof    *attest.AttestationProof
		verified bool
	}{
		{
			proof: &attest.AttestationProof{
				Version:   "att/v1",
				Subject:   "user:alice",
				Issuer:    "authority:gdpr",
				Statement: "GDPR consent",
				IssuedAt:  time.Now(),
				Nonce:     "nonce-006",
			},
			verified: true,
		},
		{
			proof: &attest.AttestationProof{
				Version:   "att/v1",
				Subject:   "user:bob",
				Issuer:    "authority:gdpr",
				Statement: "GDPR consent",
				IssuedAt:  time.Now(),
				Nonce:     "nonce-007",
			},
			verified: false,
		},
	}

	for _, p := range proofs {
		if err := store.Store(ctx, p.proof, p.verified); err != nil {
			t.Fatalf("failed to store attestation: %v", err)
		}
	}

	// Query verified only
	filter := AttestationFilter{VerifiedOnly: true}
	results, err := store.Query(ctx, filter)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 verified result, got %d", len(results))
	}

	if !results[0].Verified {
		t.Error("expected verified=true in results")
	}
}

// TestInMemoryAttestationStore_QueryWithTimeRange verifies time-based filtering.
func TestInMemoryAttestationStore_QueryWithTimeRange(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	// Store attestations with different issue times
	proofs := []*attest.AttestationProof{
		{
			Version:   "att/v1",
			Subject:   "user:alice",
			Issuer:    "authority:gdpr",
			Statement: "Old consent",
			IssuedAt:  yesterday,
			Nonce:     "nonce-008",
		},
		{
			Version:   "att/v1",
			Subject:   "user:bob",
			Issuer:    "authority:gdpr",
			Statement: "Current consent",
			IssuedAt:  now,
			Nonce:     "nonce-009",
		},
	}

	for _, proof := range proofs {
		if err := store.Store(ctx, proof, true); err != nil {
			t.Fatalf("failed to store attestation: %v", err)
		}
	}

	// Query since yesterday (should get both)
	filter := AttestationFilter{Since: &yesterday}
	results, err := store.Query(ctx, filter)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results since yesterday, got %d", len(results))
	}

	// Query until tomorrow (should get both)
	filter = AttestationFilter{Until: &tomorrow}
	results, err = store.Query(ctx, filter)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results until tomorrow, got %d", len(results))
	}

	// Query range (since now, should get 1)
	filter = AttestationFilter{Since: &now}
	results, err = store.Query(ctx, filter)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result since now, got %d", len(results))
	}
}

// TestInMemoryAttestationStore_QueryWithLimit verifies result limiting.
func TestInMemoryAttestationStore_QueryWithLimit(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()

	// Store multiple attestations
	for i := 0; i < 5; i++ {
		proof := &attest.AttestationProof{
			Version:   "att/v1",
			Subject:   "user:alice",
			Issuer:    "authority:gdpr",
			Statement: "GDPR consent",
			IssuedAt:  time.Now(),
			Nonce:     fmt.Sprintf("nonce-010-%d", i),
		}
		if err := store.Store(ctx, proof, true); err != nil {
			t.Fatalf("failed to store attestation: %v", err)
		}
	}

	// Query with limit
	filter := AttestationFilter{Subject: "user:alice", Limit: 3}
	results, err := store.Query(ctx, filter)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results (limit), got %d", len(results))
	}
}

// TestInMemoryAttestationStore_Delete verifies deletion.
func TestInMemoryAttestationStore_Delete(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()
	proof := &attest.AttestationProof{
		Version:   "att/v1",
		Subject:   "user:alice",
		Issuer:    "authority:gdpr",
		Statement: "GDPR consent",
		IssuedAt:  time.Now(),
		Nonce:     "nonce-011",
	}

	// Store attestation
	if err := store.Store(ctx, proof, true); err != nil {
		t.Fatalf("failed to store attestation: %v", err)
	}

	// Delete attestation
	if err := store.Delete(ctx, "nonce-011"); err != nil {
		t.Fatalf("failed to delete attestation: %v", err)
	}

	// Verify deletion
	_, err := store.Get(ctx, "nonce-011")
	if err == nil {
		t.Error("expected error when getting deleted attestation")
	}
}

// TestInMemoryAttestationStore_Count verifies count operation.
func TestInMemoryAttestationStore_Count(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()

	// Initial count should be 0
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Store attestations
	for i := 0; i < 3; i++ {
		proof := &attest.AttestationProof{
			Version:   "att/v1",
			Subject:   "user:alice",
			Issuer:    "authority:gdpr",
			Statement: "GDPR consent",
			IssuedAt:  time.Now(),
			Nonce:     fmt.Sprintf("nonce-012-%d", i),
		}
		if err2 := store.Store(ctx, proof, true); err2 != nil {
			t.Fatalf("failed to store attestation: %v", err2)
		}
	}

	// Count should be 3
	count, err = store.Count(ctx)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

// TestJSONLAttestationStore_Persistence verifies JSONL file persistence.
func TestJSONLAttestationStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "attestations.jsonl")

	ctx := context.Background()

	// Create store and add attestations
	store1, err := NewJSONLAttestationStore(filePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	proof := &attest.AttestationProof{
		Version:   "att/v1",
		Subject:   "user:alice",
		Issuer:    "authority:gdpr",
		Statement: "GDPR consent",
		IssuedAt:  time.Now(),
		Nonce:     "nonce-persist-001",
	}

	if err2 := store1.Store(ctx, proof, true); err2 != nil {
		t.Fatalf("failed to store attestation: %v", err2)
	}
	store1.Close()

	// Verify file exists
	if _, err2 := os.Stat(filePath); os.IsNotExist(err2) {
		t.Fatal("JSONL file was not created")
	}

	// Reload store from file
	store2, err := NewJSONLAttestationStore(filePath)
	if err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}
	defer store2.Close()

	// Verify attestation was loaded
	stored, err := store2.Get(ctx, "nonce-persist-001")
	if err != nil {
		t.Fatalf("failed to get persisted attestation: %v", err)
	}

	if stored.Proof.Subject != "user:alice" {
		t.Errorf("expected subject 'user:alice', got '%s'", stored.Proof.Subject)
	}
}

// TestAttestationStore_ValidationErrors verifies error handling.
func TestAttestationStore_ValidationErrors(t *testing.T) {
	store := NewInMemoryAttestationStore()
	defer store.Close()

	ctx := context.Background()

	// Nil proof
	err := store.Store(ctx, nil, true)
	if err == nil {
		t.Error("expected error for nil proof")
	}

	// Empty nonce
	proof := &attest.AttestationProof{
		Version:  "att/v1",
		Subject:  "user:alice",
		Issuer:   "authority:gdpr",
		IssuedAt: time.Now(),
		// Missing Nonce
	}
	err = store.Store(ctx, proof, true)
	if err == nil {
		t.Error("expected error for missing nonce")
	}

	// Get with empty nonce
	_, err = store.Get(ctx, "")
	if err == nil {
		t.Error("expected error for empty nonce in Get")
	}

	// Delete with empty nonce
	err = store.Delete(ctx, "")
	if err == nil {
		t.Error("expected error for empty nonce in Delete")
	}
}
