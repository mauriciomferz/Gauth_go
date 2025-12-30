// Package gauth - RFC-0111 Subscription Flow (Steps I-VIII)
// This implements the one-off enrollment process that was MISSING from the implementation
package gauth

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// PowerVerificationPoint interface defines identity verification operations
// This avoids import cycle with verification package
type PowerVerificationPoint interface {
	VerifyIdentityProof(ctx context.Context, request *IdentityProofRequest) (*IdentityProofResult, error)
}

// IdentityProofRequest represents an identity proof request
type IdentityProofRequest struct {
	SubjectID     string
	IdentityType  string // "natural_person", "legal_entity"
	ProofMethod   string // "eIDAS", "government_id", "commercial_register"
	ProofData     map[string]interface{}
	RequiredLevel string // "substantial", "high"
}

// IdentityProofResult represents an identity proof result
type IdentityProofResult struct {
	Valid         bool
	SubjectID     string
	Identity      string
	VerifiedAt    time.Time
	TrustLevel    string
	FailureReason string
}

// SubscriptionFlowManager manages RFC-0111 Steps I-VIII (ONE-OFF SUBSCRIPTION)
type SubscriptionFlowManager struct {
	pvpClient           PowerVerificationPoint
	pipClient           PIPClient
	commercialRegClient CommercialRegisterClient
	authChainValidator  *AuthorizationChainValidator
	formalReqValidator  *FormalRequirementsValidator
	subscriptionStore   SubscriptionStore
}

// SubscriptionStatus tracks subscription flow progress
type SubscriptionStatus string

const (
	SubscriptionStatusPending             SubscriptionStatus = "pending"
	SubscriptionStatusAwaitingIdentity    SubscriptionStatus = "awaiting_identity"
	SubscriptionStatusAwaitingAuthProof   SubscriptionStatus = "awaiting_auth_proof"
	SubscriptionStatusAwaitingClientOwner SubscriptionStatus = "awaiting_client_owner"
	SubscriptionStatusAwaitingClient      SubscriptionStatus = "awaiting_client"
	SubscriptionStatusAwaitingResource    SubscriptionStatus = "awaiting_resource"
	SubscriptionStatusCompleted           SubscriptionStatus = "completed"
	SubscriptionStatusFailed              SubscriptionStatus = "failed"
)

// Subscription represents a complete RFC-0111 subscription
type Subscription struct {
	ID        string
	Status    SubscriptionStatus
	CreatedAt time.Time
	UpdatedAt time.Time

	// Step I: Owner's Authorizer Identity Proof
	OwnersAuthorizerIdentity *IdentityProofResult

	// Step II: Owner's Authorizer Authorization Proof
	CommercialRegisterEntry *CompanyInfo
	AuthorizationProof      *AuthorizationProof

	// Step III: Client Owner Identity Proof
	ClientOwnerIdentity *IdentityProofResult

	// Step IV: Client Owner Authorization Proof
	ClientOwnerAuthProof *AuthorizationProof

	// Step V: Client Authorization
	ClientAuthorizationGrant *ClientAuthGrant

	// Step VI: Resource Owner Identity Proof
	ResourceOwnerIdentity *IdentityProofResult

	// Step VII: Resource Owner Authorization Proof
	ResourceOwnerAuthProof *AuthorizationProof

	// Step VIII: Resource Server Authorization
	ResourceServerAuth *ResourceServerAuthorization

	// Complete authorization chain
	AuthorizationChain *AuthorizationChain
}

// AuthorizationProof represents proof of authorization from commercial register
type AuthorizationProof struct {
	CommercialRegisterRef string
	ProofType             string // "managing_director", "power_of_attorney", "statutory_authority"
	DocumentRef           string
	VerifiedAt            time.Time
	VerifiedBy            string
	RegistrationNumber    string
	Jurisdiction          string
}

// ClientAuthGrant represents Step V client authorization
type ClientAuthGrant struct {
	ClientID           string
	ClientOwnerID      string
	AuthorizedAt       time.Time
	AuthorizationScope *poa.AuthorizationScope
	PoACredential      *poa.PoADefinition
	IdentityShared     bool
	PromptingEnabled   bool
}

// ResourceServerAuthorization represents Step VIII
type ResourceServerAuthorization struct {
	ServerID          string
	ServerEndpoint    string
	AuthorizedAt      time.Time
	ResourceTypes     []string
	AllowedOperations []string
}

// NewSubscriptionFlowManager creates a new subscription flow manager
func NewSubscriptionFlowManager(
	pvpClient PowerVerificationPoint,
	pipClient PIPClient,
	commercialRegClient CommercialRegisterClient,
	authChainValidator *AuthorizationChainValidator,
	formalReqValidator *FormalRequirementsValidator,
	subscriptionStore SubscriptionStore,
) *SubscriptionFlowManager {
	return &SubscriptionFlowManager{
		pvpClient:           pvpClient,
		pipClient:           pipClient,
		commercialRegClient: commercialRegClient,
		authChainValidator:  authChainValidator,
		formalReqValidator:  formalReqValidator,
		subscriptionStore:   subscriptionStore,
	}
}

// InitiateSubscription starts a new RFC-0111 subscription flow
func (m *SubscriptionFlowManager) InitiateSubscription(ctx context.Context) (*Subscription, error) {
	sub := &Subscription{
		ID:        generateSubscriptionID(),
		Status:    SubscriptionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.subscriptionStore.CreateSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return sub, nil
}

// ExecuteStepI performs Step I: Owner's Authorizer Identity Proof
// RFC-0111: "The owner's authorizer, who is authorized to act on behalf of the client
// owner, proves their identity to the authorization server"
func (m *SubscriptionFlowManager) ExecuteStepI(
	ctx context.Context,
	subscriptionID string,
	identityProofRequest *IdentityProofRequest,
) error {
	// Verify identity through PVP (Trust Service Provider)
	proof, err := m.pvpClient.VerifyIdentityProof(ctx, identityProofRequest)
	if err != nil {
		return fmt.Errorf("step I failed: identity verification failed: %w", err)
	}

	if !proof.Valid {
		return &AgentAuthError{
			Code:    "step_i_identity_invalid",
			Message: fmt.Sprintf("Owner's authorizer identity could not be verified: %s", proof.FailureReason),
		}
	}

	// Update subscription
	sub, err := m.subscriptionStore.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	sub.OwnersAuthorizerIdentity = proof
	sub.Status = SubscriptionStatusAwaitingAuthProof
	sub.UpdatedAt = time.Now()

	return m.subscriptionStore.SaveSubscription(ctx, sub)
}

// ExecuteStepII performs Step II: Owner's Authorizer Authorization Proof
// RFC-0111: "The owner's authorizer proves their authority to the authorization server,
// e.g., via a commercial register entry"
func (m *SubscriptionFlowManager) ExecuteStepII(
	ctx context.Context,
	subscriptionID string,
	commercialRegisterRef string,
	jurisdiction string,
) error {
	sub, err := m.subscriptionStore.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if sub.OwnersAuthorizerIdentity == nil {
		return &AgentAuthError{
			Code:    "step_ii_prerequisite_failed",
			Message: "Step I must be completed before Step II",
		}
	}

	// Check if Step II has already been completed
	if sub.AuthorizationProof != nil {
		return &AgentAuthError{
			Code:    "step_ii_already_completed",
			Message: "Step II has already been completed for this subscription",
		}
	}

	// Verify authorization through commercial register
	entry, err := m.commercialRegClient.VerifyCompany(ctx, jurisdiction, commercialRegisterRef)
	if err != nil {
		return fmt.Errorf("step II failed: commercial register lookup failed: %w", err)
	}

	if !entry.Active {
		return &AgentAuthError{
			Code:    "step_ii_register_not_active",
			Message: "Commercial register entry not active",
		}
	}

	// Verify the owner's authorizer is actually authorized
	if !m.verifyAuthorizerInRegister(sub.OwnersAuthorizerIdentity, entry) {
		return &AgentAuthError{
			Code:    "step_ii_authorization_invalid",
			Message: "Owner's authorizer is not listed in commercial register",
		}
	}

	sub.CommercialRegisterEntry = entry
	sub.AuthorizationProof = &AuthorizationProof{
		CommercialRegisterRef: commercialRegisterRef,
		ProofType:             determineProofType(entry),
		DocumentRef:           fmt.Sprintf("%s/%s", entry.RegisterType, entry.RegistrationNumber),
		VerifiedAt:            time.Now(),
		VerifiedBy:            "commercial_register",
		RegistrationNumber:    entry.RegistrationNumber,
		Jurisdiction:          jurisdiction,
	}
	sub.Status = SubscriptionStatusAwaitingClientOwner
	sub.UpdatedAt = time.Now()

	return m.subscriptionStore.SaveSubscription(ctx, sub)
}

// ExecuteStepIII performs Step III: Client Owner Identity Proof
// RFC-0111: "The client owner (owner of the AI system) proves their identity
// to the authorization server"
func (m *SubscriptionFlowManager) ExecuteStepIII(
	ctx context.Context,
	subscriptionID string,
	identityProofRequest *IdentityProofRequest,
) error {
	sub, err := m.subscriptionStore.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if sub.AuthorizationProof == nil {
		return &AgentAuthError{
			Code:    "step_iii_prerequisite_failed",
			Message: "Step II must be completed before Step III",
		}
	}

	// Verify client owner identity through PVP
	proof, err := m.pvpClient.VerifyIdentityProof(ctx, identityProofRequest)
	if err != nil {
		return fmt.Errorf("step III failed: identity verification failed: %w", err)
	}

	if !proof.Valid {
		return &AgentAuthError{
			Code:    "step_iii_identity_invalid",
			Message: fmt.Sprintf("Client owner identity could not be verified: %s", proof.FailureReason),
		}
	}

	sub.ClientOwnerIdentity = proof
	sub.Status = SubscriptionStatusAwaitingClient
	sub.UpdatedAt = time.Now()

	return m.subscriptionStore.SaveSubscription(ctx, sub)
}

// ExecuteStepIV performs Step IV: Client Owner Authorization Proof
// RFC-0111: "The client owner is authorized by the owner's authorizer
// to register clients with the authorization server"
func (m *SubscriptionFlowManager) ExecuteStepIV(
	ctx context.Context,
	subscriptionID string,
	authorizationChain *AuthorizationChain,
) error {
	sub, err := m.subscriptionStore.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if sub.ClientOwnerIdentity == nil {
		return &AgentAuthError{
			Code:    "step_iv_prerequisite_failed",
			Message: "Step III must be completed before Step IV",
		}
	}

	// Validate authorization chain from owner's authorizer to client owner
	chainResult, err := m.authChainValidator.ValidateAuthorizationChain(ctx, authorizationChain)
	if err != nil {
		return fmt.Errorf("step IV failed: authorization chain validation failed: %w", err)
	}

	if !chainResult.Valid {
		return &AgentAuthError{
			Code:    "step_iv_chain_invalid",
			Message: fmt.Sprintf("Authorization chain validation failed: %s", chainResult.FailureReason),
		}
	}

	// Verify chain links owner's authorizer to client owner
	if !m.verifyChainConnectsParties(authorizationChain, sub.OwnersAuthorizerIdentity.SubjectID, sub.ClientOwnerIdentity.SubjectID) {
		return &AgentAuthError{
			Code:    "step_iv_chain_incomplete",
			Message: "Authorization chain does not connect owner's authorizer to client owner",
		}
	}

	sub.ClientOwnerAuthProof = &AuthorizationProof{
		ProofType:   "authorization_chain",
		DocumentRef: authorizationChain.ChainIntegrity,
		VerifiedAt:  time.Now(),
		VerifiedBy:  "authorization_chain_validator",
	}
	sub.AuthorizationChain = authorizationChain
	sub.UpdatedAt = time.Now()

	return m.subscriptionStore.SaveSubscription(ctx, sub)
}

// ExecuteStepV performs Step V: Client Authorization
// RFC-0111: "The client owner authorizes a client (AI system) to act with the
// authorization server, including identity sharing and prompting"
func (m *SubscriptionFlowManager) ExecuteStepV(
	ctx context.Context,
	subscriptionID string,
	clientID string,
	poaCredential *poa.PoADefinition,
	enableIdentitySharing bool,
	enablePrompting bool,
) error {
	sub, err := m.subscriptionStore.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if sub.ClientOwnerAuthProof == nil {
		return &AgentAuthError{
			Code:    "step_v_prerequisite_failed",
			Message: "Step IV must be completed before Step V",
		}
	}

	// Validate PoA credential (if provided)
	if poaCredential != nil {
		if err := poa.ValidatePoADefinition(*poaCredential); err != nil {
			return fmt.Errorf("step V failed: PoA validation failed: %w", err)
		}

		// Validate formal requirements
		formalResult, err := m.formalReqValidator.ValidateFormalRequirements(
			ctx,
			poaCredential,
			nil, // Notarial certificate (if required)
			nil, // Identity documents (if required)
			nil, // Digital signatures
		)
		if err != nil {
			return fmt.Errorf("step V failed: formal requirements validation failed: %w", err)
		}

		if !formalResult.Valid {
			issuesStr := ""
			if len(formalResult.Issues) > 0 {
				issuesStr = fmt.Sprintf(": %s", formalResult.Issues[0])
			}
			return &AgentAuthError{
				Code:    "step_v_formal_requirements_failed",
				Message: fmt.Sprintf("Formal requirements validation failed%s", issuesStr),
			}
		}
	}

	grant := &ClientAuthGrant{
		ClientID:         clientID,
		ClientOwnerID:    sub.ClientOwnerIdentity.SubjectID,
		AuthorizedAt:     time.Now(),
		PoACredential:    poaCredential,
		IdentityShared:   enableIdentitySharing,
		PromptingEnabled: enablePrompting,
	}

	if poaCredential != nil {
		grant.AuthorizationScope = &poaCredential.Authorization
	}

	sub.ClientAuthorizationGrant = grant
	sub.Status = SubscriptionStatusAwaitingResource
	sub.UpdatedAt = time.Now()

	return m.subscriptionStore.SaveSubscription(ctx, sub)
}

// ExecuteStepVI performs Step VI: Resource Owner Identity Proof
// RFC-0111: "The resource owner proves their identity to the authorization server"
func (m *SubscriptionFlowManager) ExecuteStepVI(
	ctx context.Context,
	subscriptionID string,
	identityProofRequest *IdentityProofRequest,
) error {
	sub, err := m.subscriptionStore.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if sub.ClientAuthorizationGrant == nil {
		return &AgentAuthError{
			Code:    "step_vi_prerequisite_failed",
			Message: "Step V must be completed before Step VI",
		}
	}

	// Verify resource owner identity through PVP
	proof, err := m.pvpClient.VerifyIdentityProof(ctx, identityProofRequest)
	if err != nil {
		return fmt.Errorf("step VI failed: identity verification failed: %w", err)
	}

	if !proof.Valid {
		return &AgentAuthError{
			Code:    "step_vi_identity_invalid",
			Message: fmt.Sprintf("Resource owner identity could not be verified: %s", proof.FailureReason),
		}
	}

	sub.ResourceOwnerIdentity = proof
	sub.UpdatedAt = time.Now()

	return m.subscriptionStore.SaveSubscription(ctx, sub)
}

// ExecuteStepVII performs Step VII: Resource Owner Authorization Proof
// RFC-0111: "The resource owner is authorized by the owner's authorizer
// to control resources on the resource server"
func (m *SubscriptionFlowManager) ExecuteStepVII(
	ctx context.Context,
	subscriptionID string,
	authorizationChain *AuthorizationChain,
) error {
	sub, err := m.subscriptionStore.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if sub.ResourceOwnerIdentity == nil {
		return &AgentAuthError{
			Code:    "step_vii_prerequisite_failed",
			Message: "Step VI must be completed before Step VII",
		}
	}

	// Validate authorization chain from owner's authorizer to resource owner
	chainResult, err := m.authChainValidator.ValidateAuthorizationChain(ctx, authorizationChain)
	if err != nil {
		return fmt.Errorf("step VII failed: authorization chain validation failed: %w", err)
	}

	if !chainResult.Valid {
		return &AgentAuthError{
			Code:    "step_vii_chain_invalid",
			Message: fmt.Sprintf("Authorization chain validation failed: %s", chainResult.FailureReason),
		}
	}

	sub.ResourceOwnerAuthProof = &AuthorizationProof{
		ProofType:   "authorization_chain",
		DocumentRef: authorizationChain.ChainIntegrity,
		VerifiedAt:  time.Now(),
		VerifiedBy:  "authorization_chain_validator",
	}
	sub.UpdatedAt = time.Now()

	return m.subscriptionStore.SaveSubscription(ctx, sub)
}

// ExecuteStepVIII performs Step VIII: Resource Server Authorization
// RFC-0111: "The resource server is authorized to serve resources under the
// authorization server's governance"
func (m *SubscriptionFlowManager) ExecuteStepVIII(
	ctx context.Context,
	subscriptionID string,
	serverID string,
	serverEndpoint string,
	resourceTypes []string,
	allowedOperations []string,
) error {
	sub, err := m.subscriptionStore.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	if sub.ResourceOwnerAuthProof == nil {
		return &AgentAuthError{
			Code:    "step_viii_prerequisite_failed",
			Message: "Step VII must be completed before Step VIII",
		}
	}

	sub.ResourceServerAuth = &ResourceServerAuthorization{
		ServerID:          serverID,
		ServerEndpoint:    serverEndpoint,
		AuthorizedAt:      time.Now(),
		ResourceTypes:     resourceTypes,
		AllowedOperations: allowedOperations,
	}

	sub.Status = SubscriptionStatusCompleted
	sub.UpdatedAt = time.Now()

	return m.subscriptionStore.SaveSubscription(ctx, sub)
}

// GetSubscriptionStatus returns the current status of a subscription
func (m *SubscriptionFlowManager) GetSubscriptionStatus(ctx context.Context, subscriptionID string) (*Subscription, error) {
	return m.subscriptionStore.GetSubscription(ctx, subscriptionID)
}

// Helper functions

func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

func determineProofType(entry *CompanyInfo) string {
	// Simplified logic - determine proof type from company info
	if len(entry.ManagingDirectors) > 0 {
		return "managing_director"
	}
	if len(entry.AuthorizedSignatories) > 0 {
		return "power_of_attorney"
	}
	return "statutory_authority"
}

func (m *SubscriptionFlowManager) verifyAuthorizerInRegister(
	identity *IdentityProofResult,
	entry *CompanyInfo,
) bool {
	// Verify that the identity matches an authorized person in the register
	for _, director := range entry.ManagingDirectors {
		if director.PersonID == identity.SubjectID {
			return true
		}
	}
	for _, signatory := range entry.AuthorizedSignatories {
		if signatory.PersonID == identity.SubjectID {
			return true
		}
	}
	return false
}

func (m *SubscriptionFlowManager) verifyChainConnectsParties(
	chain *AuthorizationChain,
	fromPartyID string,
	toPartyID string,
) bool {
	// Verify chain connects fromParty to toParty through authorization links
	// Check if OwnersAuthorizer matches fromParty
	if chain.OwnersAuthorizer != nil && chain.OwnersAuthorizer.EntityID == fromPartyID {
		// Check if the chain reaches toParty
		if chain.ClientOwner != nil && chain.ClientOwner.EntityID == toPartyID {
			return true
		}
		if chain.Client != nil && chain.Client.EntityID == toPartyID {
			return true
		}
	}

	return false
}
