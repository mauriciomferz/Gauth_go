package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	token "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"
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
	service   *token.Service
	store     token.Store
	blacklist token.Blacklist
	Validator token.ValidationChain
}

func NewOAuth2Flow() *OAuth2Flow {
	store := token.NewMemoryStore()
	blacklist := token.NewBlacklist()
	config := &token.Config{
		DefaultTTL:     time.Hour,
		MaxTTL:         24 * time.Hour,
		SigningKey:     []byte("dummy-signing-key"),
		Algorithm:      token.RS256,
		ValidateExpiry: true,
	}
	service := token.NewService(store, blacklist, config)
	validator := token.NewValidationChain(token.ValidationConfig{
		CheckExpiry:    true,
		CheckBlacklist: true,
		CheckSignature: true,
		MaxAge:         time.Hour,
	})
	return &OAuth2Flow{
		service:   service,
		store:     store,
		blacklist: blacklist,
		Validator: validator,
	}
}

func (f *OAuth2Flow) AuthorizationCodeFlow(ctx context.Context, clientID, userID string, scopes []string) (string, string, error) {
	accessToken := &token.Token{
		ID:        token.GenerateID(),
		Value:     "valid-oauth2-access-token-value-12345",
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
		Value:     "valid-oauth2-refresh-token-value-12345",
		Type:      token.Refresh,
		Subject:   userID,
		Issuer:    "oauth2-server",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Scopes:    []string{"refresh"},
		Audience:  []string{"example-client"},
		Metadata: &token.Metadata{
			AppData: map[string]interface{}{
				"original_scopes": joinScopes(scopes),
			},
		},
	}

	// Save tokens to the store so tests can retrieve them
	if err := f.store.Save(ctx, accessToken.ID, accessToken); err != nil {
		return "", "", err
	}
	if err := f.store.Save(ctx, refreshToken.ID, refreshToken); err != nil {
		return "", "", err
	}
	return accessToken.ID, refreshToken.ID, nil
}

func (f *OAuth2Flow) RefreshTokenFlow(ctx context.Context, refreshTokenID string) (string, error) {
	// Simulate refresh: create a new access token and save it
	refreshTokenObj, err := f.store.Get(ctx, refreshTokenID)
	if err != nil {
		return "", err
	}
	userID := refreshTokenObj.Subject
	// Restore original scopes from refresh token metadata if present
	scopes := []string{"read"}
	if refreshTokenObj.Metadata != nil && refreshTokenObj.Metadata.AppData != nil {
		if orig, ok := refreshTokenObj.Metadata.AppData["original_scopes"]; ok {
			if str, ok := orig.(string); ok {
				// Split comma-separated scopes
				var parsed []string
				for _, s := range splitAndTrim(str, ",") {
					if s != "" {
						parsed = append(parsed, s)
					}
				}
				if len(parsed) > 0 {
					scopes = parsed
				}
			}
		}
	}
	newAccessToken := &token.Token{
		ID:        token.GenerateID(),
		Value:     "valid-oauth2-access-token-value-67890",
		Type:      token.Access,
		Subject:   userID,
		Issuer:    "oauth2-server",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    scopes,
		Audience:  []string{"example-client"},
	}
	if err := f.store.Save(ctx, newAccessToken.ID, newAccessToken); err != nil {
		return "", err
	}
	return newAccessToken.ID, nil
}

// splitAndTrim splits a string by sep and trims whitespace from each part
func splitAndTrim(s, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		out = append(out, trimmed)
	}
	return out
}

func main() {
	fmt.Println("OAuth2 token management example loaded.")
}
