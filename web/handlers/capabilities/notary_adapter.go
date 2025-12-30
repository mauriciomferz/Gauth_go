package capabilities

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/notary"
	"github.com/mauriciomferz/AgentAuth/pkg/ledger"
)

// NotaryAdapter bridges ledger.ExternalAnchorClient to the capabilities.AnchorClient interface.
type NotaryAdapter struct {
	client       *ledger.ExternalAnchorClient
	totalAnchors int64
}

// NewNotaryAdapter creates a new adapter for the given ledger anchor client.
func NewNotaryAdapter(client *ledger.ExternalAnchorClient) *NotaryAdapter {
	if client == nil {
		return nil
	}
	return &NotaryAdapter{
		client: client,
	}
}

// Anchor submits the hash to the ledger's external anchor client and returns a notary receipt.
func (a *NotaryAdapter) Anchor(hash string) (*notary.Receipt, error) {
	start := time.Now()
	if err := a.client.Anchor(hash); err != nil {
		return nil, fmt.Errorf("ledger anchor failed: %w", err)
	}
	atomic.AddInt64(&a.totalAnchors, 1)
	elapsed := time.Since(start).Seconds()

	// Get the latest receipt from the client to build the return value
	latest := a.client.Latest()

	return &notary.Receipt{
		Hash:           latest.Hash,
		Timestamp:      latest.Timestamp.Format(time.RFC3339Nano),
		Provider:       latest.Provider,
		Version:        latest.Version,
		Success:        true,
		LatencySeconds: elapsed,
	}, nil
}

// TotalAnchors returns the total number of anchors performed by this adapter.
func (a *NotaryAdapter) TotalAnchors() int64 {
	return atomic.LoadInt64(&a.totalAnchors)
}
