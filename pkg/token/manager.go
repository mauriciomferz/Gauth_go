package token

import (
	"context"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/tokenstore"
)

// ManagerConfig holds the configuration for a token manager
type ManagerConfig struct {
	Issuer     string
	KeyID      string
	SigningKey []byte
	Store      tokenstore.Store
	Monitor    *Monitor}

// Manager provides token management functionality
type Manager struct {
	config ManagerConfig
}

// NewManager creates a new token manager with the given configuration
func NewManager(config ManagerConfig) *Manager {
	return &Manager{
		config: config,
	}
}

// CreateToken creates a new token with the given claims and duration
func (m *Manager) CreateToken(ctx context.Context, claims map[string]interface{}, duration time.Duration) (string, error) {
	// Generate a new token
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	// Create token data
	data := tokenstore.TokenData{
		Valid:      true,
		ValidUntil: time.Now().Add(duration),
		ClientID:   m.extractClientID(claims),
		OwnerID:    m.extractOwnerID(claims),
		Scope:      m.extractScope(claims),
	}

	// Store token
	err = m.config.Store.Store(token, data)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidateToken validates a token and returns its claims
func (m *Manager) ValidateToken(ctx context.Context, token string) (map[string]interface{}, error) {
	data, exists := m.config.Store.Get(token)
	if !exists {
		return nil, ErrInvalidToken
	}

	if !data.Valid || time.Now().After(data.ValidUntil) {
		return nil, ErrTokenExpired
	}

	// Convert back to claims
	claims := make(map[string]interface{})
	claims["sub"] = data.OwnerID
	claims["client_id"] = data.ClientID
	claims["scope"] = data.Scope
	claims["exp"] = data.ValidUntil.Unix()

	return claims, nil
}

// RevokeToken revokes a token
func (m *Manager) RevokeToken(ctx context.Context, token string) error {
	return m.config.Store.Delete(token)
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
	m.config.KeyID = newKeyID
	m.config.SigningKey = newSigningKey
	return nil
}

// CompleteRotation finalizes a key rotation process
func (m *Manager) CompleteRotation() error {
	// In a real implementation, this would clean up old keys
	// For now, just return success
	return nil
}

// Monitor provides token monitoring functionality
type Monitor struct {
	// Add monitor fields as needed
}

// NewMonitor creates a new token monitor
func NewMonitor() *Monitor {
	return &Monitor{}
}

// MonitorStats represents monitoring statistics
type MonitorStats struct {
	TokensCreated   int
	TokensExpired   int
	TokensRevoked   int
	TokensValidated int
}

// GetStats returns monitoring statistics
func (m *Monitor) GetStats() MonitorStats {
	return MonitorStats{
		TokensCreated: 0,
		TokensExpired: 0,
		TokensRevoked: 0,
	}
}
