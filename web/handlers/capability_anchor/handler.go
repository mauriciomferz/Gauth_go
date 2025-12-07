package capability_anchor

import (
	"context"
	"fmt"
	"sync"
	"time"

	anchorint "github.com/mauriciomferz/Gauth_go/internal/anchor"
)

// ReceiptStore interface defines persistence for external anchor receipts.
type ReceiptStore interface {
	Append(anchorint.ExternalAnchorReceipt) (anchorint.StoredExternalAnchorReceipt, error)
	Latest() anchorint.StoredExternalAnchorReceipt
	Entries() []anchorint.StoredExternalAnchorReceipt
	Load() error
	VerifyIncremental() (string, int, string)
}

// Handler manages external anchoring for capability registries.
type Handler struct {
	mu           sync.RWMutex
	Provider     anchorint.Provider
	Store        ReceiptStore
	LastReceipt  anchorint.Receipt
	LastHashLen  int
	LastAge      uint64
	RegistryHash string // Current registry hash being anchored
}

// NewHandler creates a new capability anchor handler.
func NewHandler(provider anchorint.Provider, store ReceiptStore) *Handler {
	return &Handler{
		Provider: provider,
		Store:    store,
	}
}

// SetProvider updates the anchor provider dynamically.
func (h *Handler) SetProvider(p anchorint.Provider) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Provider = p
}

// SetRegistryHash updates the hash of the current capability registry.
func (h *Handler) SetRegistryHash(hash string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.RegistryHash = hash
}

// Anchor manually triggers an anchoring operation for the current registry hash.
func (h *Handler) Anchor(ctx context.Context) (anchorint.Receipt, error) {
	h.mu.RLock()
	provider := h.Provider
	hash := h.RegistryHash
	h.mu.RUnlock()

	if provider == nil {
		return anchorint.Receipt{}, fmt.Errorf("no anchor provider configured")
	}
	if hash == "" {
		return anchorint.Receipt{}, fmt.Errorf("no registry hash available to anchor")
	}

	receipt, err := provider.Anchor(hash)
	if err != nil {
		return anchorint.Receipt{}, err
	}

	h.UpdateReceipt(receipt)

	// Persist
	if h.Store != nil {
		// Convert Receipt to ExternalAnchorReceipt (assuming compatibility or mapping)
		// anchorint.Receipt has {Hash, Timestamp(Time), Provider, Version}
		// anchorint.ExternalAnchorReceipt has {Hash, Timestamp(string), Provider, Version, LatencySeconds}
		// We need to match what Append expects.
		er := anchorint.ExternalAnchorReceipt{
			Hash:           receipt.Hash,
			Timestamp:      receipt.Timestamp.UTC().Format(time.RFC3339Nano),
			Provider:       receipt.Provider,
			Version:        receipt.Version,
			LatencySeconds: 0, // Manual/Handler doesn't track latency here easily unless passed down, or we compute it.
		}
		// We might want to pass latency in UpdateReceipt or similar?
		// For now simplifying.
		_, _ = h.Store.Append(er)
	}

	return receipt, nil
}

// Latest returns the latest receipt from the provider, or cached.
func (h *Handler) Latest(ctx context.Context) (anchorint.Receipt, error) {
	h.mu.RLock()
	provider := h.Provider
	h.mu.RUnlock()

	if provider == nil {
		return anchorint.Receipt{}, fmt.Errorf("no anchor provider configured")
	}

	// Some providers might just fetch latest known
	receipt := provider.Latest()
	if receipt.Hash != "" {
		h.UpdateReceipt(receipt)
	}
	return receipt, nil
}

// Verify verifies a receipt against the provider.
func (h *Handler) Verify(ctx context.Context, r anchorint.Receipt) error {
	h.mu.RLock()
	provider := h.Provider
	h.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("no anchor provider configured")
	}
	return provider.Verify(r)
}

// UpdateReceipt updates the last known receipt and derived metrics.
func (h *Handler) UpdateReceipt(r anchorint.Receipt) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.LastReceipt = r
	h.LastHashLen = len(r.Hash)
	if !r.Timestamp.IsZero() {
		h.LastAge = uint64(time.Since(r.Timestamp).Seconds())
	} else {
		h.LastAge = 0
	}
}

// GetLastReceipt returns the last cached receipt safely.
func (h *Handler) GetLastReceipt() anchorint.Receipt {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.LastReceipt
}
