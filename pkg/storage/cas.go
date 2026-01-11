package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// EvidenceStore defines the interface for content-addressable storage of evidence.
type EvidenceStore interface {
	// Put stores the data and returns its content-based hash (SHA-256).
	// Ideally idempotent (storing same data twice returns same hash).
	Put(ctx context.Context, data []byte) (hash string, err error)

	// Get retrieves data by its hash.
	Get(ctx context.Context, hash string) (data []byte, err error)

	// VerifyIntegrity checks if the data stored under 'hash' actually matches that hash.
	// Returns nil if integrity is intact, error otherwise.
	VerifyIntegrity(ctx context.Context, hash string) error
}

// LocalCAS implements EvidenceStore using the local filesystem.
type LocalCAS struct {
	rootDir string
	mu      sync.RWMutex
}

// NewLocalCAS creates a new LocalCAS instance using the specified directory.
func NewLocalCAS(rootDir string) (*LocalCAS, error) {
	if rootDir == "" {
		return nil, errors.New("rootDir is required")
	}
	// #nosec G301
	if err := os.MkdirAll(rootDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create storage root: %w", err)
	}
	return &LocalCAS{rootDir: rootDir}, nil
}

// Put stores data and returns the SHA-256 hash.
func (s *LocalCAS) Put(ctx context.Context, data []byte) (string, error) {
	if data == nil {
		return "", errors.New("data cannot be nil")
	}

	// Compute hash
	h := sha256.New()
	h.Write(data)
	hash := hex.EncodeToString(h.Sum(nil))

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.rootDir, hash)

	// Check if already exists (deduplication)
	if _, err := os.Stat(path); err == nil {
		return hash, nil
	}

	// Write to temporary file first to ensure atomicity
	tmpPattern := fmt.Sprintf("tmp-%s-*", hash)
	f, err := os.CreateTemp(s.rootDir, tmpPattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := f.Name()

	// Ensure cleanup if rename fails
	defer func() {
		_ = f.Close()
		// If rename didn't happen (file still exists at tmp path), remove it
		if _, err := os.Stat(tmpName); err == nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := f.Write(data); err != nil {
		return "", fmt.Errorf("failed to write data: %w", err)
	}
	if err := f.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync data: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Rename to final path
	if err := os.Rename(tmpName, path); err != nil {
		return "", fmt.Errorf("failed to rename to final path: %w", err)
	}

	// Set read-only permissions to discourage tampering (best-effort hardening).
	_ = os.Chmod(path, 0o400)

	return hash, nil
}

// Get retrieves data by hash.
func (s *LocalCAS) Get(ctx context.Context, hash string) ([]byte, error) {
	if len(hash) != 64 {
		return nil, errors.New("invalid hash format")
	}
	// Check for non-hex characters
	if _, err := hex.DecodeString(hash); err != nil {
		return nil, errors.New("invalid hash format")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.rootDir, hash)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("evidence not found: %s", hash)
		}
		return nil, fmt.Errorf("failed to read evidence: %w", err)
	}

	return data, nil
}

// VerifyIntegrity re-computes the hash of the file on disk.
func (s *LocalCAS) VerifyIntegrity(ctx context.Context, hash string) error {
	data, err := s.Get(ctx, hash)
	if err != nil {
		return err
	}

	h := sha256.New()
	h.Write(data)
	computedHash := hex.EncodeToString(h.Sum(nil))

	if computedHash != hash {
		return fmt.Errorf("integrity failure: expected %s, got %s", hash, computedHash)
	}

	return nil
}
