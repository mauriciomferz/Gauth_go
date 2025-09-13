package auth

import (
	"context"
	"testing"
	"time"
)

func BenchmarkBasicAuth(b *testing.B) {
	config := Config{
		Type:              TypeBasic,
		AccessTokenExpiry: time.Hour,
	}

	auth, _ := NewAuthenticator(config)
	basicAuth := auth.(*basicAuthenticator)
	basicAuth.AddClient("testuser", "testpass")

	ctx := context.Background()

	b.Run("ValidateCredentials", func(b *testing.B) {
		creds := basicCredentials{
			Username: "testuser",
			Password: "testpass",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			auth.ValidateCredentials(ctx, creds)
		}
	})

	b.Run("GenerateToken", func(b *testing.B) {
		req := TokenRequest{
			Subject: "testuser",
			Scopes:  []string{"read", "write"},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			auth.GenerateToken(ctx, req)
		}
	})

	b.Run("ValidateToken", func(b *testing.B) {
		req := TokenRequest{
			Subject: "testuser",
			Scopes:  []string{"read", "write"},
		}
		resp, _ := auth.GenerateToken(ctx, req)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			auth.ValidateToken(ctx, resp.Token)
		}
	})
}

func BenchmarkJWTAuth(b *testing.B) {
	config := Config{
		Type:              TypeJWT,
		ClientID:          "testclient",
		ClientSecret:      "testsecret",
		AccessTokenExpiry: time.Hour,
		TokenValidation: TokenValidationConfig{
			ValidateSignature: true,
			AllowedIssuers:    []string{"testclient"},
			AllowedAudiences:  []string{"testapp"},
			RequiredScopes:    []string{"read"},
		},
	}

	auth, _ := NewAuthenticator(config)
	ctx := context.Background()

	b.Run("GenerateToken", func(b *testing.B) {
		req := TokenRequest{
			Subject:  "testuser",
			Audience: "testapp",
			Scopes:   []string{"read", "write"},
			Metadata: map[string]interface{}{
				"role": "admin",
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			auth.GenerateToken(ctx, req)
		}
	})

	b.Run("ValidateToken", func(b *testing.B) {
		req := TokenRequest{
			Subject:  "testuser",
			Audience: "testapp",
			Scopes:   []string{"read", "write"},
			Metadata: map[string]interface{}{
				"role": "admin",
			},
		}
		resp, _ := auth.GenerateToken(ctx, req)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			auth.ValidateToken(ctx, resp.Token)
		}
	})

	b.Run("ValidateToken_WithFullValidation", func(b *testing.B) {
		req := TokenRequest{
			Subject:  "testuser",
			Audience: "testapp",
			Scopes:   []string{"read", "write"},
			Metadata: map[string]interface{}{
				"role": "admin",
			},
		}
		resp, _ := auth.GenerateToken(ctx, req)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			auth.ValidateToken(ctx, resp.Token)
		}
	})
}
