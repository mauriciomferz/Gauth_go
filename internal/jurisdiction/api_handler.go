package jurisdiction

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/compliance"
)

// APIHandler provides REST API endpoints for jurisdiction management.
type APIHandler struct {
	integration *ServerIntegration
}

// NewAPIHandler creates a new API handler.
func NewAPIHandler(integration *ServerIntegration) *APIHandler {
	return &APIHandler{
		integration: integration,
	}
}

// RegisterRoutes registers jurisdiction API routes with Gin router.
func (h *APIHandler) RegisterRoutes(r *gin.Engine) {
	jurisdictionGroup := r.Group("/api/v1/jurisdiction")
	{
		jurisdictionGroup.GET("/status", h.getStatus)
		jurisdictionGroup.GET("/supported", h.getSupportedJurisdictions)
		jurisdictionGroup.GET("/rules/:jurisdiction", h.getJurisdictionRules)
		jurisdictionGroup.GET("/metrics", h.getMetrics)
		// Prometheus exposition (text/plain) for jurisdiction enforcement metrics.
		jurisdictionGroup.GET("/metrics/prometheus", h.getMetricsPrometheus)
		// Validator (legal framework) metrics JSON + Prometheus
		jurisdictionGroup.GET("/validator/metrics", h.getValidatorMetrics)
		jurisdictionGroup.GET("/validator/metrics/prometheus", h.getValidatorMetricsPrometheus)
		jurisdictionGroup.POST("/enforce", h.enforceAction)
		jurisdictionGroup.POST("/validate", h.validateAction)
		jurisdictionGroup.POST("/simulate", h.simulateEnforcement)
		jurisdictionGroup.POST("/enable", h.enableEnforcement)
		jurisdictionGroup.POST("/disable", h.disableEnforcement)
		jurisdictionGroup.GET("/health", h.healthCheck)
	}
}

// getStatus returns the current status of jurisdiction enforcement.
func (h *APIHandler) getStatus(c *gin.Context) {
	engine := h.integration.GetEnforcementEngine()
	validator := engine.validator

	c.JSON(http.StatusOK, gin.H{
		"enabled":                 engine.IsEnabled(),
		"supported_jurisdictions": validator.GetSupportedJurisdictions(),
		"metrics":                 engine.GetMetrics(),
	})
}

// getSupportedJurisdictions returns list of supported jurisdictions.
func (h *APIHandler) getSupportedJurisdictions(c *gin.Context) {
	engine := h.integration.GetEnforcementEngine()
	validator := engine.validator

	jurisdictions := validator.GetSupportedJurisdictions()
	jurisdictionInfo := make([]map[string]interface{}, 0, len(jurisdictions))

	for _, jurisdiction := range jurisdictions {
		rules, err := validator.GetJurisdictionRules(jurisdiction)
		if err != nil {
			continue
		}

		jurisdictionInfo = append(jurisdictionInfo, map[string]interface{}{
			"jurisdiction":          jurisdiction,
			"supported_entities":    rules.SupportedEntities,
			"compliance_frameworks": extractFrameworks(rules.ComplianceRules),
			"has_value_limits":      len(rules.ValueLimits) > 0,
			"has_time_restrictions": len(rules.TimeRestrictions) > 0,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"jurisdictions": jurisdictionInfo,
		"total":         len(jurisdictionInfo),
	})
}

// getJurisdictionRules returns detailed rules for a jurisdiction.
func (h *APIHandler) getJurisdictionRules(c *gin.Context) {
	jurisdictionStr := c.Param("jurisdiction")
	jurisdiction := compliance.Jurisdiction(jurisdictionStr)

	engine := h.integration.GetEnforcementEngine()
	validator := engine.validator

	rules, err := validator.GetJurisdictionRules(jurisdiction)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":        "jurisdiction not found",
			"jurisdiction": jurisdictionStr,
		})
		return
	}

	enforcement, err := engine.GetJurisdictionEnforcement(jurisdiction)
	if err != nil {
		// Return basic rules even if no enforcement config
		c.JSON(http.StatusOK, gin.H{
			"jurisdiction":           jurisdiction,
			"supported_entities":     rules.SupportedEntities,
			"compliance_rules":       formatComplianceRules(rules.ComplianceRules),
			"value_limits":           rules.ValueLimits,
			"required_approvals":     rules.RequiredApprovals,
			"time_restrictions":      rules.TimeRestrictions,
			"has_custom_enforcement": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jurisdiction":           jurisdiction,
		"supported_entities":     rules.SupportedEntities,
		"compliance_rules":       formatComplianceRules(rules.ComplianceRules),
		"value_limits":           rules.ValueLimits,
		"required_approvals":     rules.RequiredApprovals,
		"time_restrictions":      rules.TimeRestrictions,
		"has_custom_enforcement": true,
		"enforcement": gin.H{
			"strict_mode":           enforcement.StrictMode,
			"blocked_actions":       enforcement.BlockedActions,
			"cross_border_rules":    enforcement.CrossBorderRules,
			"data_residency_rules":  enforcement.DataResidencyRules,
			"has_custom_validators": len(enforcement.CustomValidators) > 0,
		},
	})
}

// getMetrics returns current enforcement metrics.
func (h *APIHandler) getMetrics(c *gin.Context) {
	metrics := h.integration.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"total_enforcements":        metrics.TotalEnforcements,
		"allowed_count":             metrics.AllowedCount,
		"denied_count":              metrics.DeniedCount,
		"allow_rate":                calculateAllowRate(metrics),
		"jurisdiction_breakdown":    metrics.JurisdictionBreakdown,
		"violations_by_type":        metrics.ViolationsByType,
		"average_latency_ms":        metrics.AverageLatencyMs,
		"cross_border_attempts":     metrics.CrossBorderAttempts,
		"cross_border_denials":      metrics.CrossBorderDenials,
		"cross_border_success_rate": calculateCrossBorderSuccessRate(metrics),
		"data_residency_violations": metrics.DataResidencyViolations,
	})
}

// getMetricsPrometheus returns enforcement metrics in Prometheus exposition format.
// Metric set kept minimal and stable; numerical values are counters or gauges as appropriate:
// gauth_jurisdiction_enforcements_total
// gauth_jurisdiction_enforcements_allowed_total
// gauth_jurisdiction_enforcements_denied_total
// gauth_jurisdiction_average_latency_ms (gauge)
// gauth_jurisdiction_cross_border_attempts_total
// gauth_jurisdiction_cross_border_denials_total
// gauth_jurisdiction_data_residency_violations_total
// Per-jurisdiction breakdown: gauth_jurisdiction_enforcements_total{jurisdiction="EU"}
// Violation types: gauth_jurisdiction_violation_total{type="ccpa_opt_out"}
func (h *APIHandler) getMetricsPrometheus(c *gin.Context) {
	m := h.integration.GetMetrics()
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	// Build deterministic ordering for maps
	// Jurisdiction breakdown
	jKeys := make([]string, 0, len(m.JurisdictionBreakdown))
	for j := range m.JurisdictionBreakdown {
		jKeys = append(jKeys, string(j))
	}
	// Violations by type
	vKeys := make([]string, 0, len(m.ViolationsByType))
	for vt := range m.ViolationsByType {
		vKeys = append(vKeys, vt)
	}
	// Simple insertion sort (maps are small)
	for i := 1; i < len(jKeys); i++ {
		for j := i; j > 0 && jKeys[j] < jKeys[j-1]; j-- {
			jKeys[j], jKeys[j-1] = jKeys[j-1], jKeys[j]
		}
	}
	for i := 1; i < len(vKeys); i++ {
		for j := i; j > 0 && vKeys[j] < vKeys[j-1]; j-- {
			vKeys[j], vKeys[j-1] = vKeys[j-1], vKeys[j]
		}
	}
	b := &strings.Builder{}
	b.WriteString("# HELP gauth_jurisdiction_enforcements_total Total jurisdiction enforcement attempts.\n")
	b.WriteString("# TYPE gauth_jurisdiction_enforcements_total counter\n")
	fmt.Fprintf(b, "gauth_jurisdiction_enforcements_total %d\n", m.TotalEnforcements)
	b.WriteString("# HELP gauth_jurisdiction_enforcements_allowed_total Total allowed jurisdiction decisions.\n")
	b.WriteString("# TYPE gauth_jurisdiction_enforcements_allowed_total counter\n")
	fmt.Fprintf(b, "gauth_jurisdiction_enforcements_allowed_total %d\n", m.AllowedCount)
	b.WriteString("# HELP gauth_jurisdiction_enforcements_denied_total Total denied jurisdiction decisions.\n")
	b.WriteString("# TYPE gauth_jurisdiction_enforcements_denied_total counter\n")
	fmt.Fprintf(b, "gauth_jurisdiction_enforcements_denied_total %d\n", m.DeniedCount)
	b.WriteString("# HELP gauth_jurisdiction_average_latency_ms Average jurisdiction enforcement latency (EMA).\n")
	b.WriteString("# TYPE gauth_jurisdiction_average_latency_ms gauge\n")
	fmt.Fprintf(b, "gauth_jurisdiction_average_latency_ms %.6f\n", m.AverageLatencyMs)
	b.WriteString("# HELP gauth_jurisdiction_cross_border_attempts_total Total cross-border enforcement attempts.\n")
	b.WriteString("# TYPE gauth_jurisdiction_cross_border_attempts_total counter\n")
	fmt.Fprintf(b, "gauth_jurisdiction_cross_border_attempts_total %d\n", m.CrossBorderAttempts)
	b.WriteString("# HELP gauth_jurisdiction_cross_border_denials_total Total cross-border denials.\n")
	b.WriteString("# TYPE gauth_jurisdiction_cross_border_denials_total counter\n")
	fmt.Fprintf(b, "gauth_jurisdiction_cross_border_denials_total %d\n", m.CrossBorderDenials)
	b.WriteString("# HELP gauth_jurisdiction_data_residency_violations_total Total data residency violations.\n")
	b.WriteString("# TYPE gauth_jurisdiction_data_residency_violations_total counter\n")
	fmt.Fprintf(b, "gauth_jurisdiction_data_residency_violations_total %d\n", m.DataResidencyViolations)
	// Jurisdiction breakdown metrics
	b.WriteString("# HELP gauth_jurisdiction_enforcements_by_jurisdiction_total Total enforcements per jurisdiction.\n")
	b.WriteString("# TYPE gauth_jurisdiction_enforcements_by_jurisdiction_total counter\n")
	for _, jk := range jKeys {
		fmt.Fprintf(b, "gauth_jurisdiction_enforcements_by_jurisdiction_total{jurisdiction=\"%s\"} %d\n", jk, m.JurisdictionBreakdown[compliance.Jurisdiction(jk)])
	}
	// Violation types
	b.WriteString("# HELP gauth_jurisdiction_violation_total Jurisdiction enforcement violation occurrences by type.\n")
	b.WriteString("# TYPE gauth_jurisdiction_violation_total counter\n")
	for _, vk := range vKeys {
		fmt.Fprintf(b, "gauth_jurisdiction_violation_total{type=\"%s\"} %d\n", vk, m.ViolationsByType[vk])
	}
	c.String(200, b.String())
}

// getValidatorMetrics returns legal framework validator metrics in JSON.
func (h *APIHandler) getValidatorMetrics(c *gin.Context) {
	engine := h.integration.GetEnforcementEngine()
	validator := engine.validator
	m := validator.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"validation_attempts":         m.ValidationAttempts,
		"validation_successes":        m.ValidationSuccesses,
		"validation_failures":         m.ValidationFailures,
		"entity_validation_attempts":  m.EntityValidationAttempts,
		"entity_validation_failures":  m.EntityValidationFailures,
		"value_limit_checks":          m.ValueLimitChecks,
		"value_limit_violations":      m.ValueLimitViolations,
		"approval_checks":             m.ApprovalChecks,
		"approval_failures":           m.ApprovalFailures,
		"board_approval_checks":       m.BoardApprovalChecks,
		"board_approval_failures":     m.BoardApprovalFailures,
		"total_validation_latency_ms": float64(m.TotalValidationLatencyNs) / 1e6,
		"last_validation_latency_ms":  float64(m.LastValidationLatencyNs) / 1e6,
		"jurisdiction_counts":         m.JurisdictionCounts,
		"violation_counts":            m.ViolationCounts,
	})
}

// getValidatorMetricsPrometheus exposes validator metrics in Prometheus 0.0.4 format.
func (h *APIHandler) getValidatorMetricsPrometheus(c *gin.Context) {
	engine := h.integration.GetEnforcementEngine()
	validator := engine.validator
	m := validator.GetMetrics()
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	// Deterministic ordering of maps
	jKeys := make([]string, 0, len(m.JurisdictionCounts))
	for j := range m.JurisdictionCounts {
		jKeys = append(jKeys, string(j))
	}
	vKeys := make([]string, 0, len(m.ViolationCounts))
	for vt := range m.ViolationCounts {
		vKeys = append(vKeys, vt)
	}
	for i := 1; i < len(jKeys); i++ {
		for j := i; j > 0 && jKeys[j] < jKeys[j-1]; j-- {
			jKeys[j], jKeys[j-1] = jKeys[j-1], jKeys[j]
		}
	}
	for i := 1; i < len(vKeys); i++ {
		for j := i; j > 0 && vKeys[j] < vKeys[j-1]; j-- {
			vKeys[j], vKeys[j-1] = vKeys[j-1], vKeys[j]
		}
	}
	b := &strings.Builder{}
	b.WriteString("# HELP gauth_validator_validation_attempts_total Total validator jurisdiction validation attempts.\n")
	b.WriteString("# TYPE gauth_validator_validation_attempts_total counter\n")
	fmt.Fprintf(b, "gauth_validator_validation_attempts_total %d\n", m.ValidationAttempts)
	b.WriteString("# HELP gauth_validator_validation_successes_total Successful validator jurisdiction validations.\n")
	b.WriteString("# TYPE gauth_validator_validation_successes_total counter\n")
	fmt.Fprintf(b, "gauth_validator_validation_successes_total %d\n", m.ValidationSuccesses)
	b.WriteString("# HELP gauth_validator_validation_failures_total Failed validator jurisdiction validations.\n")
	b.WriteString("# TYPE gauth_validator_validation_failures_total counter\n")
	fmt.Fprintf(b, "gauth_validator_validation_failures_total %d\n", m.ValidationFailures)
	b.WriteString("# HELP gauth_validator_entity_validation_attempts_total Entity type validation attempts.\n")
	b.WriteString("# TYPE gauth_validator_entity_validation_attempts_total counter\n")
	fmt.Fprintf(b, "gauth_validator_entity_validation_attempts_total %d\n", m.EntityValidationAttempts)
	b.WriteString("# HELP gauth_validator_entity_validation_failures_total Entity type validation failures.\n")
	b.WriteString("# TYPE gauth_validator_entity_validation_failures_total counter\n")
	fmt.Fprintf(b, "gauth_validator_entity_validation_failures_total %d\n", m.EntityValidationFailures)
	b.WriteString("# HELP gauth_validator_value_limit_checks_total Value limit checks performed.\n")
	b.WriteString("# TYPE gauth_validator_value_limit_checks_total counter\n")
	fmt.Fprintf(b, "gauth_validator_value_limit_checks_total %d\n", m.ValueLimitChecks)
	b.WriteString("# HELP gauth_validator_value_limit_violations_total Value limit violations detected.\n")
	b.WriteString("# TYPE gauth_validator_value_limit_violations_total counter\n")
	fmt.Fprintf(b, "gauth_validator_value_limit_violations_total %d\n", m.ValueLimitViolations)
	b.WriteString("# HELP gauth_validator_approval_checks_total Approval requirement checks performed.\n")
	b.WriteString("# TYPE gauth_validator_approval_checks_total counter\n")
	fmt.Fprintf(b, "gauth_validator_approval_checks_total %d\n", m.ApprovalChecks)
	b.WriteString("# HELP gauth_validator_approval_failures_total Approval validation failures.\n")
	b.WriteString("# TYPE gauth_validator_approval_failures_total counter\n")
	fmt.Fprintf(b, "gauth_validator_approval_failures_total %d\n", m.ApprovalFailures)
	b.WriteString("# HELP gauth_validator_board_approval_checks_total Board approval checks performed.\n")
	b.WriteString("# TYPE gauth_validator_board_approval_checks_total counter\n")
	fmt.Fprintf(b, "gauth_validator_board_approval_checks_total %d\n", m.BoardApprovalChecks)
	b.WriteString("# HELP gauth_validator_board_approval_failures_total Board approval validation failures.\n")
	b.WriteString("# TYPE gauth_validator_board_approval_failures_total counter\n")
	fmt.Fprintf(b, "gauth_validator_board_approval_failures_total %d\n", m.BoardApprovalFailures)
	b.WriteString("# HELP gauth_validator_total_validation_latency_ms Cumulative jurisdiction validation latency in milliseconds.\n")
	b.WriteString("# TYPE gauth_validator_total_validation_latency_ms counter\n")
	fmt.Fprintf(b, "gauth_validator_total_validation_latency_ms %.6f\n", float64(m.TotalValidationLatencyNs)/1e6)
	b.WriteString("# HELP gauth_validator_last_validation_latency_ms Last jurisdiction validation latency in milliseconds.\n")
	b.WriteString("# TYPE gauth_validator_last_validation_latency_ms gauge\n")
	fmt.Fprintf(b, "gauth_validator_last_validation_latency_ms %.6f\n", float64(m.LastValidationLatencyNs)/1e6)
	// Jurisdiction counts
	b.WriteString("# HELP gauth_validator_validations_by_jurisdiction_total Jurisdiction validation attempts by jurisdiction.\n")
	b.WriteString("# TYPE gauth_validator_validations_by_jurisdiction_total counter\n")
	for _, jk := range jKeys {
		fmt.Fprintf(b, "gauth_validator_validations_by_jurisdiction_total{jurisdiction=\"%s\"} %d\n", jk, m.JurisdictionCounts[compliance.Jurisdiction(jk)])
	}
	// Violation types
	b.WriteString("# HELP gauth_validator_violation_total Validator violation occurrences by type.\n")
	b.WriteString("# TYPE gauth_validator_violation_total counter\n")
	for _, vk := range vKeys {
		fmt.Fprintf(b, "gauth_validator_violation_total{type=\"%s\"} %d\n", vk, m.ViolationCounts[vk])
	}
	c.String(200, b.String())
}

// enforceAction enforces jurisdiction rules for a specific action.
func (h *APIHandler) enforceAction(c *gin.Context) {
	var req struct {
		Subject      string                 `json:"subject" binding:"required"`
		Resource     string                 `json:"resource" binding:"required"`
		Action       string                 `json:"action" binding:"required"`
		Jurisdiction string                 `json:"jurisdiction"`
		Claims       map[string]interface{} `json:"claims"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// Set jurisdiction claim if provided
	if req.Claims == nil {
		req.Claims = make(map[string]interface{})
	}
	if req.Jurisdiction != "" {
		req.Claims["jurisdiction"] = req.Jurisdiction
	}

	decision, err := h.integration.EnforceJurisdiction(
		c.Request.Context(),
		req.Subject,
		req.Resource,
		req.Action,
		req.Claims,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "enforcement failed",
			"details": err.Error(),
		})
		return
	}

	statusCode := http.StatusOK
	if !decision.Allowed {
		statusCode = http.StatusForbidden
	}

	c.JSON(statusCode, gin.H{
		"decision":               decision.Allowed,
		"jurisdiction":           decision.Jurisdiction,
		"applied_rules":          decision.AppliedRules,
		"required_approvals":     decision.RequiredApprovals,
		"value_limits":           decision.ValueLimits,
		"violations":             decision.Violations,
		"warnings":               decision.Warnings,
		"request_id":             decision.RequestID,
		"enforcement_latency_ms": decision.EnforcementLatency.Milliseconds(),
	})
}

// validateAction validates if an action is allowed in a jurisdiction.
func (h *APIHandler) validateAction(c *gin.Context) {
	var req struct {
		Jurisdiction string `json:"jurisdiction" binding:"required"`
		Action       string `json:"action" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	jurisdiction := compliance.Jurisdiction(req.Jurisdiction)
	err := h.integration.ValidateJurisdiction(
		c.Request.Context(),
		jurisdiction,
		req.Action,
	)

	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"valid":        false,
			"jurisdiction": jurisdiction,
			"action":       req.Action,
			"error":        err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":        true,
		"jurisdiction": jurisdiction,
		"action":       req.Action,
	})
}

// simulateEnforcement simulates an enforcement decision without recording it.
func (h *APIHandler) simulateEnforcement(c *gin.Context) {
	var req struct {
		Subject      string                 `json:"subject" binding:"required"`
		Resource     string                 `json:"resource" binding:"required"`
		Action       string                 `json:"action" binding:"required"`
		Jurisdiction string                 `json:"jurisdiction" binding:"required"`
		EntityType   string                 `json:"entity_type"`
		Value        float64                `json:"value"`
		Claims       map[string]interface{} `json:"claims"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// Build claims
	if req.Claims == nil {
		req.Claims = make(map[string]interface{})
	}
	req.Claims["jurisdiction"] = req.Jurisdiction
	if req.EntityType != "" {
		req.Claims["entity_type"] = req.EntityType
	}
	if req.Value > 0 {
		req.Claims["value"] = req.Value
	}

	// Create enforcement context
	enfCtx := &EnforcementContext{
		RequestID:    generateRequestID(),
		Subject:      req.Subject,
		Resource:     req.Resource,
		Action:       req.Action,
		Value:        req.Value,
		EntityType:   compliance.EntityType(req.EntityType),
		Jurisdiction: compliance.Jurisdiction(req.Jurisdiction),
		Claims:       req.Claims,
	}

	// Get engine
	engine := h.integration.GetEnforcementEngine()

	// Simulate without recording to metrics
	decision, err := engine.Enforce(c.Request.Context(), enfCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "simulation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"simulation":             true,
		"decision":               decision.Allowed,
		"jurisdiction":           decision.Jurisdiction,
		"applied_rules":          decision.AppliedRules,
		"required_approvals":     decision.RequiredApprovals,
		"value_limits":           decision.ValueLimits,
		"violations":             decision.Violations,
		"warnings":               decision.Warnings,
		"enforcement_latency_ms": decision.EnforcementLatency.Milliseconds(),
	})
}

// enableEnforcement enables jurisdiction enforcement.
func (h *APIHandler) enableEnforcement(c *gin.Context) {
	h.integration.SetEnabled(true)
	c.JSON(http.StatusOK, gin.H{
		"message": "jurisdiction enforcement enabled",
		"enabled": true,
	})
}

// disableEnforcement disables jurisdiction enforcement.
func (h *APIHandler) disableEnforcement(c *gin.Context) {
	h.integration.SetEnabled(false)
	c.JSON(http.StatusOK, gin.H{
		"message": "jurisdiction enforcement disabled",
		"enabled": false,
	})
}

// healthCheck returns health status of jurisdiction enforcement.
func (h *APIHandler) healthCheck(c *gin.Context) {
	engine := h.integration.GetEnforcementEngine()
	metrics := engine.GetMetrics()

	status := "healthy"
	if !engine.IsEnabled() {
		status = "disabled"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":              status,
		"enabled":             engine.IsEnabled(),
		"total_enforcements":  metrics.TotalEnforcements,
		"uptime_enforcements": metrics.TotalEnforcements,
	})
}

// Helper functions

func extractFrameworks(rules []compliance.ComplianceRule) []string {
	frameworks := make([]string, 0, len(rules))
	for _, rule := range rules {
		frameworks = append(frameworks, rule.Framework)
	}
	return frameworks
}

func formatComplianceRules(rules []compliance.ComplianceRule) []map[string]interface{} {
	formatted := make([]map[string]interface{}, 0, len(rules))
	for _, rule := range rules {
		formatted = append(formatted, map[string]interface{}{
			"framework":   rule.Framework,
			"requirement": rule.Requirement,
			"mandatory":   rule.Mandatory,
		})
	}
	return formatted
}

func calculateAllowRate(metrics *EnforcementMetrics) float64 {
	if metrics.TotalEnforcements == 0 {
		return 0.0
	}
	return float64(metrics.AllowedCount) / float64(metrics.TotalEnforcements) * 100.0
}

func calculateCrossBorderSuccessRate(metrics *EnforcementMetrics) float64 {
	if metrics.CrossBorderAttempts == 0 {
		return 0.0
	}
	successes := metrics.CrossBorderAttempts - metrics.CrossBorderDenials
	return float64(successes) / float64(metrics.CrossBorderAttempts) * 100.0
}
