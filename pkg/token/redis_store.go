//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package token

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStore implements the Store interface using Redis
type RedisStore struct {
	client     *redis.Client
	keyPrefix  string
	defaultTTL time.Duration
}

// RedisConfig holds configuration for Redis token store
type RedisConfig struct {
	// Addresses of Redis servers
	Addresses []string

	// Password for Redis authentication
	Password string

	// DB number to use
	DB int

	// KeyPrefix for Redis keys
	KeyPrefix string

	// DefaultTTL for tokens
	DefaultTTL time.Duration

	// MaxRetries for operations
	MaxRetries int

	// MinRetryBackoff for retry delays
	MinRetryBackoff time.Duration

	// MaxRetryBackoff for retry delays
	MaxRetryBackoff time.Duration
}

// NewRedisStore creates a new Redis-backed token store
func NewRedisStore(cfg RedisConfig) (*RedisStore, error) {
	if len(cfg.Addresses) == 0 {
		return nil, fmt.Errorf("%w: no Redis addresses provided", ErrInvalidConfig)
	}

	client := redis.NewClient(&redis.Options{
		Addr:            cfg.Addresses[0], // TODO: Support cluster
		Password:        cfg.Password,
		DB:              cfg.DB,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	})

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client:     client,
		keyPrefix:  cfg.KeyPrefix,
		defaultTTL: cfg.DefaultTTL,
	}, nil
}

// Save implements the Store interface
func (s *RedisStore) Save(ctx context.Context, token *Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal token: %v", ErrStorageFailure, err)
	}

	// Store by ID
	key := s.key(token.ID)
	if err := s.client.Set(ctx, key, data, s.ttl(token)).Err(); err != nil {
		return fmt.Errorf("%w: failed to save token: %v", ErrStorageFailure, err)
	}

	// Store value->ID mapping for lookups
	if token.Value != "" {
		valueKey := s.valueKey(token.Value)
		if err := s.client.Set(ctx, valueKey, token.ID, s.ttl(token)).Err(); err != nil {
			return fmt.Errorf("%w: failed to save token mapping: %v", ErrStorageFailure, err)
		}
	}

	return nil
}

// Get implements the Store interface
func (s *RedisStore) Get(ctx context.Context, id string) (*Token, error) {
	data, err := s.client.Get(ctx, s.key(id)).Bytes()
	if err == redis.Nil {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get token: %v", ErrStorageFailure, err)
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal token: %v", ErrStorageFailure, err)
	}

	return &token, nil
}

// GetByValue implements the Store interface
func (s *RedisStore) GetByValue(ctx context.Context, value string) (*Token, error) {
	// Get ID from value mapping
	id, err := s.client.Get(ctx, s.valueKey(value)).Result()
	if err == redis.Nil {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get token mapping: %v", ErrStorageFailure, err)
	}

	return s.Get(ctx, id)
}

// Delete implements the Store interface
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	// Get token to remove value mapping
	token, err := s.Get(ctx, id)
	if err != nil && err != ErrTokenNotFound {
		return err
	}

	// Remove ID->token mapping
	if err := s.client.Del(ctx, s.key(id)).Err(); err != nil {
		return fmt.Errorf("%w: failed to delete token: %v", ErrStorageFailure, err)
	}

	// Remove value->ID mapping if token exists
	if token != nil && token.Value != "" {
		if err := s.client.Del(ctx, s.valueKey(token.Value)).Err(); err != nil {
			return fmt.Errorf("%w: failed to delete token mapping: %v", ErrStorageFailure, err)
		}
	}

	return nil
}

// DeleteByValue implements the Store interface
func (s *RedisStore) DeleteByValue(ctx context.Context, value string) error {
	// Get ID from value mapping
	id, err := s.client.Get(ctx, s.valueKey(value)).Result()
	if err == redis.Nil {
		return ErrTokenNotFound
	}
	if err != nil {
		return fmt.Errorf("%w: failed to get token mapping: %v", ErrStorageFailure, err)
	}

	return s.Delete(ctx, id)
}

// List implements the Store interface
func (s *RedisStore) List(ctx context.Context, filter Filter) ([]*Token, error) {
	// Scan for all token keys
	pattern := s.key("*")
	var tokens []*Token
	var cursor uint64

	for {
		var keys []string
		var err error
		keys, cursor, err = s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("%w: failed to scan tokens: %v", ErrStorageFailure, err)
		}

		// Get tokens in parallel
		pipe := s.client.Pipeline()
		cmds := make([]*redis.StringCmd, len(keys))
		for i, key := range keys {
			cmds[i] = pipe.Get(ctx, key)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return nil, fmt.Errorf("%w: failed to get tokens: %v", ErrStorageFailure, err)
		}

		// Process results
		for _, cmd := range cmds {
			data, err := cmd.Bytes()
			if err != nil {
				continue // Skip failed tokens
			}

			var token Token
			if err := json.Unmarshal(data, &token); err != nil {
				continue // Skip invalid tokens
			}

			if s.matchesFilter(&token, filter) {
				tokens = append(tokens, &token)
			}
		}

		if cursor == 0 {
			break
		}
	}

	return tokens, nil
}

// Revoke implements the Store interface
func (s *RedisStore) Revoke(ctx context.Context, id string, reason string) error {
	token, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	token.RevocationStatus = &RevocationStatus{
		RevokedAt: time.Now(),
		Reason:    reason,
	}

	return s.Save(ctx, token)
}

// Close releases resources used by the store
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Helper methods

func (s *RedisStore) key(id string) string {
	return fmt.Sprintf("%stoken:%s", s.keyPrefix, id)
}

func (s *RedisStore) valueKey(value string) string {
	return fmt.Sprintf("%svalue:%s", s.keyPrefix, value)
}

func (s *RedisStore) ttl(token *Token) time.Duration {
	if token.ExpiresAt.IsZero() {
		return s.defaultTTL
	}
	return time.Until(token.ExpiresAt)
}

func (s *RedisStore) matchesFilter(token *Token, filter Filter) bool {
	// Subject
	if filter.Subject != "" && token.Subject != filter.Subject {
		return false
	}
	// Types
	if len(filter.Types) > 0 {
		found := false
		for _, t := range filter.Types {
			if token.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	// Issuer
	if filter.Issuer != "" && token.Issuer != filter.Issuer {
		return false
	}
	// ExpiresAfter/ExpiresBefore
	if !filter.ExpiresAfter.IsZero() && token.ExpiresAt.Before(filter.ExpiresAfter) {
		return false
	}
	if !filter.ExpiresBefore.IsZero() && token.ExpiresAt.After(filter.ExpiresBefore) {
		return false
	}
	// IssuedAfter/IssuedBefore
	if !filter.IssuedAfter.IsZero() && token.IssuedAt.Before(filter.IssuedAfter) {
		return false
	}
	if !filter.IssuedBefore.IsZero() && token.IssuedAt.After(filter.IssuedBefore) {
		return false
	}
	// Scopes
	if len(filter.Scopes) > 0 {
		if filter.RequireAllScopes {
			for _, required := range filter.Scopes {
				found := false
				for _, scope := range token.Scopes {
					if scope == required {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		} else {
			found := false
			for _, required := range filter.Scopes {
				for _, scope := range token.Scopes {
					if scope == required {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	// Active
	if filter.Active {
		now := time.Now()
		if now.After(token.ExpiresAt) || now.Before(token.NotBefore) {
			return false
		}
	}
	// Metadata (skipped: implement if needed)
	return true
}
