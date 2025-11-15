// Package gauth - RFC-0111 Public Disclosure Service
// This implements the transparency and accountability requirements from RFC-0111
// Provides APIs for resource owners to view, manage, and revoke authorizations

package gauth

import (
	"context"
	"fmt"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
)

// DisclosureService provides transparency APIs for authorization management
type DisclosureService struct {
	tokenStore        ExtendedTokenStore
	subscriptionStore SubscriptionStore
	complianceTracker ComplianceTracker
	auditLogger       AuditLogger
}

// NewDisclosureService creates a new disclosure service
func NewDisclosureService(
	tokenStore ExtendedTokenStore,
	subscriptionStore SubscriptionStore,
	complianceTracker ComplianceTracker,
	auditLogger AuditLogger,
) *DisclosureService {
	return &DisclosureService{
		tokenStore:        tokenStore,
		subscriptionStore: subscriptionStore,
		complianceTracker: complianceTracker,
		auditLogger:       auditLogger,
	}
}

// AuthorizationSummary provides high-level view of an authorization
type AuthorizationSummary struct {
	AuthorizationID    string     `json:"authorization_id"`
	ResourceOwnerID    string     `json:"resource_owner_id"`
	ClientID           string     `json:"client_id"`
	ClientType         string     `json:"client_type"`
	ClientOwner        string     `json:"client_owner"`
	OwnersAuthorizer   string     `json:"owners_authorizer"`
	GrantedScopes      []string   `json:"granted_scopes"`
	GrantedActions     []string   `json:"granted_actions"`
	Status             string     `json:"status"`
	IssuedAt           time.Time  `json:"issued_at"`
	ExpiresAt          time.Time  `json:"expires_at"`
	LastUsed           *time.Time `json:"last_used,omitempty"`
	UsageCount         int        `json:"usage_count"`
	ComplianceStatus   string     `json:"compliance_status"`
	ActiveRestrictions []string   `json:"active_restrictions"`
}

// AuthorizationDetail provides complete view of an authorization
type AuthorizationDetail struct {
	AuthorizationSummary
	PowerOfAttorney      *poa.PoADefinition         `json:"power_of_attorney"`
	AuthorizationChain   *AuthorizationChain        `json:"authorization_chain"`
	LegalFramework       *LegalFrameworkInfo        `json:"legal_framework"`
	VerificationProof    *IdentityVerificationChain `json:"verification_proof"`
	Restrictions         []PowerRestriction         `json:"restrictions"`
	AuditTrail           []AuditEntry               `json:"audit_trail"`
	ComplianceViolations []ComplianceViolation      `json:"compliance_violations,omitempty"`
	SubscriptionID       string                     `json:"subscription_id"`
}

// ComplianceViolation represents a detected compliance issue
type ComplianceViolation struct {
	ViolationID   string     `json:"violation_id"`
	DetectedAt    time.Time  `json:"detected_at"`
	ViolationType string     `json:"violation_type"`
	Severity      string     `json:"severity"`
	Description   string     `json:"description"`
	Resolved      bool       `json:"resolved"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
}

// ListActiveAuthorizationsRequest is the request for listing authorizations
type ListActiveAuthorizationsRequest struct {
	ResourceOwnerID string    `json:"resource_owner_id"`
	ClientID        string    `json:"client_id,omitempty"`
	Status          string    `json:"status,omitempty"`
	FromDate        time.Time `json:"from_date,omitempty"`
	Limit           int       `json:"limit,omitempty"`
	Offset          int       `json:"offset,omitempty"`
}

// ListActiveAuthorizationsResponse contains the list of authorizations
type ListActiveAuthorizationsResponse struct {
	Authorizations []AuthorizationSummary `json:"authorizations"`
	Total          int                    `json:"total"`
	Limit          int                    `json:"limit"`
	Offset         int                    `json:"offset"`
}

// ListActiveAuthorizations retrieves all active authorizations for a resource owner
func (s *DisclosureService) ListActiveAuthorizations(
	ctx context.Context,
	request *ListActiveAuthorizationsRequest,
) (*ListActiveAuthorizationsResponse, error) {
	if request.ResourceOwnerID == "" {
		return nil, fmt.Errorf("resource_owner_id is required")
	}

	// Set defaults
	if request.Limit <= 0 || request.Limit > 100 {
		request.Limit = 50
	}

	// Query token store for all active tokens by resource owner
	tokens, err := s.tokenStore.ListTokensByResourceOwner(ctx, request.ResourceOwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}

	// Apply client filter if specified
	if request.ClientID != "" {
		filtered := make([]*ExtendedToken, 0)
		for _, token := range tokens {
			if token.AuthorizationChain != nil && token.AuthorizationChain.Client != nil &&
				token.AuthorizationChain.Client.EntityID == request.ClientID {
				filtered = append(filtered, token)
			}
		}
		tokens = filtered
	}

	// Apply pagination
	total := len(tokens)
	start := request.Offset
	end := request.Offset + request.Limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	tokens = tokens[start:end]

	// Convert to summaries
	summaries := make([]AuthorizationSummary, 0, len(tokens))
	for _, token := range tokens {
		summary := s.tokenToSummary(token)
		summaries = append(summaries, summary)
	}

	// Log disclosure access
	s.auditLogger.LogDisclosureAccess(ctx, &AuditEntry{
		Timestamp: time.Now(),
		Action:    "list_authorizations",
		Actor:     request.ResourceOwnerID,
		Result:    "success",
		Details: map[string]interface{}{
			"count":  len(summaries),
			"status": request.Status,
		},
	})

	return &ListActiveAuthorizationsResponse{
		Authorizations: summaries,
		Total:          total,
		Limit:          request.Limit,
		Offset:         request.Offset,
	}, nil
}

// GetAuthorizationDetail retrieves complete details of a specific authorization
func (s *DisclosureService) GetAuthorizationDetail(
	ctx context.Context,
	authorizationID string,
	resourceOwnerID string,
) (*AuthorizationDetail, error) {
	if authorizationID == "" {
		return nil, fmt.Errorf("authorization_id is required")
	}
	if resourceOwnerID == "" {
		return nil, fmt.Errorf("resource_owner_id is required")
	}

	// Retrieve token
	token, err := s.tokenStore.GetToken(ctx, authorizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve authorization: %w", err)
	}

	// Verify ownership
	if token.ResourceOwner == nil || token.ResourceOwner.OwnerID != resourceOwnerID {
		return nil, fmt.Errorf("unauthorized: not the resource owner")
	}

	// Get compliance violations from tracking system
	violations := s.getComplianceViolations(ctx, authorizationID)

	// Build detail
	detail := &AuthorizationDetail{
		AuthorizationSummary: s.tokenToSummary(token),
		PowerOfAttorney:      token.PowerOfAttorney,
		AuthorizationChain:   token.AuthorizationChain,
		LegalFramework:       token.LegalFramework,
		VerificationProof:    token.VerificationProof,
		Restrictions:         token.Restrictions,
		AuditTrail:           token.AuditTrail,
		ComplianceViolations: violations,
		SubscriptionID:       token.SubscriptionID, // Link to originating subscription (RFC-0111 Steps I-VIII)
	}

	// Log disclosure access
	s.auditLogger.LogDisclosureAccess(ctx, &AuditEntry{
		Timestamp: time.Now(),
		Action:    "get_authorization_detail",
		Actor:     resourceOwnerID,
		Result:    "success",
		Details: map[string]interface{}{
			"authorization_id": authorizationID,
		},
	})

	return detail, nil
}

// RevokeAuthorizationRequest is the request to revoke an authorization
type RevokeAuthorizationRequest struct {
	AuthorizationID string `json:"authorization_id"`
	ResourceOwnerID string `json:"resource_owner_id"`
	Reason          string `json:"reason"`
	RevokedBy       string `json:"revoked_by"`
}

// RevokeAuthorizationResponse contains the revocation result
type RevokeAuthorizationResponse struct {
	AuthorizationID string    `json:"authorization_id"`
	RevokedAt       time.Time `json:"revoked_at"`
	Status          string    `json:"status"`
	Message         string    `json:"message"`
}

// RevokeAuthorization allows a resource owner to revoke an authorization
func (s *DisclosureService) RevokeAuthorization(
	ctx context.Context,
	request *RevokeAuthorizationRequest,
) (*RevokeAuthorizationResponse, error) {
	if request.AuthorizationID == "" {
		return nil, fmt.Errorf("authorization_id is required")
	}
	if request.ResourceOwnerID == "" {
		return nil, fmt.Errorf("resource_owner_id is required")
	}

	// Retrieve token
	token, err := s.tokenStore.GetToken(ctx, request.AuthorizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve authorization: %w", err)
	}

	// Verify ownership
	if token.ResourceOwner == nil || token.ResourceOwner.OwnerID != request.ResourceOwnerID {
		return nil, fmt.Errorf("unauthorized: not the resource owner")
	}

	// Check if already revoked (check via store method)
	isRevoked, _ := s.tokenStore.IsRevoked(ctx, request.AuthorizationID)
	if isRevoked {
		return &RevokeAuthorizationResponse{
			AuthorizationID: request.AuthorizationID,
			RevokedAt:       time.Now(),
			Status:          "already_revoked",
			Message:         "Authorization was already revoked",
		}, nil
	}

	// Revoke the token with reason
	revokedAt := time.Now()
	err = s.tokenStore.RevokeTokenWithReason(ctx, request.AuthorizationID, request.Reason)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke authorization: %w", err)
	}

	// Stop compliance tracking for the revoked token
	if err := s.complianceTracker.StopTracking(ctx, request.AuthorizationID); err != nil {
		// Log but don't fail - compliance tracking stop is non-critical
		// Note: Tracking may not have been active for this token
	}

	// Log revocation
	s.auditLogger.LogRevocation(ctx, &AuditEntry{
		Timestamp: revokedAt,
		Action:    "revoke_authorization",
		Actor:     request.ResourceOwnerID,
		Result:    "success",
		Details: map[string]interface{}{
			"authorization_id": request.AuthorizationID,
			"reason":           request.Reason,
			"revoked_by":       request.RevokedBy,
		},
	})

	return &RevokeAuthorizationResponse{
		AuthorizationID: request.AuthorizationID,
		RevokedAt:       revokedAt,
		Status:          "revoked",
		Message:         "Authorization successfully revoked",
	}, nil
}

// GetAuditTrail retrieves the audit trail for a specific authorization
func (s *DisclosureService) GetAuditTrail(
	ctx context.Context,
	authorizationID string,
	resourceOwnerID string,
	fromDate time.Time,
	limit int,
) ([]AuditEntry, error) {
	if authorizationID == "" {
		return nil, fmt.Errorf("authorization_id is required")
	}
	if resourceOwnerID == "" {
		return nil, fmt.Errorf("resource_owner_id is required")
	}

	// Retrieve token to verify ownership
	token, err := s.tokenStore.GetToken(ctx, authorizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve authorization: %w", err)
	}

	// Verify ownership
	if token.ResourceOwner == nil || token.ResourceOwner.OwnerID != resourceOwnerID {
		return nil, fmt.Errorf("unauthorized: not the resource owner")
	}

	// Get audit trail
	entries, err := s.auditLogger.GetAuditTrail(ctx, authorizationID, fromDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve audit trail: %w", err)
	}

	// Log disclosure access
	s.auditLogger.LogDisclosureAccess(ctx, &AuditEntry{
		Timestamp: time.Now(),
		Action:    "get_audit_trail",
		Actor:     resourceOwnerID,
		Result:    "success",
		Details: map[string]interface{}{
			"authorization_id": authorizationID,
			"entry_count":      len(entries),
			"from_date":        fromDate,
		},
	})

	return entries, nil
}

// tokenToSummary converts an ExtendedToken to an AuthorizationSummary
func (s *DisclosureService) tokenToSummary(token *ExtendedToken) AuthorizationSummary {
	// Determine if token is revoked
	status := "active"
	if s.tokenStore != nil {
		isRevoked, _ := s.tokenStore.IsRevoked(context.Background(), token.AccessToken)
		if isRevoked {
			status = "revoked"
		}
	}
	if time.Now().After(token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)) {
		status = "expired"
	}

	// Extract granted actions from Power of Attorney
	grantedActions := []string{}
	if token.PowerOfAttorney != nil {
		actions := token.PowerOfAttorney.Authorization.AuthorizedActions
		
		// Extract transactions
		for _, txn := range actions.Transactions {
			grantedActions = append(grantedActions, string(txn))
		}
		
		// Extract decisions
		for _, decision := range actions.Decisions {
			grantedActions = append(grantedActions, string(decision))
		}
		
		// Extract physical actions
		for _, physical := range actions.PhysicalActions {
			grantedActions = append(grantedActions, string(physical))
		}
		
		// Extract non-physical actions
		for _, nonPhysical := range actions.NonPhysicalActions {
			grantedActions = append(grantedActions, string(nonPhysical))
		}
	}

	summary := AuthorizationSummary{
		AuthorizationID:  token.AccessToken, // Using access token as ID
		Status:           status,
		IssuedAt:         token.IssuedAt,
		ExpiresAt:        token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second),
		ComplianceStatus: token.ComplianceLevel,
		GrantedScopes:    token.Scope,       // Already []string
		GrantedActions:   grantedActions,    // Extracted from PoA definition
	}

	if token.ResourceOwner != nil {
		summary.ResourceOwnerID = token.ResourceOwner.OwnerID
	}
	if token.ClientOwner != nil {
		summary.ClientOwner = token.ClientOwner.OwnerName
		summary.ClientID = token.ClientOwner.OwnerID
	}
	if token.OwnersAuthorizer != nil {
		summary.OwnersAuthorizer = token.OwnersAuthorizer.AuthorizerName
	}
	if token.PowerOfAttorney != nil {
		// ClientType from authorization chain Client role
		if token.AuthorizationChain != nil && token.AuthorizationChain.Client != nil {
			summary.ClientType = token.AuthorizationChain.Client.EntityType
		}
	}

	// Extract restriction summaries
	restrictions := make([]string, 0, len(token.Restrictions))
	for _, r := range token.Restrictions {
		restrictions = append(restrictions, fmt.Sprintf("%s: %s", r.RestrictionType, r.Description))
	}
	summary.ActiveRestrictions = restrictions

	return summary
}

// getComplianceViolations retrieves compliance violations for a token from the tracking system
func (s *DisclosureService) getComplianceViolations(ctx context.Context, tokenID string) []ComplianceViolation {
	violations := []ComplianceViolation{}

	// Get tracking status from compliance tracker
	status, err := s.complianceTracker.GetTrackingStatus(ctx, tokenID)
	if err != nil {
		// Token may not be tracked or tracking already stopped - return empty violations
		return violations
	}

	// Check if tracking is active and has compliance status
	if !status.Active || status.ComplianceStatus == nil {
		return violations
	}

	// Convert string violations to structured ComplianceViolation objects
	for i, v := range status.ComplianceStatus.Violations {
		violations = append(violations, ComplianceViolation{
			ViolationID:   fmt.Sprintf("%s-v%d", tokenID, i+1),
			DetectedAt:    status.LastChecked,
			ViolationType: "compliance_check",
			Severity:      s.determineViolationSeverity(v),
			Description:   v,
			Resolved:      false,
		})
	}

	return violations
}

// determineViolationSeverity analyzes violation description to determine severity
func (s *DisclosureService) determineViolationSeverity(violation string) string {
	violationLower := fmt.Sprintf("%v", violation)
	
	// Critical severity indicators
	if containsAny(violationLower, "expired", "revoked", "invalid", "unauthorized") {
		return "critical"
	}
	
	// High severity indicators
	if containsAny(violationLower, "exceeded", "breach", "violation", "denied") {
		return "high"
	}
	
	// Medium severity indicators
	if containsAny(violationLower, "warning", "approaching", "near") {
		return "medium"
	}
	
	// Default to low severity
	return "low"
}

// containsAny checks if s contains any of the substrings (case-insensitive)
func containsAny(s string, substrings ...string) bool {
	s = fmt.Sprintf("%v", s)
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				match := true
				for j := 0; j < len(substr); j++ {
					c1 := s[i+j]
					c2 := substr[j]
					// Simple case-insensitive comparison
					if c1 >= 'A' && c1 <= 'Z' {
						c1 = c1 + 32
					}
					if c2 >= 'A' && c2 <= 'Z' {
						c2 = c2 + 32
					}
					if c1 != c2 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}

// AuditLogger interface for audit trail management
type AuditLogger interface {
	LogDisclosureAccess(ctx context.Context, entry *AuditEntry) error
	LogRevocation(ctx context.Context, entry *AuditEntry) error
	GetAuditTrail(ctx context.Context, authorizationID string, fromDate time.Time, limit int) ([]AuditEntry, error)
}

// ListTokensByResourceOwner is a method that needs to be added to ExtendedTokenStore
// For now, we'll add it as a comment - it should be added to extended_token_store.go
// ListTokensByResourceOwner(ctx context.Context, resourceOwnerID, clientID, status string, limit, offset int) ([]*ExtendedToken, int, error)
