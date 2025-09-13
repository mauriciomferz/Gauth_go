package tokenmanagement

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// OAuth2Flow demonstrates token management in OAuth2 flows
type OAuth2Flow struct {
	jwtManager *token.JWTManager
	store      token.Store
	blacklist  *token.Blacklist
	validator  *token.ValidationChain
	rotator    *token.Rotator
}

func NewOAuth2Flow() *OAuth2Flow {
	store := token.NewMemoryStore(24 * time.Hour)
	blacklist := token.NewBlacklist()

	return &OAuth2Flow{
		store:     store,
		blacklist: blacklist,
		jwtManager: token.NewJWTManager(token.JWTConfig{
			SigningMethod: token.RS256,
			SigningKey:    loadPrivateKey(), // Implementation omitted
			KeyID:         "oauth-key-1",
			Audience:      []string{"example-client"},
			MaxAge:        time.Hour,
		}),
		validator: token.NewValidationChain(blacklist),
		rotator:   token.NewRotator(store, blacklist, token.Config{DefaultExpiry: time.Hour}),
	}
}

func (f *OAuth2Flow) AuthorizationCodeFlow(ctx context.Context, clientID, userID string, scopes []string) (string, string, error) {
	// Create access token
	accessToken := &token.Token{
		ID:        token.NewID(),
		Type:      token.Access,
		Subject:   userID,
		Issuer:    "oauth2-server",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    scopes,
		Metadata: map[string]string{
			"client_id":  clientID,
			"grant_type": "authorization_code",
		},
	}

	// Create refresh token
	refreshToken := &token.Token{
		ID:        token.NewID(),
		Type:      token.Refresh,
		Subject:   userID,
		Issuer:    "oauth2-server",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Scopes:    []string{"refresh"},
		Metadata: map[string]string{
			"client_id":  clientID,
			"grant_type": "authorization_code",
		},
	}

	// Store tokens
	if err := f.store.Save(ctx, accessToken); err != nil {
		return "", "", fmt.Errorf("failed to save access token: %w", err)
	}
	if err := f.store.Save(ctx, refreshToken); err != nil {
		return "", "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	// Sign tokens
	signedAccess, err := f.jwtManager.SignToken(ctx, accessToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	signedRefresh, err := f.jwtManager.SignToken(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signedAccess, signedRefresh, nil
}

func (f *OAuth2Flow) RefreshTokenFlow(ctx context.Context, refreshTokenStr string) (string, error) {
	// Verify refresh token
	refreshToken, err := f.jwtManager.VerifyToken(ctx, refreshTokenStr)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Validate refresh token
	if err := f.validator.Validate(ctx, refreshToken); err != nil {
		return "", fmt.Errorf("refresh token validation failed: %w", err)
	}

	// Create new access token
	accessToken := &token.Token{
		ID:        token.NewID(),
		Type:      token.Access,
		Subject:   refreshToken.Subject,
		Issuer:    refreshToken.Issuer,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    refreshToken.Metadata["original_scopes"],
		Metadata:  refreshToken.Metadata,
	}

	// Store and sign new access token
	if err := f.store.Save(ctx, accessToken); err != nil {
		return "", fmt.Errorf("failed to save new access token: %w", err)
	}

	signedAccess, err := f.jwtManager.SignToken(ctx, accessToken)
	if err != nil {
		return "", fmt.Errorf("failed to sign new access token: %w", err)
	}

	return signedAccess, nil
}

func main() {
	ctx := context.Background()
	flow := NewOAuth2Flow()

	// Simulate authorization code flow
	fmt.Println("1. Authorization Code Flow:")
	accessToken, refreshToken, err := flow.AuthorizationCodeFlow(
		ctx,
		"example-client",
		"user123",
		[]string{"profile", "email"},
	)
	if err != nil {
		log.Fatalf("Authorization code flow failed: %v", err)
	}
	fmt.Printf("Access Token: %s\n", accessToken)
	fmt.Printf("Refresh Token: %s\n\n", refreshToken)

	// Simulate token refresh
	fmt.Println("2. Refresh Token Flow:")
	newAccessToken, err := flow.RefreshTokenFlow(ctx, refreshToken)
	if err != nil {
		log.Fatalf("Refresh flow failed: %v", err)
	}
	fmt.Printf("New Access Token: %s\n", newAccessToken)

	// Verify new access token
	verified, err := flow.jwtManager.VerifyToken(ctx, newAccessToken)
	if err != nil {
		log.Fatalf("Token verification failed: %v", err)
	}
	fmt.Printf("\nVerified token for subject: %s\n", verified.Subject)
	fmt.Printf("Scopes: %v\n", verified.Scopes)
}

// Placeholder for key loading
func loadPrivateKey() interface{} {
	return []byte("test-key") // In practice, load RSA key from secure storage
}
