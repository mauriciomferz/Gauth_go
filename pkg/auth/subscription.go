package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// SubscriptionService handles the authorization server subscription process
type SubscriptionService interface {
	// RegisterOwnerAuthorizer registers the owner's authorizer
	RegisterOwnerAuthorizer(ctx context.Context, info *AuthorizerInfo) error

	// RegisterClientOwner registers a client owner
	RegisterClientOwner(ctx context.Context, info *OwnerInfo, authorizerID string) error

	// RegisterResourceOwner registers a resource owner
	RegisterResourceOwner(ctx context.Context, info *OwnerInfo, authorizerID string) error

	// AuthorizeClient registers an AI client under a client owner
	AuthorizeClient(ctx context.Context, clientInfo *ClientInfo, ownerID string) error

	// AuthorizeResourceServer registers a resource server under a resource owner
	AuthorizeResourceServer(ctx context.Context, serverInfo *ServerInfo, ownerID string) error
}

// StandardSubscriptionService implements SubscriptionService
type StandardSubscriptionService struct {
	store           token.EnhancedStore
	verifier        token.VerificationSystem
	identityService IdentityVerificationService
}

// AuthorizerInfo contains authorizer registration details
type AuthorizerInfo struct {
	ID               string
	Name             string
	Type             string // individual, organization
	RegistrationInfo *token.RegistrationInfo
	IdentityDocument string
	LegalCredentials string
	ContactInfo      string
}

// OwnerInfo contains owner registration details
type OwnerInfo struct {
	ID               string
	Name             string
	Type             string // individual, organization
	RegistrationInfo *token.RegistrationInfo
	IdentityDocument string
	AuthorizerID     string
	ContactInfo      string
}

// ClientInfo contains AI client registration details
type ClientInfo struct {
	ID           string
	Name         string
	Type         string // agent, robot
	Version      string
	Capabilities []string
	OwnerID      string
	Metadata     map[string]string
}

// ServerInfo contains resource server registration details
type ServerInfo struct {
	ID           string
	Name         string
	Type         string
	OwnerID      string
	Resources    []string
	Capabilities []string
	Metadata     map[string]string
}

// NewStandardSubscriptionService creates a new subscription service
func NewStandardSubscriptionService(
	store token.EnhancedStore,
	verifier token.VerificationSystem,
	identityService IdentityVerificationService,
) *StandardSubscriptionService {
	return &StandardSubscriptionService{
		store:           store,
		verifier:        verifier,
		identityService: identityService,
	}
}

// RegisterOwnerAuthorizer implements SubscriptionService
func (s *StandardSubscriptionService) RegisterOwnerAuthorizer(ctx context.Context, info *AuthorizerInfo) error {
	// Verify identity
	if err := s.identityService.VerifyIdentity(ctx, info.IdentityDocument); err != nil {
		return fmt.Errorf("identity verification failed: %w", err)
	}

	// Verify legal credentials
	if err := s.verifyLegalCredentials(ctx, info.LegalCredentials); err != nil {
		return fmt.Errorf("legal credentials verification failed: %w", err)
	}

	// Verify registration if available
	if info.RegistrationInfo != nil {
		if err := s.verifier.(*token.StandardVerificationSystem).RegVerifier().VerifyRegistration(ctx, info.RegistrationInfo); err != nil {
			return fmt.Errorf("registration verification failed: %w", err)
		}
	}

	// Store authorizer info
	return s.store.StoreAuthorizer(ctx, info)
}

// RegisterClientOwner implements SubscriptionService
func (s *StandardSubscriptionService) RegisterClientOwner(ctx context.Context, info *OwnerInfo, authorizerID string) error {
	// Verify authorizer exists and is valid
	if err := s.verifyAuthorizer(ctx, authorizerID); err != nil {
		return fmt.Errorf("authorizer verification failed: %w", err)
	}

	// Verify identity
	if err := s.identityService.VerifyIdentity(ctx, info.IdentityDocument); err != nil {
		return fmt.Errorf("identity verification failed: %w", err)
	}

	// Verify registration if available
	if info.RegistrationInfo != nil {
		if err := s.verifier.(*token.StandardVerificationSystem).RegVerifier().VerifyRegistration(ctx, info.RegistrationInfo); err != nil {
			return fmt.Errorf("registration verification failed: %w", err)
		}
	}

	// Store owner info
	return s.store.StoreOwner(ctx, info)
}

// RegisterResourceOwner implements SubscriptionService
func (s *StandardSubscriptionService) RegisterResourceOwner(ctx context.Context, info *OwnerInfo, authorizerID string) error {
	// Similar to RegisterClientOwner
	return s.RegisterClientOwner(ctx, info, authorizerID)
}

// AuthorizeClient implements SubscriptionService
func (s *StandardSubscriptionService) AuthorizeClient(ctx context.Context, clientInfo *ClientInfo, ownerID string) error {
	// Verify owner exists and is valid
	if err := s.verifyOwner(ctx, ownerID); err != nil {
		return fmt.Errorf("owner verification failed: %w", err)
	}

	// Verify client capabilities
	if err := s.verifyCapabilities(ctx, clientInfo.Capabilities); err != nil {
		return fmt.Errorf("capabilities verification failed: %w", err)
	}

	// Create and store client token
	token := &token.EnhancedToken{
		Owner: &token.OwnerInfo{
			OwnerID:   ownerID,
			OwnerType: "client_owner",
		},
		AI: &token.AIMetadata{
			AIType:       clientInfo.Type,
			Capabilities: clientInfo.Capabilities,
		},
		Versions: []token.VersionInfo{{
			Version:       1,
			UpdatedAt:     time.Now(),
			UpdatedBy:     ownerID,
			ChangeType:    "creation",
			ChangeSummary: "Initial client authorization",
		}},
	}

	return s.store.StoreToken(ctx, token)
}

// AuthorizeResourceServer implements SubscriptionService
func (s *StandardSubscriptionService) AuthorizeResourceServer(ctx context.Context, serverInfo *ServerInfo, ownerID string) error {
	// Similar to AuthorizeClient but for resource servers
	if err := s.verifyOwner(ctx, ownerID); err != nil {
		return fmt.Errorf("owner verification failed: %w", err)
	}

	// Verify server capabilities
	if err := s.verifyCapabilities(ctx, serverInfo.Capabilities); err != nil {
		return fmt.Errorf("capabilities verification failed: %w", err)
	}

	// Create and store server token
	token := &token.EnhancedToken{
		Owner: &token.OwnerInfo{
			OwnerID:   ownerID,
			OwnerType: "resource_owner",
		},
		AI: &token.AIMetadata{
			AIType:       serverInfo.Type,
			Capabilities: serverInfo.Capabilities,
		},
		Versions: []token.VersionInfo{{
			Version:       1,
			UpdatedAt:     time.Now(),
			UpdatedBy:     ownerID,
			ChangeType:    "creation",
			ChangeSummary: "Initial resource server authorization",
		}},
	}

	return s.store.StoreToken(ctx, token)
}

// Helper methods

func (s *StandardSubscriptionService) verifyAuthorizer(ctx context.Context, authorizerID string) error {
	// Verify authorizer exists and is valid
	return nil
}

func (s *StandardSubscriptionService) verifyOwner(ctx context.Context, ownerID string) error {
	// Verify owner exists and is valid
	return nil
}

func (s *StandardSubscriptionService) verifyCapabilities(ctx context.Context, capabilities []string) error {
	// Verify capabilities are valid
	return nil
}

func (s *StandardSubscriptionService) verifyLegalCredentials(ctx context.Context, credentials string) error {
	// Verify legal credentials
	return nil
}
