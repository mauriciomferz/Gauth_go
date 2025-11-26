// Package gauth - RFC-0111 Protocol Orchestrator (Steps a-i)
// This implements the REQUEST-SPECIFIC flow that connects all validation components
package gauth

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// ProtocolOrchestrator manages the complete RFC-0111 request-specific flow (steps a-i)
// This is the MISSING component that connects all validation functions
type ProtocolOrchestrator struct {
	extendedTokenService *ExtendedTokenService
	complianceValidator  *ComplianceValidator
	authChainValidator   *AuthorizationChainValidator
	formalReqValidator   *FormalRequirementsValidator
	pipClient            PIPClient
	subscriptionStore    SubscriptionStore
	complianceTracker    ComplianceTracker
}

// NewProtocolOrchestrator creates a new RFC-0111 protocol orchestrator
func NewProtocolOrchestrator(
	extendedTokenService *ExtendedTokenService,
	complianceValidator *ComplianceValidator,
	authChainValidator *AuthorizationChainValidator,
	formalReqValidator *FormalRequirementsValidator,
	pipClient PIPClient,
	subscriptionStore SubscriptionStore,
	complianceTracker ComplianceTracker,
) *ProtocolOrchestrator {
	return &ProtocolOrchestrator{
		extendedTokenService: extendedTokenService,
		complianceValidator:  complianceValidator,
		authChainValidator:   authChainValidator,
		formalReqValidator:   formalReqValidator,
		pipClient:            pipClient,
		subscriptionStore:    subscriptionStore,
		complianceTracker:    complianceTracker,
	}
}

// RFCCompliantAuthorizationRequest represents step (a): Client authorization request
type RFCCompliantAuthorizationRequest struct {
	// Client information
	ClientID      string
	ClientType    poa.ClientType
	ClientVersion string

	// Subscription reference (must have completed steps I-VIII)
	SubscriptionID string

	// Resource owner information
	ResourceOwnerID string

	// Requested authorization
	RequestedScope       *poa.AuthorizationScope
	RequestedTransaction *TransactionRequest
	RequestedDecision    *DecisionRequest
	RequestedAction      *ActionRequest

	// Power of Attorney reference
	PoACredentialRef string

	// Geographic context for scope validation (ISO 3166-1 alpha-2 or ISO 3166-2)
	Jurisdiction string

	// Additional context
	Context map[string]interface{}
}

// TransactionRequest represents a specific transaction request
type TransactionRequest struct {
	Type              poa.TransactionType
	Amount            *poa.MonetaryAmount
	Currency          string
	Counterparty      string
	Description       string
	AdditionalDetails map[string]interface{}
}

// DecisionRequest represents a specific decision request
type DecisionRequest struct {
	Type              poa.DecisionType
	Subject           string
	ImpactLevel       string // "low", "medium", "high", "critical"
	RequiresApproval  bool
	Description       string
	AdditionalDetails map[string]interface{}
}

// ActionRequest represents a specific action request
type ActionRequest struct {
	Type              string // "transaction", "decision", "action"
	IsPhysical        bool
	Location          string
	Description       string
	AdditionalDetails map[string]interface{}
}

// RFCCompliantGrantResponse represents step (c): Authorization grant issuance
type RFCCompliantGrantResponse struct {
	GrantID              string
	IssuedAt             time.Time
	ExpiresAt            time.Time
	ClientID             string
	ResourceOwnerID      string
	Scope                *poa.AuthorizationScope
	PoACredential        *poa.PoADefinition
	AuthorizationChain   *AuthorizationChain
	ComplianceValidation *RequestComplianceResult
}

// RFCCompliantTokenResponse represents step (e): Extended token issuance
type RFCCompliantTokenResponse struct {
	ExtendedToken    *ExtendedToken
	TokenType        string
	ExpiresIn        int
	Scope            *poa.AuthorizationScope
	GrantValidation  *GrantComplianceResult
	ComplianceStatus *ComplianceStatus
}

// ComplianceStatus represents ongoing compliance tracking
type ComplianceStatus struct {
	Compliant   bool
	Violations  []string
	LastChecked time.Time
	NextCheck   time.Time
}

// ExecuteRFCCompliantFlow executes the complete RFC-0111 request-specific flow (steps a-i)
// This is the MAIN METHOD that was missing - it orchestrates all validation functions
func (o *ProtocolOrchestrator) ExecuteRFCCompliantFlow(
	ctx context.Context,
	request *RFCCompliantAuthorizationRequest,
) (*RFCCompliantTokenResponse, error) {

	// STEP (a): Client Authorization Request - Already received
	// Validate basic request structure
	if err := o.validateRequestStructure(request); err != nil {
		return nil, fmt.Errorf("step (a) failed: %w", err)
	}

	// Verify subscription is complete (steps I-VIII must be done)
	subscription, err := o.subscriptionStore.GetSubscription(ctx, request.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("step (a) failed: subscription not found: %w", err)
	}

	if subscription.Status != SubscriptionStatusCompleted {
		return nil, &GAuthError{
			Code:    "subscription_incomplete",
			Message: fmt.Sprintf("Subscription must be completed (current status: %s)", subscription.Status),
		}
	}

	// STEP (b): Request Compliance Validation
	// THIS IS WHERE WE ACTUALLY CALL ValidateRequestCompliance()
	requestedActions := make([]string, 0)
	if request.RequestedTransaction != nil {
		requestedActions = append(requestedActions, "transaction")
	}
	if request.RequestedDecision != nil {
		requestedActions = append(requestedActions, "decision")
	}

	// Extract scopes from RequestedScope for compliance validation
	scopes := []string{}
	if request.RequestedScope != nil {
		// Add non-physical actions
		for _, action := range request.RequestedScope.AuthorizedActions.NonPhysicalActions {
			scopes = append(scopes, string(action))
		}
		// Add transaction types
		for _, tx := range request.RequestedScope.AuthorizedActions.Transactions {
			scopes = append(scopes, string(tx))
		}
		// Add decision types
		for _, dec := range request.RequestedScope.AuthorizedActions.Decisions {
			scopes = append(scopes, string(dec))
		}
	}
	// Ensure at least one scope for compliance validation
	if len(scopes) == 0 {
		scopes = []string{"access"}
	}

	// Extract legal framework from PoA if available
	var legalFramework *LegalFrameworkInfo
	if subscription.ClientAuthorizationGrant != nil && subscription.ClientAuthorizationGrant.PoACredential != nil {
		jurisdictionLaw := subscription.ClientAuthorizationGrant.PoACredential.Requirements.JurisdictionLaw
		legalFramework = &LegalFrameworkInfo{
			ApplicableLaws:      []string{jurisdictionLaw.GoverningLaw},
			Jurisdiction:        jurisdictionLaw.PlaceOfJurisdiction,
			ComplianceFramework: jurisdictionLaw.GoverningLaw,
		}
	}

	extendedRequest := &ExtendedAuthorizationRequest{
		AuthorizationRequest: &AuthorizationRequest{
			ClientID: request.ClientID,
			Scopes:   scopes,
		},
		PowerOfAttorney:    subscription.ClientAuthorizationGrant.PoACredential,
		AuthorizationChain: subscription.AuthorizationChain,
		LegalFramework:     legalFramework,
		RequestedActions:   requestedActions,
		TransactionContext: request.Context,
		Jurisdiction:       request.Jurisdiction,
		RequestTime:        time.Now(),
	}

	complianceResult, err := o.complianceValidator.ValidateRequestCompliance(ctx, extendedRequest)
	if err != nil {
		return nil, fmt.Errorf("step (b) failed: request compliance validation error: %w", err)
	}

	if !complianceResult.Valid {
		return nil, &GAuthError{
			Code:    "request_compliance_failed",
			Message: "Request does not comply with client's authorized powers",
		}
	}

	// STEP (c): Authorization Grant Issuance
	grant, err := o.issueAuthorizationGrant(ctx, request, subscription, complianceResult)
	if err != nil {
		return nil, fmt.Errorf("step (c) failed: grant issuance error: %w", err)
	}

	// STEP (d): Extended Token Request (implicit - client has grant)
	// No additional validation needed, grant serves as token request

	// STEP (e): Extended Token Issuance
	// THIS IS WHERE WE ACTUALLY CALL CreateExtendedToken()
	extendedTokenReq := &ExtendedTokenRequest{
		GrantID:            grant.GrantID,
		PowerOfAttorney:    subscription.ClientAuthorizationGrant.PoACredential,
		AuthorizationChain: subscription.AuthorizationChain,
		LegalFramework:     legalFramework,
		ClientOwnerInfo: &ClientOwnerInfo{
			OwnerID:   subscription.ClientOwnerIdentity.SubjectID,
			OwnerName: subscription.ClientOwnerIdentity.Identity,
		},
		OwnersAuthorizerInfo: &OwnersAuthorizerInfo{
			AuthorizerID:   subscription.OwnersAuthorizerIdentity.SubjectID,
			AuthorizerName: subscription.OwnersAuthorizerIdentity.Identity,
		},
		ResourceOwnerInfo: &ResourceOwnerInfo{
			OwnerID: request.ResourceOwnerID,
		},
	}

	extendedToken, err := o.extendedTokenService.CreateExtendedToken(ctx, extendedTokenReq)
	if err != nil {
		return nil, fmt.Errorf("step (e) failed: extended token creation error: %w", err)
	}

	// STEP (f): Grant Compliance Validation
	// THIS IS WHERE WE ACTUALLY CALL ValidateGrantCompliance()
	// Convert grant to ExtendedAuthorizationGrant format
	extendedGrant := &ExtendedAuthorizationGrant{
		AuthorizationGrant: &AuthorizationGrant{
			GrantID:    grant.GrantID,
			ClientID:   grant.ClientID,
			Scope:      scopes, // Use the same scopes we extracted earlier
			ValidUntil: grant.ExpiresAt,
		},
		ResourceOwnerID:    grant.ResourceOwnerID,
		IssuerID:           grant.ClientID, // AS is issuer
		AuthorizationChain: grant.AuthorizationChain,
		LegalFramework:     legalFramework,
		IssuedAt:           grant.IssuedAt,
		ExpiresAt:          grant.ExpiresAt,
	}

	grantValidation, err := o.complianceValidator.ValidateGrantCompliance(ctx, extendedGrant)
	if err != nil {
		return nil, fmt.Errorf("step (f) failed: grant compliance validation error: %w", err)
	}

	if !grantValidation.Valid {
		return nil, &GAuthError{
			Code:    "grant_compliance_failed",
			Message: "Grant does not comply with resource owner/server powers",
		}
	}

	// STEP (g): Transaction/Decision/Action Request
	// This happens downstream when client uses the extended token
	// We prepare the token with all necessary metadata for step (g)

	// STEP (h): Token Validation & Request Fulfillment
	// Also happens downstream at resource server
	// Extended token contains all validation information

	// STEP (i): Compliance Tracking
	// Start compliance monitoring for this authorization
	if o.complianceTracker != nil {
		monitoringPeriod := time.Duration(extendedToken.ExpiresIn) * time.Second
		err = o.complianceTracker.StartTracking(ctx, &ComplianceTrackingRequest{
			ExtendedTokenID:  extendedToken.AccessToken, // Use AccessToken as ID
			ClientID:         request.ClientID,
			ResourceOwnerID:  request.ResourceOwnerID,
			PoACredential:    subscription.ClientAuthorizationGrant.PoACredential,
			MonitoringPeriod: monitoringPeriod,
		})
		if err != nil {
			// Log error but don't fail the flow
			fmt.Printf("Warning: Failed to start compliance tracking: %v\n", err)
		}
	}

	// Build RFC-compliant response
	response := &RFCCompliantTokenResponse{
		ExtendedToken:   extendedToken,
		TokenType:       "GAuth-Extended-Token",
		ExpiresIn:       int(extendedToken.ExpiresIn),
		Scope:           request.RequestedScope,
		GrantValidation: grantValidation,
		ComplianceStatus: &ComplianceStatus{
			Compliant:   true,
			Violations:  []string{},
			LastChecked: time.Now(),
			NextCheck:   time.Now().Add(1 * time.Hour),
		},
	}

	return response, nil
}

// Helper type to store identity information simply

// validateRequestStructure performs basic validation of request structure
func (o *ProtocolOrchestrator) validateRequestStructure(request *RFCCompliantAuthorizationRequest) error {
	if request.ClientID == "" {
		return &GAuthError{Code: "missing_client_id", Message: "Client ID is required"}
	}
	if request.SubscriptionID == "" {
		return &GAuthError{Code: "missing_subscription", Message: "Subscription ID is required"}
	}
	if request.ResourceOwnerID == "" {
		return &GAuthError{Code: "missing_resource_owner", Message: "Resource owner ID is required"}
	}
	if request.RequestedScope == nil {
		return &GAuthError{Code: "missing_scope", Message: "Requested scope is required"}
	}
	return nil
}

// issueAuthorizationGrant creates an authorization grant (step c)
func (o *ProtocolOrchestrator) issueAuthorizationGrant(
	ctx context.Context,
	request *RFCCompliantAuthorizationRequest,
	subscription *Subscription,
	complianceResult *RequestComplianceResult,
) (*RFCCompliantGrantResponse, error) {

	grant := &RFCCompliantGrantResponse{
		GrantID:              generateGrantID(),
		IssuedAt:             time.Now(),
		ExpiresAt:            time.Now().Add(10 * time.Minute), // Short-lived grant
		ClientID:             request.ClientID,
		ResourceOwnerID:      request.ResourceOwnerID,
		Scope:                request.RequestedScope,
		PoACredential:        subscription.ClientAuthorizationGrant.PoACredential,
		AuthorizationChain:   subscription.AuthorizationChain,
		ComplianceValidation: complianceResult,
	}

	return grant, nil
}

func generateGrantID() string {
	return fmt.Sprintf("grant_%d", time.Now().UnixNano())
}
