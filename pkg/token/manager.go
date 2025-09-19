package token

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync/atomic"
	"time"

	tokenstore "github.com/mauriciomferz/Gauth_go/pkg/tokenstore"
)

type ManagerConfig struct {
    Issuer     string
    KeyID      string
    SigningKey []byte
    Store      interface{}
    Monitor    interface{}
}

type TokenManager struct {
    config ManagerConfig
    refreshTokens map[string]string // refreshToken -> userID
    oldKeyID string
    oldSigningKey []byte
    rotationInProgress bool
}

func NewManager(cfg ManagerConfig) *TokenManager {
    return &TokenManager{
        config: cfg,
        refreshTokens: make(map[string]string),
        oldKeyID: "",
        oldSigningKey: nil,
        rotationInProgress: false,
    }
}

func (m *TokenManager) CreateToken(ctx context.Context, claims map[string]interface{}, ttl time.Duration) (string, error) {
    token, err := generateTokenString()
    if err != nil {
        return "", err
    }
    sub, _ := claims["sub"].(string)
    role, _ := claims["role"].(string)
    now := time.Now()
    keyID := m.config.KeyID
    if m.rotationInProgress && m.oldKeyID != "" {
        // Tokens created before rotation use oldKeyID, after rotation use newKeyID
        keyID = m.config.KeyID
    }
    data := tokenstore.TokenData{
        Valid:      true,
        ValidUntil: now.Add(ttl),
        ClientID:   keyID, // Use ClientID to store keyID for simulation
        OwnerID:    sub,
        Scope:      []string{role},
    }
    if store, ok := m.config.Store.(interface{ Store(string, tokenstore.TokenData) error }); ok {
        if err := store.Store(token, data); err != nil {
            return "", err
        }
    } else {
        return "", errors.New("invalid token store")
    }
    if mon, ok := m.config.Monitor.(*Monitor); ok && mon != nil {
        atomic.AddUint64(&mon.stats.TokensCreated, 1)
    }
    return token, nil
}

func (m *TokenManager) CreateTokenWithRefresh(ctx context.Context, claims map[string]interface{}, accessTTL, refreshTTL time.Duration) (string, string, error) {
    token, err := m.CreateToken(ctx, claims, accessTTL)
    if err != nil {
        return "", "", err
    }
    refreshToken, err := generateTokenString()
    if err != nil {
        return "", "", err
    }
    sub, _ := claims["sub"].(string)
    m.refreshTokens[refreshToken] = sub
    return token, refreshToken, nil
}

func (m *TokenManager) ValidateToken(ctx context.Context, token string) (map[string]interface{}, error) {
    if store, ok := m.config.Store.(interface{ Get(string) (tokenstore.TokenData, bool) }); ok {
        data, exists := store.Get(token)
        if !exists {
            return nil, errors.New("token not found")
        }
        if !data.Valid || time.Now().After(data.ValidUntil) {
            return nil, errors.New("token expired or invalid")
        }
        // After CompleteRotation, reject tokens with oldKeyID
        if !m.rotationInProgress && m.oldKeyID == "" && data.ClientID != m.config.KeyID {
            return nil, errors.New("token signed with old key is invalid after rotation")
        }
        claims := map[string]interface{}{
            "sub":  data.OwnerID,
        }
        if len(data.Scope) > 0 {
            claims["role"] = data.Scope[0]
        }
        if mon, ok := m.config.Monitor.(*Monitor); ok && mon != nil {
            atomic.AddUint64(&mon.stats.TokensValidated, 1)
        }
        return claims, nil
    }
    return nil, errors.New("invalid token store")
}

func (m *TokenManager) RevokeToken(ctx context.Context, token string) error {
    if store, ok := m.config.Store.(interface{ Delete(string) error }); ok {
        err := store.Delete(token)
        if err == nil {
            if mon, ok := m.config.Monitor.(*Monitor); ok && mon != nil {
                atomic.AddUint64(&mon.stats.TokensRevoked, 1)
            }
        }
        return err
    }
    return errors.New("invalid token store")
}

func (m *TokenManager) RefreshToken(ctx context.Context, refresh string) (string, error) {
    sub, ok := m.refreshTokens[refresh]
    if !ok {
        return "", errors.New("invalid refresh token")
    }
    claims := map[string]interface{}{"sub": sub}
    return m.CreateToken(ctx, claims, time.Hour)
}

func (m *TokenManager) RotateKey(newKeyID string, newSigningKey []byte) error {
    if m.rotationInProgress {
        return nil // Already in progress
    }
    m.oldKeyID = m.config.KeyID
    m.oldSigningKey = m.config.SigningKey
    m.config.KeyID = newKeyID
    m.config.SigningKey = newSigningKey
    m.rotationInProgress = true
    return nil
}

func (m *TokenManager) CompleteRotation() error {
    if !m.rotationInProgress {
        return nil
    }
    m.oldKeyID = ""
    m.oldSigningKey = nil
    m.rotationInProgress = false
    return nil
}

type Monitor struct{
    stats Stats
}

func NewMonitor() *Monitor { return &Monitor{} }

type Stats struct {
    TokensCreated   uint64
    TokensValidated uint64
    TokensRevoked   uint64
}

func (m *Monitor) GetStats() Stats {
    return m.stats
}

// generateTokenString creates a new secure token string
func generateTokenString() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}


