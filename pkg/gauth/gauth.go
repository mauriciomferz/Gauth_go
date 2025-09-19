// Copyright (c) 2025 Gimel Foundation and the persons identified as the document authors.
// All rights reserved. This file is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents.
// See http://GimelFoundation.com or https://github.com/Gimel-Foundation for details.
// Code Components extracted from GiFo-RfC 0111 must include this license text and are provided without warranty.
//
// GAuth Protocol Compliance: This file implements the GAuth protocol (GiFo-RfC 0111).
//
// Protocol Usage Declaration:
//   - GAuth protocol: IMPLEMENTED throughout this file (see [GAuth] comments below)
//   - OAuth 2.0:      NOT USED anywhere in this file
//   - PKCE:           NOT USED anywhere in this file
//   - OpenID:         NOT USED anywhere in this file
//
// [GAuth] = GAuth protocol logic (GiFo-RfC 0111)
// [Other] = Placeholder for OAuth2, OpenID, PKCE, or other protocols (none present in this file)
//
// See README.md and LIBRARY.md for usage and extension points.
//
// [GAuth] Package gauth provides authentication and authorization primitives.
package gauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"      // [GAuth]
	"github.com/mauriciomferz/Gauth_go/pkg/errors"     // [GAuth]
	"github.com/mauriciomferz/Gauth_go/pkg/ratelimit"  // [GAuth]
	"github.com/mauriciomferz/Gauth_go/pkg/tokenstore" // [GAuth]
)

// Close releases any resources held by GAuth.
// For most in-memory/test use cases, this is a no-op.
func (g *GAuth) Close() error {
	// No resources to release in this stub
	return nil
}

// Authorize processes an authorization request.
// Deprecated: Use InitiateAuthorization and RequestToken for explicit flows.
func (g *GAuth) Authorize(ctx interface{}, req interface{}) (interface{}, error) {
	// Stub: return nil, nil for now
	return nil, nil
}


// AuditEventType is a string type for audit event types.
type AuditEventType string

// AuditAction is a string type for audit actions.
type AuditAction string

const (
	// AuditTypeAuthRequest is the event type for authorization requests.
	AuditTypeAuthRequest = "auth_request"
	// AuditActionInitiate is the action for initiating authorization.
	AuditActionInitiate  = "initiate_authorization"
)

// AuthRequestMetadata contains metadata for an authorization request audit event.
type AuthRequestMetadata struct {
	GrantID string
}

// AuditLogger defines the interface for pluggable audit logging.
// Implementations should be thread-safe.
type AuditLogger interface {
	// Log records an audit entry.
	Log(ctx context.Context, entry *audit.Entry)
	// GetRecentEvents returns the most recent audit events.
	GetRecentEvents(limit int) []audit.Event
}


// GAuth is the main authentication and authorization system for the GAuth protocol.
// Use New to construct a GAuth instance.
type GAuth struct {
	config      Config
	TokenStore  tokenstore.Store // Exported for use in points.go
	auditLogger AuditLogger      // Pluggable audit logger
	rateLimiter *ratelimit.Limiter
	mu          sync.RWMutex // unexported: not part of public API
}


// New creates a new GAuth instance with the provided configuration and optional pluggable components.
// If auditLogger is nil, a default in-memory logger is used.
//
// Example:
//   gauth, err := gauth.New(&gauth.Config{ClientID: "my-client", AccessTokenExpiry: time.Hour}, nil)
//   if err != nil { log.Fatal(err) }
func New(config *Config, auditLogger AuditLogger) (*GAuth, error) {
       if err := validateConfig(config); err != nil {
	       return nil, err
       }
       if auditLogger == nil {
	       auditLogger = audit.NewLogger(1000)
       }
       rateLimiter := ratelimit.NewLimiter(&ratelimit.Config{
	       RequestsPerSecond: 10,
	       BurstSize:         5,
	       WindowSize:        60, // 60 seconds
       })
       return &GAuth{
	       config:      *config,
	       TokenStore:  tokenstore.NewMemoryStore(),
	       auditLogger: auditLogger,
	       rateLimiter: rateLimiter,
       }, nil
}


// InitiateAuthorization starts the authorization process for a client.
// Returns an AuthorizationGrant if successful.
//
// Example:
//   grant, err := gauth.InitiateAuthorization(gauth.AuthorizationRequest{ClientID: "my-client", Scopes: []string{"read"}})
//   if err != nil { log.Fatal(err) }
func (g *GAuth) InitiateAuthorization(req AuthorizationRequest) (*AuthorizationGrant, error) {
       if err := g.validateAuthRequest(req); err != nil {
	       return nil, err
       }
       grantID, err := generateGrantID()
       if err != nil {
	       return nil, fmt.Errorf("failed to generate grant ID: %w", err)
       }
       grant := &AuthorizationGrant{
	       GrantID:    grantID,
	       ClientID:   req.ClientID,
	       Scope:      req.Scopes,
	       ValidUntil: time.Now().Add(g.config.AccessTokenExpiry),
       }
       g.auditLogger.Log(context.Background(), &audit.Entry{
	       Type:     AuditTypeAuthRequest,
	       ActorID:  req.ClientID,
	       Action:   AuditActionInitiate,
	       Result:   "granted",
	       Metadata: map[string]string{
		       "grant_id": grant.GrantID,
	       },
       })
       return grant, nil
}


// RequestToken issues a new token based on an authorization grant.
// Returns a TokenResponse if successful.
//
// Example:
//   resp, err := gauth.RequestToken(gauth.TokenRequest{GrantID: grant.GrantID, Scope: []string{"read"}})
//   if err != nil { log.Fatal(err) }
func (g *GAuth) RequestToken(req TokenRequest) (*TokenResponse, error) {
       if err := g.rateLimiter.Allow(req.Context, req.GrantID); err != nil {
	       return nil, errors.New(errors.ErrRateLimitExceeded, "rate limit exceeded")
       }
       token, err := generateToken()
       if err != nil {
	       return nil, errors.New(errors.ErrServerError, "failed to generate token")
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
// Returns a TokenData pointer if valid, or an error if not.
//
// Example:
//   data, err := gauth.ValidateToken(token)
//   if err != nil { log.Fatal(err) }
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


// GetAuditLogger returns the pluggable audit logger for inspection (RFC111: auditability).
func (g *GAuth) GetAuditLogger() AuditLogger {
	return g.auditLogger
}

// validateAuthRequest validates the authorization request.
func (g *GAuth) validateAuthRequest(req AuthorizationRequest) error {
	if req.ClientID == "" {
		return errors.New(errors.ErrInvalidClient, "client ID is required")
	}
	if req.ClientID != g.config.ClientID {
		return errors.New(errors.ErrUnauthorizedClient, "invalid client ID")
	}
	if len(req.Scopes) == 0 {
		return errors.New(errors.ErrInvalidScope, "at least one scope is required")
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