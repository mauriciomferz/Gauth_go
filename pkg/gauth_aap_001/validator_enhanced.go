package gauth_aap_001

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/aap"
)

// ValidationWarning represents a non-fatal validation issue that should be logged/monitored
type ValidationWarning struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Field     string      `json:"field,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	Severity  string      `json:"severity"` // "info", "warning", "error"
	Timestamp time.Time   `json:"timestamp"`
}

// ValidationResult contains both validation outcome and collected warnings
type ValidationResult struct {
	Valid    bool                   `json:"valid"`
	Error    error                  `json:"error,omitempty"`
	Warnings []ValidationWarning    `json:"warnings,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// WarningCollector interface for warning emission and collection
type WarningCollector interface {
	AddWarning(code, message, field string, value interface{}, severity string)
	GetWarnings() []ValidationWarning
	ClearWarnings()
}

// DefaultWarningCollector implements in-memory warning collection
type DefaultWarningCollector struct {
	warnings []ValidationWarning
	mu       sync.Mutex
}

func NewWarningCollector() *DefaultWarningCollector {
	return &DefaultWarningCollector{
		warnings: make([]ValidationWarning, 0),
	}
}

func (c *DefaultWarningCollector) AddWarning(code, message, field string, value interface{}, severity string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.warnings = append(c.warnings, ValidationWarning{
		Code:      code,
		Message:   message,
		Field:     field,
		Value:     value,
		Severity:  severity,
		Timestamp: time.Now(),
	})
}

func (c *DefaultWarningCollector) GetWarnings() []ValidationWarning {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]ValidationWarning, len(c.warnings))
	copy(result, c.warnings)
	return result
}

func (c *DefaultWarningCollector) ClearWarnings() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.warnings = c.warnings[:0]
}

// EnhancedPoAValidator provides comprehensive PoA semantic validation with warnings
type EnhancedPoAValidator struct {
	BasicPoAValidator
	warningCollector  WarningCollector
	dailyLimitStore   DailyLimitStore
	conditionalEngine ConditionalEngine
	currencyConverter CurrencyConverter
	validatorChain    []PoAValidator
	metricsRecorder   ValidationMetricsRecorder
}

// LimitStore interface for persistent transaction limit tracking (Daily, Weekly, Monthly)
type LimitStore interface {
	GetPeriodUsage(delegationID, periodKey string) (float64, error)
	IncrementPeriodUsage(delegationID, periodKey string, amount float64) error
	ResetPeriodUsage(delegationID, periodKey string) error
	// Deprecated: Use GetPeriodUsage
	GetDailyUsage(delegationID, date string) (float64, error)
	// Deprecated: Use IncrementPeriodUsage
	IncrementDailyUsage(delegationID, date string, amount float64) error
	// Deprecated: Use ResetPeriodUsage
	ResetDailyUsage(delegationID, date string) error
	ExportDailyLimits(ctx context.Context) (map[string]map[string]float64, error)
}

// Deprecated: Use LimitStore
type DailyLimitStore = LimitStore

// ConditionalEngine interface for evaluating complex conditional expressions
type ConditionalEngine interface {
	EvaluateCondition(condition string, context map[string]interface{}) (bool, error)
	ValidateConditionSyntax(condition string) error
}

// ValidationMetricsRecorder interface for recording validation metrics
type ValidationMetricsRecorder interface {
	RecordValidationSuccess(validatorType, scope string)
	RecordValidationFailure(validatorType, scope, reason string)
	RecordWarning(category, severity string)
	RecordDailyLimitCheck(delegationID string, used, limit float64, exceeded bool)
}

// NewEnhancedPoAValidator creates a new enhanced validator with optional components
func NewEnhancedPoAValidator(opts ...EnhancedValidatorOption) *EnhancedPoAValidator {
	validator := &EnhancedPoAValidator{
		BasicPoAValidator: BasicPoAValidator{},
		warningCollector:  NewWarningCollector(),
		validatorChain:    make([]PoAValidator, 0),
	}

	for _, opt := range opts {
		opt(validator)
	}

	return validator
}

// EnhancedValidatorOption configures the enhanced validator
type EnhancedValidatorOption func(*EnhancedPoAValidator)

// WithWarningCollector sets a custom warning collector
func WithWarningCollector(collector WarningCollector) EnhancedValidatorOption {
	return func(v *EnhancedPoAValidator) {
		v.warningCollector = collector
	}
}

// WithLimitStore sets a persistent limit store
func WithLimitStore(store LimitStore) EnhancedValidatorOption {
	return func(v *EnhancedPoAValidator) {
		v.dailyLimitStore = store
	}
}

// WithDailyLimitStore sets a persistent daily limit store
// Deprecated: Use WithLimitStore
func WithDailyLimitStore(store DailyLimitStore) EnhancedValidatorOption {
	return WithLimitStore(store)
}

// WithConditionalEngine sets a conditional evaluation engine
func WithConditionalEngine(engine ConditionalEngine) EnhancedValidatorOption {
	return func(v *EnhancedPoAValidator) {
		v.conditionalEngine = engine
	}
}

// WithCurrencyConverter sets a currency converter
func WithCurrencyConverter(converter CurrencyConverter) EnhancedValidatorOption {
	return func(v *EnhancedPoAValidator) {
		v.currencyConverter = converter
	}
}

// WithValidatorChain adds custom validators to the validation chain
func WithValidatorChain(validators ...PoAValidator) EnhancedValidatorOption {
	return func(v *EnhancedPoAValidator) {
		v.validatorChain = append(v.validatorChain, validators...)
	}
}

// WithMetricsRecorder sets a metrics recorder for validation events
func WithMetricsRecorder(recorder ValidationMetricsRecorder) EnhancedValidatorOption {
	return func(v *EnhancedPoAValidator) {
		v.metricsRecorder = recorder
	}
}

// Validate performs comprehensive PoA validation with warning collection
func (v *EnhancedPoAValidator) Validate(p *PowerOfAttorney) error {
	return v.ValidateWithContext(context.Background(), p)
}

// ValidateWithContext performs validation with context and returns detailed results
func (v *EnhancedPoAValidator) ValidateWithContext(ctx context.Context, p *PowerOfAttorney) error {
	if v.warningCollector != nil {
		v.warningCollector.ClearWarnings()
	}

	// Start with basic validation
	if err := v.BasicPoAValidator.Validate(p); err != nil {
		if v.metricsRecorder != nil {
			v.metricsRecorder.RecordValidationFailure("basic", getPoAScope(p), err.Error())
		}
		return err
	}

	// Enhanced validations with warning collection
	if err := v.validateEnhancedSemantics(ctx, p); err != nil {
		if v.metricsRecorder != nil {
			v.metricsRecorder.RecordValidationFailure("enhanced", getPoAScope(p), err.Error())
		}
		return err
	}

	// Run validator chain
	for i, validator := range v.validatorChain {
		if err := validator.Validate(p); err != nil {
			if v.metricsRecorder != nil {
				v.metricsRecorder.RecordValidationFailure(fmt.Sprintf("chain_%d", i), getPoAScope(p), err.Error())
			}
			return err
		}
	}

	// Financial limit validation (Daily, Weekly, Monthly)
	if err := v.validateFinancialLimits(ctx, p); err != nil {
		if v.metricsRecorder != nil {
			v.metricsRecorder.RecordValidationFailure("financial_limits", getPoAScope(p), err.Error())
		}
		return err
	}

	// Conditional expressions validation
	if err := v.validateConditionalExpressions(ctx, p); err != nil {
		if v.metricsRecorder != nil {
			v.metricsRecorder.RecordValidationFailure("conditional", getPoAScope(p), err.Error())
		}
		return err
	}

	if v.metricsRecorder != nil {
		v.metricsRecorder.RecordValidationSuccess("enhanced", getPoAScope(p))
	}

	return nil
}

// ValidateWithResult performs validation and returns a ValidationResult with warnings
func (v *EnhancedPoAValidator) ValidateWithResult(ctx context.Context, p *PowerOfAttorney) ValidationResult {
	if v.warningCollector != nil {
		v.warningCollector.ClearWarnings()
	}

	err := v.ValidateWithContext(ctx, p)

	var warnings []ValidationWarning
	if v.warningCollector != nil {
		warnings = v.warningCollector.GetWarnings()
	}

	return ValidationResult{
		Valid:    err == nil,
		Error:    err,
		Warnings: warnings,
		Metadata: map[string]interface{}{
			"validator_type": "enhanced",
			"timestamp":      time.Now(),
		},
	}
}

// validateEnhancedSemantics performs advanced semantic validation with warnings
func (v *EnhancedPoAValidator) validateEnhancedSemantics(ctx context.Context, p *PowerOfAttorney) error {
	// AAP002-specific semantic validation
	if err := v.validateAAP002Semantics(ctx, p); err != nil {
		return err
	}

	// Check for potentially suspicious patterns
	if len(p.Scope) > 10 {
		v.addWarning("excessive_scope", "Large number of scopes may indicate overprivileged delegation", "scope", len(p.Scope), "warning")
	}

	// Duration analysis
	duration := p.ValidUntil.Sub(p.ValidFrom)
	if duration > 365*24*time.Hour {
		v.addWarning("long_duration", "Delegation duration exceeds 1 year", "duration", duration.String(), "warning")
	}

	// Financial scope analysis
	for _, scope := range p.Scope {
		if strings.HasPrefix(scope, "transaction:") {
			if err := v.validateFinancialScope(ctx, p, scope); err != nil {
				return err
			}
		}
	}

	// Administrative scope warnings
	for _, scope := range p.Scope {
		if strings.Contains(scope, "admin") || strings.Contains(scope, "root") {
			v.addWarning("administrative_scope", "Administrative scope detected - requires elevated approval", "scope", scope, "error")
		}
	}

	// Cross-field consistency checks
	if err := v.validateCrossFieldConsistency(ctx, p); err != nil {
		return err
	}

	return nil
}

// validateAAP002Semantics validates PoA according to AAP002 semantic rules
func (v *EnhancedPoAValidator) validateAAP002Semantics(ctx context.Context, p *PowerOfAttorney) error {
	// 1. Scope syntax validation - ensure proper format
	for i, scope := range p.Scope {
		if err := v.validateScopeSyntax(scope); err != nil {
			return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("scope[%d] syntax invalid: %v", i, err))
		}
	}

	// 2. Scope semantic validation - ensure logical consistency
	if err := v.validateScopeSemantics(p.Scope); err != nil {
		return err
	}

	// 3. Action taxonomy validation - verify action classes are valid
	if err := v.validateActionTaxonomy(p); err != nil {
		return err
	}

	// 4. Temporal constraint semantics
	if err := v.validateTemporalConstraints(p); err != nil {
		return err
	}

	// 5. Authority relationship validation
	if err := v.validateAuthorityRelationship(p); err != nil {
		return err
	}

	// 6. Delegation depth semantics (if parent chain available)
	if err := v.validateDelegationDepthSemantics(ctx, p); err != nil {
		return err
	}

	// 7. Restriction semantics validation
	if err := v.validateRestrictionSemantics(p); err != nil {
		return err
	}

	return nil
}

// validateScopeSyntax validates individual scope string format
func (v *EnhancedPoAValidator) validateScopeSyntax(scope string) error {
	if scope == "" {
		return fmt.Errorf("empty scope not allowed")
	}

	// Wildcard validation
	if scope == "*" {
		return nil // Wildcard is syntactically valid
	}

	// Scope must be printable ASCII or valid UTF-8
	for _, r := range scope {
		if r < 32 || r == 127 {
			return fmt.Errorf("control characters not allowed in scope")
		}
	}

	// If scope contains colon, validate namespace:action format
	if strings.Contains(scope, ":") {
		parts := strings.SplitN(scope, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid namespace:action format")
		}

		// Namespace must be alphanumeric with underscores/hyphens
		namespace := parts[0]
		for _, r := range namespace {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '_' || r == '-') {
				return fmt.Errorf("invalid namespace characters: %s", namespace)
			}
		}
	}

	return nil
}

// validateScopeSemantics validates logical consistency of scope array
func (v *EnhancedPoAValidator) validateScopeSemantics(scopes []string) error {
	if len(scopes) == 0 {
		return aap.New(aap.ErrInvalidRequest, "scope array cannot be empty")
	}

	// Check for duplicates
	seen := make(map[string]bool, len(scopes))
	for i, scope := range scopes {
		if seen[scope] {
			v.addWarning("duplicate_scope", "Duplicate scope detected", "scope", scope, "warning")
		}
		seen[scope] = true

		// Check for wildcard with other scopes
		if scope == "*" && len(scopes) > 1 {
			return aap.New(aap.ErrInvalidRequest, "wildcard scope must be used alone")
		}

		// Check for scope subsumption (e.g., "read:*" and "read:documents")
		if strings.HasSuffix(scope, ":*") {
			prefix := scope[:len(scope)-1] // Remove "*", keep ":"
			for j, other := range scopes {
				if i != j && strings.HasPrefix(other, prefix) && other != scope {
					v.addWarning("scope_subsumption", fmt.Sprintf("Scope %s subsumes %s", scope, other), "scope", other, "info")
				}
			}
		}
	}

	return nil
}

// validateActionTaxonomy validates action classes against AAP002 taxonomy
func (v *EnhancedPoAValidator) validateActionTaxonomy(p *PowerOfAttorney) error {
	// Define valid action classes per AAP002
	validActionClasses := map[string]bool{
		"read":        true,
		"write":       true,
		"execute":     true,
		"delete":      true,
		"admin":       true,
		"transaction": true,
		"transfer":    true,
		"delegate":    true,
		"revoke":      true,
		"audit":       true,
		"regulatory":  true,
		"joint":       true,
	}

	// Validate ActionClass field if present
	if p.ActionClass != "" {
		if !validActionClasses[strings.ToLower(p.ActionClass)] {
			v.addWarning("unknown_action_class", "Action class not in AAP002 taxonomy", "action_class", p.ActionClass, "warning")
		}
	}

	// Validate scope prefixes
	for _, scope := range p.Scope {
		if strings.Contains(scope, ":") {
			parts := strings.SplitN(scope, ":", 2)
			action := strings.ToLower(parts[0])
			if action != "*" && !validActionClasses[action] {
				v.addWarning("unknown_action_prefix", "Scope action prefix not in AAP002 taxonomy", "scope", scope, "info")
			}
		}
	}

	return nil
}

// validateTemporalConstraints validates temporal semantics
func (v *EnhancedPoAValidator) validateTemporalConstraints(p *PowerOfAttorney) error {
	now := time.Now()

	// Warn about past valid_from (likely error)
	if p.ValidFrom.Before(now.Add(-24 * time.Hour)) {
		v.addWarning("past_valid_from", "Valid_from is more than 24h in the past", "valid_from", p.ValidFrom, "warning")
	}

	// Warn about very short duration (< 1 hour)
	duration := p.ValidUntil.Sub(p.ValidFrom)
	if duration < time.Hour {
		v.addWarning("very_short_duration", "Delegation duration less than 1 hour may be unintentional", "duration", duration.String(), "info")
	}

	// Validate business hour restrictions if present
	if validHours, exists := p.Restrictions["valid_hours"]; exists {
		parts := strings.Split(validHours, "-")
		if len(parts) == 2 {
			start, _ := strconv.Atoi(parts[0])
			end, _ := strconv.Atoi(parts[1])
			if start >= end {
				v.addWarning("overnight_hours", "valid_hours spans midnight - ensure this is intentional", "valid_hours", validHours, "info")
			}
		}
	}

	return nil
}

// validateAuthorityRelationship validates grantor-grantee relationship semantics
func (v *EnhancedPoAValidator) validateAuthorityRelationship(p *PowerOfAttorney) error {
	// Grantor and grantee must be different (unless wildcard delegation)
	if p.Grantor == p.Grantee {
		isWildcard := len(p.Scope) == 1 && p.Scope[0] == "*"
		if !isWildcard {
			return aap.New(aap.ErrInvalidRequest, "self-delegation only allowed for wildcard scope")
		}
	}

	// Check for service account patterns (prefix-based heuristic)
	servicePatterns := []string{"service-", "bot-", "system-", "app-"}
	isServiceGrantor := false
	isServiceGrantee := false

	for _, pattern := range servicePatterns {
		if strings.HasPrefix(strings.ToLower(p.Grantor), pattern) {
			isServiceGrantor = true
		}
		if strings.HasPrefix(strings.ToLower(p.Grantee), pattern) {
			isServiceGrantee = true
		}
	}

	// Warn about service-to-service delegation (may require elevated approval)
	if isServiceGrantor && isServiceGrantee {
		v.addWarning("service_to_service", "Service-to-service delegation detected", "grantor_grantee", fmt.Sprintf("%s -> %s", p.Grantor, p.Grantee), "warning")
	}

	return nil
}

// validateDelegationDepthSemantics validates delegation chain depth
func (v *EnhancedPoAValidator) validateDelegationDepthSemantics(ctx context.Context, p *PowerOfAttorney) error {
	// If delegation has a parent, validate depth constraints
	if p.ParentPOAID != "" {
		// Maximum depth check (environment-based)
		maxDepthStr := os.Getenv("GAUTH_MAX_DELEGATION_DEPTH")
		maxDepth := 5
		if maxDepthStr != "" {
			if iv, err := strconv.Atoi(maxDepthStr); err == nil && iv > 0 {
				maxDepth = iv
			}
		}

		if p.Depth > maxDepth {
			v.addWarning("depth_limit_exceeded", "Delegation depth exceeds system limit", "depth", p.Depth, "error")
		}

		// Check for self-contained restriction (Requirement 12)
		if limitStr, ok := p.Restrictions["max_delegation_depth"]; ok {
			if limit, err := strconv.Atoi(limitStr); err == nil {
				if p.Depth > limit {
					// This would be weird (self-contradiction), but technically possible if manually constructed
					v.addWarning("depth_restriction_mismatch", "Current depth exceeds own max_delegation_depth restriction", "depth", p.Depth, "error")
				}
			}
		}
	}

	return nil
}

// validateRestrictionSemantics validates restriction key-value semantics
func (v *EnhancedPoAValidator) validateRestrictionSemantics(p *PowerOfAttorney) error {
	// Define valid restriction keys per AAP002
	knownRestrictions := map[string]bool{
		"currency":           true,
		"max_amount":         true,
		"max_daily_amount":   true,
		"max_weekly_amount":  true, // New
		"max_monthly_amount": true, // New
		"min_amount":         true,
		"jurisdiction":       true,
		"signatures":         true,
		"valid_hours":        true,
		"valid_weekdays":     true,
		"time_condition":     true,
		"ip_whitelist":       true,
		"geo_restriction":    true,
		"purpose":            true,
		"approval_required":  true,
	}

	// Warn about unknown restrictions
	for key, value := range p.Restrictions {
		// Validating dynamic conditions (Requirement 51)
		if strings.HasPrefix(key, "condition_") {
			if v.conditionalEngine != nil {
				if err := v.conditionalEngine.ValidateConditionSyntax(value); err != nil {
					v.addWarning("invalid_condition_syntax", fmt.Sprintf("Invalid condition syntax for %s: %v", key, err), "restriction", key, "error")
				}
			}
			continue
		}

		if !knownRestrictions[key] {
			v.addWarning("unknown_restriction", "Restriction key not in AAP002 standard", "restriction", key, "info")
		}
	}

	// Validate restriction value semantics
	if purpose, exists := p.Restrictions["purpose"]; exists {
		if len(purpose) > 500 {
			return aap.New(aap.ErrInvalidRequest, "purpose restriction exceeds 500 character limit")
		}
	}

	if ipWhitelist, exists := p.Restrictions["ip_whitelist"]; exists {
		// Basic IP format validation
		ips := strings.Split(ipWhitelist, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" && !v.isValidIPOrCIDR(ip) {
				v.addWarning("invalid_ip_format", "IP whitelist contains potentially invalid entry", "ip", ip, "warning")
			}
		}
	}

	return nil
}

// isValidIPOrCIDR performs basic IP/CIDR format validation
func (v *EnhancedPoAValidator) isValidIPOrCIDR(s string) bool {
	// Simple heuristic: contains only digits, dots, slashes, colons
	for _, r := range s {
		if !((r >= '0' && r <= '9') || r == '.' || r == ':' || r == '/') {
			return false
		}
	}
	return true
}

// validateFinancialScope validates transaction-related scopes with enhanced checks
func (v *EnhancedPoAValidator) validateFinancialScope(ctx context.Context, p *PowerOfAttorney, scope string) error {
	// Require comprehensive financial restrictions
	requiredRestrictions := []string{"currency", "max_amount"}
	for _, req := range requiredRestrictions {
		if _, exists := p.Restrictions[req]; !exists {
			return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("financial scope %s requires %s restriction", scope, req))
		}
	}

	// Enhanced currency validation
	if currency, exists := p.Restrictions["currency"]; exists {
		// Use converter if available
		if v.currencyConverter != nil {
			if !v.currencyConverter.IsValidCurrency(currency) {
				return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("unsupported currency code: %s", currency))
			}
		} else if !v.isValidCurrencyCode(currency) {
			// Fallback to basic validation
			return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("invalid currency code: %s", currency))
		}
	}

	// Amount validation with warnings
	if maxAmountStr, exists := p.Restrictions["max_amount"]; exists {
		if maxAmount, err := strconv.ParseFloat(maxAmountStr, 64); err == nil {
			if maxAmount > 1000000 { // 1M limit
				v.addWarning("high_amount_limit", "Very high amount limit detected", "max_amount", maxAmount, "warning")
			}
		}
	}

	// Geographic restrictions for international transactions
	if strings.Contains(scope, "international") {
		if _, exists := p.Restrictions["jurisdiction"]; !exists {
			return aap.New(aap.ErrInvalidRequest, "international transactions require jurisdiction restriction")
		}
	}

	return nil
}

// validateFinancialLimits checks and enforces financial limits (Daily, Weekly, Monthly)
func (v *EnhancedPoAValidator) validateFinancialLimits(ctx context.Context, p *PowerOfAttorney) error {
	if v.dailyLimitStore == nil {
		return nil // No limit tracking configured
	}

	// Extract requested amount from context
	var requestedAmount float64
	if amt := ctx.Value(ctxKeyRequestedAmount); amt != nil {
		if f, ok := amt.(float64); ok {
			requestedAmount = f
		} else if s, ok := amt.(string); ok {
			requestedAmount, _ = strconv.ParseFloat(s, 64)
		}
	} else if amtLegacy := ctx.Value(LegacyCtxRequestedAmount); amtLegacy != nil {
		if f, ok := amtLegacy.(float64); ok {
			requestedAmount = f
		} else if s, ok := amtLegacy.(string); ok {
			requestedAmount, _ = strconv.ParseFloat(s, 64)
		}
	}

	// Extract currency from context, fallback to PoA's default_currency, then USD
	currency := "USD"
	if c := ctx.Value("currency"); c != nil {
		if s, ok := c.(string); ok {
			currency = strings.ToUpper(s)
		}
	} else if dc, ok := p.Restrictions["default_currency"]; ok {
		currency = strings.ToUpper(dc)
	}

	now := time.Now()

	// 1. Daily Limit
	if dailyLimitStr, ok := p.Restrictions["max_daily_amount"]; ok {
		limit, err := strconv.ParseFloat(dailyLimitStr, 64)
		if err != nil {
			return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("invalid max_daily_amount: %v", err))
		}

		// Normalize requested amount to limit currency (if different)
		limitCurrency := "USD"
		if lc, ok := p.Restrictions["max_daily_currency"]; ok {
			limitCurrency = strings.ToUpper(lc)
		} else if dc, ok := p.Restrictions["default_currency"]; ok {
			limitCurrency = strings.ToUpper(dc)
		}

		normalizedAmount := requestedAmount
		if currency != limitCurrency && v.currencyConverter != nil {
			var err error
			normalizedAmount, err = v.currencyConverter.Convert(requestedAmount, currency, limitCurrency)
			if err != nil {
				v.addWarning("currency_conversion_failed", "Daily limit check used non-normalized amount", "error", err.Error(), "error")
			}
		}

		today := now.Format("2006-01-02")
		if err := v.checkLimit(p.ID, today, normalizedAmount, limit, "daily"); err != nil {
			return err
		}
	}

	// 2. Weekly Limit
	if weeklyLimitStr, ok := p.Restrictions["max_weekly_amount"]; ok {
		limit, err := strconv.ParseFloat(weeklyLimitStr, 64)
		if err != nil {
			return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("invalid max_weekly_amount: %v", err))
		}

		limitCurrency := "USD"
		if lc, ok := p.Restrictions["max_weekly_currency"]; ok {
			limitCurrency = strings.ToUpper(lc)
		} else if dc, ok := p.Restrictions["default_currency"]; ok {
			limitCurrency = strings.ToUpper(dc)
		}

		normalizedAmount := requestedAmount
		if currency != limitCurrency && v.currencyConverter != nil {
			var err error
			normalizedAmount, err = v.currencyConverter.Convert(requestedAmount, currency, limitCurrency)
			if err != nil {
				v.addWarning("currency_conversion_failed", "Weekly limit check used non-normalized amount", "error", err.Error(), "error")
			}
		}

		year, week := now.ISOWeek()
		weekKey := fmt.Sprintf("%d-W%02d", year, week)
		if err := v.checkLimit(p.ID, weekKey, normalizedAmount, limit, "weekly"); err != nil {
			return err
		}
	}

	// 3. Monthly Limit
	if monthlyLimitStr, ok := p.Restrictions["max_monthly_amount"]; ok {
		limit, err := strconv.ParseFloat(monthlyLimitStr, 64)
		if err != nil {
			return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("invalid max_monthly_amount: %v", err))
		}

		limitCurrency := "USD"
		if lc, ok := p.Restrictions["max_monthly_currency"]; ok {
			limitCurrency = strings.ToUpper(lc)
		} else if dc, ok := p.Restrictions["default_currency"]; ok {
			limitCurrency = strings.ToUpper(dc)
		}

		normalizedAmount := requestedAmount
		if currency != limitCurrency && v.currencyConverter != nil {
			var err error
			normalizedAmount, err = v.currencyConverter.Convert(requestedAmount, currency, limitCurrency)
			if err != nil {
				v.addWarning("currency_conversion_failed", "Monthly limit check used non-normalized amount", "error", err.Error(), "error")
			}
		}

		monthKey := now.Format("2006-01")
		if err := v.checkLimit(p.ID, monthKey, normalizedAmount, limit, "monthly"); err != nil {
			return err
		}
	}

	return nil
}

// checkLimit performs generic limit check
func (v *EnhancedPoAValidator) checkLimit(delegationID, periodKey string, amount, limit float64, periodType string) error {
	currentUsage, err := v.dailyLimitStore.GetPeriodUsage(delegationID, periodKey)
	if err != nil {
		v.addWarning(fmt.Sprintf("%s_limit_check_failed", periodType), fmt.Sprintf("Could not verify %s usage", periodType), "limit", err.Error(), "error")
		return nil // Don't fail validation hard on store error, but warn
	}

	if v.metricsRecorder != nil {
		exceeded := (currentUsage + amount) > limit
		// We reuse RecordDailyLimitCheck for all periods for now, or assume this metric is generic enough
		// Ideally we would add RecordPeriodLimitCheck to interface but avoiding breaking interface change if not necessary
		v.metricsRecorder.RecordDailyLimitCheck(delegationID, currentUsage+amount, limit, exceeded)
	}

	if (currentUsage + amount) > limit {
		return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("%s limit exceeded: current=%f requested=%f limit=%f", periodType, currentUsage, amount, limit))
	}

	// Warning for approaching limit
	if (currentUsage + amount) > limit*0.8 {
		v.addWarning(fmt.Sprintf("approaching_%s_limit", periodType), fmt.Sprintf("%s usage approaching limit", strings.Title(periodType)), "usage_percentage", ((currentUsage+amount)/limit)*100, "warning")
	}
	return nil
}

// validateConditionalExpressions validates complex conditional restrictions
func (v *EnhancedPoAValidator) validateConditionalExpressions(ctx context.Context, p *PowerOfAttorney) error {
	if v.conditionalEngine == nil {
		return nil // No conditional engine configured
	}

	for key, value := range p.Restrictions {
		if strings.HasPrefix(key, "condition_") {
			if err := v.conditionalEngine.ValidateConditionSyntax(value); err != nil {
				return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("invalid condition %s: %v", key, err))
			}
		}
	}

	// Validate time-based expressions
	if timeCondition, exists := p.Restrictions["time_condition"]; exists {
		if err := v.validateTimeCondition(timeCondition); err != nil {
			return err
		}
	}

	return nil
}

// validateCrossFieldConsistency checks for logical inconsistencies across fields
func (v *EnhancedPoAValidator) validateCrossFieldConsistency(ctx context.Context, p *PowerOfAttorney) error {
	// Check for conflicting restrictions
	if maxAmount, exists1 := p.Restrictions["max_amount"]; exists1 {
		if minAmount, exists2 := p.Restrictions["min_amount"]; exists2 {
			max, err1 := strconv.ParseFloat(maxAmount, 64)
			min, err2 := strconv.ParseFloat(minAmount, 64)
			if err1 == nil && err2 == nil && min >= max {
				return aap.New(aap.ErrInvalidRequest, "min_amount must be less than max_amount")
			}
		}
	}

	// Validate scope-restriction alignment
	hasTransactionScope := false
	for _, scope := range p.Scope {
		if strings.HasPrefix(scope, "transaction:") {
			hasTransactionScope = true
			break
		}
	}

	if !hasTransactionScope {
		// Warn about financial restrictions without transaction scope
		financialRestrictions := []string{"currency", "max_amount", "max_daily_amount"}
		for _, restriction := range financialRestrictions {
			if _, exists := p.Restrictions[restriction]; exists {
				v.addWarning("unused_financial_restriction", "Financial restriction without transaction scope", "restriction", restriction, "info")
			}
		}
	}

	return nil
}

// validateTimeCondition validates time-based conditional expressions
func (v *EnhancedPoAValidator) validateTimeCondition(condition string) error {
	// Simple DSL for time conditions: "weekdays(1,2,3,4,5) AND hours(9-17)"
	// This is a basic implementation - a full DSL would be more sophisticated

	if strings.Contains(condition, "weekdays(") {
		start := strings.Index(condition, "weekdays(")
		end := strings.Index(condition[start:], ")")
		if end == -1 {
			return aap.New(aap.ErrInvalidRequest, "malformed weekdays condition")
		}
		weekdayStr := condition[start+9 : start+end]
		weekdays := strings.Split(weekdayStr, ",")
		for _, wd := range weekdays {
			wd = strings.TrimSpace(wd)
			if day, err := strconv.Atoi(wd); err != nil || day < 0 || day > 6 {
				return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("invalid weekday: %s", wd))
			}
		}
	}

	if strings.Contains(condition, "hours(") {
		start := strings.Index(condition, "hours(")
		end := strings.Index(condition[start:], ")")
		if end == -1 {
			return aap.New(aap.ErrInvalidRequest, "malformed hours condition")
		}
		hoursStr := condition[start+6 : start+end]
		if !strings.Contains(hoursStr, "-") {
			return aap.New(aap.ErrInvalidRequest, "hours condition must be in HH-HH format")
		}
		parts := strings.Split(hoursStr, "-")
		if len(parts) != 2 {
			return aap.New(aap.ErrInvalidRequest, "hours condition must have start and end")
		}
		for _, part := range parts {
			if hour, err := strconv.Atoi(strings.TrimSpace(part)); err != nil || hour < 0 || hour > 23 {
				return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("invalid hour: %s", part))
			}
		}
	}

	return nil
}

// isValidCurrencyCode validates currency codes against ISO 4217 standard (simplified)
func (v *EnhancedPoAValidator) isValidCurrencyCode(code string) bool {
	// Simplified validation - in production, use a proper ISO 4217 library
	validCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true, "CAD": true,
		"AUD": true, "CHF": true, "CNY": true, "INR": true, "BRL": true,
		"MXN": true, "SGD": true, "HKD": true, "NZD": true, "SEK": true,
		"NOK": true, "DKK": true, "PLN": true, "CZK": true, "HUF": true,
	}
	return validCurrencies[strings.ToUpper(code)]
}

// addWarning adds a warning to the collector if available
func (v *EnhancedPoAValidator) addWarning(code, message, field string, value interface{}, severity string) {
	if v.warningCollector != nil {
		v.warningCollector.AddWarning(code, message, field, value, severity)
	}
	if v.metricsRecorder != nil {
		v.metricsRecorder.RecordWarning(code, severity)
	}
}

// GetWarnings returns collected warnings from the last validation
func (v *EnhancedPoAValidator) GetWarnings() []ValidationWarning {
	if v.warningCollector == nil {
		return nil
	}
	return v.warningCollector.GetWarnings()
}

// getPoAScope extracts scope information for metrics
func getPoAScope(p *PowerOfAttorney) string {
	if p == nil || len(p.Scope) == 0 {
		return "empty"
	}
	if len(p.Scope) == 1 {
		return p.Scope[0]
	}
	return fmt.Sprintf("multiple_%d", len(p.Scope))
}

// EvaluatePoAConditions evaluates all conditional restrictions in a PoA against the provided context.
// Returns nil if all conditions pass, or an error if any condition fails.
func (v *EnhancedPoAValidator) EvaluatePoAConditions(ctx context.Context, p *PowerOfAttorney, contextData map[string]interface{}) error {
	if v.conditionalEngine == nil {
		// If no engine is configured but conditions exist, we must fail-closed or warn?
		// RFC says unprocessable critical restrictions must fail. conditions are critical.
		for key := range p.Restrictions {
			if strings.HasPrefix(key, "condition_") {
				return aap.New(aap.ErrConfiguration, "conditional restrictions present but no engine configured")
			}
		}
		return nil
	}

	for key, condition := range p.Restrictions {
		if strings.HasPrefix(key, "condition_") {
			result, err := v.conditionalEngine.EvaluateCondition(condition, contextData)
			if err != nil {
				// Evaluation error (e.g. missing field) -> Fail closed
				if v.metricsRecorder != nil {
					v.metricsRecorder.RecordValidationFailure("conditional", key, err.Error())
				}
				return aap.New(aap.ErrRestrictionExceeded, fmt.Sprintf("condition evaluation error for %s: %v", key, err))
			}
			if !result {
				// Condition unmet
				if v.metricsRecorder != nil {
					v.metricsRecorder.RecordValidationFailure("conditional", key, "false_result")
				}
				return aap.New(aap.ErrRestrictionExceeded, fmt.Sprintf("condition unmet: %s", key))
			}
			if v.metricsRecorder != nil {
				v.metricsRecorder.RecordValidationSuccess("conditional", key)
			}
		}
	}
	return nil
}

// RecordPoAConsumption increments persistent usage counters for a successful transaction.
func (v *EnhancedPoAValidator) RecordPoAConsumption(ctx context.Context, p *PowerOfAttorney, amount float64, currency string) error {
	if v.dailyLimitStore == nil {
		return nil
	}

	currency = strings.ToUpper(currency)
	now := time.Now()

	// Helper to normalize and increment
	updatePeriod := func(limitKey, currencyKey, periodKey string) {
		if _, ok := p.Restrictions[limitKey]; !ok {
			return
		}

		limitCurrency := "USD"
		if lc, ok := p.Restrictions[currencyKey]; ok {
			limitCurrency = strings.ToUpper(lc)
		} else if dc, ok := p.Restrictions["default_currency"]; ok {
			limitCurrency = strings.ToUpper(dc)
		}

		normalized := amount
		if currency != limitCurrency && v.currencyConverter != nil {
			if n, err := v.currencyConverter.Convert(amount, currency, limitCurrency); err == nil {
				normalized = n
			}
		}

		_ = v.dailyLimitStore.IncrementPeriodUsage(p.ID, periodKey, normalized)
	}

	// Update Daily
	updatePeriod("max_daily_amount", "max_daily_currency", now.Format("2006-01-02"))

	// Update Weekly
	year, week := now.ISOWeek()
	updatePeriod("max_weekly_amount", "max_weekly_currency", fmt.Sprintf("%d-W%02d", year, week))

	// Update Monthly
	updatePeriod("max_monthly_amount", "max_monthly_currency", now.Format("2006-01"))

	return nil
}
