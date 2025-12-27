package delegation

import (
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/anchor"
	"github.com/mauriciomferz/Gauth_go/pkg/ledger"
)

// ExternalRevocationAnchor implements RevocationAnchorObserver to anchor
// revocation chain SignedTreeHead events to external timestamp/notarization providers.
// It wraps ledger.ExternalAnchorClient to provide seamless integration.
type ExternalRevocationAnchor struct {
	client       *ledger.ExternalAnchorClient
	receiptStore *anchor.ExternalReceiptStore
	mu           sync.RWMutex
	lastAnchor   time.Time
	anchorCount  uint64
}

// NewExternalRevocationAnchor creates an anchor observer for revocation events.
//
// Parameters:
//   - provider: External timestamp/notarization provider (RFC3161, Memory, etc.)
//   - receiptStorePath: Path for persisting anchor receipts with hash-chain integrity
//
// Returns configured anchor observer ready to receive tree head callbacks.
func NewExternalRevocationAnchor(provider anchor.Provider, receiptStorePath string) (*ExternalRevocationAnchor, error) {
	// Create receipt store with hash-chain integrity
	receiptStore := anchor.NewExternalReceiptStore(receiptStorePath)

	// Load existing receipts if file exists
	if err := receiptStore.Load(); err != nil {
		return nil, err
	}

	// Create external anchor client (cast to interface)
	var receiptStoreInterface anchor.ReceiptStore = receiptStore
	client := ledger.NewExternalAnchorClient(provider, receiptStoreInterface)

	return &ExternalRevocationAnchor{
		client:       client,
		receiptStore: receiptStore,
		lastAnchor:   time.Time{},
		anchorCount:  0,
	}, nil
}

// OnRevocationAnchor implements RevocationAnchorObserver.
// It is called when a SignedTreeHead is generated in the revocation chain.
// Submits the tree head root hash to the external provider for timestamp/notarization.
func (e *ExternalRevocationAnchor) OnRevocationAnchor(sth *SignedTreeHead) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Use MerkleRoot as the canonical hash to anchor
	// This represents the immutable state of the entire revocation chain
	hashToAnchor := sth.MerkleRoot
	if hashToAnchor == "" {
		// Fall back to AggregateHash if MerkleRoot not available
		hashToAnchor = sth.AggregateHash
	}

	// Submit to external provider
	err := e.client.Anchor(hashToAnchor)
	if err != nil {
		return err
	}

	e.lastAnchor = time.Now()
	e.anchorCount++
	return nil
}

// Status returns anchoring status information for monitoring/debugging.
func (e *ExternalRevocationAnchor) Status() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	latest := e.client.Latest()

	return map[string]interface{}{
		"anchor_count":   e.anchorCount,
		"last_anchor_at": e.lastAnchor.Format(time.RFC3339),
		"latest_receipt": map[string]interface{}{
			"hash":      latest.Hash,
			"timestamp": latest.Timestamp.Format(time.RFC3339),
			"provider":  latest.Provider,
		},
	}
}

// LatestReceipt returns the most recent anchor receipt.
func (e *ExternalRevocationAnchor) LatestReceipt() anchor.Receipt {
	return e.client.Latest()
}

// VerifyReceipt validates a receipt against the provider.
func (e *ExternalRevocationAnchor) VerifyReceipt(receipt anchor.Receipt) error {
	return e.client.Verify(receipt)
}

// Close releases resources.
func (e *ExternalRevocationAnchor) Close() error {
	// Receipt store is file-based and doesn't need explicit closing
	return nil
}
