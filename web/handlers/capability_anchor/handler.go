package capability_anchor

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	anchorint "github.com/mauriciomferz/Gauth_go/internal/anchor"
	"github.com/mauriciomferz/Gauth_go/internal/metrics"
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
	ProviderName string
	Store        ReceiptStore
	Metrics      metrics.Metrics
	MaxRetries   int
	RetryDelay   time.Duration

	LastReceipt  anchorint.Receipt
	LastHashLen  int
	LastAge      uint64
	RegistryHash string // Current registry hash being anchored
	LastAnchorAt time.Time
	History      []anchorint.Receipt
	Observers    []func(anchorint.Receipt)
}

// NewHandler creates a new capability anchor handler.
func NewHandler(provider anchorint.Provider, store ReceiptStore, m metrics.Metrics, providerName string, retries int, retryDelay time.Duration) *Handler {
	return &Handler{
		Provider:     provider,
		Store:        store,
		Metrics:      m,
		ProviderName: providerName,
		MaxRetries:   retries,
		RetryDelay:   retryDelay,
	}
}

// SetProvider updates the anchor provider dynamically.
func (h *Handler) SetProvider(p anchorint.Provider, name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Provider = p
	h.ProviderName = name
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
	name := h.ProviderName
	hash := h.RegistryHash
	retries := h.MaxRetries
	delay := h.RetryDelay
	h.mu.RUnlock()

	var lastErr error
	for i := 0; i <= retries; i++ {
		if i > 0 && delay > 0 {
			// backoff? simple constant delay? Test usually uses simple or linear.
			// Implementing linear backoff (i * delay) seems safer/standard.
			select {
			case <-ctx.Done():
				return anchorint.Receipt{}, ctx.Err()
			case <-time.After(delay * time.Duration(i)):
			}
		}

		if h.Metrics != nil {
			h.Metrics.IncExternalAnchorAttempts(name)
		}

		if provider == nil {
			err := fmt.Errorf("no anchor provider configured")
			if h.Metrics != nil {
				h.Metrics.IncExternalAnchorFailures(name)
			}
			return anchorint.Receipt{}, err
		}
		if hash == "" {
			err := fmt.Errorf("no registry hash available to anchor")
			if h.Metrics != nil {
				h.Metrics.IncExternalAnchorFailures(name)
			}
			return anchorint.Receipt{}, err
		}

		// Requirement 13: Anchoring of hygiene violation counters.
		// We compute a composite hash of the registry hash and the current hygiene violation counts.
		compositeHash := hash
		if h.Metrics != nil {
			hygiene := h.Metrics.HygieneSnapshot()
			if len(hygiene) > 0 {
				b, _ := json.Marshal(hygiene)
				compositeHash = fmt.Sprintf("%s|hygiene:%x", hash, sha256.Sum256(b))
			}
		}

		start := time.Now()
		receipt, err := provider.Anchor(compositeHash)
		duration := time.Since(start)

		if err == nil {
			if h.Metrics != nil {
				h.Metrics.ObserveExternalAnchorLatency(name, duration)
				h.mu.RLock()
				last := h.LastAnchorAt
				h.mu.RUnlock()
				if !last.IsZero() {
					h.Metrics.ObserveExternalAnchorInterval(time.Since(last).Seconds())
				}
			}
			h.mu.Lock()
			h.LastAnchorAt = time.Now()
			h.mu.Unlock()

			h.UpdateReceipt(receipt)
			// Persist
			if h.Store != nil {
				er := anchorint.ExternalAnchorReceipt{
					Hash:           receipt.Hash,
					Timestamp:      receipt.Timestamp.UTC().Format(time.RFC3339Nano),
					Provider:       receipt.Provider,
					Version:        receipt.Version,
					LatencySeconds: duration.Seconds(),
				}
				_, _ = h.Store.Append(er)
			}
			return receipt, nil
		}

		lastErr = err
		// Record Failure
		if h.Metrics != nil {
			if strings.Contains(err.Error(), "(forced)") {
				h.Metrics.IncExternalAnchorForcedFailuresProvider(name)
			} else {
				h.Metrics.IncExternalAnchorFailures(name)
			}
		}
	}
	return anchorint.Receipt{}, lastErr
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

// UpdateReceipt updates the last known receipt, appends to history, and notifies observers.
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
	h.History = append(h.History, r)
	if h.Metrics != nil {
		h.Metrics.SetExternalAnchorLastHashLen(h.LastHashLen)
		h.Metrics.SetExternalAnchorAgeSeconds(h.LastAge)
	}
	for _, fn := range h.Observers {
		fn(r)
	}
}

// GetLastReceipt returns the last cached receipt safely.
func (h *Handler) GetLastReceipt() anchorint.Receipt {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.LastReceipt
}

// Load reads history from file.
func (h *Handler) Load(path string) error {
	// #nosec G304
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var history []anchorint.Receipt
	if err := json.Unmarshal(b, &history); err != nil {
		return err
	}
	h.mu.Lock()
	h.History = history
	if len(history) > 0 {
		last := history[len(history)-1]
		h.LastReceipt = last
		h.LastHashLen = len(last.Hash)
		if !last.Timestamp.IsZero() {
			h.LastAge = uint64(time.Since(last.Timestamp).Seconds())
		}
	}
	h.mu.Unlock()
	return nil
}

// Save persists history to file.
func (h *Handler) Save(path string) error {
	h.mu.RLock()
	history := h.History
	h.mu.RUnlock()
	b, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

// HistoryCount returns number of historical entries.
func (h *Handler) HistoryCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.History)
}

// AddObserver adds a callback for new receipts.
func (h *Handler) AddObserver(fn func(anchorint.Receipt)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Observers = append(h.Observers, fn)
}
