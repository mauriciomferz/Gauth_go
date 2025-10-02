// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package auth

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// Additional error codes for system management
const (
	ErrMemoryLimit      ErrorCode = "MEMORY_LIMIT"
	ErrConfigValidation ErrorCode = "CONFIG_VALIDATION"
	ErrCleanupFailed    ErrorCode = "CLEANUP_FAILED"
)

// MemoryManager handles memory cleanup and leak prevention
type MemoryManager struct {
	mu              sync.RWMutex
	tokenStore      map[string]*TokenEntry
	cleanupInterval time.Duration
	maxEntries      int
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// TokenEntry represents a stored token with metadata
type TokenEntry struct {
	Token        *token.Token
	CreatedAt    time.Time
	LastAccessed time.Time
	AccessCount  int64
}

// NewMemoryManager creates a memory manager with cleanup
func NewMemoryManager(cleanupInterval time.Duration, maxEntries int) *MemoryManager {
	mm := &MemoryManager{
		tokenStore:      make(map[string]*TokenEntry),
		cleanupInterval: cleanupInterval,
		maxEntries:      maxEntries,
		stopCh:          make(chan struct{}),
	}

	// Start cleanup goroutine
	mm.wg.Add(1)
	go mm.cleanupLoop()

	return mm
}

// StoreToken stores a token with automatic cleanup
func (mm *MemoryManager) StoreToken(ctx context.Context, tok *token.Token) error {
	if tok == nil {
		return NewError(ErrInvalidDocument, "token cannot be nil", nil)
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Check capacity
	if len(mm.tokenStore) >= mm.maxEntries {
		if err := mm.evictOldestEntry(); err != nil {
			return NewError(ErrMemoryLimit, "failed to evict old entries", err)
		}
	}

	entry := &TokenEntry{
		Token:        tok,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  0,
	}

	mm.tokenStore[tok.ID] = entry
	return nil
}

// GetToken retrieves a token and updates access metadata
func (mm *MemoryManager) GetToken(ctx context.Context, tokenID string) (*token.Token, error) {
	if tokenID == "" {
		return nil, NewError(ErrInvalidDocument, "token ID cannot be empty", nil)
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	entry, exists := mm.tokenStore[tokenID]
	if !exists {
		return nil, NewError(ErrNotAuthorized, "token not found", nil)
	}

	// Update access metadata
	entry.LastAccessed = time.Now()
	entry.AccessCount++

	// Check expiration
	if entry.Token.ExpiresAt.Before(time.Now()) {
		delete(mm.tokenStore, tokenID)
		return nil, NewError(ErrAuthorizationExpired, "token expired", nil)
	}

	return entry.Token, nil
}

// RemoveToken removes a token from storage
func (mm *MemoryManager) RemoveToken(ctx context.Context, tokenID string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.tokenStore[tokenID]; !exists {
		return NewError(ErrInvalidDocument, "token not found", nil)
	}

	delete(mm.tokenStore, tokenID)
	return nil
}

// cleanupLoop runs periodic cleanup
func (mm *MemoryManager) cleanupLoop() {
	defer mm.wg.Done()

	ticker := time.NewTicker(mm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mm.performCleanup()
		case <-mm.stopCh:
			return
		}
	}
}

// performCleanup removes expired and unused tokens
func (mm *MemoryManager) performCleanup() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	now := time.Now()
	expiredCount := 0
	unusedCount := 0

	for tokenID, entry := range mm.tokenStore {
		// Remove expired tokens
		if entry.Token.ExpiresAt.Before(now) {
			delete(mm.tokenStore, tokenID)
			expiredCount++
			continue
		}

		// Remove unused tokens (not accessed in 24 hours)
		if now.Sub(entry.LastAccessed) > 24*time.Hour && entry.AccessCount == 0 {
			delete(mm.tokenStore, tokenID)
			unusedCount++
		}
	}

	// Force garbage collection if significant cleanup occurred
	if expiredCount+unusedCount > 100 {
		runtime.GC()
	}
}

// evictOldestEntry removes the oldest entry to make space
func (mm *MemoryManager) evictOldestEntry() error {
	if len(mm.tokenStore) == 0 {
		return nil
	}

	var oldestID string
	var oldestTime time.Time

	for tokenID, entry := range mm.tokenStore {
		if oldestID == "" || entry.CreatedAt.Before(oldestTime) {
			oldestID = tokenID
			oldestTime = entry.CreatedAt
		}
	}

	delete(mm.tokenStore, oldestID)
	return nil
}

// GetStats returns memory manager statistics
func (mm *MemoryManager) GetStats() map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var totalAccess int64
	expiredCount := 0
	now := time.Now()

	for _, entry := range mm.tokenStore {
		totalAccess += entry.AccessCount
		if entry.Token.ExpiresAt.Before(now) {
			expiredCount++
		}
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"total_tokens":   len(mm.tokenStore),
		"expired_tokens": expiredCount,
		"total_accesses": totalAccess,
		"memory_alloc":   memStats.Alloc,
		"memory_sys":     memStats.Sys,
		"num_gc":         memStats.NumGC,
	}
}

// Shutdown gracefully shuts down the memory manager
func (mm *MemoryManager) Shutdown(ctx context.Context) error {
	close(mm.stopCh)

	// Wait for cleanup goroutine to stop
	done := make(chan struct{})
	go func() {
		mm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Final cleanup
		mm.mu.Lock()
		mm.tokenStore = make(map[string]*TokenEntry)
		mm.mu.Unlock()
		runtime.GC()
		return nil
	case <-ctx.Done():
		return NewError(ErrCleanupFailed, "shutdown timeout", ctx.Err())
	}
}

// ConfigValidator validates system configuration
type ConfigValidator struct {
	requiredFields map[string]bool
	validators     map[string]func(interface{}) error
}

// NewConfigValidator creates a configuration validator
func NewConfigValidator() *ConfigValidator {
	cv := &ConfigValidator{
		requiredFields: map[string]bool{
			"auth_server_url":      true,
			"token_expiry":         true,
			"rate_limit":           true,
			"max_delegation_depth": true,
		},
		validators: make(map[string]func(interface{}) error),
	}

	// Add field validators
	cv.validators["auth_server_url"] = cv.validateURL
	cv.validators["token_expiry"] = cv.validateDuration
	cv.validators["rate_limit"] = cv.validatePositiveInt
	cv.validators["max_delegation_depth"] = cv.validatePositiveInt

	return cv
}

// ValidateConfig validates system configuration
func (cv *ConfigValidator) ValidateConfig(config map[string]interface{}) error {
	if config == nil {
		return NewError(ErrConfigValidation, "configuration cannot be empty", nil)
	}

	// Check required fields
	for field := range cv.requiredFields {
		if _, exists := config[field]; !exists {
			return NewError(ErrConfigValidation,
				fmt.Sprintf("required field missing: %s", field), nil)
		}
	}

	// Validate each field
	for field, value := range config {
		if validator, exists := cv.validators[field]; exists {
			if err := validator(value); err != nil {
				return NewError(ErrConfigValidation,
					fmt.Sprintf("invalid %s", field), err)
			}
		}
	}

	return nil
}

// validateURL validates URL format
func (cv *ConfigValidator) validateURL(value interface{}) error {
	url, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be string")
	}
	if url == "" {
		return fmt.Errorf("cannot be empty")
	}
	// Additional URL validation would go here
	return nil
}

// validateDuration validates duration values
func (cv *ConfigValidator) validateDuration(value interface{}) error {
	duration, ok := value.(time.Duration)
	if !ok {
		return fmt.Errorf("must be duration")
	}
	if duration <= 0 {
		return fmt.Errorf("must be positive")
	}
	return nil
}

// validatePositiveInt validates positive integer values
func (cv *ConfigValidator) validatePositiveInt(value interface{}) error {
	num, ok := value.(int)
	if !ok {
		return fmt.Errorf("must be integer")
	}
	if num <= 0 {
		return fmt.Errorf("must be positive")
	}
	return nil
}
