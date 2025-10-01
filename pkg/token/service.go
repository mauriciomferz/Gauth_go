package token

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// splitScopes splits a comma-separated string into a slice of scopes
func splitScopes(s string) []string {
	return strings.Split(s, ",")
}

// ServiceAPI defines the interface for token management functionality
type ServiceAPI interface {
	GetToken(ctx context.Context, id string) (*Token, error)
	Validate(ctx context.Context, token *Token) error
	Revoke(ctx context.Context, token *Token) error
	Issue(ctx context.Context, token *Token) (*Token, error)
	Refresh(ctx context.Context, refreshToken *Token) (*Token, error)
	List(ctx context.Context, filter Filter) ([]*Token, error)
}

// Service provides token management functionality
type Service struct {
	config Config
	store  Store
} // GetToken retrieves a token by its ID.
func (s *Service) GetToken(ctx context.Context, id string) (*Token, error) {
	return s.store.Get(ctx, id)
}

// NewService creates a new token service with the given configuration
func NewService(config Config, store Store) ServiceAPI {
	svc := &Service{
		config: config,
		store:  store,
	}

	// Start cleanup goroutine if interval is set
	if config.CleanupInterval > 0 {
		go svc.periodicCleanup()
	}

	return svc
}

// Issue creates, signs and stores a new token
func (s *Service) Issue(ctx context.Context, token *Token) (*Token, error) {
	// Set defaults if not provided
	if token.ExpiresAt.IsZero() {
		if token.Type == Refresh {
			token.ExpiresAt = time.Now().Add(s.config.RefreshPeriod)
		} else {
			token.ExpiresAt = time.Now().Add(s.config.ValidityPeriod)
		}
	}

	if token.IssuedAt.IsZero() {
		token.IssuedAt = time.Now()
	}

	if token.NotBefore.IsZero() {
		token.NotBefore = token.IssuedAt
	}

	if token.Algorithm == "" {
		token.Algorithm = s.config.SigningMethod
	}

	if len(token.Scopes) == 0 {
		token.Scopes = s.config.DefaultScopes
	}

	// Basic validation
	if err := s.validateConfig(token); err != nil {
		return nil, NewValidationErrorWithCause(ValidationCodeInvalidConfig, "token fails config validation", err)
	}

	// Generate signed token value
	signedValue, err := s.signToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}
	token.Value = signedValue

	// Store token
	if err := s.store.Save(ctx, token.ID, token); err != nil {
		return nil, NewValidationErrorWithCause(ValidationCodeStorageFailure, "failed to store token", err)
	}

	return token, nil
}

// Validate checks if a token is valid
func (s *Service) Validate(ctx context.Context, token *Token) error {
	if err := s.validateSignature(token); err != nil {
		return err
	}

	if err := s.validateTimeClaims(token); err != nil {
		return err
	}

	if err := s.validateIssuerAndAudience(token); err != nil {
		return err
	}

	return s.validateTokenStorage(ctx, token)
}

func (s *Service) validateSignature(token *Token) error {
	if err := s.verifySignature(token); err != nil {
		return NewValidationErrorWithCause(ValidationCodeInvalidSignature, "invalid token signature", err)
	}
	return nil
}

func (s *Service) validateTimeClaims(token *Token) error {
	now := time.Now()

	if now.After(token.ExpiresAt) {
		return NewValidationError(ValidationCodeExpired, "token has expired")
	}

	if now.Before(token.NotBefore) {
		return NewValidationError(ValidationCodeNotYetValid, "token not yet valid")
	}

	return nil
}

func (s *Service) validateIssuerAndAudience(token *Token) error {
	if err := s.validateTokenIssuer(token); err != nil {
		return err
	}

	return s.validateTokenAudience(token)
}

func (s *Service) validateTokenIssuer(token *Token) error {
	if !s.config.ValidateIssuer {
		return nil
	}

	for _, iss := range s.config.AllowedIssuers {
		if token.Issuer == iss {
			return nil
		}
	}
	return NewValidationError(ValidationCodeInvalidIssuer, "token issuer not allowed")
}

func (s *Service) validateTokenAudience(token *Token) error {
	if !s.config.ValidateAudience {
		return nil
	}

	for _, aud := range token.Audience {
		for _, allowed := range s.config.AllowedAudiences {
			if aud == allowed {
				return nil
			}
		}
	}
	return NewValidationError(ValidationCodeInvalidAudience, "token audience not allowed")
}

func (s *Service) validateTokenStorage(ctx context.Context, token *Token) error {
	stored, err := s.store.Get(ctx, token.ID)
	if err != nil {
		if err == ErrTokenNotFound {
			return NewValidationError(ValidationCodeRevoked, "token has been revoked")
		}
		return NewValidationErrorWithCause(ValidationCodeStorageFailure, "failed to verify token status", err)
	}

	if stored.Value != token.Value {
		return NewValidationError(ValidationCodeInvalid, "token does not match stored value")
	}

	return nil
}

// Revoke invalidates a token before its natural expiration
func (s *Service) Revoke(ctx context.Context, token *Token) error {
	return s.store.Delete(ctx, token.ID)
}

// Refresh exchanges a refresh token for a new access token
func (s *Service) Refresh(ctx context.Context, refreshToken *Token) (*Token, error) {
	// Validate refresh token
	if err := s.Validate(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if refreshToken.Type != Refresh {
		return nil, NewValidationError(ValidationCodeInvalidType, "token is not a refresh token")
	}

	// Create new access token
	var scopes []string
	if refreshToken.Metadata != nil && refreshToken.Metadata.AppData != nil {
		if orig, ok := refreshToken.Metadata.AppData["original_scopes"]; ok && orig != "" {
			// Split comma-separated string into slice
			for _, s := range splitScopes(orig) {
				if s != "" {
					scopes = append(scopes, s)
				}
			}
		}
	}
	if len(scopes) == 0 {
		scopes = refreshToken.Scopes
	}
	accessToken := &Token{
		ID:        GenerateID(),
		Type:      Access,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(s.config.ValidityPeriod),
		NotBefore: time.Now(),
		Issuer:    refreshToken.Issuer,
		Subject:   refreshToken.Subject,
		Audience:  refreshToken.Audience,
		Scopes:    scopes,
		Algorithm: s.config.SigningMethod,
	}

	// Helper to split comma-separated scopes
	// (define at file scope if not present)
	// func splitScopes(s string) []string { return strings.Split(s, ",") }

	// Issue new token
	return s.Issue(ctx, accessToken)
}

// List returns tokens matching the given filter
func (s *Service) List(ctx context.Context, filter Filter) ([]*Token, error) {
	return s.store.List(ctx, filter)
}

func (s *Service) validateConfig(_ *Token) error {
	if s.config.SigningKey == nil {
		return fmt.Errorf("signing key not configured")
	}

	if s.config.ValidateIssuer && len(s.config.AllowedIssuers) == 0 {
		return fmt.Errorf("issuer validation enabled but no allowed issuers configured")
	}

	if s.config.ValidateAudience && len(s.config.AllowedAudiences) == 0 {
		return fmt.Errorf("audience validation enabled but no allowed audiences configured")
	}

	return nil
}

func (s *Service) signToken(token *Token) (string, error) {
	// This is a placeholder - actual signing would use JWT, PASETO, etc.
	tokenBytes := []byte(fmt.Sprintf("%s.%s.%s", token.ID, token.Subject, token.Type))

	// Hash the tokenBytes using SHA256
	hash := crypto.SHA256.New()
	hash.Write(tokenBytes)
	hashed := hash.Sum(nil)

	// Use rsa.SignPKCS1v15 for signing
	rsaPriv, ok := s.config.SigningKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("SigningKey is not an *rsa.PrivateKey")
	}
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaPriv, crypto.SHA256, hashed)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(signature), nil
}

func (s *Service) verifySignature(_ *Token) error {
	// This is a placeholder - actual verification would use JWT, PASETO, etc.
	return nil
}

func (s *Service) periodicCleanup() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		filter := Filter{
			ExpiresBefore: time.Now(),
		}

		tokens, err := s.store.List(ctx, filter)
		if err != nil {
			continue
		}

		for _, token := range tokens {
			_ = s.store.Delete(ctx, token.ID)
		}
	}
}

// GenerateID generates a random token ID
func GenerateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.
