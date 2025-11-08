package policy

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	pkgpolicy "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/policy"
	bolt "go.etcd.io/bbolt"
)

// PolicyVersionStore provides persistent storage for policy versions.
type PolicyVersionStore interface {
	// SaveVersion persists a policy version with metadata
	SaveVersion(version int, bundle pkgpolicy.Bundle, metadata *PolicyVersionMetadata) error

	// LoadVersion retrieves a policy version
	LoadVersion(version int) (*pkgpolicy.Bundle, *PolicyVersionMetadata, error)

	// ListVersions returns all stored version numbers
	ListVersions() ([]int, error)

	// SaveActiveVersion persists the currently active version
	SaveActiveVersion(version int) error

	// LoadActiveVersion retrieves the active version
	LoadActiveVersion() (int, error)

	// SaveAuditEvent persists a version audit event
	SaveAuditEvent(event VersionAuditEvent) error

	// LoadAuditEvents retrieves audit events (optionally filtered by version)
	LoadAuditEvents(version int) ([]VersionAuditEvent, error)

	// Close closes the store
	Close() error
}

// BoltPolicyVersionStore implements PolicyVersionStore using BoltDB.
type BoltPolicyVersionStore struct {
	db   *bolt.DB
	path string
	mu   sync.RWMutex
}

const (
	versionMetadataBucket = "version_metadata" // version -> PolicyVersionMetadata JSON
	bundlesBucket         = "bundles"          // version -> Bundle JSON
	activeVersionKey      = "active_version"   // singleton key for active version
	auditEventsBucket     = "audit_events"     // event_id -> VersionAuditEvent JSON
	auditIndexBucket      = "audit_index"      // version -> []event_id (for filtering)
)

// NewBoltPolicyVersionStore creates a new BoltDB-backed policy version store.
func NewBoltPolicyVersionStore(dbPath string) (*BoltPolicyVersionStore, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bolt db: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err2 := tx.CreateBucketIfNotExists([]byte(versionMetadataBucket)); err2 != nil {
			return fmt.Errorf("create version_metadata bucket: %w", err2)
		}
		if _, err2 := tx.CreateBucketIfNotExists([]byte(bundlesBucket)); err2 != nil {
			return fmt.Errorf("create bundles bucket: %w", err2)
		}
		if _, err2 := tx.CreateBucketIfNotExists([]byte(auditEventsBucket)); err2 != nil {
			return fmt.Errorf("create audit_events bucket: %w", err2)
		}
		if _, err2 := tx.CreateBucketIfNotExists([]byte(auditIndexBucket)); err2 != nil {
			return fmt.Errorf("create audit_index bucket: %w", err2)
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &BoltPolicyVersionStore{
		db:   db,
		path: dbPath,
	}, nil
}

// SaveVersion persists a policy version with metadata.
func (s *BoltPolicyVersionStore) SaveVersion(version int, bundle pkgpolicy.Bundle, metadata *PolicyVersionMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		// Serialize metadata
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}

		// Serialize bundle
		bundleJSON, err := json.Marshal(bundle)
		if err != nil {
			return fmt.Errorf("marshal bundle: %w", err)
		}

		// Save metadata
		metadataBucket := tx.Bucket([]byte(versionMetadataBucket))
		versionKey := []byte(strconv.Itoa(version))
		if err := metadataBucket.Put(versionKey, metadataJSON); err != nil {
			return fmt.Errorf("put metadata: %w", err)
		}

		// Save bundle
		bundlesBucket := tx.Bucket([]byte(bundlesBucket))
		if err := bundlesBucket.Put(versionKey, bundleJSON); err != nil {
			return fmt.Errorf("put bundle: %w", err)
		}

		return nil
	})
}

// LoadVersion retrieves a policy version.
func (s *BoltPolicyVersionStore) LoadVersion(version int) (*pkgpolicy.Bundle, *PolicyVersionMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bundle pkgpolicy.Bundle
	var metadata PolicyVersionMetadata

	err := s.db.View(func(tx *bolt.Tx) error {
		versionKey := []byte(strconv.Itoa(version))

		// Load metadata
		metadataBucket := tx.Bucket([]byte(versionMetadataBucket))
		metadataJSON := metadataBucket.Get(versionKey)
		if metadataJSON == nil {
			return fmt.Errorf("version %d not found", version)
		}
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return fmt.Errorf("unmarshal metadata: %w", err)
		}

		// Load bundle
		bundlesBucket := tx.Bucket([]byte(bundlesBucket))
		bundleJSON := bundlesBucket.Get(versionKey)
		if bundleJSON == nil {
			return fmt.Errorf("bundle for version %d not found", version)
		}
		if err := json.Unmarshal(bundleJSON, &bundle); err != nil {
			return fmt.Errorf("unmarshal bundle: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return &bundle, &metadata, nil
}

// ListVersions returns all stored version numbers.
func (s *BoltPolicyVersionStore) ListVersions() ([]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var versions []int

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(versionMetadataBucket))
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			version, err := strconv.Atoi(string(k))
			if err != nil {
				continue // Skip invalid keys
			}
			versions = append(versions, version)
		}

		return nil
	})

	return versions, err
}

// SaveActiveVersion persists the currently active version.
func (s *BoltPolicyVersionStore) SaveActiveVersion(version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(versionMetadataBucket))
		return b.Put([]byte(activeVersionKey), []byte(strconv.Itoa(version)))
	})
}

// LoadActiveVersion retrieves the active version.
func (s *BoltPolicyVersionStore) LoadActiveVersion() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var version int

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(versionMetadataBucket))
		versionBytes := b.Get([]byte(activeVersionKey))
		if versionBytes == nil {
			return fmt.Errorf("active version not set")
		}

		var err error
		version, err = strconv.Atoi(string(versionBytes))
		return err
	})

	return version, err
}

// SaveAuditEvent persists a version audit event.
func (s *BoltPolicyVersionStore) SaveAuditEvent(event VersionAuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		// Generate event ID
		eventID := fmt.Sprintf("%d-%s-%d", event.Timestamp.UnixNano(), event.EventType, event.Version)

		// Serialize event
		eventJSON, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal audit event: %w", err)
		}

		// Save to audit_events bucket
		auditBucket := tx.Bucket([]byte(auditEventsBucket))
		if err2 := auditBucket.Put([]byte(eventID), eventJSON); err2 != nil {
			return fmt.Errorf("put audit event: %w", err)
		}

		// Update index (version -> event IDs)
		indexBucket := tx.Bucket([]byte(auditIndexBucket))
		versionKey := []byte(strconv.Itoa(event.Version))

		// Load existing event IDs for this version
		var eventIDs []string
		existingJSON := indexBucket.Get(versionKey)
		if existingJSON != nil {
			if err2 := json.Unmarshal(existingJSON, &eventIDs); err2 != nil {
				// Ignore unmarshal errors, start fresh
				eventIDs = []string{}
			}
		}

		// Append new event ID
		eventIDs = append(eventIDs, eventID)

		// Save updated index
		updatedJSON, err := json.Marshal(eventIDs)
		if err != nil {
			return fmt.Errorf("marshal event IDs: %w", err)
		}
		if err := indexBucket.Put(versionKey, updatedJSON); err != nil {
			return fmt.Errorf("put event index: %w", err)
		}

		return nil
	})
}

// LoadAuditEvents retrieves audit events (optionally filtered by version).
// If version is 0, returns all events. Otherwise, returns events for the specified version.
func (s *BoltPolicyVersionStore) LoadAuditEvents(version int) ([]VersionAuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var events []VersionAuditEvent

	err := s.db.View(func(tx *bolt.Tx) error {
		auditBucket := tx.Bucket([]byte(auditEventsBucket))

		if version == 0 {
			// Return all events
			c := auditBucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				var event VersionAuditEvent
				if err := json.Unmarshal(v, &event); err != nil {
					continue // Skip malformed events
				}
				events = append(events, event)
			}
		} else {
			// Return events for specific version
			indexBucket := tx.Bucket([]byte(auditIndexBucket))
			versionKey := []byte(strconv.Itoa(version))
			eventIDsJSON := indexBucket.Get(versionKey)
			if eventIDsJSON == nil {
				return nil // No events for this version
			}

			var eventIDs []string
			if err := json.Unmarshal(eventIDsJSON, &eventIDs); err != nil {
				return fmt.Errorf("unmarshal event IDs: %w", err)
			}

			// Load each event
			for _, eventID := range eventIDs {
				eventJSON := auditBucket.Get([]byte(eventID))
				if eventJSON == nil {
					continue // Event not found (orphaned index entry)
				}

				var event VersionAuditEvent
				if err := json.Unmarshal(eventJSON, &event); err != nil {
					continue // Skip malformed events
				}
				events = append(events, event)
			}
		}

		return nil
	})

	return events, err
}

// Close closes the store.
func (s *BoltPolicyVersionStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

// Stats returns statistics about the policy version store.
func (s *BoltPolicyVersionStore) Stats() (*PolicyVersionStoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &PolicyVersionStoreStats{}

	err := s.db.View(func(tx *bolt.Tx) error {
		// Count versions
		metadataBucket := tx.Bucket([]byte(versionMetadataBucket))
		stats.TotalVersions = metadataBucket.Stats().KeyN - 1 // Exclude active_version key

		// Count bundles
		bundlesBucket := tx.Bucket([]byte(bundlesBucket))
		stats.TotalBundles = bundlesBucket.Stats().KeyN

		// Count audit events
		auditBucket := tx.Bucket([]byte(auditEventsBucket))
		stats.TotalAuditEvents = auditBucket.Stats().KeyN

		// Get active version
		activeVersionBytes := metadataBucket.Get([]byte(activeVersionKey))
		if activeVersionBytes != nil {
			activeVersion, err := strconv.Atoi(string(activeVersionBytes))
			if err == nil {
				stats.ActiveVersion = activeVersion
			}
		}

		return nil
	})

	return stats, err
}

// PolicyVersionStoreStats contains statistics about the policy version store.
type PolicyVersionStoreStats struct {
	TotalVersions    int `json:"total_versions"`
	TotalBundles     int `json:"total_bundles"`
	TotalAuditEvents int `json:"total_audit_events"`
	ActiveVersion    int `json:"active_version"`
}
