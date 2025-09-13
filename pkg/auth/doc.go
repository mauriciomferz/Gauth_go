// Package auth provides authentication functionality for the GAuth framework.
//
// The auth package implements the core GAuth authentication components including:
//
//   - Authentication mechanisms for verifying identity
//   - Credentials management and secure storage
//   - Claims handling and validation
//   - Authentication services and integrations
//   - Legal and compliance frameworks
//   - Multi-factor authentication support
//
// # Basic Usage
//
// To use the auth package in your application:
//
//	import "github.com/Gimel-Foundation/gauth/pkg/auth"
//
//	// Create auth service with options
//	authService := auth.NewAuthService(auth.Options{
//	    TokenService: tokenService,
//	    Store:        store,
//	})
//
//	// Generate token
//	token, err := auth.GenerateToken(ctx, auth.Claims{
//		Subject: "user123",
//		Scopes: []string{"read", "write"},
//	})
//
//	// Validate token
//	claims, err := auth.ValidateToken(ctx, token)
//
// Thread Safety:
//
// All public methods are thread-safe and can be called concurrently.
// The package uses internal synchronization to protect shared resources.
//
// Error Handling:
//
// The package uses strongly typed errors that can be checked using errors.Is():
//
//	var ErrInvalidToken = errors.New("invalid token")
//	var ErrExpiredToken = errors.New("token expired")
//	var ErrInvalidClaims = errors.New("invalid claims")
//
// Configuration:
//
// The package can be configured using the Config struct:
//
//	type Config struct {
//		SigningKey    []byte
//		TokenExpiry   time.Duration
//		RefreshExpiry time.Duration
//		ClockSkew     time.Duration
//	}
//
// Best Practices:
//
//   - Use short-lived access tokens (1 hour or less)
//   - Implement token rotation for long-running sessions
//   - Always validate tokens before granting access
//   - Use scopes to limit token permissions
//   - Store tokens securely using the provided Store interface
package auth
