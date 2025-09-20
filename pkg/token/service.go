package token

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"
)

// splitScopes splits a comma-separated string into a slice of scopes.
// Used internally for parsing scope lists.
func splitScopes(s string) []string {
	return strings.Split(s, ",")
}

// ServiceAPI defines the interface for token management functionality.
// Provides methods for issuing, validating, revoking, refreshing, and listing tokens.
type ServiceAPI interface {
	// GetToken retrieves a token by its ID.
	GetToken(ctx context.Context, id string) (*Token, error)
	// Validate checks if a token is valid.
	Validate(ctx context.Context, token *Token) error
	// Revoke invalidates a token.
	Revoke(ctx context.Context, token *Token) error
	// Issue creates, signs, and stores a new token.
	Issue(ctx context.Context, token *Token) (*Token, error)
	// Refresh issues a new token using a refresh token.
	Refresh(ctx context.Context, refreshToken *Token) (*Token, error)
	// List returns tokens matching a filter.
	List(ctx context.Context, filter Filter) ([]*Token, error)
}

// Service provides token management functionality.
// Implements ServiceAPI for issuing, validating, revoking, and refreshing tokens.
type Service struct {
	config *Config
	store  Store
}

// GetToken retrieves a token by its ID.
// Returns the token if found, or an error if not found.
func (s *Service) GetToken(ctx context.Context, id string) (*Token, error) {
	return s.store.Get(ctx, id)
}

// NewService creates a new token service with the given configuration and store.
// Returns a ServiceAPI implementation. Starts a cleanup goroutine if CleanupInterval is set.
func NewService(config *Config, store Store) ServiceAPI {
	   log.Printf("DEBUG: SigningKey in token.NewService: %v, addr: %p", config.SigningKey, config.SigningKey)
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
	log.Printf("DEBUG: SigningKey in token.Service.Issue: %v, addr: %p", s.config.SigningKey, s.config.SigningKey)
	if s.config.SigningKey == nil {
		log.Printf("ERROR: SigningKey is nil in token.Service.Issue")
	}

	// Sign the token
	sig, err := s.signToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}
	token.Value = sig

       // Store the token
       if err := s.store.Save(ctx, token.ID, token); err != nil {
	       return nil, fmt.Errorf("failed to store token: %w", err)
       }

	return token, nil
}

// Validate checks if a token is valid
func (s *Service) Validate(ctx context.Context, token *Token) error {
	// Check signature
	if err := s.verifySignature(token); err != nil {
		return NewValidationErrorWithCause(ValidationCodeInvalidSignature, "invalid token signature", err)
	}

	// Check expiry
	if time.Now().After(token.ExpiresAt) {
		return NewValidationError(ValidationCodeExpired, "token has expired")
	}

	// Check not before
	if time.Now().Before(token.NotBefore) {
		return NewValidationError(ValidationCodeNotYetValid, "token not yet valid")
	}

	// Check issuer
	if s.config.ValidateIssuer {
		validIssuer := false
		for _, iss := range s.config.AllowedIssuers {
			if token.Issuer == iss {
				validIssuer = true
				break
			}
		}
		if !validIssuer {
			return NewValidationError(ValidationCodeInvalidIssuer, "token issuer not allowed")
		}
	}

	// Check audience
	if s.config.ValidateAudience {
		validAudience := false
		for _, aud := range token.Audience {
			for _, allowed := range s.config.AllowedAudiences {
				if aud == allowed {
					validAudience = true
					break
				}
			}
			if validAudience {
				break
			}
		}
		if !validAudience {
			return NewValidationError(ValidationCodeInvalidAudience, "token audience not allowed")
		}
	}

	// Check if revoked
	stored, err := s.store.Get(ctx, token.ID)
	if err != nil {
		if err == ErrTokenNotFound {
			return NewValidationError(ValidationCodeRevoked, "token has been revoked")
		}
		return NewValidationErrorWithCause(ValidationCodeStorageFailure, "failed to verify token status", err)
	}

	// Compare with stored token
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
       id, err := GenerateID()
       if err != nil {
	       return nil, fmt.Errorf("failed to generate token ID: %w", err)
       }
       accessToken := &Token{
	       ID:        id,
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

func (s *Service) verifySignature(token *Token) error {
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
func GenerateID() (string, error) {
       b := make([]byte, 16)
       _, err := rand.Read(b)
       if err != nil {
	       return "", err
       }
       return base64.URLEncoding.EncodeToString(b), nil
}

//
// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.
