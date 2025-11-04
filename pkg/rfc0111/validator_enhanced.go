package rfc0111

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc"
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
	validatorChain    []PoAValidator
	metricsRecorder   ValidationMetricsRecorder
}

// DailyLimitStore interface for persistent daily transaction limit tracking
type DailyLimitStore interface {
	GetDailyUsage(delegationID, date string) (float64, error)
	IncrementDailyUsage(delegationID, date string, amount float64) error
	ResetDailyUsage(delegationID, date string) error
	ExportDailyLimits(ctx context.Context) (map[string]map[string]float64, error)
}

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

// WithDailyLimitStore sets a persistent daily limit store
func WithDailyLimitStore(store DailyLimitStore) EnhancedValidatorOption {
	return func(v *EnhancedPoAValidator) {
		v.dailyLimitStore = store
	}
}

// WithConditionalEngine sets a conditional evaluation engine
func WithConditionalEngine(engine ConditionalEngine) EnhancedValidatorOption {
	return func(v *EnhancedPoAValidator) {
		v.conditionalEngine = engine
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

	// Daily limit validation
	if err := v.validateDailyLimits(ctx, p); err != nil {
		if v.metricsRecorder != nil {
			v.metricsRecorder.RecordValidationFailure("daily_limits", getPoAScope(p), err.Error())
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

// validateFinancialScope validates transaction-related scopes with enhanced checks
func (v *EnhancedPoAValidator) validateFinancialScope(ctx context.Context, p *PowerOfAttorney, scope string) error {
	// Require comprehensive financial restrictions
	requiredRestrictions := []string{"currency", "max_amount"}
	for _, req := range requiredRestrictions {
		if _, exists := p.Restrictions[req]; !exists {
			return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("financial scope %s requires %s restriction", scope, req))
		}
	}

	// Enhanced currency validation
	if currency, exists := p.Restrictions["currency"]; exists {
		if !v.isValidCurrencyCode(currency) {
			return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("invalid currency code: %s", currency))
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
			return rfc.New(rfc.ErrInvalidRequest, "international transactions require jurisdiction restriction")
		}
	}

	return nil
}

// validateDailyLimits checks and enforces daily transaction limits
func (v *EnhancedPoAValidator) validateDailyLimits(ctx context.Context, p *PowerOfAttorney) error {
	if v.dailyLimitStore == nil {
		return nil // No daily limit tracking configured
	}

	dailyLimitStr, hasDailyLimit := p.Restrictions["max_daily_amount"]
	if !hasDailyLimit {
		return nil // No daily limit set
	}

	dailyLimit, err := strconv.ParseFloat(dailyLimitStr, 64)
	if err != nil {
		return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("invalid max_daily_amount: %v", err))
	}

	today := time.Now().Format("2006-01-02")
	currentUsage, err := v.dailyLimitStore.GetDailyUsage(p.ID, today)
	if err != nil {
		v.addWarning("daily_limit_check_failed", "Could not verify daily usage", "daily_limit", err.Error(), "error")
		return nil // Don't fail validation, but log warning
	}

	if v.metricsRecorder != nil {
		exceeded := currentUsage >= dailyLimit
		v.metricsRecorder.RecordDailyLimitCheck(p.ID, currentUsage, dailyLimit, exceeded)
	}

	if currentUsage >= dailyLimit {
		return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("daily limit exceeded: %f/%f", currentUsage, dailyLimit))
	}

	// Warning for approaching limit
	if currentUsage > dailyLimit*0.8 {
		v.addWarning("approaching_daily_limit", "Daily usage approaching limit", "usage_percentage", (currentUsage/dailyLimit)*100, "warning")
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
				return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("invalid condition %s: %v", key, err))
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
				return rfc.New(rfc.ErrInvalidRequest, "min_amount must be less than max_amount")
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
			return rfc.New(rfc.ErrInvalidRequest, "malformed weekdays condition")
		}
		weekdayStr := condition[start+9 : start+end]
		weekdays := strings.Split(weekdayStr, ",")
		for _, wd := range weekdays {
			wd = strings.TrimSpace(wd)
			if day, err := strconv.Atoi(wd); err != nil || day < 0 || day > 6 {
				return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("invalid weekday: %s", wd))
			}
		}
	}

	if strings.Contains(condition, "hours(") {
		start := strings.Index(condition, "hours(")
		end := strings.Index(condition[start:], ")")
		if end == -1 {
			return rfc.New(rfc.ErrInvalidRequest, "malformed hours condition")
		}
		hoursStr := condition[start+6 : start+end]
		if !strings.Contains(hoursStr, "-") {
			return rfc.New(rfc.ErrInvalidRequest, "hours condition must be in HH-HH format")
		}
		parts := strings.Split(hoursStr, "-")
		if len(parts) != 2 {
			return rfc.New(rfc.ErrInvalidRequest, "hours condition must have start and end")
		}
		for _, part := range parts {
			if hour, err := strconv.Atoi(strings.TrimSpace(part)); err != nil || hour < 0 || hour > 23 {
				return rfc.New(rfc.ErrInvalidRequest, fmt.Sprintf("invalid hour: %s", part))
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
