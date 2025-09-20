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
			   creds := BasicCredentials{
			Username: "testuser",
			Password: "testpass",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			   if err := auth.ValidateCredentials(ctx, creds); err != nil {
				   b.Fatalf("ValidateCredentials failed: %v", err)
			   }
		}
	})

	b.Run("GenerateToken", func(b *testing.B) {
		req := TokenRequest{
			Subject: "testuser",
			Scopes:  []string{"read", "write"},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			   if _, err := auth.GenerateToken(ctx, req); err != nil {
				   b.Fatalf("GenerateToken failed: %v", err)
			   }
		}
	})

	b.Run("ValidateToken", func(b *testing.B) {
		req := TokenRequest{
			Subject: "testuser",
			Scopes:  []string{"read", "write"},
		}
		resp, err := auth.GenerateToken(ctx, req)
		if err != nil {
			b.Fatalf("GenerateToken failed: %v", err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			   if _, err := auth.ValidateToken(ctx, resp.Token); err != nil {
				   b.Fatalf("ValidateToken failed: %v", err)
			   }
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
			   if _, err := auth.GenerateToken(ctx, req); err != nil {
				   b.Fatalf("GenerateToken failed: %v", err)
			   }
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
		resp, err := auth.GenerateToken(ctx, req)
		if err != nil {
			b.Fatalf("GenerateToken failed: %v", err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			   if _, err := auth.ValidateToken(ctx, resp.Token); err != nil {
				   b.Fatalf("ValidateToken failed: %v", err)
			   }
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
		resp, err := auth.GenerateToken(ctx, req)
		if err != nil {
			b.Fatalf("GenerateToken failed: %v", err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			   if _, err := auth.ValidateToken(ctx, resp.Token); err != nil {
				   b.Fatalf("ValidateToken failed: %v", err)
			   }
		}
	})
}
