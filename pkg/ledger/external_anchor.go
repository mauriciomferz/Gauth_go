package ledger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/anchor"
)

// ExternalAnchorClient integrates external timestamping providers with audit ledger.
// It bridges the ledger.AnchorClient interface with external anchor.Provider implementations.
type ExternalAnchorClient struct {
	provider     anchor.Provider
	receiptStore *anchor.ExternalReceiptStore
	mu           sync.RWMutex
	lastReceipt  anchor.Receipt
}

// NewExternalAnchorClient creates a client that submits ledger hashes to external providers.
func NewExternalAnchorClient(provider anchor.Provider, receiptStore *anchor.ExternalReceiptStore) *ExternalAnchorClient {
	if provider == nil {
		return nil
	}
	return &ExternalAnchorClient{
		provider:     provider,
		receiptStore: receiptStore,
	}
}

// Anchor submits the ledger hash to the external provider and persists the receipt.
// This implements the ledger.AnchorClient interface for external timestamping.
func (eac *ExternalAnchorClient) Anchor(hash string) error {
	if hash == "" {
		return fmt.Errorf("cannot anchor empty hash")
	}

	start := time.Now()
	
	// Submit to external provider
	receipt, err := eac.provider.Anchor(hash)
	if err != nil {
		return fmt.Errorf("external anchor failed: %w", err)
	}

	latency := time.Since(start)
	
	// Update last receipt
	eac.mu.Lock()
	eac.lastReceipt = receipt
	eac.mu.Unlock()

	// Persist receipt if store configured
	if eac.receiptStore != nil {
		extReceipt := anchor.ExternalAnchorReceipt{
			Hash:           receipt.Hash,
			Timestamp:      receipt.Timestamp.UTC().Format(time.RFC3339Nano),
			Provider:       receipt.Provider,
			Version:        receipt.Version,
			LatencySeconds: latency.Seconds(),
		}
		
		if _, err := eac.receiptStore.Append(extReceipt); err != nil {
			// Non-fatal - receipt persistence failure shouldn't break anchoring
			return fmt.Errorf("external anchor succeeded but receipt persistence failed: %w", err)
		}
	}

	return nil
}

// Latest returns the most recent external anchor receipt.
func (eac *ExternalAnchorClient) Latest() anchor.Receipt {
	eac.mu.RLock()
	defer eac.mu.RUnlock()
	return eac.lastReceipt
}

// Verify validates a receipt against the external provider.
func (eac *ExternalAnchorClient) Verify(receipt anchor.Receipt) error {
	return eac.provider.Verify(receipt)
}

// ExternalAuditLedger extends BoltDB ledger with automatic external anchoring.
// It wraps the base boltStore and adds external anchor submission on every append.
type ExternalAuditLedger struct {
	*boltStore
	externalAnchor *ExternalAnchorClient
	anchorInterval time.Duration
	mu             sync.RWMutex
	lastAnchor     time.Time
}

// NewExternalAuditLedger creates a BoltDB ledger with external anchoring capabilities.
func NewExternalAuditLedger(dbPath string, provider anchor.Provider, receiptStorePath string, anchorInterval time.Duration) (*ExternalAuditLedger, error) {
	// Create base BoltDB store
	baseStore, err := NewBoltStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create bolt store: %w", err)
	}

	// Setup external anchor client
	var receiptStore *anchor.ExternalReceiptStore
	if receiptStorePath != "" {
		receiptStore = anchor.NewExternalReceiptStore(receiptStorePath)
		if err := receiptStore.Load(); err != nil {
			return nil, fmt.Errorf("failed to load receipt store: %w", err)
		}
	}

	externalAnchor := NewExternalAnchorClient(provider, receiptStore)
	if externalAnchor == nil {
		return nil, fmt.Errorf("failed to create external anchor client")
	}

	if anchorInterval <= 0 {
		anchorInterval = 60 * time.Second // Default 60s anchor interval
	}

	return &ExternalAuditLedger{
		boltStore:      baseStore.(*boltStore),
		externalAnchor: externalAnchor,
		anchorInterval: anchorInterval,
	}, nil
}

// Append adds an entry to the ledger and triggers external anchoring if interval elapsed.
func (eal *ExternalAuditLedger) Append(ctx context.Context, e *Entry) error {
	// Append to base ledger first
	if err := eal.boltStore.Append(ctx, e); err != nil {
		return err
	}

	// Check if we should submit to external anchor
	now := time.Now()
	eal.mu.RLock()
	shouldAnchor := now.Sub(eal.lastAnchor) >= eal.anchorInterval
	eal.mu.RUnlock()

	if shouldAnchor {
		// Get current chain tip for anchoring
		tip, err := eal.boltStore.lastHash()
		if err == nil && tip != "" {
			// Submit asynchronously to avoid blocking append
			go func() {
				if anchorErr := eal.externalAnchor.Anchor(tip); anchorErr == nil {
					eal.mu.Lock()
					eal.lastAnchor = now
					eal.mu.Unlock()
				}
				// Errors logged but don't fail the append operation
			}()
		}
	}

	return nil
}

// EnableAnchorFile configures both base ledger and external anchor file emission.
func (eal *ExternalAuditLedger) EnableAnchorFile(path string, interval time.Duration) error {
	return eal.boltStore.EnableAnchorFile(path, interval)
}

// ForceExternalAnchor immediately submits current chain tip to external provider.
func (eal *ExternalAuditLedger) ForceExternalAnchor() error {
	tip, err := eal.boltStore.lastHash()
	if err != nil {
		return fmt.Errorf("failed to get chain tip: %w", err)
	}
	if tip == "" {
		return fmt.Errorf("no entries to anchor")
	}

	if err := eal.externalAnchor.Anchor(tip); err != nil {
		return fmt.Errorf("external anchor failed: %w", err)
	}

	eal.mu.Lock()
	eal.lastAnchor = time.Now()
	eal.mu.Unlock()

	return nil
}

// ExternalAnchorStatus returns status information about external anchoring.
func (eal *ExternalAuditLedger) ExternalAnchorStatus() map[string]interface{} {
	eal.mu.RLock()
	lastAnchor := eal.lastAnchor
	eal.mu.RUnlock()

	latest := eal.externalAnchor.Latest()
	
	status := map[string]interface{}{
		"configured":      true,
		"interval":        eal.anchorInterval.String(),
		"last_anchor_at":  lastAnchor.Format(time.RFC3339Nano),
		"age_seconds":     time.Since(lastAnchor).Seconds(),
	}

	if latest.Hash != "" {
		status["latest_receipt"] = map[string]interface{}{
			"hash":      latest.Hash,
			"timestamp": latest.Timestamp.Format(time.RFC3339Nano),
			"provider":  latest.Provider,
			"version":   latest.Version,
		}
	}

	// Include receipt store verification if available
	if eal.externalAnchor.receiptStore != nil {
		verifyStatus, _, _ := eal.externalAnchor.receiptStore.VerifyIncremental()
		status["receipt_chain_status"] = verifyStatus
	}

	return status
}

// Close closes both the base store and external anchor resources.
func (eal *ExternalAuditLedger) Close() error {
	return eal.boltStore.Close()
}