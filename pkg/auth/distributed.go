package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// DistributedValidatorConfig contains configuration for distributed token validation
type DistributedValidatorConfig struct {
	// Redis client for distributed state
	RedisClient *redis.Client
	// Prefix for Redis keys
	KeyPrefix string
	// TTL for token validation results
	CacheTTL time.Duration
	// Underlying authenticator
	BaseAuthenticator Authenticator
}

// distributedValidator implements distributed token validation
type distributedValidator struct {
	config DistributedValidatorConfig
	cache  sync.Map
}

// NewDistributedValidator creates a new distributed token validator
func NewDistributedValidator(config DistributedValidatorConfig) (Authenticator, error) {
	if config.RedisClient == nil {
		return nil, errors.New("redis client is required")
	}

	if config.BaseAuthenticator == nil {
		return nil, errors.New("base authenticator is required")
	}

	if config.KeyPrefix == "" {
		config.KeyPrefix = "token:"
	}

	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	return &distributedValidator{
		config: config,
	}, nil
}

func (v *distributedValidator) Initialize(ctx context.Context) error {
	return v.config.BaseAuthenticator.Initialize(ctx)
}

func (v *distributedValidator) Close() error {
	return v.config.BaseAuthenticator.Close()
}

func (v *distributedValidator) ValidateCredentials(ctx context.Context, creds interface{}) error {
	return v.config.BaseAuthenticator.ValidateCredentials(ctx, creds)
}

func (v *distributedValidator) GenerateToken(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	token, err := v.config.BaseAuthenticator.GenerateToken(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store token validation info in Redis
	tokenData := &TokenData{
		Valid:     true,
		Subject:   req.Subject,
		Scope:     req.Scopes,
		ExpiresAt: time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
	}

	if err := v.storeValidationData(ctx, token.Token, tokenData); err != nil {
		return nil, fmt.Errorf("failed to store validation data: %w", err)
	}

	return token, nil
}

func (v *distributedValidator) ValidateToken(ctx context.Context, tokenStr string) (*TokenData, error) {
	// Check local cache first
	if cached, ok := v.cache.Load(tokenStr); ok {
		data := cached.(*TokenData)
		if time.Now().After(data.ExpiresAt) {
			v.cache.Delete(tokenStr)
		} else {
			return data, nil
		}
	}

	// Check Redis cache
	data, err := v.getValidationData(ctx, tokenStr)
	if err == nil {
		// Cache the validation result locally
		v.cache.Store(tokenStr, data)
		return data, nil
	}

	// If not in cache, validate using base authenticator
	data, err = v.config.BaseAuthenticator.ValidateToken(ctx, tokenStr)
	if err != nil {
		return nil, err
	}

	// Store validation result in Redis
	if err := v.storeValidationData(ctx, tokenStr, data); err != nil {
		return nil, fmt.Errorf("failed to store validation data: %w", err)
	}

	// Cache locally
	v.cache.Store(tokenStr, data)
	return data, nil
}

func (v *distributedValidator) RevokeToken(ctx context.Context, tokenStr string) error {
	// Remove from local cache
	v.cache.Delete(tokenStr)

	// Remove from Redis
	key := v.config.KeyPrefix + tokenStr
	if err := v.config.RedisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove token from Redis: %w", err)
	}

	return v.config.BaseAuthenticator.RevokeToken(ctx, tokenStr)
}

func (v *distributedValidator) storeValidationData(ctx context.Context, tokenStr string, data *TokenData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	key := v.config.KeyPrefix + tokenStr
	ttl := time.Until(data.ExpiresAt)
	if ttl > v.config.CacheTTL {
		ttl = v.config.CacheTTL
	}

	if err := v.config.RedisClient.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store token data in Redis: %w", err)
	}

	return nil
}

func (v *distributedValidator) getValidationData(ctx context.Context, tokenStr string) (*TokenData, error) {
	key := v.config.KeyPrefix + tokenStr
	jsonData, err := v.config.RedisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get token data from Redis: %w", err)
	}

	var data TokenData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	if time.Now().After(data.ExpiresAt) {
		v.config.RedisClient.Del(ctx, key)
		return nil, ErrTokenExpired
	}

	return &data, nil
}
