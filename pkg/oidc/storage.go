package oidc

import (
	"context"
	"sync"
	"time"
)

// StorageBackend defines the interface for persistent storage implementations.
// Implementations can use PostgreSQL, MySQL, Redis, or other storage systems.
type StorageBackend interface {
	// RefreshToken operations
	StoreRefreshToken(ctx context.Context, token *RefreshTokenEntry) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenEntry, error)
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
	ListRefreshTokensByUser(ctx context.Context, userID string) ([]*RefreshTokenEntry, error)
	ListRefreshTokensByClient(ctx context.Context, clientID string) ([]*RefreshTokenEntry, error)
	CleanupExpiredRefreshTokens(ctx context.Context) (int, error)

	// Revoked token operations
	StoreRevokedToken(ctx context.Context, entry *RevokedTokenEntry) error
	IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error)
	CleanupExpiredRevocations(ctx context.Context) (int, error)

	// Device code operations
	StoreDeviceCode(ctx context.Context, entry *DeviceCodeEntry) error
	GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*DeviceCodeEntry, error)
	GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*DeviceCodeEntry, error)
	UpdateDeviceCodeStatus(ctx context.Context, deviceCode string, entry *DeviceCodeEntry) error
	DeleteDeviceCode(ctx context.Context, deviceCode string) error
	CleanupExpiredDeviceCodes(ctx context.Context) (int, error)

	// PAR request URI operations
	StorePARRequest(ctx context.Context, requestURI string, entry *RequestURIEntry) error
	GetPARRequest(ctx context.Context, requestURI string) (*RequestURIEntry, error)
	DeletePARRequest(ctx context.Context, requestURI string) error
	MarkPARRequestUsed(ctx context.Context, requestURI string) error
	CleanupExpiredPARRequests(ctx context.Context) (int, error)

	// Health check
	Ping(ctx context.Context) error

	// Close connections
	Close() error
}

// InMemoryStorage implements StorageBackend using in-memory maps.
// Suitable for development and testing. Not recommended for production.
type InMemoryStorage struct {
	refreshTokens map[string]*RefreshTokenEntry
	revokedTokens map[string]*RevokedTokenEntry
	deviceCodes   map[string]*DeviceCodeEntry // key: deviceCode
	userCodes     map[string]*DeviceCodeEntry // key: userCode -> deviceCode entry
	parRequests   map[string]*RequestURIEntry
	mu            sync.RWMutex
}

// NewInMemoryStorage creates a new in-memory storage backend.
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		refreshTokens: make(map[string]*RefreshTokenEntry),
		revokedTokens: make(map[string]*RevokedTokenEntry),
		deviceCodes:   make(map[string]*DeviceCodeEntry),
		userCodes:     make(map[string]*DeviceCodeEntry),
		parRequests:   make(map[string]*RequestURIEntry),
	}
}

// RefreshToken operations

func (s *InMemoryStorage) StoreRefreshToken(ctx context.Context, token *RefreshTokenEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshTokens[token.RefreshToken] = token
	return nil
}

func (s *InMemoryStorage) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.refreshTokens[tokenHash]
	if !exists {
		return nil, ErrTokenNotFound
	}
	return entry, nil
}

func (s *InMemoryStorage) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.refreshTokens, tokenHash)
	return nil
}

func (s *InMemoryStorage) ListRefreshTokensByUser(ctx context.Context, userID string) ([]*RefreshTokenEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var tokens []*RefreshTokenEntry
	for _, token := range s.refreshTokens {
		if token.Subject == userID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (s *InMemoryStorage) ListRefreshTokensByClient(ctx context.Context, clientID string) ([]*RefreshTokenEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var tokens []*RefreshTokenEntry
	for _, token := range s.refreshTokens {
		if token.ProviderID == clientID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (s *InMemoryStorage) CleanupExpiredRefreshTokens(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for hash, token := range s.refreshTokens {
		if now.After(token.ExpiresAt) {
			delete(s.refreshTokens, hash)
			count++
		}
	}
	return count, nil
}

// Revoked token operations

func (s *InMemoryStorage) StoreRevokedToken(ctx context.Context, entry *RevokedTokenEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokedTokens[entry.TokenID] = entry
	return nil
}

func (s *InMemoryStorage) IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.revokedTokens[tokenHash]
	return exists, nil
}

func (s *InMemoryStorage) CleanupExpiredRevocations(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for hash, entry := range s.revokedTokens {
		if now.After(entry.ExpiresAt) {
			delete(s.revokedTokens, hash)
			count++
		}
	}
	return count, nil
}

// Device code operations

func (s *InMemoryStorage) StoreDeviceCode(ctx context.Context, entry *DeviceCodeEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deviceCodes[entry.DeviceCode] = entry
	s.userCodes[entry.UserCode] = entry
	return nil
}

func (s *InMemoryStorage) GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*DeviceCodeEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.deviceCodes[deviceCode]
	if !exists {
		return nil, ErrDeviceCodeNotFound
	}
	return entry, nil
}

func (s *InMemoryStorage) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*DeviceCodeEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.userCodes[userCode]
	if !exists {
		return nil, ErrUserCodeNotFound
	}
	return entry, nil
}

func (s *InMemoryStorage) UpdateDeviceCodeStatus(ctx context.Context, deviceCode string, entry *DeviceCodeEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.deviceCodes[deviceCode]; !exists {
		return ErrDeviceCodeNotFound
	}
	s.deviceCodes[deviceCode] = entry
	s.userCodes[entry.UserCode] = entry
	return nil
}

func (s *InMemoryStorage) DeleteDeviceCode(ctx context.Context, deviceCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, exists := s.deviceCodes[deviceCode]; exists {
		delete(s.userCodes, entry.UserCode)
		delete(s.deviceCodes, deviceCode)
	}
	return nil
}

func (s *InMemoryStorage) CleanupExpiredDeviceCodes(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for deviceCode, entry := range s.deviceCodes {
		if now.After(entry.ExpiresAt) {
			delete(s.userCodes, entry.UserCode)
			delete(s.deviceCodes, deviceCode)
			count++
		}
	}
	return count, nil
}

// PAR request URI operations

func (s *InMemoryStorage) StorePARRequest(ctx context.Context, requestURI string, entry *RequestURIEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parRequests[requestURI] = entry
	return nil
}

func (s *InMemoryStorage) GetPARRequest(ctx context.Context, requestURI string) (*RequestURIEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, exists := s.parRequests[requestURI]
	if !exists {
		return nil, ErrRequestURINotFound
	}
	return entry, nil
}

func (s *InMemoryStorage) DeletePARRequest(ctx context.Context, requestURI string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.parRequests, requestURI)
	return nil
}

func (s *InMemoryStorage) MarkPARRequestUsed(ctx context.Context, requestURI string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.parRequests[requestURI]
	if !exists {
		return ErrRequestURINotFound
	}
	entry.Used = true
	entry.UsedAt = time.Now()
	return nil
}

func (s *InMemoryStorage) CleanupExpiredPARRequests(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for uri, entry := range s.parRequests {
		if now.After(entry.ExpiresAt) {
			delete(s.parRequests, uri)
			count++
		}
	}
	return count, nil
}

// Health check

func (s *InMemoryStorage) Ping(ctx context.Context) error {
	// In-memory storage is always available
	return nil
}

func (s *InMemoryStorage) Close() error {
	// No resources to close for in-memory storage
	return nil
}

// Storage error types
var (
	ErrTokenNotFound      = &OIDCError{ErrorCode: "token_not_found", ErrorDescription: "refresh token not found"}
	ErrDeviceCodeNotFound = &OIDCError{ErrorCode: "device_code_not_found", ErrorDescription: "device code not found"}
	ErrUserCodeNotFound   = &OIDCError{ErrorCode: "user_code_not_found", ErrorDescription: "user code not found"}
	ErrRequestURINotFound = &OIDCError{ErrorCode: "request_uri_not_found", ErrorDescription: "request URI not found"}
)
