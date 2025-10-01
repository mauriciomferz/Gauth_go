// Package store provides token storage implementations for GAuth
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisConfig contains Redis-specific configuration
type RedisConfig struct {
	// Base configuration
	Config

	// Redis connection options
	Addr     string
	Password string
	DB       int

	// Key prefix for Redis keys
	KeyPrefix string

	// Key expiration multiplier (relative to token expiration)
	// For example, 1.1 means keys will expire 10% after token expiration
	ExpirationMultiplier float64
}

// DefaultRedisConfig returns default Redis configuration
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Config:               DefaultConfig(),
		Addr:                 "localhost:6379",
		DB:                   0,
		KeyPrefix:            "gauth:token:",
		ExpirationMultiplier: 1.1,
	}
}

// RedisStore implements TokenStore using Redis
type RedisStore struct {
	client   *redis.Client
	config   RedisConfig
	stopChan chan struct{}
}

// NewRedisStore creates a new Redis-backed token store
func NewRedisStore(cfg RedisConfig) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, &StorageError{
			Op:     "connect",
			Err:    err,
			Detail: "failed to connect to Redis",
		}
	}

	store := &RedisStore{
		client:   client,
		config:   cfg,
		stopChan: make(chan struct{}),
	}

	// Start cleanup routine if enabled
	if cfg.CleanupInterval > 0 {
		go store.startCleanupRoutine()
	}

	return store, nil
}

// startCleanupRoutine starts a background goroutine to clean up expired tokens
func (s *RedisStore) startCleanupRoutine() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = s.Cleanup(ctx)
			cancel()
		case <-s.stopChan:
			return
		}
	}
}

// Close releases resources and stops background tasks
func (s *RedisStore) Close() error {
	close(s.stopChan)
	return s.client.Close()
}

// tokenKey generates a Redis key for the token
func (s *RedisStore) tokenKey(token string) string {
	return s.config.KeyPrefix + "raw:" + token
}

// idKey generates a Redis key for looking up tokens by ID
func (s *RedisStore) idKey(id string) string {
	return s.config.KeyPrefix + "id:" + id
}

// revocationKey generates a Redis key for token revocation status
func (s *RedisStore) revocationKey(token string) string {
	return s.config.KeyPrefix + "revoked:" + token
}

// subjectKey generates a Redis key for listing tokens by subject
func (s *RedisStore) subjectKey(subject string) string {
	return s.config.KeyPrefix + "subject:" + subject
}

// Store stores a token with its metadata
func (s *RedisStore) Store(ctx context.Context, token string, metadata TokenMetadata) error {
	if token == "" || metadata.ID == "" {
		return &StorageError{
			Op:     "store",
			Key:    token,
			Err:    ErrInvalidMetadata,
			Detail: "token or ID is empty",
		}
	}

	// Set default expiration if not provided
	if metadata.ExpiresAt.IsZero() {
		metadata.ExpiresAt = time.Now().Add(s.config.DefaultTTL)
	}

	// Set default issuedAt if not provided
	if metadata.IssuedAt.IsZero() {
		metadata.IssuedAt = time.Now()
	}

	// Serialize metadata
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return &StorageError{
			Op:     "store",
			Key:    token,
			Err:    err,
			Detail: "failed to serialize metadata",
		}
	}

	// Calculate TTL based on token expiration
	ttl := time.Until(metadata.ExpiresAt)
	if ttl <= 0 {
		return &StorageError{
			Op:     "store",
			Key:    token,
			Err:    ErrTokenExpired,
			Detail: "token is already expired",
		}
	}

	// Apply expiration multiplier
	redisTTL := time.Duration(float64(ttl) * s.config.ExpirationMultiplier)

	// Begin transaction
	pipe := s.client.TxPipeline()

	// Store token metadata
	pipe.Set(ctx, s.tokenKey(token), metadataBytes, redisTTL)

	// Create ID -> token mapping
	pipe.Set(ctx, s.idKey(metadata.ID), token, redisTTL)

	// Add to subject index
	pipe.SAdd(ctx, s.subjectKey(metadata.Subject), token)
	pipe.Expire(ctx, s.subjectKey(metadata.Subject), redisTTL)

	// Execute transaction
	_, err = pipe.Exec(ctx)
	if err != nil {
		return &StorageError{
			Op:     "store",
			Key:    token,
			Err:    err,
			Detail: "Redis transaction failed",
		}
	}

	return nil
}

// Get retrieves token metadata by token string
func (s *RedisStore) Get(ctx context.Context, token string) (*TokenMetadata, error) {
	// Check if token is revoked
	revoked, err := s.client.Exists(ctx, s.revocationKey(token)).Result()
	if err != nil {
		return nil, &StorageError{
			Op:     "get",
			Key:    token,
			Err:    err,
			Detail: "failed to check revocation status",
		}
	}

	if revoked > 0 {
		return nil, &StorageError{
			Op:     "get",
			Key:    token,
			Err:    ErrTokenRevoked,
			Detail: "token has been revoked",
		}
	}

	// Get token metadata
	metadataBytes, err := s.client.Get(ctx, s.tokenKey(token)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, &StorageError{
				Op:     "get",
				Key:    token,
				Err:    ErrTokenNotFound,
				Detail: "token not found in Redis",
			}
		}
		return nil, &StorageError{
			Op:     "get",
			Key:    token,
			Err:    err,
			Detail: "failed to retrieve token from Redis",
		}
	}

	// Deserialize metadata
	var metadata TokenMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, &StorageError{
			Op:     "get",
			Key:    token,
			Err:    err,
			Detail: "failed to deserialize token metadata",
		}
	}

	// Check if token is expired
	if !metadata.ExpiresAt.IsZero() && time.Now().After(metadata.ExpiresAt) {
		return nil, &StorageError{
			Op:     "get",
			Key:    token,
			Err:    ErrTokenExpired,
			Detail: "token expired",
		}
	}

	return &metadata, nil
}

// GetByID retrieves token metadata by token ID
func (s *RedisStore) GetByID(ctx context.Context, id string) (*TokenMetadata, error) {
	// Get token string from ID
	token, err := s.client.Get(ctx, s.idKey(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, &StorageError{
				Op:     "getbyid",
				Key:    id,
				Err:    ErrTokenNotFound,
				Detail: "token ID not found",
			}
		}
		return nil, &StorageError{
			Op:     "getbyid",
			Key:    id,
			Err:    err,
			Detail: "failed to retrieve token by ID",
		}
	}

	// Get token metadata using token string
	return s.Get(ctx, token)
}

// Delete removes a token from storage
func (s *RedisStore) Delete(ctx context.Context, token string) error {
	// Get token metadata first to get the ID
	metadataBytes, err := s.client.Get(ctx, s.tokenKey(token)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &StorageError{
				Op:     "delete",
				Key:    token,
				Err:    ErrTokenNotFound,
				Detail: "cannot delete non-existent token",
			}
		}
		return &StorageError{
			Op:     "delete",
			Key:    token,
			Err:    err,
			Detail: "failed to retrieve token for deletion",
		}
	}

	// Deserialize metadata
	var metadata TokenMetadata
	if unmarshalErr := json.Unmarshal(metadataBytes, &metadata); unmarshalErr != nil {
		return &StorageError{
			Op:     "delete",
			Key:    token,
			Err:    unmarshalErr,
			Detail: "failed to deserialize token metadata",
		}
	}

	// Begin transaction
	pipe := s.client.TxPipeline()

	// Delete token metadata
	pipe.Del(ctx, s.tokenKey(token))

	// Delete ID -> token mapping
	pipe.Del(ctx, s.idKey(metadata.ID))

	// Delete from revocation set if exists
	pipe.Del(ctx, s.revocationKey(token))

	// Remove from subject index
	pipe.SRem(ctx, s.subjectKey(metadata.Subject), token)

	// Execute transaction
	_, err = pipe.Exec(ctx)
	if err != nil {
		return &StorageError{
			Op:     "delete",
			Key:    token,
			Err:    err,
			Detail: "Redis transaction failed during deletion",
		}
	}

	return nil
}

// List returns all tokens for a subject
func (s *RedisStore) List(ctx context.Context, subject string) ([]TokenMetadata, error) {
	// Get token strings from subject index
	tokenStrings, err := s.client.SMembers(ctx, s.subjectKey(subject)).Result()
	if err != nil {
		return nil, &StorageError{
			Op:     "list",
			Key:    subject,
			Err:    err,
			Detail: "failed to retrieve subject token list",
		}
	}

	var tokens []TokenMetadata
	for _, tokenStr := range tokenStrings {
		// Get token metadata and check validity/expiration
		metadata, err := s.Get(ctx, tokenStr)
		if err != nil {
			// Skip expired, revoked, or otherwise invalid tokens
			continue
		}
		tokens = append(tokens, *metadata)
	}

	return tokens, nil
}

// Revoke marks a token as revoked
func (s *RedisStore) Revoke(ctx context.Context, token string) error {
	// Check if token exists
	exists, err := s.client.Exists(ctx, s.tokenKey(token)).Result()
	if err != nil {
		return &StorageError{
			Op:     "revoke",
			Key:    token,
			Err:    err,
			Detail: "failed to check token existence",
		}
	}

	if exists == 0 {
		return &StorageError{
			Op:     "revoke",
			Key:    token,
			Err:    ErrTokenNotFound,
			Detail: "cannot revoke non-existent token",
		}
	}

	// Get token metadata to find expiration
	metadata, err := s.Get(ctx, token)
	if err != nil {
		return &StorageError{
			Op:     "revoke",
			Key:    token,
			Err:    err,
			Detail: "failed to retrieve token metadata for revocation",
		}
	}

	// Calculate TTL based on token expiration
	ttl := time.Until(metadata.ExpiresAt)
	if ttl <= 0 {
		ttl = 24 * time.Hour // Default TTL for already expired tokens
	}

	// Mark token as revoked
	err = s.client.Set(ctx, s.revocationKey(token), time.Now().Format(time.RFC3339), ttl).Err()
	if err != nil {
		return &StorageError{
			Op:     "revoke",
			Key:    token,
			Err:    err,
			Detail: "failed to mark token as revoked",
		}
	}

	return nil
}

// IsRevoked checks if a token is revoked
func (s *RedisStore) IsRevoked(ctx context.Context, token string) (bool, error) {
	// Check if token exists
	exists, err := s.client.Exists(ctx, s.tokenKey(token)).Result()
	if err != nil {
		return false, &StorageError{
			Op:     "isrevoked",
			Key:    token,
			Err:    err,
			Detail: "failed to check token existence",
		}
	}

	if exists == 0 {
		return false, &StorageError{
			Op:     "isrevoked",
			Key:    token,
			Err:    ErrTokenNotFound,
			Detail: "cannot check revocation status of non-existent token",
		}
	}

	// Check if token is in revocation set
	revoked, err := s.client.Exists(ctx, s.revocationKey(token)).Result()
	if err != nil {
		return false, &StorageError{
			Op:     "isrevoked",
			Key:    token,
			Err:    err,
			Detail: "failed to check token revocation",
		}
	}

	return revoked > 0, nil
}

// Cleanup removes expired tokens
// This is less critical in Redis because we use Redis's built-in expiration mechanism,
// but we can use this to perform maintenance operations
func (s *RedisStore) Cleanup(ctx context.Context) error {
	// In Redis implementation, Redis automatically removes expired keys
	// However, we can still perform some maintenance tasks:

	// 1. Scan for subject indexes with no members and remove them
	var cursor uint64
	for {
		var keys []string
		var err error
		pattern := s.config.KeyPrefix + "subject:*"

		keys, cursor, err = s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return &StorageError{
				Op:     "cleanup",
				Err:    err,
				Detail: "failed to scan for empty subject indexes",
			}
		}

		for _, key := range keys {
			count, err := s.client.SCard(ctx, key).Result()
			if err != nil {
				continue // Skip keys that error out
			}

			if count == 0 {
				// Remove empty sets
				s.client.Del(ctx, key)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}
