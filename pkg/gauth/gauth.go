// Package gauth provides authentication and authorization primitives.
//
// See README.md and LIBRARY.md for usage and extension points.
package gauth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/audit"
	"github.com/Gimel-Foundation/gauth/internal/errors"
	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
	"github.com/Gimel-Foundation/gauth/internal/tokenstore"
	"github.com/Gimel-Foundation/gauth/pkg/metrics"
)

// Close releases any resources held by GAuth (stub for test compatibility)
func (g *GAuth) Close() error {
	// No resources to release in this stub
	return nil
}

// Authorize processes an authorization request (stub for test compatibility)
func (g *GAuth) Authorize(ctx interface{}, req interface{}) (interface{}, error) {
	// Stub: return nil, nil for now
	return nil, nil
}

// AuditEventType and AuditAction are typed constants for audit logging.
type AuditEventType string
type AuditAction string

const (
	AuditTypeAuthRequest audit.Type   = "auth_request"
	AuditActionInitiate  audit.Action = "initiate_authorization"
)

// AuthRequestMetadata is a typed struct for audit event metadata.
type AuthRequestMetadata struct {
	GrantID string
}

// GAuth represents the main authentication and authorization system.
type GAuth struct {
	config      Config
	TokenStore  tokenstore.Store // Exported for use in points.go
	auditLogger *audit.Logger
	rateLimiter *ratelimit.Limiter
	mu          sync.RWMutex //nolint:unused // unexported: not part of public API, reserved for concurrent operations
}

// New creates a new GAuth instance with the provided configuration.
func New(config Config) (*GAuth, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	
	// Register metrics (replaces removed init functions)
	metrics.RegisterMetrics()
	metrics.RegisterHTTPMetrics()
	
	auditLogger := audit.NewLogger(1000)
	rateLimiter := ratelimit.NewLimiter(&ratelimit.Config{
		RequestsPerSecond: 10,
		BurstSize:         5,
		WindowSize:        60, // 60 seconds
	})
	return &GAuth{
		config:      config,
		TokenStore:  tokenstore.NewMemoryStore(),
		auditLogger: auditLogger,
		rateLimiter: rateLimiter,
	}, nil
}

// InitiateAuthorization starts the authorization process.
func (g *GAuth) InitiateAuthorization(req AuthorizationRequest) (*AuthorizationGrant, error) {
	if err := g.validateAuthRequest(req); err != nil {
		return nil, err
	}
	grantID := generateGrantID()
	grant := &AuthorizationGrant{
		GrantID:    grantID,
		ClientID:   req.ClientID,
		Scope:      req.Scopes,
		ValidUntil: time.Now().Add(g.config.AccessTokenExpiry),
	}
	g.auditLogger.Log(audit.Event{
		Type:    AuditTypeAuthRequest,
		ActorID: req.ClientID,
		Action:  AuditActionInitiate,
		Status:  "granted",
		Metadata: map[string]string{
			"grant_id": grant.GrantID,
		},
	})
	return grant, nil
}

// RequestToken issues a new token based on an authorization grant.
func (g *GAuth) RequestToken(req TokenRequest) (*TokenResponse, error) {
	if err := g.rateLimiter.Allow(req.Context, req.GrantID); err != nil {
		return nil, errors.New(errors.ErrRateLimitExceeded, "rate limit exceeded")
	}
	token, err := generateToken()
	if err != nil {
		return nil, errors.New(errors.ErrInternalError, "failed to generate token")
	}
	tokenData := tokenstore.TokenData{
		Valid:      true,
		ValidUntil: time.Now().Add(g.config.AccessTokenExpiry),
		ClientID:   req.GrantID,
		Scope:      req.Scope,
	}
	if err := g.TokenStore.Store(token, tokenData); err != nil {
		return nil, err
	}
	return &TokenResponse{
		Token:        token,
		ValidUntil:   tokenData.ValidUntil,
		Scope:        tokenData.Scope,
		Restrictions: req.Restrictions,
	}, nil
}

// ValidateToken checks if a token is valid and returns its associated data.
func (g *GAuth) ValidateToken(token string) (*tokenstore.TokenData, error) {
	data, exists := g.TokenStore.Get(token)
	if !exists {
		return nil, errors.New(errors.ErrInvalidToken, "token not found")
	}
	if !data.Valid || time.Now().After(data.ValidUntil) {
		return nil, errors.New(errors.ErrTokenExpired, "token has expired")
	}
	return &data, nil
}

// GetAuditLogger returns the audit logger for inspection (RFC111: auditability).
func (g *GAuth) GetAuditLogger() *audit.Logger {
	return g.auditLogger
}

// validateAuthRequest validates the authorization request.
func (g *GAuth) validateAuthRequest(req AuthorizationRequest) error {
	if req.ClientID == "" {
		return errors.New(errors.ErrInvalidConfig, "client ID is required")
	}
	if req.ClientID != g.config.ClientID {
		return errors.New(errors.ErrUnauthorized, "invalid client ID")
	}
	if len(req.Scopes) == 0 {
		return errors.New(errors.ErrInvalidConfig, "at least one scope is required")
	}
	return nil
}

// generateToken creates a random token string for demonstration/testing purposes.
func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
