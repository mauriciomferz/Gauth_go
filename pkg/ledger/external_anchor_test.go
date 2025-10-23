package ledger

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/anchor"
	"github.com/stretchr/testify/require"
)

// TestExternalAnchorClient tests the basic external anchor client functionality.
func TestExternalAnchorClient(t *testing.T) {
	provider := anchor.NewMemoryProvider()
	
	tempDir := t.TempDir()
	receiptPath := filepath.Join(tempDir, "receipts.json")
	receiptStore := anchor.NewExternalReceiptStore(receiptPath)
	
	client := NewExternalAnchorClient(provider, receiptStore)
	require.NotNil(t, client)

	// Test anchoring
	testHash := "0123456789abcdef"
	err := client.Anchor(testHash)
	require.NoError(t, err)

	// Verify latest receipt
	latest := client.Latest()
	require.Equal(t, testHash, latest.Hash)
	require.Equal(t, "memory", latest.Provider)
	require.Equal(t, 1, latest.Version)
	require.False(t, latest.Timestamp.IsZero())

	// Test verification
	err = client.Verify(latest)
	require.NoError(t, err)

	// Test empty hash rejection
	err = client.Anchor("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty hash")
}

// TestExternalAnchorClientWithoutReceiptStore tests client without receipt persistence.
func TestExternalAnchorClientWithoutReceiptStore(t *testing.T) {
	provider := anchor.NewMemoryProvider()
	client := NewExternalAnchorClient(provider, nil)
	require.NotNil(t, client)

	testHash := "test-hash-no-receipt"
	err := client.Anchor(testHash)
	require.NoError(t, err)

	latest := client.Latest()
	require.Equal(t, testHash, latest.Hash)
}

// TestExternalAuditLedger tests the complete external audit ledger functionality.
func TestExternalAuditLedger(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "audit.db")
	receiptPath := filepath.Join(tempDir, "receipts.json")

	provider := anchor.NewMemoryProvider()
	anchorInterval := 100 * time.Millisecond

	ledger, err := NewExternalAuditLedger(dbPath, provider, receiptPath, anchorInterval)
	require.NoError(t, err)
	require.NotNil(t, ledger)
	defer ledger.Close()

	ctx := context.Background()

	// Create test entries
	entry1 := &Entry{
		ID:      "test-1",
		TS:      time.Now().UTC(),
		Type:    "audit",
		Subject: "user1",
		Object:  "resource1",
		Metadata: map[string]interface{}{
			"action": "create",
			"result": "success",
		},
	}

	entry2 := &Entry{
		ID:      "test-2", 
		TS:      time.Now().UTC().Add(time.Second),
		Type:    "audit",
		Subject: "user2",
		Object:  "resource2",
		Metadata: map[string]interface{}{
			"action": "update", 
			"result": "success",
		},
	}

	// Append entries
	err = ledger.Append(ctx, entry1)
	require.NoError(t, err)

	err = ledger.Append(ctx, entry2)
	require.NoError(t, err)

	// Wait for anchor interval + some buffer
	time.Sleep(150 * time.Millisecond)

	// Verify entries can be retrieved
	retrieved, err := ledger.Get(ctx, "test-1")
	require.NoError(t, err)
	require.Equal(t, entry1.ID, retrieved.ID)
	require.Equal(t, entry1.Subject, retrieved.Subject)

	// Test chain verification
	result, err := ledger.VerifyChain(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, result.Count)
	require.Zero(t, result.Mismatches)

	// Test external anchor status
	status := ledger.ExternalAnchorStatus()
	require.Equal(t, true, status["configured"])
	require.Equal(t, anchorInterval.String(), status["interval"])
	require.Contains(t, status, "last_anchor_at")
	require.Contains(t, status, "age_seconds")
	require.Contains(t, status, "receipt_chain_status")
}

// TestExternalAuditLedgerForceAnchor tests manual external anchoring.
func TestExternalAuditLedgerForceAnchor(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "audit.db")
	receiptPath := filepath.Join(tempDir, "receipts.json")

	provider := anchor.NewMemoryProvider()
	ledger, err := NewExternalAuditLedger(dbPath, provider, receiptPath, time.Hour) // Long interval
	require.NoError(t, err)
	t.Cleanup(func() { ledger.Close() })

	ctx := context.Background()

	// Add entry
	entry := &Entry{
		ID:      "force-test",
		TS:      time.Now().UTC(),
		Type:    "audit",
		Subject: "user",
		Object:  "resource",
		Metadata: map[string]interface{}{
			"action": "test",
			"result": "success",
		},
	}

	err = ledger.Append(ctx, entry)
	require.NoError(t, err)

	// Force external anchor
	err = ledger.ForceExternalAnchor()
	require.NoError(t, err)

	// Verify anchor was created
	latest := ledger.externalAnchor.Latest()
	require.NotEmpty(t, latest.Hash)
	require.Equal(t, "memory", latest.Provider)

	status := ledger.ExternalAnchorStatus()
	require.Contains(t, status, "latest_receipt")
}

// TestExternalAuditLedgerEmptyForceAnchor tests force anchor on empty ledger.
func TestExternalAuditLedgerEmptyForceAnchor(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "empty.db")

	provider := anchor.NewMemoryProvider()
	ledger, err := NewExternalAuditLedger(dbPath, provider, "", time.Hour)
	require.NoError(t, err)
	defer ledger.Close()

	// Try to force anchor on empty ledger
	err = ledger.ForceExternalAnchor()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no entries to anchor")
}

// TestExternalAnchorClientWithTSAStub tests external anchoring with TSA stub provider.
func TestExternalAnchorClientWithTSAStub(t *testing.T) {
	// Create TSA stub with minimal latency
	provider := anchor.NewTSAStubProvider(1, 5, 0.0) // 1-5ms, no failures

	tempDir := t.TempDir()
	receiptPath := filepath.Join(tempDir, "tsa_receipts.json")
	receiptStore := anchor.NewExternalReceiptStore(receiptPath)

	client := NewExternalAnchorClient(provider, receiptStore)
	require.NotNil(t, client)

	testHash := "tsa-test-hash"
	err := client.Anchor(testHash)
	require.NoError(t, err)

	latest := client.Latest()
	require.Equal(t, testHash, latest.Hash)
	require.Equal(t, "tsa-stub", latest.Provider)
	require.NotEmpty(t, latest.Proof)

	// Verify receipt was persisted
	receiptEntries := receiptStore.Entries()
	require.Len(t, receiptEntries, 1)
	require.Equal(t, testHash, receiptEntries[0].Hash)
	require.Equal(t, "tsa-stub", receiptEntries[0].Provider)
}

// TestExternalAnchorClientFailure tests handling of provider failures.
func TestExternalAnchorClientFailure(t *testing.T) {
	// Create TSA stub that always fails
	provider := anchor.NewTSAStubProvider(1, 5, 1.0) // 100% failure rate

	client := NewExternalAnchorClient(provider, nil)
	require.NotNil(t, client)

	err := client.Anchor("will-fail")
	require.Error(t, err)
	require.Contains(t, err.Error(), "external anchor failed")
}

// TestExternalAuditLedgerAnchorInterval tests automatic anchoring based on intervals.
func TestExternalAuditLedgerAnchorInterval(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "interval.db")

	provider := anchor.NewMemoryProvider()
	shortInterval := 50 * time.Millisecond

	ledger, err := NewExternalAuditLedger(dbPath, provider, "", shortInterval)
	require.NoError(t, err)
	defer ledger.Close()

	ctx := context.Background()

	// Add first entry - should trigger anchor
	entry1 := &Entry{
		ID:      "interval-1",
		TS:      time.Now().UTC(),
		Type:    "audit",
		Subject: "user",
		Object:  "resource",
		Metadata: map[string]interface{}{
			"action": "action1",
			"result": "success",
		},
	}

	err = ledger.Append(ctx, entry1)
	require.NoError(t, err)

	// Wait for interval
	time.Sleep(60 * time.Millisecond)

	// Add second entry - should trigger another anchor  
	entry2 := &Entry{
		ID:      "interval-2",
		TS:      time.Now().UTC(),
		Type:    "audit",
		Subject: "user",
		Object:  "resource",
		Metadata: map[string]interface{}{
			"action": "action2", 
			"result": "success",
		},
	}

	err = ledger.Append(ctx, entry2)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(60 * time.Millisecond)

	// Verify we have anchor activity
	status := ledger.ExternalAnchorStatus()
	require.Equal(t, true, status["configured"])
	ageSeconds := status["age_seconds"].(float64)
	require.Less(t, ageSeconds, 1.0) // Should be recent
}

// TestExternalAuditLedgerWithBoltAnchorFile tests integration with base ledger anchor files.
func TestExternalAuditLedgerWithBoltAnchorFile(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "combined.db")
	anchorFilePath := filepath.Join(tempDir, "anchor.json")

	provider := anchor.NewMemoryProvider()
	ledger, err := NewExternalAuditLedger(dbPath, provider, "", time.Hour)
	require.NoError(t, err)
	t.Cleanup(func() { ledger.Close() })

	ctx := context.Background()

	// Add entry first (anchor file needs entries to emit)
	entry := &Entry{
		ID:      "combined-test",
		TS:      time.Now().UTC(),
		Type:    "audit",
		Subject: "user",
		Object:  "resource", 
		Metadata: map[string]interface{}{
			"action": "test",
			"result": "success",
		},
	}

	err = ledger.Append(ctx, entry)
	require.NoError(t, err)

	// Enable base ledger anchor file after adding entry
	err = ledger.EnableAnchorFile(anchorFilePath, 100*time.Millisecond)
	require.NoError(t, err)

	// Wait for anchor file emission
	time.Sleep(150 * time.Millisecond)

	// Verify anchor file was created (should be created immediately if entries exist)
	require.FileExists(t, anchorFilePath)

	// Also force external anchor
	err = ledger.ForceExternalAnchor()
	require.NoError(t, err)

	// Verify both anchoring systems work
	status := ledger.ExternalAnchorStatus()
	require.Contains(t, status, "latest_receipt")
}