package token

import (
	"context"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/tokenstore"
)

// ManagerConfig holds the configuration for a token manager
type ManagerConfig struct {
	Issuer     string
	KeyID      string
	SigningKey []byte
	Store      tokenstore.Store
	Monitor    *Monitor
}

// Manager provides token management functionality
type Manager struct {
	config               ManagerConfig
	keyRotationCompleted bool
	keyRotationTime      time.Time
	mu                   sync.RWMutex
}

// NewManager creates a new token manager with the given configuration
func NewManager(config ManagerConfig) *Manager {
	return &Manager{
		config: config,
	}
}

// CreateToken creates a new token with the given claims and duration
func (m *Manager) CreateToken(_ context.Context, claims map[string]interface{}, duration time.Duration) (string, error) {
	// Generate a new token
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	// Create token data
	now := time.Now()
	data := tokenstore.TokenData{
		Valid:      true,
		ValidUntil: now.Add(duration),
		CreatedAt:  now, // Track when the token was created
		ClientID:   m.extractClientID(claims),
		OwnerID:    m.extractOwnerID(claims),
		Scope:      m.extractScope(claims),
		Claims:     claims, // Store all original claims
	}

	// Store token
	err = m.config.Store.Store(token, data)
	if err != nil {
		return "", err
	}

	// Update monitor if available
	if m.config.Monitor != nil {
		m.config.Monitor.IncrementTokensCreated()
	}

	return token, nil
}

// ValidateToken validates a token and returns its claims
func (m *Manager) ValidateToken(_ context.Context, token string) (map[string]interface{}, error) {
	data, exists := m.config.Store.Get(token)
	if !exists {
		return nil, ErrInvalidToken
	}

	if !data.Valid || time.Now().After(data.ValidUntil) {
		return nil, ErrTokenExpired
	}

	// Check if key rotation was completed and token was created before rotation
	m.mu.RLock()
	if m.keyRotationCompleted && data.CreatedAt.Before(m.keyRotationTime) {
		m.mu.RUnlock()
		return nil, ErrInvalidToken
	}
	m.mu.RUnlock()

	// Return original claims with updated values
	claims := make(map[string]interface{})

	// Start with stored claims to preserve all original data
	if data.Claims != nil {
		for k, v := range data.Claims {
			claims[k] = v
		}
	}

	// Only override specific standard claims if we have valid data
	if data.OwnerID != "" {
		claims["sub"] = data.OwnerID
	}
	if data.ClientID != "" {
		claims["client_id"] = data.ClientID
	}
	if len(data.Scope) > 0 {
		claims["scope"] = data.Scope
	}
	claims["exp"] = data.ValidUntil.Unix()

	// Update monitor if available
	if m.config.Monitor != nil {
		m.config.Monitor.IncrementTokensValidated()
	}

	return claims, nil
}

// RevokeToken revokes a token
func (m *Manager) RevokeToken(_ context.Context, token string) error {
	err := m.config.Store.Delete(token)

	// Update monitor if available and revocation was successful
	if err == nil && m.config.Monitor != nil {
		m.config.Monitor.IncrementTokensRevoked()
	}

	return err
}

// CreateTokenWithRefresh creates both access and refresh tokens
func (m *Manager) CreateTokenWithRefresh(ctx context.Context, claims map[string]interface{}, accessDuration, refreshDuration time.Duration) (string, string, error) {
	// Create access token
	accessToken, err := m.CreateToken(ctx, claims, accessDuration)
	if err != nil {
		return "", "", err
	}

	// Create refresh token
	refreshToken, err := m.CreateToken(ctx, claims, refreshDuration)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// RefreshToken creates a new access token using a refresh token
func (m *Manager) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Validate refresh token
	claims, err := m.ValidateToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	// Create new access token
	return m.CreateToken(ctx, claims, 1*time.Hour)
}

// Helper methods to extract data from claims
func (m *Manager) extractClientID(claims map[string]interface{}) string {
	if clientID, ok := claims["client_id"].(string); ok {
		return clientID
	}
	return ""
}

func (m *Manager) extractOwnerID(claims map[string]interface{}) string {
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}

func (m *Manager) extractScope(claims map[string]interface{}) []string {
	if scope, ok := claims["scope"].([]string); ok {
		return scope
	}
	if scope, ok := claims["scope"].(string); ok {
		return []string{scope}
	}
	return []string{}
}

// RotateKey rotates the signing key for the manager
func (m *Manager) RotateKey(newKeyID string, newSigningKey []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.KeyID = newKeyID
	m.config.SigningKey = newSigningKey
	m.keyRotationTime = time.Now()

	return nil
}

// CompleteRotation finalizes a key rotation process
func (m *Manager) CompleteRotation() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keyRotationCompleted = true

	return nil
}

// Monitor provides token monitoring functionality
type Monitor struct {
	tokensCreated   uint64
	tokensExpired   uint64
	tokensRevoked   uint64
	tokensValidated uint64
	mu              sync.RWMutex
}

// NewMonitor creates a new token monitor
func NewMonitor() *Monitor {
	return &Monitor{}
}

// MonitorStats represents monitoring statistics
type MonitorStats struct {
	TokensCreated   uint64
	TokensExpired   uint64
	TokensRevoked   uint64
	TokensValidated uint64
}

// GetStats returns monitoring statistics
func (m *Monitor) GetStats() MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MonitorStats{
		TokensCreated:   m.tokensCreated,
		TokensExpired:   m.tokensExpired,
		TokensRevoked:   m.tokensRevoked,
		TokensValidated: m.tokensValidated,
	}
}

// IncrementTokensCreated increments the created tokens counter
func (m *Monitor) IncrementTokensCreated() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokensCreated++
}

// IncrementTokensValidated increments the validated tokens counter
func (m *Monitor) IncrementTokensValidated() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokensValidated++
}

// IncrementTokensRevoked increments the revoked tokens counter
func (m *Monitor) IncrementTokensRevoked() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokensRevoked++
}

// IncrementTokensExpired increments the expired tokens counter
func (m *Monitor) IncrementTokensExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokensExpired++
}
