package ledger

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/anchor"
	"github.com/mauriciomferz/Gauth_go/pkg/ledger/rfc3161"
)

// ExternalAnchorClient integrates external timestamping providers with audit ledger.
// It bridges the ledger.AnchorClient interface with external anchor.Provider implementations.
type ExternalAnchorClient struct {
	provider     anchor.Provider
	receiptStore anchor.ReceiptStore
	mu           sync.RWMutex
	lastReceipt  anchor.Receipt
}

// NewExternalAnchorClient creates a client that submits ledger hashes to external providers.
func NewExternalAnchorClient(provider anchor.Provider, receiptStore anchor.ReceiptStore) *ExternalAnchorClient {
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
	// Setup external anchor client
	var receiptStore anchor.ReceiptStore

	if receiptStorePath != "" {
		// Legacy/File mode
		fs := anchor.NewExternalReceiptStore(receiptStorePath)
		if err := fs.Load(); err != nil {
			return nil, fmt.Errorf("failed to load receipt store: %w", err)
		}
		receiptStore = fs
	} else {
		// BoltDB mode (Embedded in ledger DB)
		// We know baseStore is *boltStore because we just created it
		if bs, ok := baseStore.(*boltStore); ok {
			bs, err := NewBoltReceiptStore(bs.db)
			if err != nil {
				return nil, fmt.Errorf("failed to init bolt receipt store: %w", err)
			}
			receiptStore = bs
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
	eal.mu.RLock()
	shouldAnchor := time.Since(eal.lastAnchor) >= eal.anchorInterval
	eal.mu.RUnlock()

	if shouldAnchor {
		// Get current chain tip for anchoring
		tip, err := eal.boltStore.lastHash()
		if err == nil && tip != "" {
			// Submit asynchronously to avoid blocking append
			go func() {
				if anchorErr := eal.externalAnchor.Anchor(tip); anchorErr == nil {
					eal.mu.Lock()
					eal.lastAnchor = time.Now()
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
		"configured":     true,
		"interval":       eal.anchorInterval.String(),
		"last_anchor_at": lastAnchor.Format(time.RFC3339Nano),
		"age_seconds":    time.Since(lastAnchor).Seconds(),
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

// RFC3161Provider integrates the RFC 3161 client as an anchor provider.
type RFC3161Provider struct {
	primaryClient   *rfc3161.Client
	secondaryClient *rfc3161.Client // Optional fallback TSA
	latest          anchor.Receipt
	mu              sync.RWMutex
	config          TSAConfig
}

// TSAConfig holds production TSA configuration.
type TSAConfig struct {
	PrimaryURL       string        // Primary TSA endpoint
	SecondaryURL     string        // Secondary/fallback TSA endpoint (optional)
	CertBundle       string        // Path to trusted CA certificates (optional)
	Timeout          time.Duration // Request timeout (default: 30s)
	ValidateCerts    bool          // Enable certificate validation (default: true)
	AllowLocalOnFail bool          // Fall back to local timestamp if all TSAs fail
}

// NewRFC3161Provider creates a new provider pointing to a TSA URL.
func NewRFC3161Provider(url string) *RFC3161Provider {
	return NewRFC3161ProviderWithConfig(TSAConfig{
		PrimaryURL:    url,
		Timeout:       30 * time.Second,
		ValidateCerts: true,
	})
}

// NewRFC3161ProviderWithConfig creates a provider with production configuration.
func NewRFC3161ProviderWithConfig(config TSAConfig) *RFC3161Provider {
	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	primaryClient := rfc3161.NewClient(config.PrimaryURL)

	var secondaryClient *rfc3161.Client
	if config.SecondaryURL != "" {
		secondaryClient = rfc3161.NewClient(config.SecondaryURL)
	}

	return &RFC3161Provider{
		primaryClient:   primaryClient,
		secondaryClient: secondaryClient,
		config:          config,
	}
}

// Anchor submits the hash to the RFC 3161 TSA with fallback support.
func (r *RFC3161Provider) Anchor(hash string) (anchor.Receipt, error) {
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return anchor.Receipt{}, fmt.Errorf("invalid hash: %w", err)
	}

	// Try primary TSA
	receipt, err := r.tryTSA(r.primaryClient, "primary", hashBytes)
	if err == nil {
		r.mu.Lock()
		r.latest = receipt
		r.mu.Unlock()
		return receipt, nil
	}

	primaryErr := err

	// Try secondary TSA if configured
	if r.secondaryClient != nil {
		receipt, err = r.tryTSA(r.secondaryClient, "secondary", hashBytes)
		if err == nil {
			r.mu.Lock()
			r.latest = receipt
			r.mu.Unlock()
			return receipt, nil
		}
	}

	// If both TSAs failed and local fallback is allowed, create local receipt
	if r.config.AllowLocalOnFail {
		receipt = anchor.Receipt{
			Hash:      hash,
			Timestamp: time.Now().UTC(),
			Proof:     []byte("LOCAL_FALLBACK"),
			Provider:  "local",
			Version:   1,
		}
		r.mu.Lock()
		r.latest = receipt
		r.mu.Unlock()
		return receipt, nil
	}

	// Return primary TSA error if no fallback worked
	return anchor.Receipt{}, fmt.Errorf("all TSAs failed, primary error: %w", primaryErr)
}

// tryTSA attempts to get a timestamp from a specific TSA client.
func (r *RFC3161Provider) tryTSA(client *rfc3161.Client, name string, hashBytes []byte) (anchor.Receipt, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)
	defer cancel()

	// Submit timestamp request using the Anchor method
	receipt, err := client.Anchor(ctx, hashBytes)
	if err != nil {
		return anchor.Receipt{}, fmt.Errorf("%s TSA request failed: %w", name, err)
	}

	// If certificate validation is enabled, verify the timestamp token
	if r.config.ValidateCerts {
		if err := client.Verify(receipt); err != nil {
			return anchor.Receipt{}, fmt.Errorf("%s TSA verification failed: %w", name, err)
		}
	}

	// Update the receipt provider to indicate which TSA was used
	receipt.Provider = fmt.Sprintf("rfc3161-%s", name)

	return receipt, nil
}

// Latest returns the most recent receipt.
func (p *RFC3161Provider) Latest() anchor.Receipt {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.latest
}

// Verify performs a basic sanity check on the receipt.
func (r *RFC3161Provider) Verify(receipt anchor.Receipt) error {
	// Use the primary client for verification
	return r.primaryClient.Verify(receipt)
}
