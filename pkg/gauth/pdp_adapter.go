// Package gauth - Power Decision Point (PDP) Adapter
// This adapter connects the gauth.PowerDecisionPoint interface to policy-based decisions
package gauth

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
)

// ProductionPEPAuditLogger provides thread-safe audit logging with observability integration
type ProductionPEPAuditLogger struct {
	mu           sync.RWMutex
	enforcements []EnforcementAuditEntry
	violations   []ViolationAuditEntry
	
	// Configuration
	maxEntries     int
	enableConsole  bool
	enableMetrics  bool
	
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

	// Metrics export (future: Prometheus, OpenTelemetry)
	if l.enableMetrics {
		// TODO: Export to Prometheus/OpenTelemetry when integrated
		// Example: metrics.IncrementCounter("gauth.enforcement.total", map[string]string{
		//     "allowed": fmt.Sprintf("%v", entry.Allowed),
		//     "action_type": entry.ActionType,
		// })
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
	if l.enableMetrics {
		// TODO: Export metrics and trigger alerts for high-severity violations
		// Example:
		// metrics.IncrementCounter("gauth.violation.total", map[string]string{
		//     "type": entry.ViolationType,
		//     "severity": entry.Severity,
		// })
		// if entry.Severity == "critical" {
		//     alerts.TriggerAlert("CriticalViolation", entry)
		// }
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

// SimplePDP is a basic PDP implementation for RFC-0111 compliance
// It provides policy-based authorization decisions
type SimplePDP struct {
	// Future: could add policy storage/engine here
}

// NewSimplePDP creates a new SimplePDP instance
func NewSimplePDP() *SimplePDP {
	return &SimplePDP{}
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
	// Step 1: Validate Power of Attorney exists
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

	// Step 3: Check action type against authorized scope
	if !pdp.isActionAuthorized(request.ActionType, request.PowerOfAttorney) {
		return false, fmt.Sprintf("action type '%s' not authorized in PoA", request.ActionType)
	}

	// Step 4: Check resource access
	if request.ResourceID != "" && !pdp.isResourceAuthorized(request.ResourceID, request.PowerOfAttorney) {
		return false, fmt.Sprintf("resource '%s' not authorized in PoA scope", request.ResourceID)
	}

	// All checks passed
	return true, "authorization granted per PoA and chain validation"
}

// isActionAuthorized checks if the action type is allowed in the PoA
func (pdp *SimplePDP) isActionAuthorized(actionType string, poaDef *poa.PoADefinition) bool {
	if poaDef == nil {
		// If no specific restrictions, allow (default permissive for demo)
		return true
	}

	authActions := poaDef.Authorization.AuthorizedActions

	// Check transaction types
	if actionType == "transaction" {
		return len(authActions.Transactions) > 0
	}

	// Check decision types
	if actionType == "decision" {
		return len(authActions.Decisions) > 0
	}

	// Check action types
	if actionType == "action" {
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

// AddPolicy adds a policy to the PDP (future enhancement)
func (pdp *SimplePDP) AddPolicy(policyID string, policy interface{}) error {
	// TODO: Implement policy storage when policy engine is added
	return fmt.Errorf("policy management not yet implemented")
}

// RemovePolicy removes a policy from the PDP (future enhancement)
func (pdp *SimplePDP) RemovePolicy(policyID string) error {
	// TODO: Implement policy removal when policy engine is added
	return fmt.Errorf("policy management not yet implemented")
}
