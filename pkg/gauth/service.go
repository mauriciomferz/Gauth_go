package gauth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/audit"
	"github.com/Gimel-Foundation/gauth/pkg/events"
	"github.com/Gimel-Foundation/gauth/pkg/rate"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// Service represents the main GAuth service
type Service struct {
	config      Config
	rateLimiter rate.Limiter
	tokenSvc    *token.Service
	eventBus    *events.EventBus
	audit       *audit.Logger

	mu     sync.RWMutex
	grants map[string]*AuthorizationGrant
}

// Authorize handles an authorization request
func (s *Service) Authorize(ctx context.Context, req *AuthorizationRequest) (*AuthorizationGrant, error) {
	// Apply rate limiting
	if err := s.rateLimiter.Allow(ctx, req.ClientID); err != nil {
		s.audit.Log(ctx, audit.NewEntry(audit.TypeAuth).
			WithActor(req.ClientID, audit.ActorUser).
			WithAction(audit.ActionLogin).
			WithResult(audit.ResultDenied).
			WithMetadata("reason", "rate_limited"),
		)
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Validate request
	if err := s.validateAuthRequest(req); err != nil {
		s.audit.Log(ctx, audit.NewEntry(audit.TypeAuth).
			WithActor(req.ClientID, audit.ActorUser).
			WithAction(audit.ActionLogin).
			WithResult(audit.ResultDenied).
			WithMetadata("reason", err.Error()),
		)
		return nil, err
	}

	// Create grant
	grant := &AuthorizationGrant{
		GrantID:    generateGrantID(),
		ClientID:   req.ClientID,
		Scope:      req.Scopes,
		ValidUntil: time.Now().Add(s.config.AccessTokenExpiry),
	}

	// Store grant
	s.mu.Lock()
	s.grants[grant.GrantID] = grant
	s.mu.Unlock()

	// Emit event
	s.eventBus.Publish(events.Event{
		Type:      events.EventTypeAuth,
		Action:    "grant",
		Subject:   req.ClientID,
		Resource:  "auth_grant",
		Timestamp: time.Now(),
		Metadata:  nil, // Add as needed
	})

	// Audit log
	s.audit.Log(ctx, audit.NewEntry(audit.TypeAuth).
		WithActor(req.ClientID, audit.ActorUser).
		WithAction(audit.ActionLogin).
		WithResult(audit.ResultSuccess).
		WithMetadata("grant_id", grant.GrantID).
		WithMetadata("scopes", fmt.Sprintf("%v", req.Scopes)),
	)

	return grant, nil
}

// RequestToken handles a token request
func (s *Service) RequestToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	// Validate grant
	s.mu.RLock()
	grant, exists := s.grants[req.GrantID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid grant ID")
	}

	if time.Now().After(grant.ValidUntil) {
		return nil, fmt.Errorf("grant expired")
	}

	// Generate token

	tok := &token.Token{
		Subject:   grant.ClientID,
		Scopes:    grant.Scope,
		ExpiresAt: time.Now().Add(s.config.AccessTokenExpiry),
		IssuedAt:  time.Now(),
		Type:      token.Access,
	}
	issued, err := s.tokenSvc.Issue(ctx, tok)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	resp := &TokenResponse{
		Token:      issued.Value,
		ValidUntil: issued.ExpiresAt,
		Scope:      grant.Scope,
	}

	// Emit event
	s.eventBus.Publish(events.Event{
		Type:      events.EventTypeToken,
		Action:    "issue",
		Subject:   grant.ClientID,
		Resource:  "token",
		Timestamp: time.Now(),
		Metadata:  nil, // Add as needed
	})

	// Audit log
	s.audit.Log(ctx, audit.NewEntry(audit.TypeToken).
		WithActor(grant.ClientID, audit.ActorUser).
		WithAction(audit.ActionTokenCreate).
		WithResult(audit.ResultSuccess).
		WithMetadata("grant_id", grant.GrantID),
	)

	return resp, nil
}

// RevokeToken revokes a token
func (s *Service) RevokeToken(ctx context.Context, token string) error {

	// TODO: Implement revoke using tokenSvc and token.Token struct
	// if err := s.tokenSvc.Revoke(ctx, &token.Token{Value: token}); err != nil {
	//     return fmt.Errorf("failed to revoke token: %w", err)
	// }

	// Emit event
	s.eventBus.Publish(events.Event{
		Type:      events.EventTypeToken,
		Action:    "revoke",
		Subject:   "unknown", // TODO: fill with correct subject
		Resource:  "token",
		Timestamp: time.Now(),
		Metadata:  nil, // Add as needed
	})

	// Audit log
	s.audit.Log(ctx, audit.NewEntry(audit.TypeToken).
		WithAction(audit.ActionTokenRevoke).
		WithResult(audit.ResultSuccess).
		WithMetadata("token", token),
	)

	return nil
}

// Close releases all resources
func (s *Service) Close() error {
	var errs []error

	if err := s.audit.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing service: %v", errs)
	}

	return nil
}

// Internal methods

func (s *Service) validateAuthRequest(req *AuthorizationRequest) error {
	if req.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if len(req.Scopes) == 0 {
		return fmt.Errorf("at least one scope is required")
	}
	return nil
}

func (s *Service) handleAuthGrant(data interface{}) {
	// Handle authorization grant event
}

func (s *Service) handleTokenIssue(data interface{}) {
	// Handle token issuance event
}

func (s *Service) handleTokenRevoke(data interface{}) {
	// Handle token revocation event
}

func validateConfig(config Config) error {
	if config.AuthServerURL == "" {
		return fmt.Errorf("auth server URL is required")
	}
	if config.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if config.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}
	if config.AccessTokenExpiry <= 0 {
		return fmt.Errorf("access token expiry must be positive")
	}
	return nil
}
