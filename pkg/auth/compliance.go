package auth

import (
	"context"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// ComplianceChecker handles compliance verification
type ComplianceChecker interface {
	// CheckCompliance verifies compliance with rules and restrictions
	CheckCompliance(ctx context.Context, token *token.EnhancedToken, action string) error

	// ValidateRestrictions checks if an action meets restrictions
	ValidateRestrictions(ctx context.Context, restrictions *token.Restrictions, action string) error

	// TrackCompliance records compliance events
	TrackCompliance(ctx context.Context, token *token.EnhancedToken, action string, compliant bool) error
}

// StandardComplianceChecker implements ComplianceChecker
type StandardComplianceChecker struct {
	config  *Config
	tracker ComplianceTracker
}

// ComplianceTracker tracks compliance events
type ComplianceTracker interface {
	// TrackEvent records a compliance event
	TrackEvent(ctx context.Context, event ComplianceEvent) error

	// GetEvents retrieves compliance events for a token
	GetEvents(ctx context.Context, tokenID string) ([]ComplianceEvent, error)

	// GetStatistics gets compliance statistics
	GetStatistics(ctx context.Context, tokenID string) (*ComplianceStats, error)
}

// ComplianceEvent represents a compliance tracking event
type ComplianceEvent struct {
	// Event metadata
	TokenID   string
	Action    string
	Timestamp time.Time
	Compliant bool

	// Violation details (if any)
	ViolationType   string
	ViolationRules  []string
	ViolationDetail string
}

// ComplianceStats contains compliance statistics
type ComplianceStats struct {
	// Overall stats
	TotalActions     int
	CompliantActions int
	ViolationActions int

	// Violation breakdowns
	ViolationsByType map[string]int
	ViolationsByRule map[string]int

	// Time-based stats
	LastViolation time.Time
	LastCompliant time.Time
}

// NewStandardComplianceChecker creates a new compliance checker
func NewStandardComplianceChecker(config *Config, tracker ComplianceTracker) *StandardComplianceChecker {
	return &StandardComplianceChecker{
		config:  config,
		tracker: tracker,
	}
}

// CheckCompliance implements ComplianceChecker
func (c *StandardComplianceChecker) CheckCompliance(ctx context.Context, token *token.EnhancedToken, action string) error {
	// Check basic token validity
	if token.IsExpired() {
		return NewError(ErrAuthorizationExpired, "token has expired", nil)
	}

	// Check AI restrictions
	if token.AI != nil && token.AI.Restrictions != nil {
		if err := c.ValidateRestrictions(ctx, token.AI.Restrictions, action); err != nil {
			return err
		}
	}

	// Check delegation guidelines
	if err := c.checkDelegationGuidelines(ctx, token, action); err != nil {
		return err
	}

	// Check approval rules
	if err := c.checkApprovalRules(ctx, token, action); err != nil {
		return err
	}

	// Track compliance
	if err := c.TrackCompliance(ctx, token, action, true); err != nil {
		return err
	}

	return nil
}

// ValidateRestrictions implements ComplianceChecker
func (c *StandardComplianceChecker) ValidateRestrictions(ctx context.Context, restrictions *token.Restrictions, action string) error {
	// Check value limits
	if restrictions.ValueLimits != nil {
		if err := c.checkValueLimits(ctx, restrictions.ValueLimits, action); err != nil {
			return err
		}
	}

	// Check geographic constraints
	if len(restrictions.GeographicConstraints) > 0 {
		if err := c.checkGeographicConstraints(ctx, restrictions.GeographicConstraints, action); err != nil {
			return err
		}
	}

	// Check time constraints
	if restrictions.TimeConstraints != nil {
		if err := c.checkTimeConstraints(ctx, restrictions.TimeConstraints); err != nil {
			return err
		}
	}

	// Check custom limits
	if len(restrictions.CustomLimits) > 0 {
		// Convert map[string]float64 to map[string]interface{} for compatibility
		converted := make(map[string]interface{}, len(restrictions.CustomLimits))
		for k, v := range restrictions.CustomLimits {
			converted[k] = v
		}
		if err := c.checkCustomLimits(ctx, converted, action); err != nil {
			return err
		}
	}

	return nil
}

// TrackCompliance implements ComplianceChecker
func (c *StandardComplianceChecker) TrackCompliance(ctx context.Context, token *token.EnhancedToken, action string, compliant bool) error {
	event := ComplianceEvent{
		TokenID:   token.ID,
		Action:    action,
		Timestamp: time.Now(),
		Compliant: compliant,
	}

	return c.tracker.TrackEvent(ctx, event)
}

// Helper methods

func (c *StandardComplianceChecker) checkDelegationGuidelines(ctx context.Context, token *token.EnhancedToken, action string) error {
	if token.AI == nil || len(token.AI.DelegationGuidelines) == 0 {
		return nil
	}

	// Implement delegation guidelines checking
	// This would typically involve checking if the action is allowed by the guidelines
	return nil
}

func (c *StandardComplianceChecker) checkApprovalRules(ctx context.Context, token *token.EnhancedToken, action string) error {
	for _, rule := range c.config.ApprovalRules {
		if matches, err := c.matchesRule(ctx, rule, token, action); err != nil {
			return err
		} else if matches {
			if err := c.enforceRule(ctx, rule, token, action); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *StandardComplianceChecker) checkValueLimits(ctx context.Context, limits *token.ValueLimits, action string) error {
	// Implement value limits checking
	// This would typically involve checking transaction amounts against limits
	return nil
}

func (c *StandardComplianceChecker) checkGeographicConstraints(ctx context.Context, constraints []string, action string) error {
	// Implement geographic constraints checking
	// This would typically involve checking location information
	return nil
}

func (c *StandardComplianceChecker) checkTimeConstraints(ctx context.Context, constraints *token.TimeConstraints) error {
	if len(constraints.AllowedTimeWindows) == 0 {
		return nil
	}

	now := time.Now()
	loc, err := time.LoadLocation(constraints.TimeZone)
	if err != nil {
		return NewError(ErrRuleViolation, "invalid timezone", err)
	}

	localNow := now.In(loc)
	weekday := localNow.Weekday()

	for _, window := range constraints.AllowedTimeWindows {
		// Check if current day is allowed
		dayAllowed := false
		for _, day := range window.DaysOfWeek {
			if int(weekday) == day {
				dayAllowed = true
				break
			}
		}
		if !dayAllowed {
			continue
		}

		// Check if current time is within window
		if isTimeInWindow(localNow, window) {
			return nil
		}
	}

	return NewError(ErrRuleViolation, "action not allowed at current time", nil)
}

func (c *StandardComplianceChecker) checkCustomLimits(ctx context.Context, limits map[string]interface{}, action string) error {
	// Implement custom limits checking
	// This would typically involve checking against application-specific rules
	return nil
}

func (c *StandardComplianceChecker) matchesRule(ctx context.Context, rule ApprovalRule, token *token.EnhancedToken, action string) (bool, error) {
	// Implement rule matching logic
	return false, nil
}

func (c *StandardComplianceChecker) enforceRule(ctx context.Context, rule ApprovalRule, token *token.EnhancedToken, action string) error {
	// Implement rule enforcement logic
	return nil
}

// Helper function to check if a time is within a time window
func isTimeInWindow(t time.Time, window token.TimeWindow) bool {
	// Parse window times
	start, err := time.Parse("15:04", window.StartTime)
	if err != nil {
		return false
	}
	end, err := time.Parse("15:04", window.EndTime)
	if err != nil {
		return false
	}

	// Extract time of day
	timeOfDay := time.Date(0, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
	start = time.Date(0, 1, 1, start.Hour(), start.Minute(), 0, 0, time.UTC)
	end = time.Date(0, 1, 1, end.Hour(), end.Minute(), 0, 0, time.UTC)

	// Handle windows that span midnight
	if end.Before(start) {
		return timeOfDay.After(start) || timeOfDay.Before(end)
	}

	return timeOfDay.After(start) && timeOfDay.Before(end)
}
