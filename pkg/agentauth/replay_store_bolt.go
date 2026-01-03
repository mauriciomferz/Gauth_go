// Copyright 2025 AgentAuth Contributors
// SPDX-License-Identifier: Apache-2.0

package agentauth

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/security"
	bolt "go.etcd.io/bbolt"
)

// BoltReplayStore implements durable replay detection using BoltDB.
// Addresses gap sec6.item1 (P1): Durable replay persistence with eviction controls.
//
// SECURITY WARNING (V-2025-005): BoltDB is UNSAFE for containerized deployments
// due to ephemeral storage vulnerability. See SECURITY_AUDIT_CRITICAL_REVIEW.md.
//
// DEPRECATED for production use in containers. Use Redis or other distributed
// store for production deployments. See REPLAY_STORE_MIGRATION_GUIDE.md.
type BoltReplayStore struct {
	db         *bolt.DB
	bucketName []byte
	ttl        time.Duration
	quit       chan struct{}
	wg         sync.WaitGroup
}

// NewBoltReplayStore creates a new BoltDB-backed replay store.
// The path parameter specifies where the database file should be created.
// TTL determines how long JTI entries are retained before expiration.
//
// SECURITY: This function performs container environment detection. If running
// in a containerized environment (Docker, Kubernetes, Podman) with ephemeral
// storage paths (/tmp, /var/tmp, emptyDir), it will FAIL with a detailed error
// message explaining the security vulnerability and remediation options.
//
// To bypass this check (NOT RECOMMENDED), set AGENTAUTH_ALLOW_UNSAFE_BOLTDB=1.
// This should ONLY be used for development/testing, never in production.
func NewBoltReplayStore(path string, ttl time.Duration) (*BoltReplayStore, error) {
	// SECURITY CHECK: Validate path is safe for persistent storage in containers
	// This prevents CV-2025-005 vulnerability (ephemeral storage replay bypass)
	if security.ShouldEnforceContainerSafety() {
		// Allow bypass for development/testing ONLY
		if os.Getenv("AGENTAUTH_ALLOW_UNSAFE_BOLTDB") != "1" {
			if err := security.ValidatePathForPersistence(path, "replay protection"); err != nil {
				return nil, fmt.Errorf("BoltDB SECURITY VIOLATION (CV-2025-005): %w - "+
					"BoltDB is DEPRECATED for production use in containers, "+
					"use Redis (recommended) or PostgreSQL for distributed replay protection, "+
					"for development/testing ONLY, set AGENTAUTH_ALLOW_UNSAFE_BOLTDB=1 to bypass, "+
					"see REPLAY_STORE_MIGRATION_GUIDE.md for migration instructions", err)
			}
		} else {
			// Log warning when bypass is used
			fmt.Fprintf(os.Stderr, "[SECURITY WARNING] AGENTAUTH_ALLOW_UNSAFE_BOLTDB=1 detected\n")
			fmt.Fprintf(os.Stderr, "[SECURITY WARNING] BoltDB container safety checks BYPASSED\n")
			fmt.Fprintf(os.Stderr, "[SECURITY WARNING] %s\n", security.GetContainerInfo())
			fmt.Fprintf(os.Stderr, "[SECURITY WARNING] Path: %s (may be ephemeral)\n", path)
			fmt.Fprintf(os.Stderr, "[SECURITY WARNING] This is UNSAFE for production use!\n")
		}
	}

	// Ensure directory exists with restricted permissions (0750 instead of 0755)
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
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
		_, err2 := tx.CreateBucketIfNotExists(bucketName)
		return err2
	})
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "replay_store: failed to close db after error: %v\n", closeErr)
		}
		return nil, fmt.Errorf("replay_store: create bucket: %w", err)
	}

	store := &BoltReplayStore{
		db:         db,
		bucketName: bucketName,
		ttl:        ttl,
		quit:       make(chan struct{}),
	}

	// Start background cleanup goroutine
	store.wg.Add(1)
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
			// Check if JTI exists and is not expired
			expiry := int64(binary.BigEndian.Uint64(existing)) // #nosec G115: conversion safe for Unix timestamp
			if now < expiry {
				return fmt.Errorf("replay detected: jti=%s already used", jti)
			}
		}

		// Record new JTI with expiration timestamp
		value := make([]byte, 8)
		// #nosec G115: timestamp conversion, safe for Unix time values
		binary.BigEndian.PutUint64(value, uint64(expiresAt))
		return bucket.Put(key, value)
	})

	return err
}

// cleanupExpired removes expired JTI entries periodically.
// Runs every 5 minutes to maintain database size.
func (s *BoltReplayStore) cleanupExpired() {
	defer s.wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Ignore error in background task
			_, _ = s.Cleanup(context.Background())
		case <-s.quit:
			return
		}
	}
}

// Cleanup manually triggers expiration of stale entries.
// Returns the number of evicted items.
// Exposed for maintenance and testing (Bonus 5).
func (s *BoltReplayStore) Cleanup(ctx context.Context) (int, error) {
	var evictedCount int
	now := time.Now().Unix()

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucketName)
		if bucket == nil {
			return nil
		}

		// Collect expired keys
		var expiredKeys [][]byte
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			// Check for cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if len(v) >= 8 {
				// G115 fix: Validate timestamp boundary before uint64→int64 conversion
				expiryUint := binary.BigEndian.Uint64(v)
				if expiryUint > math.MaxInt64 {
					// Timestamp beyond int64 max (year 2262+), treat as far future
					continue
				}
				expiry := int64(expiryUint)
				if now >= expiry {
					expiredKeys = append(expiredKeys, append([]byte(nil), k...))
				}
			}
		}

		// Delete expired entries
		for _, key := range expiredKeys {
			if err := bucket.Delete(key); err == nil {
				evictedCount++
			}
		}

		return nil
	})

	return evictedCount, err
}

// Close closes the BoltDB database and stops background tasks.
func (s *BoltReplayStore) Close() error {
	// Signal background routines to stop
	close(s.quit)
	// Wait for them to finish
	s.wg.Wait()

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
				// G115 fix: Validate timestamp boundary before uint64→int64 conversion
				expiryUint := binary.BigEndian.Uint64(v)
				if expiryUint > math.MaxInt64 {
					// Timestamp beyond int64 max (year 2262+), treat as not expired
					count++
					continue
				}
				expiry := int64(expiryUint)
				if now < expiry {
					count++
				}
			}
		}

		return nil
	})

	return count, err
}
