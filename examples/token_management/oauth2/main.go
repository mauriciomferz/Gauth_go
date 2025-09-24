package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"time"

	token "github.com/Gimel-Foundation/gauth/pkg/token"
)

// joinScopes joins a slice of scopes into a comma-separated string
func joinScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	result := scopes[0]
	for _, s := range scopes[1:] {
		result += "," + s
	}
	return result
}

type OAuth2Flow struct {
	service   token.ServiceAPI
	store     token.Store
	blacklist *token.Blacklist
	Validator *token.ValidationChain // Exported for test access
	rotator   *token.Rotator
}

func NewOAuth2Flow() *OAuth2Flow {
	store := token.NewMemoryStore()
	blacklist := token.NewBlacklist()
	// Generate a test RSA private key for signing
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("failed to generate test RSA key: " + err.Error())
	}
	config := token.Config{
		SigningMethod:    token.RS256,
		SigningKey:       privKey,
		ValidityPeriod:   time.Hour,
		RefreshPeriod:    24 * time.Hour,
		DefaultScopes:    []string{"read"},
		ValidateAudience: true,
		AllowedAudiences: []string{"example-client"},
		ValidateIssuer:   true,
		AllowedIssuers:   []string{"oauth2-server"},
	}
	service := token.NewService(config, store)
	validator := token.NewValidationChain(token.ValidationConfig{
		AllowedIssuers:   config.AllowedIssuers,
		AllowedAudiences: config.AllowedAudiences,
		ClockSkew:        2 * time.Minute,
	}, blacklist)
	rotator := token.NewRotator(store, blacklist, config)
	return &OAuth2Flow{
		service:   service,
		store:     store,
		blacklist: blacklist,
		Validator: validator,
		rotator:   rotator,
	}
}

func (f *OAuth2Flow) AuthorizationCodeFlow(ctx context.Context, clientID, userID string, scopes []string) (string, string, error) {
	accessToken := &token.Token{
		ID:        token.GenerateID(),
		Type:      token.Access,
		Subject:   userID,
		Issuer:    "oauth2-server",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    scopes,
		Audience:  []string{"example-client"},
	}

	refreshToken := &token.Token{
		ID:        token.GenerateID(),
		Type:      token.Refresh,
		Subject:   userID,
		Issuer:    "oauth2-server",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Scopes:    []string{"refresh"},
		Audience:  []string{"example-client"},
		Metadata: &token.Metadata{
			AppData: map[string]string{
				"original_scopes": joinScopes(scopes),
			},
		},
	}

	issuedAccess, err := f.service.Issue(ctx, accessToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to issue access token: %w", err)
	}
	issuedRefresh, err := f.service.Issue(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to issue refresh token: %w", err)
	}

	// Return token IDs for store lookup
	return issuedAccess.ID, issuedRefresh.ID, nil
}

func (f *OAuth2Flow) RefreshTokenFlow(ctx context.Context, refreshTokenID string) (string, error) {
	refreshToken, err := f.store.Get(ctx, refreshTokenID)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}
	issuedAccess, err := f.service.Refresh(ctx, refreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to refresh access token: %w", err)
	}
	return issuedAccess.ID, nil
}

func main() {
	fmt.Println("OAuth2 token management example loaded.")
}
