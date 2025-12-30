package jurisdiction

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/compliance"
)

// EnforcementDecision represents the result of a jurisdiction enforcement check.
type EnforcementDecision struct {
	Allowed            bool
	Jurisdiction       compliance.Jurisdiction
	AppliedRules       []string
	RequiredApprovals  []compliance.ApprovalLevel
	ValueLimits        map[string]float64
	Violations         []string
	Warnings           []string
	Timestamp          time.Time
	RequestID          string
	EnforcementLatency time.Duration
}

// EnforcementContext contains the context information for enforcement decisions.
type EnforcementContext struct {
	RequestID    string
	Subject      string                  // The subject making the request
	Resource     string                  // The resource being accessed
	Action       string                  // The action being performed
	Value        float64                 // Monetary or numeric value (if applicable)
	EntityType   compliance.EntityType   // Type of entity making the request
	Jurisdiction compliance.Jurisdiction // Jurisdiction where the action occurs
	Claims       map[string]interface{}  // Additional claims from token
	Timestamp    time.Time
}

// EnforcementEngine provides runtime jurisdiction-specific enforcement.
type EnforcementEngine struct {
	mu                       sync.RWMutex
	validator                *compliance.LegalFrameworkValidator
	enabled                  bool
	metrics                  *EnforcementMetrics
	auditCallback            func(decision EnforcementDecision)
	jurisdictionRules        map[compliance.Jurisdiction]*JurisdictionEnforcement
	requireDetachedSignature bool // if true, deny requests missing detached signature artifact
}

// JurisdictionEnforcement contains enforcement-specific configuration for a jurisdiction.
type JurisdictionEnforcement struct {
	Jurisdiction       compliance.Jurisdiction
	StrictMode         bool                // If true, enforce all rules strictly
	AllowedActions     map[string]bool     // Whitelist of allowed actions
	BlockedActions     map[string]bool     // Blacklist of blocked actions
	CrossBorderRules   map[string][]string // action -> list of allowed destination jurisdictions
	DataResidencyRules map[string]bool     // data_type -> must_stay_local
	CustomValidators   map[string]func(ctx *EnforcementContext) error
}

// EnforcementMetrics tracks enforcement metrics.
type EnforcementMetrics struct {
	mu                      sync.RWMutex
	TotalEnforcements       int64
	AllowedCount            int64
	DeniedCount             int64
	JurisdictionBreakdown   map[compliance.Jurisdiction]int64
	ViolationsByType        map[string]int64
	AverageLatencyMs        float64
	CrossBorderAttempts     int64
	CrossBorderDenials      int64
	DataResidencyViolations int64
}

// NewEnforcementEngine creates a new jurisdiction enforcement engine.
func NewEnforcementEngine() *EnforcementEngine {
	engine := &EnforcementEngine{
		validator: compliance.NewLegalFrameworkValidator(),
		enabled:   true,
		metrics: &EnforcementMetrics{
			JurisdictionBreakdown: make(map[compliance.Jurisdiction]int64),
			ViolationsByType:      make(map[string]int64),
		},
		jurisdictionRules:        make(map[compliance.Jurisdiction]*JurisdictionEnforcement),
		requireDetachedSignature: parseBoolEnv("AGENTAUTH_REQUIRE_DETACHED_SIGNATURE"),
	}

	// Attempt external rules load first (allows overriding built-ins without code changes).
	if path := os.Getenv("AGENTAUTH_JURISDICTION_RULES_PATH"); path != "" {
		if err := engine.loadRulesFromFile(path); err == nil {
			// Successfully loaded external rules; skip defaults.
			return engine
		} else {
			// Fall back to defaults on failure (log best-effort; avoid panics in constructor).
			fmt.Fprintf(os.Stderr, "[jurisdiction] external rules load failed path=%s err=%v (falling back to defaults)\n", path, err)
		}
	}

	// Initialize default jurisdiction enforcement rules when no external override.
	engine.initializeDefaultEnforcement()
	return engine
}

// parseBoolEnv returns true if environment variable is set to a truthy value: "1","true","yes" (case-insensitive).
func parseBoolEnv(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// errExternalLoad formats nil-safe error strings.
// (removed helper errExternalLoad - direct logging used)

// jurisdictionRulesFile models the external JSON configuration schema.
// Example schema:
//
//	{
//	  "jurisdictions": [
//	    {
//	      "jurisdiction": "UNITED_STATES",
//	      "strict_mode": true,
//	      "allowed_actions": ["transfer","pay"],
//	      "blocked_actions": ["high_value_transfer"],
//	      "cross_border_rules": {"transfer": ["CANADA","UNITED_KINGDOM"]},
//	      "data_residency_rules": {"personal_data": true}
//	    }
//	  ]
//	}
type jurisdictionRulesFile struct {
	Jurisdictions []struct {
		Jurisdiction       string              `json:"jurisdiction"`
		StrictMode         bool                `json:"strict_mode"`
		AllowedActions     []string            `json:"allowed_actions"`
		BlockedActions     []string            `json:"blocked_actions"`
		CrossBorderRules   map[string][]string `json:"cross_border_rules"`
		DataResidencyRules map[string]bool     `json:"data_residency_rules"`
	} `json:"jurisdictions"`
}

// loadRulesFromFile loads jurisdiction enforcement overrides from a JSON file.
// On success it replaces any existing in-memory rules. Unknown jurisdictions are accepted
// (they will still pass validator checks only if validator supports them). Returns error on parse/read.
func (e *EnforcementEngine) loadRulesFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg jurisdictionRulesFile
	if err := json.Unmarshal(b, &cfg); err != nil {
		return err
	}
	if len(cfg.Jurisdictions) == 0 {
		return fmt.Errorf("no jurisdictions defined in rules file")
	}
	newMap := make(map[compliance.Jurisdiction]*JurisdictionEnforcement, len(cfg.Jurisdictions))
	for _, j := range cfg.Jurisdictions {
		jid := compliance.Jurisdiction(strings.TrimSpace(strings.ToUpper(j.Jurisdiction)))
		allowed := make(map[string]bool)
		for _, a := range j.AllowedActions {
			allowed[a] = true
		}
		blocked := make(map[string]bool)
		for _, b2 := range j.BlockedActions {
			blocked[b2] = true
		}
		newMap[jid] = &JurisdictionEnforcement{
			Jurisdiction:       jid,
			StrictMode:         j.StrictMode,
			AllowedActions:     allowed,
			BlockedActions:     blocked,
			CrossBorderRules:   j.CrossBorderRules,
			DataResidencyRules: j.DataResidencyRules,
			CustomValidators:   make(map[string]func(ctx *EnforcementContext) error),
		}
	}
	e.mu.Lock()
	e.jurisdictionRules = newMap
	e.mu.Unlock()
	fmt.Fprintf(os.Stderr, "[jurisdiction] loaded external enforcement rules jurisdictions=%d path=%s\n", len(newMap), path)
	return nil
}

// Enforce performs jurisdiction-specific enforcement of an action.
func (e *EnforcementEngine) Enforce(ctx context.Context, enfCtx *EnforcementContext) (*EnforcementDecision, error) {
	startTime := time.Now()

	e.mu.RLock()
	enabled := e.enabled
	e.mu.RUnlock()

	if !enabled {
		return &EnforcementDecision{
			Allowed:      true,
			Jurisdiction: enfCtx.Jurisdiction,
			Warnings:     []string{"jurisdiction enforcement is disabled"},
			Timestamp:    time.Now(),
			RequestID:    enfCtx.RequestID,
		}, nil
	}

	decision := &EnforcementDecision{
		Allowed:           true,
		Jurisdiction:      enfCtx.Jurisdiction,
		AppliedRules:      []string{},
		RequiredApprovals: []compliance.ApprovalLevel{},
		ValueLimits:       make(map[string]float64),
		Violations:        []string{},
		Warnings:          []string{},
		Timestamp:         time.Now(),
		RequestID:         enfCtx.RequestID,
	}

	// Step 0 (optional): Detached signature enforcement
	if e.requireDetachedSignature {
		if _, ok := enfCtx.Claims["detached_signature"]; !ok {
			decision.Allowed = false
			decision.Violations = append(decision.Violations, "missing_detached_signature")
			e.recordDenial(enfCtx.Jurisdiction, "missing_detached_signature")
			decision.EnforcementLatency = time.Since(startTime)
			// Record latency even on early return so metrics AverageLatencyMs is never zero when denials occur.
			e.updateLatency(decision.EnforcementLatency)
			e.notifyAudit(*decision)
			return decision, nil
		}
		decision.AppliedRules = append(decision.AppliedRules, "detached_signature_present")
	}

	// Step 1: Validate jurisdiction is supported
	if err := e.validator.ValidateJurisdiction(ctx, enfCtx.Jurisdiction, enfCtx.Action); err != nil {
		decision.Allowed = false
		decision.Violations = append(decision.Violations, fmt.Sprintf("jurisdiction validation failed: %v", err))
		e.recordDenial(enfCtx.Jurisdiction, "jurisdiction_validation_failed")
		decision.EnforcementLatency = time.Since(startTime)
		e.updateLatency(decision.EnforcementLatency)
		e.notifyAudit(*decision)
		return decision, nil
	}
	decision.AppliedRules = append(decision.AppliedRules, "jurisdiction_validation")

	// Step 2: Validate entity type
	if enfCtx.EntityType != "" {
		if err := e.validator.ValidateEntityType(enfCtx.Jurisdiction, enfCtx.EntityType); err != nil {
			decision.Allowed = false
			decision.Violations = append(decision.Violations, fmt.Sprintf("entity type not supported: %v", err))
			e.recordDenial(enfCtx.Jurisdiction, "entity_type_unsupported")
			decision.EnforcementLatency = time.Since(startTime)
			e.updateLatency(decision.EnforcementLatency)
			e.notifyAudit(*decision)
			return decision, nil
		}
		decision.AppliedRules = append(decision.AppliedRules, "entity_type_validation")
	}

	// Step 3: Get jurisdiction requirements
	requirements, err := e.validator.GetJurisdictionRules(enfCtx.Jurisdiction)
	if err != nil {
		decision.Allowed = false
		decision.Violations = append(decision.Violations, fmt.Sprintf("failed to get jurisdiction rules: %v", err))
		e.recordDenial(enfCtx.Jurisdiction, "rules_retrieval_failed")
		decision.EnforcementLatency = time.Since(startTime)
		e.updateLatency(decision.EnforcementLatency)
		e.notifyAudit(*decision)
		return decision, nil
	}

	// Step 4: Check value limits
	if enfCtx.Value > 0 {
		if limit, exists := requirements.ValueLimits[enfCtx.Action]; exists {
			decision.ValueLimits[enfCtx.Action] = limit
			if enfCtx.Value > limit {
				decision.Allowed = false
				decision.Violations = append(decision.Violations,
					fmt.Sprintf("value %.2f exceeds limit %.2f for action %s", enfCtx.Value, limit, enfCtx.Action))
				e.recordDenial(enfCtx.Jurisdiction, "value_limit_exceeded")
			} else {
				decision.AppliedRules = append(decision.AppliedRules, "value_limit_check")
			}
		}
	}

	// Step 5: Check required approvals
	if approvalLevel, exists := requirements.RequiredApprovals[enfCtx.Action]; exists {
		decision.RequiredApprovals = append(decision.RequiredApprovals, approvalLevel)
		decision.AppliedRules = append(decision.AppliedRules, "approval_level_determined")
	}

	// Step 6: Apply jurisdiction-specific enforcement rules
	e.mu.RLock()
	jurisdictionEnforcement, hasCustomRules := e.jurisdictionRules[enfCtx.Jurisdiction]
	e.mu.RUnlock()

	if hasCustomRules {
		if err := e.applyJurisdictionEnforcement(enfCtx, jurisdictionEnforcement, decision); err != nil {
			decision.Allowed = false
			decision.Violations = append(decision.Violations, fmt.Sprintf("jurisdiction enforcement failed: %v", err))
			e.recordDenial(enfCtx.Jurisdiction, "custom_enforcement_failed")
		}
	}

	// Step 7: Apply compliance rule validations
	for _, rule := range requirements.ComplianceRules {
		if rule.Mandatory && rule.Validation != nil {
			if err := rule.Validation(enfCtx.Action); err != nil {
				decision.Allowed = false
				decision.Violations = append(decision.Violations,
					fmt.Sprintf("compliance rule %s failed: %v", rule.Framework, err))
				e.recordDenial(enfCtx.Jurisdiction, fmt.Sprintf("compliance_%s_failed", rule.Framework))
			} else {
				decision.AppliedRules = append(decision.AppliedRules, fmt.Sprintf("compliance_%s", rule.Framework))
			}
		}
	}

	// Record metrics
	decision.EnforcementLatency = time.Since(startTime)
	if decision.Allowed {
		e.recordAllow(enfCtx.Jurisdiction)
	} else {
		e.recordDenial(enfCtx.Jurisdiction, "enforcement_denied")
	}
	e.updateLatency(decision.EnforcementLatency)

	// Notify audit
	e.notifyAudit(*decision)

	return decision, nil
}

// applyJurisdictionEnforcement applies jurisdiction-specific enforcement rules.
func (e *EnforcementEngine) applyJurisdictionEnforcement(
	enfCtx *EnforcementContext,
	enforcement *JurisdictionEnforcement,
	decision *EnforcementDecision,
) error {
	// Check blocked actions
	if enforcement.BlockedActions != nil {
		if blocked, exists := enforcement.BlockedActions[enfCtx.Action]; exists && blocked {
			return fmt.Errorf("action %s is blocked in jurisdiction %s", enfCtx.Action, enfCtx.Jurisdiction)
		}
	}

	// Check allowed actions (if whitelist mode)
	if len(enforcement.AllowedActions) > 0 {
		if allowed, exists := enforcement.AllowedActions[enfCtx.Action]; !exists || !allowed {
			return fmt.Errorf("action %s is not in allowed list for jurisdiction %s", enfCtx.Action, enfCtx.Jurisdiction)
		}
	}

	// Check cross-border rules
	if destinationJurisdiction, ok := enfCtx.Claims["destination_jurisdiction"].(string); ok {
		e.metrics.mu.Lock()
		e.metrics.CrossBorderAttempts++
		e.metrics.mu.Unlock()

		if allowedDestinations, exists := enforcement.CrossBorderRules[enfCtx.Action]; exists {
			allowed := false
			for _, dest := range allowedDestinations {
				if dest == destinationJurisdiction {
					allowed = true
					break
				}
			}
			if !allowed {
				e.metrics.mu.Lock()
				e.metrics.CrossBorderDenials++
				e.metrics.mu.Unlock()
				return fmt.Errorf("cross-border action %s to %s not allowed from %s",
					enfCtx.Action, destinationJurisdiction, enfCtx.Jurisdiction)
			}
			decision.AppliedRules = append(decision.AppliedRules, "cross_border_validation")
		}
	}

	// Check data residency rules
	if dataType, ok := enfCtx.Claims["data_type"].(string); ok {
		if mustStayLocal, exists := enforcement.DataResidencyRules[dataType]; exists && mustStayLocal {
			// Check if data is leaving jurisdiction
			if destJurisdiction, ok := enfCtx.Claims["destination_jurisdiction"].(string); ok {
				if string(enfCtx.Jurisdiction) != destJurisdiction {
					e.metrics.mu.Lock()
					e.metrics.DataResidencyViolations++
					e.metrics.mu.Unlock()
					return fmt.Errorf("data residency violation: %s data must remain in %s jurisdiction",
						dataType, enfCtx.Jurisdiction)
				}
			}
			decision.AppliedRules = append(decision.AppliedRules, "data_residency_validation")
		}
	}

	// Apply custom validators
	if enforcement.CustomValidators != nil {
		if validator, exists := enforcement.CustomValidators[enfCtx.Action]; exists {
			if err := validator(enfCtx); err != nil {
				return fmt.Errorf("custom validation failed: %w", err)
			}
			decision.AppliedRules = append(decision.AppliedRules, "custom_validation")
		}
	}

	return nil
}

// initializeDefaultEnforcement sets up default jurisdiction enforcement rules.
func (e *EnforcementEngine) initializeDefaultEnforcement() {
	// EU - GDPR strict enforcement
	e.jurisdictionRules[compliance.JurisdictionEU] = &JurisdictionEnforcement{
		Jurisdiction: compliance.JurisdictionEU,
		StrictMode:   true,
		BlockedActions: map[string]bool{
			"unrestricted_data_export": true,
			"automated_profiling":      true,
			"bulk_data_transfer":       true,
		},
		CrossBorderRules: map[string][]string{
			"personal_data_transfer": {"EU", "UK"},       // Only allow to adequacy countries
			"financial_data_export":  {"EU", "UK", "US"}, // Financial data has more flexibility
		},
		DataResidencyRules: map[string]bool{
			"personal_data":  true,  // Must stay in EU
			"health_data":    true,  // Must stay in EU
			"financial_data": false, // Can cross borders with safeguards
		},
		CustomValidators: map[string]func(ctx *EnforcementContext) error{
			"gdpr_data_processing": func(ctx *EnforcementContext) error {
				// Check for GDPR consent
				if consent, ok := ctx.Claims["gdpr_consent"].(bool); !ok || !consent {
					return fmt.Errorf("GDPR consent required for data processing")
				}
				return nil
			},
		},
	}

	// US - CCPA and sector-specific regulations
	e.jurisdictionRules[compliance.JurisdictionUS] = &JurisdictionEnforcement{
		Jurisdiction: compliance.JurisdictionUS,
		StrictMode:   false, // More flexible
		BlockedActions: map[string]bool{
			"autonomous_high_risk_decision": true,
		},
		CrossBorderRules: map[string][]string{
			"personal_data_transfer": {"US", "EU", "UK", "CA", "AU", "JP"}, // More permissive
			"financial_data_export":  {"US", "EU", "UK", "CA", "AU", "JP"},
		},
		DataResidencyRules: map[string]bool{
			"personal_data":  false, // CCPA allows transfer with notice
			"health_data":    false, // HIPAA has separate rules
			"financial_data": false,
		},
		CustomValidators: map[string]func(ctx *EnforcementContext) error{
			"ccpa_data_processing": func(ctx *EnforcementContext) error {
				// Check for CCPA opt-out
				if optOut, ok := ctx.Claims["ccpa_opt_out"].(bool); ok && optOut {
					return fmt.Errorf("CCPA opt-out prevents data processing")
				}
				return nil
			},
		},
	}

	// UK - Post-Brexit UK GDPR
	e.jurisdictionRules[compliance.JurisdictionUK] = &JurisdictionEnforcement{
		Jurisdiction: compliance.JurisdictionUK,
		StrictMode:   true,
		BlockedActions: map[string]bool{
			"unrestricted_data_export": true,
		},
		CrossBorderRules: map[string][]string{
			"personal_data_transfer": {"UK", "EU"}, // Maintain EU adequacy
			"financial_data_export":  {"UK", "EU", "US"},
		},
		DataResidencyRules: map[string]bool{
			"personal_data":  true, // Similar to GDPR
			"health_data":    true,
			"financial_data": false,
		},
	}

	// Canada - PIPEDA
	e.jurisdictionRules[compliance.JurisdictionCA] = &JurisdictionEnforcement{
		Jurisdiction: compliance.JurisdictionCA,
		StrictMode:   false,
		CrossBorderRules: map[string][]string{
			"personal_data_transfer": {"CA", "US", "EU", "UK"}, // More permissive
			"financial_data_export":  {"CA", "US", "EU", "UK"},
		},
		DataResidencyRules: map[string]bool{
			"personal_data":  false, // PIPEDA allows transfer with safeguards
			"health_data":    true,  // Provincial health laws stricter
			"financial_data": false,
		},
	}

	// Australia - Privacy Act
	e.jurisdictionRules[compliance.JurisdictionAU] = &JurisdictionEnforcement{
		Jurisdiction: compliance.JurisdictionAU,
		StrictMode:   false,
		CrossBorderRules: map[string][]string{
			"personal_data_transfer": {"AU", "US", "EU", "UK", "JP"}, // APPs allow with accountability
			"financial_data_export":  {"AU", "US", "EU", "UK", "JP"},
		},
		DataResidencyRules: map[string]bool{
			"personal_data":  false,
			"health_data":    false,
			"financial_data": false,
		},
	}

	// Japan - APPI
	e.jurisdictionRules[compliance.JurisdictionJP] = &JurisdictionEnforcement{
		Jurisdiction: compliance.JurisdictionJP,
		StrictMode:   true,
		CrossBorderRules: map[string][]string{
			"personal_data_transfer": {"JP", "EU"}, // Requires white-listed countries
			"financial_data_export":  {"JP", "US", "EU", "UK"},
		},
		DataResidencyRules: map[string]bool{
			"personal_data":  true, // APPI strict on data export
			"health_data":    true,
			"financial_data": false,
		},
	}
}

// ExtractJurisdictionFromClaims extracts jurisdiction from token claims.
func ExtractJurisdictionFromClaims(claims map[string]interface{}) compliance.Jurisdiction {
	if jurisdiction, ok := claims["jurisdiction"].(string); ok {
		return compliance.Jurisdiction(strings.ToUpper(jurisdiction))
	}
	if location, ok := claims["location"].(string); ok {
		// Try to map location to jurisdiction
		locationUpper := strings.ToUpper(location)
		if strings.Contains(locationUpper, "EU") || strings.Contains(locationUpper, "EUROPE") {
			return compliance.JurisdictionEU
		}
		if strings.Contains(locationUpper, "US") || strings.Contains(locationUpper, "USA") {
			return compliance.JurisdictionUS
		}
		// Support multiple UK identifiers ("UK", "Britain", "United Kingdom")
		if strings.Contains(locationUpper, "UK") || strings.Contains(locationUpper, "BRITAIN") || strings.Contains(locationUpper, "UNITED KINGDOM") {
			return compliance.JurisdictionUK
		}
		if strings.Contains(locationUpper, "CA") || strings.Contains(locationUpper, "CANADA") {
			return compliance.JurisdictionCA
		}
		if strings.Contains(locationUpper, "AU") || strings.Contains(locationUpper, "AUSTRALIA") {
			return compliance.JurisdictionAU
		}
		if strings.Contains(locationUpper, "JP") || strings.Contains(locationUpper, "JAPAN") {
			return compliance.JurisdictionJP
		}
	}
	// Default to US if not specified
	return compliance.JurisdictionUS
}

// SetEnabled enables or disables enforcement.
func (e *EnforcementEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
}

// IsEnabled returns whether enforcement is currently enabled.
func (e *EnforcementEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// SetAuditCallback sets the audit callback function.
func (e *EnforcementEngine) SetAuditCallback(callback func(decision EnforcementDecision)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.auditCallback = callback
}

// GetMetrics returns current enforcement metrics.
func (e *EnforcementEngine) GetMetrics() *EnforcementMetrics {
	e.metrics.mu.RLock()
	defer e.metrics.mu.RUnlock()
	return e.metrics
}

// GetJurisdictionEnforcement returns enforcement rules for a jurisdiction.
func (e *EnforcementEngine) GetJurisdictionEnforcement(jurisdiction compliance.Jurisdiction) (*JurisdictionEnforcement, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	enforcement, exists := e.jurisdictionRules[jurisdiction]
	if !exists {
		return nil, fmt.Errorf("no enforcement rules for jurisdiction %s", jurisdiction)
	}

	return enforcement, nil
}

// recordAllow records a successful enforcement.
func (e *EnforcementEngine) recordAllow(jurisdiction compliance.Jurisdiction) {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()
	e.metrics.TotalEnforcements++
	e.metrics.AllowedCount++
	e.metrics.JurisdictionBreakdown[jurisdiction]++
}

// recordDenial records a denied enforcement.
func (e *EnforcementEngine) recordDenial(jurisdiction compliance.Jurisdiction, violationType string) {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()
	e.metrics.TotalEnforcements++
	e.metrics.DeniedCount++
	e.metrics.JurisdictionBreakdown[jurisdiction]++
	e.metrics.ViolationsByType[violationType]++
}

// updateLatency updates average latency metrics.
func (e *EnforcementEngine) updateLatency(latency time.Duration) {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()

	// Very fast operations may be < 1µs causing Microseconds() to truncate to 0.
	// Normalize any sub-microsecond or non-positive durations to 1µs so average is never zero after first update.
	if latency <= 0 || latency.Microseconds() == 0 {
		latency = time.Microsecond
	}
	latencyMs := float64(latency.Microseconds()) / 1000.0
	alpha := 0.1
	if e.metrics.AverageLatencyMs == 0 {
		e.metrics.AverageLatencyMs = latencyMs
	} else {
		e.metrics.AverageLatencyMs = alpha*latencyMs + (1-alpha)*e.metrics.AverageLatencyMs
	}
}

// notifyAudit notifies the audit callback if set.
func (e *EnforcementEngine) notifyAudit(decision EnforcementDecision) {
	e.mu.RLock()
	callback := e.auditCallback
	e.mu.RUnlock()

	if callback != nil {
		go callback(decision)
	}
}
