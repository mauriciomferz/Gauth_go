// Copyright 2025 Gimel Foundation
// SPDX-License-Identifier: Apache-2.0

package gauth

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

// BoltReplayStore implements durable replay detection using BoltDB.
// Addresses gap sec6.item1 (P1): Durable replay persistence with eviction controls.
type BoltReplayStore struct {
	db         *bolt.DB
	bucketName []byte
	ttl        time.Duration
}

// NewBoltReplayStore creates a new BoltDB-backed replay store.
// The path parameter specifies where the database file should be created.
// TTL determines how long JTI entries are retained before expiration.
func NewBoltReplayStore(path string, ttl time.Duration) (*BoltReplayStore, error) {
	// Ensure directory exists
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("replay_store: mkdir failed: %w", err)
		}
	}

	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("replay_store: open boltdb: %w", err)
	}

	bucketName := []byte("jti_replay")

	// Create bucket if it doesn't exist
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("replay_store: create bucket: %w", err)
	}

	store := &BoltReplayStore{
		db:         db,
		bucketName: bucketName,
		ttl:        ttl,
	}

	// Start background cleanup goroutine
	go store.cleanupExpired()

	return store, nil
}

// CheckAndRecord checks if JTI has been seen and records it if not.
// Returns error if JTI already exists (replay detected).
func (s *BoltReplayStore) CheckAndRecord(jti string) error {
	now := time.Now().Unix()
	expiresAt := now + int64(s.ttl.Seconds())

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucketName)
		if bucket == nil {
			return fmt.Errorf("replay_store: bucket not found")
		}

		key := []byte(jti)
		existing := bucket.Get(key)

		// Check if JTI exists and is not expired
		if existing != nil {
			expiry := int64(binary.BigEndian.Uint64(existing))
			if now < expiry {
				return fmt.Errorf("replay detected: jti=%s already used", jti)
			}
		}

		// Record new JTI with expiration timestamp
		value := make([]byte, 8)
		binary.BigEndian.PutUint64(value, uint64(expiresAt))
		return bucket.Put(key, value)
	})

	return err
}

// cleanupExpired removes expired JTI entries periodically.
// Runs every 5 minutes to maintain database size.
func (s *BoltReplayStore) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()

		_ = s.db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(s.bucketName)
			if bucket == nil {
				return nil
			}

			// Collect expired keys
			var expiredKeys [][]byte
			cursor := bucket.Cursor()

			for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
				if len(v) >= 8 {
					expiry := int64(binary.BigEndian.Uint64(v))
					if now >= expiry {
						expiredKeys = append(expiredKeys, append([]byte(nil), k...))
					}
				}
			}

			// Delete expired entries
			for _, key := range expiredKeys {
				_ = bucket.Delete(key)
			}

			return nil
		})
	}
}

// Close closes the BoltDB database.
func (s *BoltReplayStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Count returns the number of active (non-expired) JTI entries.
// Useful for monitoring and testing.
func (s *BoltReplayStore) Count() (int, error) {
	var count int
	now := time.Now().Unix()

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucketName)
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			if len(v) >= 8 {
				expiry := int64(binary.BigEndian.Uint64(v))
				if now < expiry {
					count++
				}
			}
		}

		return nil
	})

	return count, err
}
