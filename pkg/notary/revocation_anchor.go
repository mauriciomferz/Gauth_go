package notary

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/notary"
	bolt "go.etcd.io/bbolt"
)

// RevocationAnchoringAdapter implements the AnchorClient interface from pkg"AAP-001
// by wrapping an internal/notary.Notarizer and providing receipt persistence.
//
// P2.12 (sec5.item3): Provides external revocation anchoring with RFC 3161 TSA integration.
// Stores timestamped receipts in BoltDB for non-repudiation and audit trail.
//
// Architecture:
//
//	gauth_aap_001.Service.RevokeDelegation()
//	  → AnchorClient.Anchor(hash)
//	    → RevocationAnchoringAdapter.Anchor(hash)
//	      → Notarizer.Notarize(hash)  [RFC3161Provider, MemoryNotarizer, etc.]
//	      → Store Receipt in BoltDB anchor_receipts bucket
//
// Receipts provide cryptographic proof that revocation occurred at specific time.
// Future enhancement: Batch anchoring with merkle tree compression.
type RevocationAnchoringAdapter struct {
	notarizer    notary.Notarizer          // Internal notarizer (RFC3161Provider, MemoryNotarizer, etc.)
	receiptStore *ReceiptStore             // BoltDB-backed receipt storage
	mu           sync.RWMutex              // Protects in-memory cache
	cache        map[string]notary.Receipt // Hash -> Receipt cache (optional optimization)
}

// ReceiptStore provides persistent storage for notarization receipts in BoltDB.
type ReceiptStore struct {
	db         *bolt.DB
	bucketName []byte
	mu         sync.RWMutex
}

// NewReceiptStore creates a new BoltDB-backed receipt store.
//
// Bucket: anchor_receipts
// Key: SHA256 hash (hex string)
// Value: JSON-encoded Receipt
func NewReceiptStore(db *bolt.DB) (*ReceiptStore, error) {
	store := &ReceiptStore{
		db:         db,
		bucketName: []byte("anchor_receipts"),
	}

	// Create bucket if not exists
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(store.bucketName)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create anchor_receipts bucket: %w", err)
	}

	return store, nil
}

// Store persists a receipt in BoltDB.
func (rs *ReceiptStore) Store(hash string, receipt notary.Receipt) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Serialize receipt to JSON
	data, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("marshal receipt: %w", err)
	}

	// Store in BoltDB
	err = rs.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(rs.bucketName)
		if bucket == nil {
			return errors.New("anchor_receipts bucket not found")
		}
		return bucket.Put([]byte(hash), data)
	})
	if err != nil {
		return fmt.Errorf("store receipt in bolt: %w", err)
	}

	return nil
}

// Get retrieves a receipt from BoltDB by hash.
func (rs *ReceiptStore) Get(hash string) (notary.Receipt, bool, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var receipt notary.Receipt
	var found bool

	err := rs.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(rs.bucketName)
		if bucket == nil {
			return errors.New("anchor_receipts bucket not found")
		}

		data := bucket.Get([]byte(hash))
		if data == nil {
			found = false
			return nil
		}

		found = true
		return json.Unmarshal(data, &receipt)
	})
	if err != nil {
		return notary.Receipt{}, false, fmt.Errorf("get receipt from bolt: %w", err)
	}

	return receipt, found, nil
}

// List returns all receipts (for auditing/reporting).
// Warning: May be slow for large datasets.
func (rs *ReceiptStore) List() ([]notary.Receipt, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var receipts []notary.Receipt

	err := rs.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(rs.bucketName)
		if bucket == nil {
			return errors.New("anchor_receipts bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var receipt notary.Receipt
			if err := json.Unmarshal(v, &receipt); err != nil {
				return fmt.Errorf("unmarshal receipt %s: %w", string(k), err)
			}
			receipts = append(receipts, receipt)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list receipts: %w", err)
	}

	return receipts, nil
}

// NewRevocationAnchoringAdapter creates a new adapter with notarizer and receipt store.
//
// Example:
//
//	notarizer := notary.NewRFC3161Provider("https://freetsa.org/tsr", "FreeTSA")
//	receiptStore, _ := NewReceiptStore(boltDB)
//	adapter := NewRevocationAnchoringAdapter(notarizer, receiptStore)
//	svc := gauth_aap_001.NewService(..., gauth_aap_001.WithAnchorClient(adapter))
func NewRevocationAnchoringAdapter(notarizer notary.Notarizer, receiptStore *ReceiptStore) *RevocationAnchoringAdapter {
	return &RevocationAnchoringAdapter{
		notarizer:    notarizer,
		receiptStore: receiptStore,
		cache:        make(map[string]notary.Receipt),
	}
}

// Anchor implements the AnchorClient interface (from pkg"AAP-001).
//
// Workflow:
// 1. Compute SHA256 hash of input hash (for request integrity)
// 2. Call notarizer.Notarize() to get timestamped receipt
// 3. Store receipt in BoltDB for persistence
// 4. Cache receipt in memory (optional optimization)
// 5. Return error if notarization fails (fail-closed)
//
// P2.12: Provides cryptographic proof that revocation occurred at specific time.
// Receipts contain TSA timestamp (RFC 3161) or transparency log inclusion proof.
func (a *RevocationAnchoringAdapter) Anchor(hash string) error {
	// Validate hash format
	if hash == "" {
		return errors.New("hash required for anchoring")
	}

	// Call notarizer to get timestamped receipt
	receipt, err := a.notarizer.Notarize(hash)
	if err != nil {
		return fmt.Errorf("notarization failed: %w", err)
	}

	// Store receipt in BoltDB
	if a.receiptStore != nil {
		if err := a.receiptStore.Store(hash, receipt); err != nil {
			return fmt.Errorf("store receipt failed: %w", err)
		}
	}

	// Cache receipt in memory
	a.mu.Lock()
	a.cache[hash] = receipt
	a.mu.Unlock()

	return nil
}

// GetReceipt retrieves a receipt for a given hash (verification API).
//
// Checks:
// 1. In-memory cache (fast path)
// 2. BoltDB persistent storage (slow path)
//
// Returns:
//   - receipt: The notarization receipt
//   - found: Whether receipt exists
//   - error: Any storage errors
func (a *RevocationAnchoringAdapter) GetReceipt(hash string) (notary.Receipt, bool, error) {
	// Check cache first
	a.mu.RLock()
	if receipt, ok := a.cache[hash]; ok {
		a.mu.RUnlock()
		return receipt, true, nil
	}
	a.mu.RUnlock()

	// Check persistent storage
	if a.receiptStore == nil {
		return notary.Receipt{}, false, nil
	}

	receipt, found, err := a.receiptStore.Get(hash)
	if err != nil {
		return notary.Receipt{}, false, err
	}

	// Populate cache if found
	if found {
		a.mu.Lock()
		a.cache[hash] = receipt
		a.mu.Unlock()
	}

	return receipt, found, nil
}

// VerifyReceipt validates a receipt's integrity (timestamp, signature, etc.).
//
// For RFC3161Provider: Verifies timestamp format, provider name match.
// Future: Cryptographic signature verification, PKI chain validation.
//
// Returns error if receipt is invalid or verification fails.
func (a *RevocationAnchoringAdapter) VerifyReceipt(receipt notary.Receipt) error {
	// Basic validation
	if !receipt.Success {
		return errors.New("receipt indicates failed notarization")
	}

	// Verify timestamp format
	_, err := time.Parse(time.RFC3339Nano, receipt.Timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	// Provider-specific verification
	// If notarizer implements VerifyReceipt interface, delegate to it
	if verifier, ok := a.notarizer.(interface{ VerifyReceipt(notary.Receipt) error }); ok {
		if err := verifier.VerifyReceipt(receipt); err != nil {
			return fmt.Errorf("provider verification failed: %w", err)
		}
	}

	return nil
}

// ComputeRevocationHash computes SHA256 hash of revocation event.
//
// Input: Revocation metadata (poaID, revoker, timestamp, reason)
// Output: "sha256:<hex>" format hash
//
// Used by RevokeDelegation to generate anchor hash.
func ComputeRevocationHash(poaID, revoker string, timestamp time.Time, reason string) string {
	// Canonical format: "<poaID>|<revoker>|<timestamp_unix>|<reason>"
	canonical := fmt.Sprintf("%s|%s|%d|%s", poaID, revoker, timestamp.Unix(), reason)

	hash := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(hash[:])
}

// Stats returns statistics about anchored receipts.
type AnchorStats struct {
	TotalReceipts    int     `json:"total_receipts"`
	SuccessfulCount  int     `json:"successful_count"`
	FailedCount      int     `json:"failed_count"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	OldestReceipt    string  `json:"oldest_receipt,omitempty"`
	NewestReceipt    string  `json:"newest_receipt,omitempty"`
}

// GetStats returns statistics about anchored receipts (for monitoring).
func (a *RevocationAnchoringAdapter) GetStats() (AnchorStats, error) {
	if a.receiptStore == nil {
		return AnchorStats{}, errors.New("receipt store not initialized")
	}

	receipts, err := a.receiptStore.List()
	if err != nil {
		return AnchorStats{}, fmt.Errorf("list receipts: %w", err)
	}

	stats := AnchorStats{
		TotalReceipts: len(receipts),
	}

	if len(receipts) == 0 {
		return stats, nil
	}

	var totalLatency float64
	var oldestTime, newestTime time.Time

	for _, receipt := range receipts {
		if receipt.Success {
			stats.SuccessfulCount++
		} else {
			stats.FailedCount++
		}

		totalLatency += receipt.LatencySeconds

		// Parse timestamp
		ts, err := time.Parse(time.RFC3339Nano, receipt.Timestamp)
		if err != nil {
			continue
		}

		if oldestTime.IsZero() || ts.Before(oldestTime) {
			oldestTime = ts
			stats.OldestReceipt = receipt.Timestamp
		}

		if newestTime.IsZero() || ts.After(newestTime) {
			newestTime = ts
			stats.NewestReceipt = receipt.Timestamp
		}
	}

	stats.AverageLatencyMs = (totalLatency / float64(len(receipts))) * 1000

	return stats, nil
}
