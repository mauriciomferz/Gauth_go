// Package agentauth - Power Decision Point (PDP) Adapter
// This adapter connects the agentauth.PowerDecisionPoint interface to policy-based decisions
package agentauth

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

const (
	actionTypeTransaction = "transaction"
	actionTypeDecision    = "decision"
	actionTypeAction      = "action"
)

// ProductionPEPAuditLogger provides thread-safe audit logging with observability integration
type ProductionPEPAuditLogger struct {
	mu           sync.RWMutex
	enforcements []EnforcementAuditEntry
	violations   []ViolationAuditEntry

	// Configuration
	maxEntries    int
	enableConsole bool
	enableMetrics bool
	metrics       metrics.Metrics

	// Statistics
	totalEnforcements int64
	totalViolations   int64
}

// NewProductionPEPAuditLogger creates a production-ready audit logger
func NewProductionPEPAuditLogger(maxEntries int, enableConsole, enableMetrics bool) *ProductionPEPAuditLogger {
	if maxEntries <= 0 {
		maxEntries = 10000 // Default to 10k entries
	}

	return &ProductionPEPAuditLogger{
		enforcements:  make([]EnforcementAuditEntry, 0, maxEntries),
		violations:    make([]ViolationAuditEntry, 0, maxEntries),
		maxEntries:    maxEntries,
		enableConsole: enableConsole,
		enableMetrics: enableMetrics,
		metrics:       metrics.Noop, // Default to noop, can be overridden with SetMetrics
	}
}

// SetMetrics configures the metrics collector for the audit logger
func (l *ProductionPEPAuditLogger) SetMetrics(m metrics.Metrics) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if m != nil {
		l.metrics = m
	}
}

// LogEnforcement logs an enforcement action with observability
func (l *ProductionPEPAuditLogger) LogEnforcement(ctx context.Context, entry *EnforcementAuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Store in memory (with rotation)
	if len(l.enforcements) >= l.maxEntries {
		// Remove oldest entry (FIFO)
		l.enforcements = l.enforcements[1:]
	}
	l.enforcements = append(l.enforcements, *entry)
	l.totalEnforcements++

	// Console logging for debugging
	if l.enableConsole {
		log.Printf("[ENFORCEMENT] ID=%s Action=%s Resource=%s Allowed=%v Outcome=%s Reason=%s Violations=%d Timestamp=%v",
			entry.EnforcementID,
			entry.ActionType,
			entry.ResourceID,
			entry.Allowed,
			entry.Outcome,
			entry.Reason,
			entry.ViolationCount,
			entry.Timestamp)
	}

	// Metrics export
	if l.enableMetrics && l.metrics != nil {
		l.metrics.IncPEPEnforcements(entry.Allowed, entry.ActionType)
		l.metrics.SetPEPAuditBufferSize(len(l.enforcements), len(l.violations))
	}

	return nil
}

// LogViolation logs a violation with observability and alerting
func (l *ProductionPEPAuditLogger) LogViolation(ctx context.Context, entry *ViolationAuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Store in memory (with rotation)
	if len(l.violations) >= l.maxEntries {
		// Remove oldest entry (FIFO)
		l.violations = l.violations[1:]
	}
	l.violations = append(l.violations, *entry)
	l.totalViolations++

	// Console logging with severity
	if l.enableConsole {
		log.Printf("[VIOLATION] Enforcement=%s Type=%s Severity=%s Action=%s Resource=%s Description=%s Timestamp=%v",
			entry.EnforcementID,
			entry.ViolationType,
			entry.Severity,
			entry.ActionType,
			entry.ResourceID,
			entry.Description,
			entry.Timestamp)
	}

	// Metrics and alerting
	if l.enableMetrics && l.metrics != nil {
		l.metrics.IncPEPViolations(entry.ViolationType, entry.Severity)
		l.metrics.SetPEPAuditBufferSize(len(l.enforcements), len(l.violations))
		// Note: High-severity violations (critical/high) should be monitored via
		// Prometheus alerts configured externally; check violations_total rate
	}

	return nil
}

// GetEnforcements returns recent enforcement logs (thread-safe)
func (l *ProductionPEPAuditLogger) GetEnforcements(limit int) []EnforcementAuditEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit <= 0 || limit > len(l.enforcements) {
		limit = len(l.enforcements)
	}

	// Return most recent entries
	start := len(l.enforcements) - limit
	if start < 0 {
		start = 0
	}

	result := make([]EnforcementAuditEntry, limit)
	copy(result, l.enforcements[start:])
	return result
}

// GetViolations returns recent violation logs (thread-safe)
func (l *ProductionPEPAuditLogger) GetViolations(limit int) []ViolationAuditEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit <= 0 || limit > len(l.violations) {
		limit = len(l.violations)
	}

	// Return most recent entries
	start := len(l.violations) - limit
	if start < 0 {
		start = 0
	}

	result := make([]ViolationAuditEntry, limit)
	copy(result, l.violations[start:])
	return result
}

// GetStatistics returns audit log statistics
func (l *ProductionPEPAuditLogger) GetStatistics() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return map[string]interface{}{
		"total_enforcements":        l.totalEnforcements,
		"total_violations":          l.totalViolations,
		"stored_enforcements":       len(l.enforcements),
		"stored_violations":         len(l.violations),
		"max_entries":               l.maxEntries,
		"enforcement_storage_usage": float64(len(l.enforcements)) / float64(l.maxEntries) * 100,
		"violation_storage_usage":   float64(len(l.violations)) / float64(l.maxEntries) * 100,
	}
}

// noopPEPAuditLogger is a simple no-op audit logger for testing/minimal setups
type noopPEPAuditLogger struct{}

func (n *noopPEPAuditLogger) LogEnforcement(ctx context.Context, entry *EnforcementAuditEntry) error {
	return nil
}

func (n *noopPEPAuditLogger) LogViolation(ctx context.Context, entry *ViolationAuditEntry) error {
	return nil
}

// simpleTokenValidator adapts ExtendedTokenService to TokenValidator interface
type simpleTokenValidator struct {
	extTokenService *ExtendedTokenService
}

func (v *simpleTokenValidator) ValidateExtendedToken(ctx context.Context, token string) (*ExtendedToken, error) {
	result, err := v.extTokenService.ValidateExtendedToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !result.Valid {
		return nil, fmt.Errorf("token validation failed")
	}
	return result.ExtendedToken, nil
}

// SimplePDP is a PDP implementation with PAP integration for AAP001 compliance
// It provides policy-based authorization decisions with centralized policy management
type SimplePDP struct {
	pap                    *PowerAdministrationPoint
	AgentAuthPlusValidator *AgentAuthPlusValidator
	enforceAgentAuthPlus   bool
}

// NewSimplePDP creates a new SimplePDP instance
func NewSimplePDP() *SimplePDP {
	return &SimplePDP{
		pap:                  nil,   // No PAP integration by default (backward compatible)
		enforceAgentAuthPlus: false, // AgentAuth+ disabled by default
	}
}

// NewSimplePDPWithPAP creates a new SimplePDP instance with PAP integration
func NewSimplePDPWithPAP(pap *PowerAdministrationPoint) *SimplePDP {
	return &SimplePDP{
		pap:                  pap,
		enforceAgentAuthPlus: false,
	}
}

// SetAgentAuthPlusValidator sets the AgentAuth+ validator and enables enforcement
func (pdp *SimplePDP) SetAgentAuthPlusValidator(validator *AgentAuthPlusValidator) {
	pdp.AgentAuthPlusValidator = validator
	pdp.enforceAgentAuthPlus = true
}

// SetEnforceAgentAuthPlus enables/disables AgentAuth+ enforcement
func (pdp *SimplePDP) SetEnforceAgentAuthPlus(enforce bool) {
	pdp.enforceAgentAuthPlus = enforce
}

// MakeDecision implements the PowerDecisionPoint interface
func (pdp *SimplePDP) MakeDecision(
	ctx context.Context,
	request *AuthorizationDecisionRequest,
) (*AuthorizationDecision, error) {
	if request == nil {
		return &AuthorizationDecision{
			Authorized: false,
			Reason:     "nil request",
		}, nil
	}

	// Extract authorization scope from PoA
	authorized, reason := pdp.evaluateRequest(request)

	return &AuthorizationDecision{
		DecisionID:     fmt.Sprintf("pdp-decision-%d", time.Now().UnixNano()),
		Authorized:     authorized,
		Reason:         reason,
		Conditions:     []string{},
		ValidUntil:     time.Now().Add(5 * time.Minute), // Decision valid for 5 minutes
		RequiresReview: false,
	}, nil
}

// evaluateRequest performs the actual authorization logic
func (pdp *SimplePDP) evaluateRequest(request *AuthorizationDecisionRequest) (bool, string) {
	// Step 1: Validate Proof of Authorization exists
	if request.PowerOfAttorney == nil {
		return false, "missing power of attorney credential"
	}

	// Step 2: Validate Authorization Chain
	if request.AuthorizationChain == nil {
		return false, "missing authorization chain"
	}

	if !request.AuthorizationChain.ChainValidated {
		return false, "authorization chain not validated"
	}

	// Step 3: Check AgentAuth+ policies (if enabled)
	if pdp.enforceAgentAuthPlus && pdp.AgentAuthPlusValidator != nil {
		agentID := request.PowerOfAttorney.Parties.AuthorizedClient.Identity
		// Note: Using agent identity as PoA ID placeholder
		// In production, track PoA ID separately in AuthorizationDecisionRequest
		poaID := agentID // TODO: Get actual PoA ID from request
		AgentAuthPlusResult, err := pdp.AgentAuthPlusValidator.ValidatePoAWithAgentAuthPlus(
			context.Background(),
			poaID,
			request.PowerOfAttorney,
			agentID,
			request.ActionType,
		)

		if err != nil {
			return false, fmt.Sprintf("AgentAuth+ validation error: %v", err)
		}

		if !AgentAuthPlusResult.Valid {
			return false, fmt.Sprintf("AgentAuth+ policy violation: %s", AgentAuthPlusResult.FailureReason)
		}

		// Log any AgentAuth+ warnings (successor takeover, capability expiration, etc.)
		if len(AgentAuthPlusResult.Warnings) > 0 {
			for _, warning := range AgentAuthPlusResult.Warnings {
				log.Printf("AgentAuth+ Warning: %s", warning)
			}
		}
	}

	// Step 4: Check action type against authorized scope
	if !pdp.isActionAuthorized(request.ActionType, request.PowerOfAttorney) {
		return false, fmt.Sprintf("action type '%s' not authorized in PoA", request.ActionType)
	}

	// Step 5: Check resource access
	if request.ResourceID != "" && !pdp.isResourceAuthorized(request.ResourceID, request.PowerOfAttorney) {
		return false, fmt.Sprintf("resource '%s' not authorized in PoA scope", request.ResourceID)
	}

	// All checks passed
	return true, "authorization granted per PoA, chain, and AgentAuth+ validation"
}

// isActionAuthorized checks if the action type is allowed in the PoA
func (pdp *SimplePDP) isActionAuthorized(actionType string, poaDef *poa.PoADefinition) bool {
	if poaDef == nil {
		// If no specific restrictions, allow (default permissive for demo)
		return true
	}

	authActions := poaDef.Authorization.AuthorizedActions

	// Check transaction types
	if actionType == actionTypeTransaction {
		return len(authActions.Transactions) > 0
	}

	// Check decision types
	if actionType == actionTypeDecision {
		return len(authActions.Decisions) > 0
	}

	// Check action types
	if actionType == actionTypeAction {
		physicalCount := len(authActions.PhysicalActions)
		nonPhysicalCount := len(authActions.NonPhysicalActions)
		return physicalCount > 0 || nonPhysicalCount > 0
	}

	// Unknown action type - deny by default
	return false
}

// isResourceAuthorized checks if the resource is within PoA scope
func (pdp *SimplePDP) isResourceAuthorized(resourceID string, poaDef *poa.PoADefinition) bool {
	// For now, use simple logic:
	// If geographic scope includes global, allow all resources
	if len(poaDef.Authorization.ApplicableRegions) > 0 {
		for _, region := range poaDef.Authorization.ApplicableRegions {
			if region.Type == poa.GeoTypeGlobal {
				return true
			}
		}
	}

	// If sectors defined, assume resource is authorized
	// (In production, this would check resource sector against authorized sectors)
	if len(poaDef.Authorization.ApplicableSectors) > 0 {
		return true
	}

	// Default: allow if no specific restrictions
	return true
}

// AddPolicy adds a policy to the PDP via PAP integration
func (pdp *SimplePDP) AddPolicy(policyID string, policy interface{}) error {
	if pdp.pap == nil {
		return fmt.Errorf("PAP not configured - use NewSimplePDPWithPAP() to enable policy management")
	}

	// Convert policy interface to AuthorizationPolicy
	authPolicy, ok := policy.(*AuthorizationPolicy)
	if !ok {
		return fmt.Errorf("policy must be of type *AuthorizationPolicy")
	}

	// Create policy via PAP
	request := &PolicyCreateRequest{
		PolicyName:       authPolicy.PolicyName,
		PolicyType:       authPolicy.PolicyType,
		Description:      authPolicy.Description,
		ClientOwner:      authPolicy.ClientOwner,
		OwnersAuthorizer: authPolicy.OwnersAuthorizer,
		PolicyRules:      authPolicy.PolicyRules,
		Scope:            authPolicy.Scope,
		Restrictions:     authPolicy.Restrictions,
		PoATemplate:      authPolicy.PoATemplate,
		ExpiresAt:        authPolicy.ExpiresAt,
		Tags:             authPolicy.Tags,
		Metadata:         authPolicy.Metadata,
	}

	createdPolicy, err := pdp.pap.CreatePolicy(context.Background(), request)
	if err != nil {
		return fmt.Errorf("failed to create policy via PAP: %w", err)
	}

	// Update the policyID if provided policy didn't have one
	if authPolicy.PolicyID == "" {
		authPolicy.PolicyID = createdPolicy.PolicyID
	}

	return nil
}

// RemovePolicy removes a policy from the PDP via PAP integration
func (pdp *SimplePDP) RemovePolicy(policyID string) error {
	if pdp.pap == nil {
		return fmt.Errorf("PAP not configured - use NewSimplePDPWithPAP() to enable policy management")
	}

	// Delete policy via PAP
	err := pdp.pap.DeletePolicy(context.Background(), policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy via PAP: %w", err)
	}

	return nil
}

// GetPolicy retrieves a policy from the PDP via PAP integration
func (pdp *SimplePDP) GetPolicy(policyID string) (*AuthorizationPolicy, error) {
	if pdp.pap == nil {
		return nil, fmt.Errorf("PAP not configured - use NewSimplePDPWithPAP() to enable policy management")
	}

	policy, err := pdp.pap.GetPolicy(context.Background(), policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve policy via PAP: %w", err)
	}

	return policy, nil
}

// ListActivePolicies retrieves all active policies from PAP
func (pdp *SimplePDP) ListActivePolicies() ([]*AuthorizationPolicy, error) {
	if pdp.pap == nil {
		return nil, fmt.Errorf("PAP not configured - use NewSimplePDPWithPAP() to enable policy management")
	}

	activeStatus := PolicyStatusActive
	policies, err := pdp.pap.ListPolicies(context.Background(), &activeStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to list active policies via PAP: %w", err)
	}

	return policies, nil
}
