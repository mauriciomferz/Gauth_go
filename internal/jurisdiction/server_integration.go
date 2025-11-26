package jurisdiction

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/compliance"
)

// ServerIntegration integrates jurisdiction enforcement with the GAuth server.
type ServerIntegration struct {
	engine *EnforcementEngine
}

// NewServerIntegration creates a new server integration instance.
func NewServerIntegration() *ServerIntegration {
	return &ServerIntegration{
		engine: NewEnforcementEngine(),
	}
}

// EnforceJurisdiction enforces jurisdiction-specific rules for a request.
// This is the main entry point for server-side enforcement.
func (si *ServerIntegration) EnforceJurisdiction(
	ctx context.Context,
	subject string,
	resource string,
	action string,
	claims map[string]interface{},
) (*EnforcementDecision, error) {
	// Extract jurisdiction from claims
	jurisdiction := ExtractJurisdictionFromClaims(claims)

	// Extract entity type from claims
	entityType := extractEntityType(claims)

	// Extract value if present
	value := extractValue(claims)

	// Create enforcement context
	enfCtx := &EnforcementContext{
		RequestID:    generateRequestID(),
		Subject:      subject,
		Resource:     resource,
		Action:       action,
		Value:        value,
		EntityType:   entityType,
		Jurisdiction: jurisdiction,
		Claims:       claims,
		Timestamp:    time.Now(),
	}

	// Perform enforcement
	return si.engine.Enforce(ctx, enfCtx)
}

// ValidateJurisdiction validates if an action is allowed in a jurisdiction.
func (si *ServerIntegration) ValidateJurisdiction(
	ctx context.Context,
	jurisdiction compliance.Jurisdiction,
	action string,
) error {
	enfCtx := &EnforcementContext{
		RequestID:    generateRequestID(),
		Action:       action,
		Jurisdiction: jurisdiction,
		Timestamp:    time.Now(),
		Claims:       make(map[string]interface{}),
	}

	decision, err := si.engine.Enforce(ctx, enfCtx)
	if err != nil {
		return err
	}

	if !decision.Allowed {
		return fmt.Errorf("jurisdiction validation failed: %v", decision.Violations)
	}

	return nil
}

// GetEnforcementEngine returns the underlying enforcement engine.
func (si *ServerIntegration) GetEnforcementEngine() *EnforcementEngine {
	return si.engine
}

// SetAuditCallback sets the audit callback for enforcement decisions.
func (si *ServerIntegration) SetAuditCallback(callback func(decision EnforcementDecision)) {
	si.engine.SetAuditCallback(callback)
}

// GetMetrics returns current enforcement metrics.
func (si *ServerIntegration) GetMetrics() *EnforcementMetrics {
	return si.engine.GetMetrics()
}

// SetEnabled enables or disables jurisdiction enforcement.
func (si *ServerIntegration) SetEnabled(enabled bool) {
	si.engine.SetEnabled(enabled)
}

// IsEnabled returns whether jurisdiction enforcement is enabled.
func (si *ServerIntegration) IsEnabled() bool {
	return si.engine.IsEnabled()
}

// extractEntityType extracts entity type from claims.
func extractEntityType(claims map[string]interface{}) compliance.EntityType {
	if entityTypeStr, ok := claims["entity_type"].(string); ok {
		return compliance.EntityType(entityTypeStr)
	}
	if entityTypeStr, ok := claims["entity"].(string); ok {
		return compliance.EntityType(entityTypeStr)
	}
	// Default to individual if not specified
	return compliance.EntityTypeIndividual
}

// extractValue extracts monetary/numeric value from claims.
func extractValue(claims map[string]interface{}) float64 {
	if value, ok := claims["value"].(float64); ok {
		return value
	}
	if value, ok := claims["amount"].(float64); ok {
		return value
	}
	if value, ok := claims["transaction_value"].(float64); ok {
		return value
	}
	return 0.0
}

// generateRequestID generates a unique request ID for tracking.
func generateRequestID() string {
	return fmt.Sprintf("jur-%d", time.Now().UnixNano())
}

// ExtendStandardCapabilityEnforcement extends standard capability enforcement with jurisdiction rules.
// This can be called from the main capability enforcement logic to add jurisdiction-specific checks.
func (si *ServerIntegration) ExtendStandardCapabilityEnforcement(
	ctx context.Context,
	standardDecision interface{}, // The standard capability decision
	claims map[string]interface{},
	action string,
	resource string,
) (*EnforcementDecision, error) {
	// Extract subject from claims
	subject := "unknown"
	if sub, ok := claims["sub"].(string); ok {
		subject = sub
	}

	// Perform jurisdiction enforcement
	return si.EnforceJurisdiction(ctx, subject, resource, action, claims)
}
