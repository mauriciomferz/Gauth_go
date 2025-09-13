// GAuth: Gimel Foundation Authorization Framework
//
// Copyright (c) 2025 Gimel Foundation and contributors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This implementation references and builds upon:
//   - OAuth 2.0 (RFC 6749, RFC 7636, Best Practices) [Apache 2.0]
//   - OpenID Connect (Discovery, Dynamic Client Registration, Session Management) [Apache 2.0]
//   - Model Context Protocol (MCP) [MIT]
//
// Exclusions: This implementation MUST NOT include Web3/blockchain, DNA-based identity, or AI-controlled GAuth logic, as per GiFo-RfC 0111 GAuth (September 2025).
//
// For more information, see: https://gimelfoundation.com/standards/gauth and GiFo-RfC 0111 GAuth.
//
// Package gauth implements the core GAuth functionality
package gauth

import (
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/audit"
	"github.com/Gimel-Foundation/gauth/internal/errors"
	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
	"github.com/Gimel-Foundation/gauth/internal/tokenstore"
)

// GAuth represents the main authentication and authorization system
type GAuth struct {
	config      Config
	tokenStore  tokenstore.Store
	auditLogger *audit.Logger
	rateLimiter *ratelimit.Limiter
	mu          sync.RWMutex
}

// New creates a new GAuth instance with the provided configuration
func New(config Config) (*GAuth, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return &GAuth{
		config:      config,
		tokenStore:  tokenstore.NewMemoryStore(),
		auditLogger: audit.NewLogger(1000),
		rateLimiter: ratelimit.NewLimiter(&ratelimit.Config{
			RequestsPerSecond: config.RateLimit.RequestsPerSecond,
			WindowSize:        1, // 1 second window
			BurstSize:         config.RateLimit.BurstSize,
		}),
	}, nil
}

// InitiateAuthorization starts the authorization process
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
		Type:    "auth_request",
		ActorID: req.ClientID,
		Action:  "initiate_authorization",
		Status:  "granted",
		Metadata: map[string]string{
			"grant_id": grant.GrantID,
		},
	})

	return grant, nil
}

// RequestToken issues a new token based on an authorization grant
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

	if err := g.tokenStore.Store(token, tokenData); err != nil {
		return nil, err
	}

	return &TokenResponse{
		Token:        token,
		ValidUntil:   tokenData.ValidUntil,
		Scope:        tokenData.Scope,
		Restrictions: req.Restrictions,
	}, nil
}

// ValidateToken checks if a token is valid and returns its associated data
// ValidateToken validates the given token and returns its data
func (g *GAuth) ValidateToken(token string) (*tokenstore.TokenData, error) {
	data, exists := g.tokenStore.Get(token)
	if !exists {
		return nil, errors.New(errors.ErrInvalidToken, "token not found")
	}

	if !data.Valid || time.Now().After(data.ValidUntil) {
		return nil, errors.New(errors.ErrTokenExpired, "token has expired")
	}

	return &data, nil
}

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
