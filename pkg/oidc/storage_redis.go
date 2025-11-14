package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStorage implements StorageBackend using Redis.
type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage creates a new Redis storage backend.
// addr format: "localhost:6379"
func NewRedisStorage(addr string, password string, db int) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &RedisStorage{client: client}, nil
}

// RefreshToken operations

func (s *RedisStorage) StoreRefreshToken(ctx context.Context, token *RefreshTokenEntry) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal refresh token: %w", err)
	}

	key := fmt.Sprintf("refresh_token:%s", token.RefreshToken)
	ttl := time.Until(token.ExpiresAt)

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Index by user
	userKey := fmt.Sprintf("user_tokens:%s", token.Subject)
	if err := s.client.SAdd(ctx, userKey, token.RefreshToken).Err(); err != nil {
		return fmt.Errorf("failed to index by user: %w", err)
	}
	s.client.Expire(ctx, userKey, ttl)

	// Index by client
	clientKey := fmt.Sprintf("client_tokens:%s", token.ProviderID)
	if err := s.client.SAdd(ctx, clientKey, token.RefreshToken).Err(); err != nil {
		return fmt.Errorf("failed to index by client: %w", err)
	}
	s.client.Expire(ctx, clientKey, ttl)

	return nil
}

func (s *RedisStorage) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenEntry, error) {
	key := fmt.Sprintf("refresh_token:%s", tokenHash)
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	var entry RefreshTokenEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh token: %w", err)
	}

	return &entry, nil
}

func (s *RedisStorage) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	// Get token first to clean up indices
	token, err := s.GetRefreshToken(ctx, tokenHash)
	if err != nil && err != ErrTokenNotFound {
		return err
	}

	if token != nil {
		// Remove from user index
		userKey := fmt.Sprintf("user_tokens:%s", token.Subject)
		s.client.SRem(ctx, userKey, tokenHash)

		// Remove from client index
		clientKey := fmt.Sprintf("client_tokens:%s", token.ProviderID)
		s.client.SRem(ctx, clientKey, tokenHash)
	}

	key := fmt.Sprintf("refresh_token:%s", tokenHash)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	return nil
}

func (s *RedisStorage) ListRefreshTokensByUser(ctx context.Context, userID string) ([]*RefreshTokenEntry, error) {
	userKey := fmt.Sprintf("user_tokens:%s", userID)
	tokenHashes, err := s.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list user tokens: %w", err)
	}

	var tokens []*RefreshTokenEntry
	for _, hash := range tokenHashes {
		token, err := s.GetRefreshToken(ctx, hash)
		if err == ErrTokenNotFound {
			// Token expired, remove from index
			s.client.SRem(ctx, userKey, hash)
			continue
		}
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (s *RedisStorage) ListRefreshTokensByClient(ctx context.Context, clientID string) ([]*RefreshTokenEntry, error) {
	clientKey := fmt.Sprintf("client_tokens:%s", clientID)
	tokenHashes, err := s.client.SMembers(ctx, clientKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list client tokens: %w", err)
	}

	var tokens []*RefreshTokenEntry
	for _, hash := range tokenHashes {
		token, err := s.GetRefreshToken(ctx, hash)
		if err == ErrTokenNotFound {
			// Token expired, remove from index
			s.client.SRem(ctx, clientKey, hash)
			continue
		}
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (s *RedisStorage) CleanupExpiredRefreshTokens(ctx context.Context) (int, error) {
	// Redis automatically expires keys based on TTL, so this is a no-op
	return 0, nil
}

// Revoked token operations

func (s *RedisStorage) StoreRevokedToken(ctx context.Context, entry *RevokedTokenEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal revoked token: %w", err)
	}

	key := fmt.Sprintf("revoked_token:%s", entry.TokenID)
	ttl := time.Until(entry.ExpiresAt)

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store revoked token: %w", err)
	}

	return nil
}

func (s *RedisStorage) IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	key := fmt.Sprintf("revoked_token:%s", tokenHash)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check revocation: %w", err)
	}
	return exists > 0, nil
}

func (s *RedisStorage) CleanupExpiredRevocations(ctx context.Context) (int, error) {
	// Redis automatically expires keys based on TTL, so this is a no-op
	return 0, nil
}

// Device code operations

func (s *RedisStorage) StoreDeviceCode(ctx context.Context, entry *DeviceCodeEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal device code: %w", err)
	}

	deviceKey := fmt.Sprintf("device_code:%s", entry.DeviceCode)
	userKey := fmt.Sprintf("user_code:%s", entry.UserCode)
	ttl := time.Until(entry.ExpiresAt)

	// Store by device code
	if err := s.client.Set(ctx, deviceKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store device code: %w", err)
	}

	// Store mapping from user code to device code
	if err := s.client.Set(ctx, userKey, entry.DeviceCode, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store user code mapping: %w", err)
	}

	return nil
}

func (s *RedisStorage) GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*DeviceCodeEntry, error) {
	key := fmt.Sprintf("device_code:%s", deviceCode)
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrDeviceCodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device code: %w", err)
	}

	var entry DeviceCodeEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device code: %w", err)
	}

	return &entry, nil
}

func (s *RedisStorage) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*DeviceCodeEntry, error) {
	// Get device code from user code mapping
	userKey := fmt.Sprintf("user_code:%s", userCode)
	deviceCode, err := s.client.Get(ctx, userKey).Result()
	if err == redis.Nil {
		return nil, ErrUserCodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user code mapping: %w", err)
	}

	return s.GetDeviceCodeByDeviceCode(ctx, deviceCode)
}

func (s *RedisStorage) UpdateDeviceCodeStatus(ctx context.Context, deviceCode string, entry *DeviceCodeEntry) error {
	// Check if exists
	key := fmt.Sprintf("device_code:%s", deviceCode)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check device code existence: %w", err)
	}
	if exists == 0 {
		return ErrDeviceCodeNotFound
	}

	// Store updated entry
	return s.StoreDeviceCode(ctx, entry)
}

func (s *RedisStorage) DeleteDeviceCode(ctx context.Context, deviceCode string) error {
	// Get entry to find user code
	entry, err := s.GetDeviceCodeByDeviceCode(ctx, deviceCode)
	if err != nil && err != ErrDeviceCodeNotFound {
		return err
	}

	// Delete device code
	deviceKey := fmt.Sprintf("device_code:%s", deviceCode)
	if err := s.client.Del(ctx, deviceKey).Err(); err != nil {
		return fmt.Errorf("failed to delete device code: %w", err)
	}

	// Delete user code mapping if we found the entry
	if entry != nil {
		userKey := fmt.Sprintf("user_code:%s", entry.UserCode)
		s.client.Del(ctx, userKey)
	}

	return nil
}

func (s *RedisStorage) CleanupExpiredDeviceCodes(ctx context.Context) (int, error) {
	// Redis automatically expires keys based on TTL, so this is a no-op
	return 0, nil
}

// PAR request URI operations

func (s *RedisStorage) StorePARRequest(ctx context.Context, requestURI string, entry *RequestURIEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal PAR request: %w", err)
	}

	key := fmt.Sprintf("par_request:%s", requestURI)
	ttl := time.Until(entry.ExpiresAt)

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store PAR request: %w", err)
	}

	return nil
}

func (s *RedisStorage) GetPARRequest(ctx context.Context, requestURI string) (*RequestURIEntry, error) {
	key := fmt.Sprintf("par_request:%s", requestURI)
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrRequestURINotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get PAR request: %w", err)
	}

	var entry RequestURIEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PAR request: %w", err)
	}

	return &entry, nil
}

func (s *RedisStorage) DeletePARRequest(ctx context.Context, requestURI string) error {
	key := fmt.Sprintf("par_request:%s", requestURI)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete PAR request: %w", err)
	}
	return nil
}

func (s *RedisStorage) MarkPARRequestUsed(ctx context.Context, requestURI string) error {
	entry, err := s.GetPARRequest(ctx, requestURI)
	if err != nil {
		return err
	}

	entry.Used = true
	entry.UsedAt = time.Now()

	return s.StorePARRequest(ctx, requestURI, entry)
}

func (s *RedisStorage) CleanupExpiredPARRequests(ctx context.Context) (int, error) {
	// Redis automatically expires keys based on TTL, so this is a no-op
	return 0, nil
}

// Health check

func (s *RedisStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *RedisStorage) Close() error {
	return s.client.Close()
}
