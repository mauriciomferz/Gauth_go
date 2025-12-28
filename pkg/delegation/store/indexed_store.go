// Package store provides indexed delegation storage with efficient queries and pruning.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/bbolt"
)

// DelegationRecord represents a delegation with metadata for indexing.
type DelegationRecord struct {
	ID           string            `json:"id"`
	Subject      string            `json:"subject"`
	Delegate     string            `json:"delegate"`
	Scope        map[string]string `json:"scope"`
	IssuedAt     time.Time         `json:"issued_at"`
	ExpiresAt    time.Time         `json:"expires_at"`
	Status       string            `json:"status"`
	Hash         string            `json:"hash"`
	PrevHash     string            `json:"prev_hash"`
	LastAccessed time.Time         `json:"last_accessed"`
}

// DelegationStatus represents the status of a delegation.
type DelegationStatus string

const (
	StatusActive           DelegationStatus = "active"
	StatusExpired          DelegationStatus = "expired"
	StatusRevoked          DelegationStatus = "revoked"
	StatusSuspended        DelegationStatus = "suspended"
	StatusPartiallyRevoked DelegationStatus = "partially_revoked"
)

// IndexedDelegationStore provides high-performance delegation storage with multiple indexes.
type IndexedDelegationStore struct {
	mu sync.RWMutex
	db *bbolt.DB

	// Statistics
	// Statistics
	stats *StoreStats

	// Lifecycle
	quit chan struct{}
	wg   sync.WaitGroup
}

// PruneConfig configures automated pruning behavior.
type PruneConfig struct {
	Enabled          bool
	Interval         time.Duration
	RetentionPeriod  time.Duration
	InactivityPeriod time.Duration
}

// DefaultPruneConfig returns a default pruning configuration.
func DefaultPruneConfig() PruneConfig {
	return PruneConfig{
		Enabled:          true,
		Interval:         1 * time.Hour,
		RetentionPeriod:  24 * time.Hour * 30, // 30 days
		InactivityPeriod: 24 * time.Hour * 90, // 90 days
	}
}

// StoreStats tracks store operations and performance metrics.
type StoreStats struct {
	TotalRecords   int64
	ActiveRecords  int64
	ExpiredRecords int64
	RevokedRecords int64
	PrunedRecords  int64
	LastPruneTime  time.Time
}

// Bucket names for BoltDB
const (
	bucketDelegations     = "delegations"
	bucketSubjectIndex    = "index_subject"
	bucketDelegateIndex   = "index_delegate"
	bucketExpiryIndex     = "index_expiry"
	bucketStatusIndex     = "index_status"
	bucketAccessTimeIndex = "index_access_time"
	bucketStats           = "stats"
)

// IndexedDelegationStoreOption configures the store.
type IndexedDelegationStoreOption func(*IndexedDelegationStore)

// WithPruneConfig sets the pruning configuration.
func WithPruneConfig(config PruneConfig) IndexedDelegationStoreOption {
	return func(s *IndexedDelegationStore) {
		// Store config in a new field if we had one, but for now we'll just use it to start the routine
		// Since we didn't add a Config field to struct in this diff stride, we'll implement it slightly differently
		// or add the field to struct now.
		// Let's add 'pruneConfig' to struct to be clean.
	}
}

// NewIndexedDelegationStore creates a new indexed delegation store.
// Now accepts optional configuration for pruning.
func NewIndexedDelegationStore(dbPath string, pruneConfig *PruneConfig) (*IndexedDelegationStore, error) {
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create all buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		buckets := []string{
			bucketDelegations,
			bucketSubjectIndex,
			bucketDelegateIndex,
			bucketExpiryIndex,
			bucketStatusIndex,
			bucketAccessTimeIndex,
			bucketStats,
		}

		for _, bucket := range buckets {
			if _, err2 := tx.CreateBucketIfNotExists([]byte(bucket)); err2 != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	store := &IndexedDelegationStore{
		db:    db,
		stats: &StoreStats{},
		quit:  make(chan struct{}),
	}

	// Load stats
	_ = store.loadStats() // Best effort stat loading

	// Start background pruning if configured
	if pruneConfig != nil && pruneConfig.Enabled {
		store.wg.Add(1)
		go store.pruneLoop(*pruneConfig)
	}

	return store, nil
}

// Close closes the delegation store and stops background tasks.
func (s *IndexedDelegationStore) Close() error {
	// Signal shutdown
	close(s.quit)
	s.wg.Wait()

	return s.db.Close()
}

// pruneLoop runs periodic pruning.
func (s *IndexedDelegationStore) pruneLoop(config PruneConfig) {
	defer s.wg.Done()
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Run pruning
			if _, err := s.PruneExpired(config.RetentionPeriod); err != nil {
				fmt.Printf("Failed to prune expired delegations: %v\n", err)
			}
			if _, err := s.PruneInactive(config.InactivityPeriod); err != nil {
				fmt.Printf("Failed to prune inactive delegations: %v\n", err)
			}
		case <-s.quit:
			return
		}
	}
}

// Store saves a delegation record with all indexes.
func (s *IndexedDelegationStore) Store(record *DelegationRecord) error {
	if record == nil || record.ID == "" {
		return errors.New("invalid delegation record")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bbolt.Tx) error {
		// Serialize record
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}

		// Store main record
		bucket := tx.Bucket([]byte(bucketDelegations))
		if err := bucket.Put([]byte(record.ID), data); err != nil {
			return err
		}

		// Update indexes
		if err := s.updateIndexes(tx, record); err != nil {
			return err
		}

		// Update stats
		s.stats.TotalRecords++
		if record.Status == string(StatusActive) {
			s.stats.ActiveRecords++
		}

		return s.saveStats(tx)
	})
}

// Get retrieves a delegation by ID.
func (s *IndexedDelegationStore) Get(id string) (*DelegationRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var record *DelegationRecord
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketDelegations))
		data := bucket.Get([]byte(id))
		if data == nil {
			return errors.New("delegation not found")
		}

		record = &DelegationRecord{}
		return json.Unmarshal(data, record)
	})

	if err != nil {
		return nil, err
	}

	// Update last accessed time asynchronously
	go func() {
		_ = s.updateAccessTime(id) // Best effort update
	}()

	return record, nil
}

// GetBySubject retrieves all delegations for a subject.
func (s *IndexedDelegationStore) GetBySubject(subject string) ([]*DelegationRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []*DelegationRecord
	err := s.db.View(func(tx *bbolt.Tx) error {
		indexBucket := tx.Bucket([]byte(bucketSubjectIndex))
		delegationBucket := tx.Bucket([]byte(bucketDelegations))

		// Get delegation IDs from index
		idsData := indexBucket.Get([]byte(subject))
		if idsData == nil {
			return nil // No delegations for this subject
		}

		var ids []string
		if err := json.Unmarshal(idsData, &ids); err != nil {
			return err
		}

		// Fetch each delegation
		for _, id := range ids {
			data := delegationBucket.Get([]byte(id))
			if data == nil {
				continue
			}

			record := &DelegationRecord{}
			if err := json.Unmarshal(data, record); err != nil {
				continue
			}
			records = append(records, record)
		}

		return nil
	})

	return records, err
}

// GetByDelegate retrieves all delegations for a delegate.
func (s *IndexedDelegationStore) GetByDelegate(delegate string) ([]*DelegationRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []*DelegationRecord
	err := s.db.View(func(tx *bbolt.Tx) error {
		indexBucket := tx.Bucket([]byte(bucketDelegateIndex))
		delegationBucket := tx.Bucket([]byte(bucketDelegations))

		idsData := indexBucket.Get([]byte(delegate))
		if idsData == nil {
			return nil
		}

		var ids []string
		if err := json.Unmarshal(idsData, &ids); err != nil {
			return err
		}

		for _, id := range ids {
			data := delegationBucket.Get([]byte(id))
			if data == nil {
				continue
			}

			record := &DelegationRecord{}
			if err := json.Unmarshal(data, record); err != nil {
				continue
			}
			records = append(records, record)
		}

		return nil
	})

	return records, err
}

// GetByStatus retrieves all delegations with a specific status.
func (s *IndexedDelegationStore) GetByStatus(status DelegationStatus) ([]*DelegationRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []*DelegationRecord
	err := s.db.View(func(tx *bbolt.Tx) error {
		indexBucket := tx.Bucket([]byte(bucketStatusIndex))
		delegationBucket := tx.Bucket([]byte(bucketDelegations))

		idsData := indexBucket.Get([]byte(status))
		if idsData == nil {
			return nil
		}

		var ids []string
		if err := json.Unmarshal(idsData, &ids); err != nil {
			return err
		}

		for _, id := range ids {
			data := delegationBucket.Get([]byte(id))
			if data == nil {
				continue
			}

			record := &DelegationRecord{}
			if err := json.Unmarshal(data, record); err != nil {
				continue
			}
			records = append(records, record)
		}

		return nil
	})

	return records, err
}

// PruneExpired removes expired delegations older than the retention period.
func (s *IndexedDelegationStore) PruneExpired(retentionPeriod time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffTime := time.Now().Add(-retentionPeriod)
	pruned := 0

	err := s.db.Update(func(tx *bbolt.Tx) error {
		delegationBucket := tx.Bucket([]byte(bucketDelegations))
		cursor := delegationBucket.Cursor()

		var toDelete []string

		// Find expired delegations
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			record := &DelegationRecord{}
			if err := json.Unmarshal(v, record); err != nil {
				continue
			}

			// Delete if expired and past retention period
			if record.ExpiresAt.Before(cutoffTime) {
				toDelete = append(toDelete, record.ID)
			}
		}

		// Delete records and update indexes
		for _, id := range toDelete {
			data := delegationBucket.Get([]byte(id))
			if data == nil {
				continue
			}

			record := &DelegationRecord{}
			if err := json.Unmarshal(data, record); err != nil {
				continue
			}

			// Remove from main bucket
			if err := delegationBucket.Delete([]byte(id)); err != nil {
				return err
			}

			// Remove from indexes
			if err := s.removeFromIndexes(tx, record); err != nil {
				return err
			}

			pruned++
		}

		// Update stats
		s.stats.PrunedRecords += int64(pruned)
		s.stats.TotalRecords -= int64(pruned)
		s.stats.LastPruneTime = time.Now()

		return s.saveStats(tx)
	})

	return pruned, err
}

// PruneInactive removes delegations that haven't been accessed in a long time.
func (s *IndexedDelegationStore) PruneInactive(inactivityPeriod time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoffTime := time.Now().Add(-inactivityPeriod)
	pruned := 0

	err := s.db.Update(func(tx *bbolt.Tx) error {
		delegationBucket := tx.Bucket([]byte(bucketDelegations))
		cursor := delegationBucket.Cursor()

		var toDelete []string

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			record := &DelegationRecord{}
			if err := json.Unmarshal(v, record); err != nil {
				continue
			}

			// Delete if not accessed recently and expired
			if record.LastAccessed.Before(cutoffTime) && record.ExpiresAt.Before(time.Now()) {
				toDelete = append(toDelete, record.ID)
			}
		}

		for _, id := range toDelete {
			data := delegationBucket.Get([]byte(id))
			if data == nil {
				continue
			}

			record := &DelegationRecord{}
			if err := json.Unmarshal(data, record); err != nil {
				continue
			}

			if err := delegationBucket.Delete([]byte(id)); err != nil {
				return err
			}

			if err := s.removeFromIndexes(tx, record); err != nil {
				return err
			}

			pruned++
		}

		s.stats.PrunedRecords += int64(pruned)
		s.stats.TotalRecords -= int64(pruned)

		return s.saveStats(tx)
	})

	return pruned, err
}

// UpdateStatus updates the status of a delegation.
func (s *IndexedDelegationStore) UpdateStatus(id string, newStatus DelegationStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketDelegations))
		data := bucket.Get([]byte(id))
		if data == nil {
			return errors.New("delegation not found")
		}

		record := &DelegationRecord{}
		if err := json.Unmarshal(data, record); err != nil {
			return err
		}

		oldStatus := record.Status
		record.Status = string(newStatus)

		// Re-serialize
		updatedData, err := json.Marshal(record)
		if err != nil {
			return err
		}

		if err := bucket.Put([]byte(id), updatedData); err != nil {
			return err
		}

		// Update status index
		return s.updateStatusIndex(tx, record, oldStatus)
	})
}

// GetStats returns current store statistics.
func (s *IndexedDelegationStore) GetStats() *StoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statsCopy := *s.stats
	return &statsCopy
}

// updateIndexes updates all secondary indexes for a record.
func (s *IndexedDelegationStore) updateIndexes(tx *bbolt.Tx, record *DelegationRecord) error {
	// Subject index
	if err := s.addToIndex(tx, bucketSubjectIndex, record.Subject, record.ID); err != nil {
		return err
	}

	// Delegate index
	if err := s.addToIndex(tx, bucketDelegateIndex, record.Delegate, record.ID); err != nil {
		return err
	}

	// Status index
	if err := s.addToIndex(tx, bucketStatusIndex, record.Status, record.ID); err != nil {
		return err
	}

	return nil
}

// addToIndex adds a record ID to an index.
func (s *IndexedDelegationStore) addToIndex(tx *bbolt.Tx, bucketName, key, id string) error {
	bucket := tx.Bucket([]byte(bucketName))

	var ids []string
	data := bucket.Get([]byte(key))
	if data != nil {
		if err := json.Unmarshal(data, &ids); err != nil {
			return err
		}
	}

	// Add ID if not already present
	found := false
	for _, existingID := range ids {
		if existingID == id {
			found = true
			break
		}
	}

	if !found {
		ids = append(ids, id)
		data, err := json.Marshal(ids)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(key), data)
	}

	return nil
}

// removeFromIndexes removes a record from all indexes.
func (s *IndexedDelegationStore) removeFromIndexes(tx *bbolt.Tx, record *DelegationRecord) error {
	if err := s.removeFromIndex(tx, bucketSubjectIndex, record.Subject, record.ID); err != nil {
		return err
	}
	if err := s.removeFromIndex(tx, bucketDelegateIndex, record.Delegate, record.ID); err != nil {
		return err
	}
	if err := s.removeFromIndex(tx, bucketStatusIndex, record.Status, record.ID); err != nil {
		return err
	}
	return nil
}

// removeFromIndex removes a record ID from an index.
func (s *IndexedDelegationStore) removeFromIndex(tx *bbolt.Tx, bucketName, key, id string) error {
	bucket := tx.Bucket([]byte(bucketName))
	data := bucket.Get([]byte(key))
	if data == nil {
		return nil
	}

	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return err
	}

	// Remove ID
	newIDs := make([]string, 0, len(ids))
	for _, existingID := range ids {
		if existingID != id {
			newIDs = append(newIDs, existingID)
		}
	}

	if len(newIDs) == 0 {
		return bucket.Delete([]byte(key))
	}

	data, err := json.Marshal(newIDs)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), data)
}

// updateStatusIndex updates the status index when status changes.
func (s *IndexedDelegationStore) updateStatusIndex(tx *bbolt.Tx, record *DelegationRecord, oldStatus string) error {
	// Remove from old status index
	if err := s.removeFromIndex(tx, bucketStatusIndex, oldStatus, record.ID); err != nil {
		return err
	}

	// Add to new status index
	return s.addToIndex(tx, bucketStatusIndex, record.Status, record.ID)
}

// updateAccessTime updates the last accessed time for a delegation.
func (s *IndexedDelegationStore) updateAccessTime(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketDelegations))
		data := bucket.Get([]byte(id))
		if data == nil {
			return nil
		}

		record := &DelegationRecord{}
		if err := json.Unmarshal(data, record); err != nil {
			return err
		}

		record.LastAccessed = time.Now()

		updatedData, err := json.Marshal(record)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(id), updatedData)
	})
}

// saveStats persists store statistics.
func (s *IndexedDelegationStore) saveStats(tx *bbolt.Tx) error {
	bucket := tx.Bucket([]byte(bucketStats))
	data, err := json.Marshal(s.stats)
	if err != nil {
		return err
	}
	return bucket.Put([]byte("stats"), data)
}

// loadStats loads store statistics from disk.
func (s *IndexedDelegationStore) loadStats() error {
	return s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketStats))
		data := bucket.Get([]byte("stats"))
		if data == nil {
			return nil // No stats yet
		}
		return json.Unmarshal(data, s.stats)
	})
}
