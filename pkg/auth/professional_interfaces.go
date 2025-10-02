// Professional interfaces for mesh integration
// This file provides professional interfaces that replace the amateur ones

package auth

import (
	"context"
	"time"
)

// ProfessionalAuthService defines the interface for professional authentication
// This replaces the amateur auth.Authenticator interface
type ProfessionalAuthService interface {
	// CreateToken creates a JWT token for a user with specified scopes
	CreateToken(userID string, scopes []string, duration time.Duration) (string, error)
	
	// ValidateToken validates a JWT token and returns claims
	ValidateToken(tokenString string) (*CustomClaims, error)
	
	// ValidateServiceToken validates service-to-service authentication
	ValidateServiceToken(ctx context.Context, token string) (*ServiceClaims, error)
}

// ServiceClaims represents claims for service-to-service authentication
type ServiceClaims struct {
	*CustomClaims
	ServiceID string `json:"service_id"`
	Mesh      string `json:"mesh"`
}

// ProfessionalConfig defines configuration for professional authentication
// This replaces the amateur auth.Config type
type ProfessionalConfig struct {
	// JWT Configuration
	Issuer      string        `json:"issuer"`
	Audience    string        `json:"audience"`
	TokenExpiry time.Duration `json:"token_expiry"`
	
	// Service Configuration
	ServiceID string `json:"service_id"`
	MeshID    string `json:"mesh_id"`
	
	// Security Configuration
	UseSecureDefaults bool `json:"use_secure_defaults"`
}

// NewProfessionalAuthService creates a new professional authentication service
// This replaces the amateur auth.NewAuthenticator function
func NewProfessionalAuthService(config ProfessionalConfig) (ProfessionalAuthService, error) {
	return NewProfessionalAuthServiceAdapter(config)
}

// professionalAuthServiceAdapter adapts ProperJWTService to ProfessionalAuthService
type professionalAuthServiceAdapter struct {
	*ProperJWTService
	config ProfessionalConfig
}

// ValidateServiceToken implements service-to-service token validation
func (p *professionalAuthServiceAdapter) ValidateServiceToken(ctx context.Context, token string) (*ServiceClaims, error) {
	claims, err := p.ValidateToken(token)
	if err != nil {
		return nil, err
	}
	
	// Create service claims from custom claims
	serviceClaims := &ServiceClaims{
		CustomClaims: claims,
		ServiceID:    p.config.ServiceID,
		Mesh:         p.config.MeshID,
	}
	
	return serviceClaims, nil
}

// NewProfessionalAuthServiceAdapter creates an adapter that implements all professional interfaces
func NewProfessionalAuthServiceAdapter(config ProfessionalConfig) (ProfessionalAuthService, error) {
	jwtService, err := NewProperJWTService(config.Issuer, config.Audience)
	if err != nil {
		return nil, err
	}
	
	return &professionalAuthServiceAdapter{
		ProperJWTService: jwtService,
		config:          config,
	}, nil
}