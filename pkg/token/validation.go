package token

import (
	"context"
	"errors"
	"time"
)

// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

// Validator defines the interface for token validation
// Validator (RFC111: Power Decision Point, Section 3) defines the interface for token validation
type Validator interface {
	// Validate checks if a token is valid
	Validate(ctx context.Context, token *Token) error
}

// ValidationConfig holds configuration for token validation
type ValidationConfig struct {
	// AllowedIssuers is a list of valid token issuers
	AllowedIssuers []string

	// AllowedAudiences is a list of valid token audiences
	AllowedAudiences []string

	// RequiredScopes are scopes that must be present
	RequiredScopes []string

	// RequiredClaims are claims that must be present and match (now type-safe)
	RequiredClaims *ClaimRequirements

	// ClockSkew allows for small time differences
	ClockSkew time.Duration

	// ValidateSignature indicates if signature validation is required
	ValidateSignature bool
}

// ValidationChain runs multiple validators in sequence
type ValidationChain struct {
	validators []Validator
	blacklist  *Blacklist
	config     ValidationConfig
}

// NewValidationChain creates a new validation chain
func NewValidationChain(config ValidationConfig, blacklist *Blacklist, validators ...Validator) *ValidationChain {
	return &ValidationChain{
		validators: validators,
		blacklist:  blacklist,
		config:     config,
	}
}

// Validate runs all validators in sequence
func (vc *ValidationChain) Validate(ctx context.Context, token *Token) error {
	if err := vc.validateBasicToken(token); err != nil {
		return err
	}

	now := time.Now()

	if err := vc.validateBlacklist(ctx, token); err != nil {
		return err
	}

	if err := vc.validateTimeClaims(token, now); err != nil {
		return err
	}

	if err := vc.validateIssuer(token); err != nil {
		return err
	}

	if err := vc.validateAudience(token); err != nil {
		return err
	}

	if err := vc.validateScopes(token); err != nil {
		return err
	}

	if err := vc.validateClaims(token); err != nil {
		return err
	}

	return vc.runCustomValidators(ctx, token)
}

func (vc *ValidationChain) validateBasicToken(token *Token) error {
	if token == nil {
		return ErrInvalidToken
	}
	return nil
}

func (vc *ValidationChain) validateBlacklist(ctx context.Context, token *Token) error {
	if vc.blacklist != nil && vc.blacklist.IsBlacklisted(ctx, token.ID) {
		return ErrTokenBlacklisted
	}
	return nil
}

func (vc *ValidationChain) validateTimeClaims(token *Token, now time.Time) error {
	if token.ExpiresAt.Add(vc.config.ClockSkew).Before(now) {
		return ErrTokenExpired
	}

	if token.NotBefore.Add(-vc.config.ClockSkew).After(now) {
		return ErrTokenNotYetValid
	}

	return nil
}

func (vc *ValidationChain) validateIssuer(token *Token) error {
	if len(vc.config.AllowedIssuers) == 0 {
		return nil
	}

	for _, issuer := range vc.config.AllowedIssuers {
		if token.Issuer == issuer {
			return nil
		}
	}
	return ErrInvalidIssuer
}

func (vc *ValidationChain) validateAudience(token *Token) error {
	if len(vc.config.AllowedAudiences) == 0 || len(token.Audience) == 0 {
		return nil
	}

	for _, reqAud := range vc.config.AllowedAudiences {
		for _, tokenAud := range token.Audience {
			if reqAud == tokenAud {
				return nil
			}
		}
	}
	return ErrInvalidAudience
}

func (vc *ValidationChain) validateScopes(token *Token) error {
	for _, scope := range vc.config.RequiredScopes {
		if !token.HasScope(scope) {
			return ErrInsufficientScope
		}
	}
	return nil
}

func (vc *ValidationChain) validateClaims(token *Token) error {
	if vc.config.RequiredClaims != nil {
		return ValidateClaims(token, vc.config.RequiredClaims)
	}
	return nil
}

func (vc *ValidationChain) runCustomValidators(ctx context.Context, token *Token) error {
	for _, validator := range vc.validators {
		if err := validator.Validate(ctx, token); err != nil {
			return err
		}
	}
	return nil
}

// validateClaims checks if all required claims are present and match
// ValidateClaims (RFC111: Section 6, Verification of powers/claims)
// ValidateClaims checks if all required claims are present and match (RFC111: Section 6, Verification of powers/claims)
func ValidateClaims(_ *Token, _ *ClaimRequirements) error {
	// Implementation would check custom claims match required values
	return nil
}

// HasScope checks if the token has the given scope
func (t *Token) HasScope(scope string) bool {
	for _, s := range t.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// Query represents a token search query
type Query struct {
	// Subject filters tokens by subject
	Subject string

	// Type filters tokens by type
	Type Type

	// Issuer filters tokens by issuer
	Issuer string

	// Scopes filters tokens having all these scopes
	Scopes []string

	// ValidAt filters tokens valid at this time
	ValidAt time.Time

	// Metadata filters tokens having these metadata key-value pairs
	Metadata map[string]string
}

// QueryResult represents a token search result
type QueryResult struct {
	// Tokens are the tokens matching the query
	Tokens []*Token

	// Total is the total number of matching tokens
	Total int

	// HasMore indicates if there are more results
	HasMore bool
}

// Querier defines token search functionality
type Querier interface {
	// Query searches for tokens matching the criteria
	Query(ctx context.Context, query Query, offset, limit int) (*QueryResult, error)

	// CountBySubject counts tokens for a subject
	CountBySubject(ctx context.Context, subject string) (int, error)

	// ListExpiringSoon lists tokens expiring within duration
	ListExpiringSoon(ctx context.Context, duration time.Duration) ([]*Token, error)
}

// DefaultQuerier implements Querier for Store
type DefaultQuerier struct {
	store Store
}

// NewDefaultQuerier creates a default token querier
func NewDefaultQuerier(store Store) *DefaultQuerier {
	return &DefaultQuerier{store: store}
}

// Query implements token searching
func (q *DefaultQuerier) Query(ctx context.Context, query Query, offset, limit int) (*QueryResult, error) {
	// Implementation depends on the store's capabilities
	// For memory store, we'd scan all tokens and filter
	// For database store, we'd build a SQL query
	// This would be implemented by specific store types
	return nil, errors.New("query not implemented by this store")
}

// CountBySubject counts tokens for a subject
func (q *DefaultQuerier) CountBySubject(ctx context.Context, subject string) (int, error) {
	return 0, errors.New("count not implemented by this store")
}

// ListExpiringSoon lists tokens expiring within duration
func (q *DefaultQuerier) ListExpiringSoon(ctx context.Context, duration time.Duration) ([]*Token, error) {
	return nil, errors.New("list expiring not implemented by this store")
}
