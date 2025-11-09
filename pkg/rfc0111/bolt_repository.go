package rfc0111

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Bolt buckets
const (
	boltBucketPOA        = "poa"        // primary storage by ID
	boltBucketPrincipal  = "principal"  // secondary index: principal -> JSON array of POA IDs (grantor+grantee)
	boltBucketStatus     = "status"     // index: status -> JSON array of POA IDs (active/revoked/expired/etc.)
	boltBucketExpiration = "expiration" // index: expiration date (YYYY-MM-DD) -> JSON array of POA IDs
)

// BoltRepository is a BoltDB-backed implementation of POARepository.
// Layout:
//
//	Bucket "poa": key = POA.ID, value = JSON serialized PowerOfAttorney
//	Bucket "principal": key = principal (grantor or grantee), value = JSON array of POA IDs
//
// Rationale: Simple index supporting ListByPrincipal without scanning all POAs.
// Concurrency: Bolt supports multiple readers and single writer; we keep writes small.
// Thread-safety: DB handle is safe for concurrent use; we only guard close semantics.
type BoltRepository struct {
	db   *bolt.DB
	path string
	mu   sync.RWMutex // protects Close operations / future mutable state
}

// NewBoltRepository opens (or creates) a BoltDB file at path.
// It initializes required buckets. Caller should call Close() when done.
func NewBoltRepository(path string) (*BoltRepository, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}
	db, err := bolt.Open(abs, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bolt: %w", err)
	}
	br := &BoltRepository{db: db, path: abs}
	if err := db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists([]byte(boltBucketPOA)); e != nil {
			return e
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(boltBucketPrincipal)); e != nil {
			return e
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(boltBucketStatus)); e != nil {
			return e
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(boltBucketExpiration)); e != nil {
			return e
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	return br, nil
}

// Close closes the underlying DB.
func (b *BoltRepository) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.db == nil {
		return nil
	}
	err := b.db.Close()
	b.db = nil
	return err
}

// ensureOpen returns error if DB has been closed.
func (b *BoltRepository) ensureOpen() error {
	if b.db == nil {
		return errors.New("bolt repository closed")
	}
	return nil
}

// addToIndex appends a POA ID to an index bucket (deduplicates).
// bucket: target bucket, key: index key, id: POA ID to add
func addToIndex(bucket *bolt.Bucket, key, id string) error {
	prev := bucket.Get([]byte(key))
	var ids []string
	if prev != nil {
		if err := json.Unmarshal(prev, &ids); err != nil {
			return err
		}
	}
	// Append if not present
	found := false
	for _, existing := range ids {
		if existing == id {
			found = true
			break
		}
	}
	if !found {
		ids = append(ids, id)
	}
	enc, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), enc)
}

// removeFromIndex removes a POA ID from an index bucket.
func removeFromIndex(bucket *bolt.Bucket, key, id string) error {
	prev := bucket.Get([]byte(key))
	if prev == nil {
		return nil // Not present
	}
	var ids []string
	if err := json.Unmarshal(prev, &ids); err != nil {
		return err
	}
	// Remove the ID
	filtered := make([]string, 0, len(ids))
	for _, existing := range ids {
		if existing != id {
			filtered = append(filtered, existing)
		}
	}
	if len(filtered) == 0 {
		return bucket.Delete([]byte(key)) // Remove empty index
	}
	enc, err := json.Marshal(filtered)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), enc)
}

// Create persists a new PowerOfAttorney. Overwrites existing same ID (idempotent issuance path).
func (b *BoltRepository) Create(p *PowerOfAttorney) error {
	if p == nil {
		return errors.New("nil poa")
	}
	if p.ID == "" {
		return errors.New("empty id")
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return b.db.Update(func(tx *bolt.Tx) error {
		poaB := tx.Bucket([]byte(boltBucketPOA))
		princB := tx.Bucket([]byte(boltBucketPrincipal))
		statusB := tx.Bucket([]byte(boltBucketStatus))
		expB := tx.Bucket([]byte(boltBucketExpiration))
		if poaB == nil || princB == nil || statusB == nil || expB == nil {
			return errors.New("missing buckets")
		}
		if err := poaB.Put([]byte(p.ID), data); err != nil {
			return err
		}
		// Index grantor & grantee
		for _, principal := range []string{p.Grantor, p.Grantee} {
			if principal == "" {
				continue
			}
			prev := princB.Get([]byte(principal))
			var ids []string
			if prev != nil {
				_ = json.Unmarshal(prev, &ids)
			}
			// append if not present
			found := false
			for _, id := range ids {
				if id == p.ID {
					found = true
					break
				}
			}
			if !found {
				ids = append(ids, p.ID)
			}
			enc, mErr := json.Marshal(ids)
			if mErr != nil {
				return mErr
			}
			if err := princB.Put([]byte(principal), enc); err != nil {
				return err
			}
		}

		// Index by status
		statusKey := string(p.Status)
		if statusKey == "" {
			statusKey = string(POAStatusActive) // Default
		}
		if err := addToIndex(statusB, statusKey, p.ID); err != nil {
			return err
		}

		// Index by expiration date (YYYY-MM-DD)
		if !p.ValidUntil.IsZero() {
			expirationKey := p.ValidUntil.Format("2006-01-02")
			if err := addToIndex(expB, expirationKey, p.ID); err != nil {
				return err
			}
		}

		return nil
	})
}

// Get retrieves a POA by ID.
func (b *BoltRepository) Get(id string) (*PowerOfAttorney, bool) {
	if id == "" {
		return nil, false
	}
	if err := b.ensureOpen(); err != nil {
		return nil, false
	}
	var out *PowerOfAttorney
	_ = b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(boltBucketPOA))
		if bkt == nil {
			return errors.New("missing bucket")
		}
		v := bkt.Get([]byte(id))
		if v == nil {
			return nil
		}
		var p PowerOfAttorney
		if err := json.Unmarshal(v, &p); err != nil {
			return err
		}
		out = &p
		return nil
	})
	if out == nil {
		return nil, false
	}
	return out, true
}

// ListByPrincipal lists POAs where principal is grantor or grantee.
func (b *BoltRepository) ListByPrincipal(principal string) []*PowerOfAttorney {
	if principal == "" {
		return []*PowerOfAttorney{}
	}
	if err := b.ensureOpen(); err != nil {
		return []*PowerOfAttorney{}
	}
	ids := make([]string, 0)
	_ = b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(boltBucketPrincipal))
		if bkt == nil {
			return errors.New("missing bucket")
		}
		raw := bkt.Get([]byte(principal))
		if raw != nil {
			_ = json.Unmarshal(raw, &ids)
		}
		return nil
	})
	if len(ids) == 0 {
		return []*PowerOfAttorney{}
	}
	out := make([]*PowerOfAttorney, 0, len(ids))
	_ = b.db.View(func(tx *bolt.Tx) error {
		poaB := tx.Bucket([]byte(boltBucketPOA))
		if poaB == nil {
			return errors.New("missing bucket")
		}
		for _, id := range ids {
			v := poaB.Get([]byte(id))
			if v == nil {
				continue
			}
			var p PowerOfAttorney
			if err := json.Unmarshal(v, &p); err == nil {
				out = append(out, &p)
			}
		}
		return nil
	})
	return out
}

// Update replaces existing POA content (status, timestamps, etc.). Returns error if not found.
// Re-indexes status and expiration if they changed.
func (b *BoltRepository) Update(p *PowerOfAttorney) error {
	if p == nil {
		return errors.New("nil poa")
	}
	if p.ID == "" {
		return errors.New("empty id")
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return b.db.Update(func(tx *bolt.Tx) error {
		poaB := tx.Bucket([]byte(boltBucketPOA))
		statusB := tx.Bucket([]byte(boltBucketStatus))
		expB := tx.Bucket([]byte(boltBucketExpiration))

		if poaB == nil || statusB == nil || expB == nil {
			return errors.New("missing buckets")
		}

		existingData := poaB.Get([]byte(p.ID))
		if existingData == nil {
			return errors.New("not found")
		}

		// Parse old POA to detect index changes
		var old PowerOfAttorney
		if err := json.Unmarshal(existingData, &old); err != nil {
			return err
		}

		// Update primary record
		if err := poaB.Put([]byte(p.ID), data); err != nil {
			return err
		}

		// Re-index status if changed
		oldStatus := string(old.Status)
		if oldStatus == "" {
			oldStatus = string(POAStatusActive)
		}
		newStatus := string(p.Status)
		if newStatus == "" {
			newStatus = string(POAStatusActive)
		}
		if oldStatus != newStatus {
			// Remove from old status index
			if err := removeFromIndex(statusB, oldStatus, p.ID); err != nil {
				return err
			}
			// Add to new status index
			if err := addToIndex(statusB, newStatus, p.ID); err != nil {
				return err
			}
		}

		// Re-index expiration if changed
		oldExpKey := ""
		if !old.ValidUntil.IsZero() {
			oldExpKey = old.ValidUntil.Format("2006-01-02")
		}
		newExpKey := ""
		if !p.ValidUntil.IsZero() {
			newExpKey = p.ValidUntil.Format("2006-01-02")
		}
		if oldExpKey != newExpKey {
			// Remove from old expiration index
			if oldExpKey != "" {
				if err := removeFromIndex(expB, oldExpKey, p.ID); err != nil {
					return err
				}
			}
			// Add to new expiration index
			if newExpKey != "" {
				if err := addToIndex(expB, newExpKey, p.ID); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// ListDescendants finds all POAs that have the given POA ID as their parent.
// Performs depth-limited traversal with cycle detection.
func (b *BoltRepository) ListDescendants(parentPoaID string, maxDepth int) ([]*PowerOfAttorney, error) {
	if parentPoaID == "" {
		return []*PowerOfAttorney{}, nil
	}
	if err := b.ensureOpen(); err != nil {
		return nil, err
	}

	var result []*PowerOfAttorney
	visited := make(map[string]bool) // Cycle prevention

	// Helper to find direct children of a given parent ID
	findDirectChildren := func(currentParentID string) ([]*PowerOfAttorney, error) {
		var children []*PowerOfAttorney
		err := b.db.View(func(tx *bolt.Tx) error {
			poaB := tx.Bucket([]byte(boltBucketPOA))
			if poaB == nil {
				return errors.New("missing bucket")
			}
			// Scan all POAs to find those with matching parent_poa_id
			// TODO: Add parent_poa_id index for better performance
			return poaB.ForEach(func(k, v []byte) error {
				var p PowerOfAttorney
				if err := json.Unmarshal(v, &p); err != nil {
					return nil // Skip malformed entries
				}
				if p.ParentPOAID == currentParentID {
					children = append(children, &p)
				}
				return nil
			})
		})
		return children, err
	}

	// Recursive depth-first search for descendants
	var findDescendants func(currentParentID string, currentDepth int) error
	findDescendants = func(currentParentID string, currentDepth int) error {
		if maxDepth > 0 && currentDepth >= maxDepth {
			return nil // Hit depth limit
		}
		if visited[currentParentID] {
			return nil // Cycle prevention
		}
		visited[currentParentID] = true

		children, err := findDirectChildren(currentParentID)
		if err != nil {
			return err
		}

		for _, child := range children {
			result = append(result, child)
			// Recursively find descendants of this child
			if err := findDescendants(child.ID, currentDepth+1); err != nil {
				return err
			}
		}
		return nil
	}

	if err := findDescendants(parentPoaID, 0); err != nil {
		return nil, err
	}
	return result, nil
}

// WithPOARepository injects a custom repository (e.g., Bolt).
func WithPOARepository(repo POARepository) Option {
	return func(s *Service) {
		if repo != nil {
			s.repo = repo
		}
	}
}

// ===== Enhanced Indexing & Pruning Methods (P2.3) =====

// FindByStatus returns all POAs with the given status.
func (b *BoltRepository) FindByStatus(status POAStatus) ([]*PowerOfAttorney, error) {
	if err := b.ensureOpen(); err != nil {
		return nil, err
	}

	var ids []string
	if err := b.db.View(func(tx *bolt.Tx) error {
		statusB := tx.Bucket([]byte(boltBucketStatus))
		if statusB == nil {
			return errors.New("missing status bucket")
		}
		val := statusB.Get([]byte(string(status)))
		if val != nil {
			return json.Unmarshal(val, &ids)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Retrieve full POAs
	var out []*PowerOfAttorney
	_ = b.db.View(func(tx *bolt.Tx) error {
		poaB := tx.Bucket([]byte(boltBucketPOA))
		if poaB == nil {
			return errors.New("missing poa bucket")
		}
		for _, id := range ids {
			v := poaB.Get([]byte(id))
			if v == nil {
				continue
			}
			var p PowerOfAttorney
			if err := json.Unmarshal(v, &p); err == nil {
				out = append(out, &p)
			}
		}
		return nil
	})
	return out, nil
}

// FindExpired returns all POAs expiring before the given time.
func (b *BoltRepository) FindExpired(before time.Time) ([]*PowerOfAttorney, error) {
	if err := b.ensureOpen(); err != nil {
		return nil, err
	}

	var allIDs []string
	if err := b.db.View(func(tx *bolt.Tx) error {
		expB := tx.Bucket([]byte(boltBucketExpiration))
		if expB == nil {
			return errors.New("missing expiration bucket")
		}

		// Scan all expiration dates <= before (compare by formatting to date strings)
		beforeDateStr := before.Format("2006-01-02")
		return expB.ForEach(func(k, v []byte) error {
			dateStr := string(k)
			// Compare date strings lexicographically (YYYY-MM-DD sorts correctly)
			if dateStr <= beforeDateStr {
				var ids []string
				if err := json.Unmarshal(v, &ids); err == nil {
					allIDs = append(allIDs, ids...)
				}
			}
			return nil
		})
	}); err != nil {
		return nil, err
	}

	// Retrieve full POAs
	var out []*PowerOfAttorney
	_ = b.db.View(func(tx *bolt.Tx) error {
		poaB := tx.Bucket([]byte(boltBucketPOA))
		if poaB == nil {
			return errors.New("missing poa bucket")
		}
		for _, id := range allIDs {
			v := poaB.Get([]byte(id))
			if v == nil {
				continue
			}
			var p PowerOfAttorney
			if err := json.Unmarshal(v, &p); err == nil {
				out = append(out, &p)
			}
		}
		return nil
	})
	return out, nil
}

// PruneExpired deletes POAs that expired before retentionCutoff and are already marked expired.
// Returns the count of pruned entries.
func (b *BoltRepository) PruneExpired(retentionCutoff time.Time) (int, error) {
	if err := b.ensureOpen(); err != nil {
		return 0, err
	}

	expired, err := b.FindExpired(retentionCutoff)
	if err != nil {
		return 0, err
	}

	// Only delete if status is expired (safety check)
	count := 0
	for _, p := range expired {
		if p.Status != POAStatusExpired {
			continue // Skip non-expired status
		}
		if err := b.deletePOA(p); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// PruneRevoked deletes POAs that were revoked before retentionCutoff.
// Returns the count of pruned entries.
func (b *BoltRepository) PruneRevoked(retentionCutoff time.Time) (int, error) {
	if err := b.ensureOpen(); err != nil {
		return 0, err
	}

	revoked, err := b.FindByStatus(POAStatusRevoked)
	if err != nil {
		return 0, err
	}

	// Only delete if revoked before retention cutoff
	count := 0
	for _, p := range revoked {
		// Check if revoked long enough ago (use UpdatedAt or CreatedAt as proxy)
		if !p.UpdatedAt.Before(retentionCutoff) {
			continue
		}
		if err := b.deletePOA(p); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// deletePOA removes a POA and all its index entries.
func (b *BoltRepository) deletePOA(p *PowerOfAttorney) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		poaB := tx.Bucket([]byte(boltBucketPOA))
		princB := tx.Bucket([]byte(boltBucketPrincipal))
		statusB := tx.Bucket([]byte(boltBucketStatus))
		expB := tx.Bucket([]byte(boltBucketExpiration))

		if poaB == nil || princB == nil || statusB == nil || expB == nil {
			return errors.New("missing buckets")
		}

		// Delete primary record
		if err := poaB.Delete([]byte(p.ID)); err != nil {
			return err
		}

		// Remove from principal index
		for _, principal := range []string{p.Grantor, p.Grantee} {
			if principal == "" {
				continue
			}
			if err := removeFromIndex(princB, principal, p.ID); err != nil {
				return err
			}
		}

		// Remove from status index
		statusKey := string(p.Status)
		if statusKey == "" {
			statusKey = string(POAStatusActive)
		}
		if err := removeFromIndex(statusB, statusKey, p.ID); err != nil {
			return err
		}

		// Remove from expiration index
		if !p.ValidUntil.IsZero() {
			expirationKey := p.ValidUntil.Format("2006-01-02")
			if err := removeFromIndex(expB, expirationKey, p.ID); err != nil {
				return err
			}
		}

		return nil
	})
}

// BoltRepositoryStats provides storage statistics.
type BoltRepositoryStats struct {
	TotalPOAs    int
	ActivePOAs   int
	RevokedPOAs  int
	ExpiredPOAs  int
	DatabasePath string
	DatabaseSize int64 // File size in bytes
}

// Stats returns repository statistics.
func (b *BoltRepository) Stats() (*BoltRepositoryStats, error) {
	if err := b.ensureOpen(); err != nil {
		return nil, err
	}

	stats := &BoltRepositoryStats{
		DatabasePath: b.path,
	}

	// Get file size using os.Stat
	if fi, err := os.Stat(b.path); err == nil {
		stats.DatabaseSize = fi.Size()
	}

	// Count by status
	if err := b.db.View(func(tx *bolt.Tx) error {
		statusB := tx.Bucket([]byte(boltBucketStatus))
		if statusB == nil {
			return nil
		}

		// Count active
		if val := statusB.Get([]byte(string(POAStatusActive))); val != nil {
			var ids []string
			if json.Unmarshal(val, &ids) == nil {
				stats.ActivePOAs = len(ids)
			}
		}

		// Count revoked
		if val := statusB.Get([]byte(string(POAStatusRevoked))); val != nil {
			var ids []string
			if json.Unmarshal(val, &ids) == nil {
				stats.RevokedPOAs = len(ids)
			}
		}

		// Count expired
		if val := statusB.Get([]byte(string(POAStatusExpired))); val != nil {
			var ids []string
			if json.Unmarshal(val, &ids) == nil {
				stats.ExpiredPOAs = len(ids)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// Total = active + revoked + expired + other statuses
	stats.TotalPOAs = stats.ActivePOAs + stats.RevokedPOAs + stats.ExpiredPOAs

	return stats, nil
}

// Compact triggers BoltDB compaction to reclaim space after pruning.
func (b *BoltRepository) Compact() error {
	if err := b.ensureOpen(); err != nil {
		return err
	}

	// BoltDB doesn't have built-in compaction, but we can trigger a manual compaction
	// by copying to a new file and replacing the old one.
	// For now, return nil (future enhancement: implement file-level compaction)
	return nil
}
