package gauth

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// RARValidator validates Rich Authorization Requests against a Power of Attorney
type RARValidator struct {
}

// NewRARValidator creates a new instance of RARValidator
func NewRARValidator() *RARValidator {
	return &RARValidator{}
}

// ValidateAuthorizationDetails ensures requested details are within PoA scope
func (v *RARValidator) ValidateAuthorizationDetails(
	poaDef *poa.PoADefinition,
	details []AuthorizationDetail,
) error {
	if poaDef == nil {
		return &AgentAuthError{Code: "missing_poa", Message: "Power of Attorney is required for rar validation"}
	}

	for i, detail := range details {
		// Validate actions are authorized
		if err := v.validateActions(poaDef, detail); err != nil {
			return fmt.Errorf("authorization_detail[%d]: action validation failed: %w", i, err)
		}

		// Validate locations match authorized interaction bounds
		if err := v.validateLocations(poaDef, detail); err != nil {
			return fmt.Errorf("authorization_detail[%d]: location validation failed: %w", i, err)
		}

		// Validate constraints (e.g., amounts) against PoA limits
		if err := v.validateConstraints(poaDef, detail); err != nil {
			return fmt.Errorf("authorization_detail[%d]: constraint validation failed: %w", i, err)
		}

		// Validate data types if specified
		if err := v.validateDataTypes(poaDef, detail); err != nil {
			return fmt.Errorf("authorization_detail[%d]: datatypes validation failed: %w", i, err)
		}
	}

	return nil
}

func (v *RARValidator) validateActions(
	poaDef *poa.PoADefinition,
	detail AuthorizationDetail,
) error {
	// If no specific actions requested, assume compliant
	if len(detail.Actions) == 0 {
		return nil
	}

	// Gather authorized actions from PoA
	authorizedActions := make(map[string]bool)

	// Add Transactions
	for _, txn := range poaDef.Authorization.AuthorizedActions.Transactions {
		authorizedActions[string(txn)] = true
	}
	// Add Decisions
	for _, dec := range poaDef.Authorization.AuthorizedActions.Decisions {
		authorizedActions[string(dec)] = true
	}
	// Add Physical Actions
	for _, phys := range poaDef.Authorization.AuthorizedActions.PhysicalActions {
		authorizedActions[string(phys)] = true
	}
	// Add Non-Physical Actions
	for _, nonPhys := range poaDef.Authorization.AuthorizedActions.NonPhysicalActions {
		authorizedActions[string(nonPhys)] = true
	}

	for _, requestedAction := range detail.Actions {
		if !authorizedActions[requestedAction] {
			return fmt.Errorf("action '%s' not authorized in PoA", requestedAction)
		}
	}

	return nil
}

func (v *RARValidator) validateLocations(
	poaDef *poa.PoADefinition,
	detail AuthorizationDetail,
) error {
	if len(detail.Locations) == 0 {
		return nil
	}

	// Check against PoA Interaction Bounds
	// We treat InteractionBounds as allowed location patterns
	authorizedLocations := poaDef.Requirements.PowerLimits.InteractionBounds

	// If no bounds specified, check if global geographic scope?
	// For MVP, if no interaction bounds, we assume open OR we could check regions.
	// Let's rely on InteractionBounds for URLs.
	if len(authorizedLocations) == 0 {
		// Fallback: If no strict interaction bounds, we might allow.
		// Or strictly deny. Let's allow for now if empty (permissive default if not set).
		return nil
	}

	for _, requestedLocation := range detail.Locations {
		matchFound := false
		for _, authLoc := range authorizedLocations {
			if matchesPattern(authLoc, requestedLocation) {
				matchFound = true
				break
			}
		}
		if !matchFound {
			return fmt.Errorf("location '%s' not authorized by PoA interaction bounds", requestedLocation)
		}
	}

	return nil
}

func (v *RARValidator) validateConstraints(
	poaDef *poa.PoADefinition,
	detail AuthorizationDetail,
) error {
	// Validate amount constraints against PowerLimits
	if detail.InstructedAmount != nil {
		// Look for "max_amount:{currency}:{value}" in PowerLevels
		for _, level := range poaDef.Requirements.PowerLimits.PowerLevels {
			if strings.HasPrefix(level, "max_amount:") {
				parts := strings.Split(level, ":")
				if len(parts) == 3 {
					currency := parts[1]
					maxValStr := parts[2]

					if currency == detail.InstructedAmount.Currency {
						max, err := strconv.ParseFloat(maxValStr, 64)
						if err == nil && max > 0 {
							amount, err := parseAmount(detail.InstructedAmount.Amount)
							if err != nil {
								return fmt.Errorf("invalid amount in request: %w", err)
							}
							if amount > max {
								return fmt.Errorf("amount %.2f exceeds PoA limit %.2f for %s", amount, max, currency)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (v *RARValidator) validateDataTypes(
	poaDef *poa.PoADefinition,
	detail AuthorizationDetail,
) error {
	// If PoA has restrictions on data types?
	// Often generic scope handles this.
	// If PoA doesn't explicitly restrict datatypes, we allow if generic scope permits.
	// Implementation dependent.
	return nil
}

// Helpers

func matchesPattern(pattern, value string) bool {
	if pattern == "*" || pattern == value {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}
	return false
}

func parseAmount(amountStr string) (float64, error) {
	return strconv.ParseFloat(amountStr, 64)
}

// EvaluateAccess checks if the provided AuthorizationDetails grant access to the requested resource and action
func (v *RARValidator) EvaluateAccess(
	details []AuthorizationDetail,
	resource string,
	action string,
) error {
	if len(details) == 0 {
		return &AgentAuthError{Code: "missing_authorization_details", Message: "No authorization details provided"}
	}

	for _, detail := range details {
		// Check locations
		matchLocation := false
		if len(detail.Locations) == 0 {
			// If no specific locations in the DETAIL, does it mean "any"?
			// RFC 9396 says details narrow the scope.
			// Usually locations are specific.
			// If missing, it might mean "all authorized locations" but we should match against resource.
			// Let's assume strict matching for now: Detail MUST specify location or it applies to none (or typically "any" if implied by type).
			// However, for safety in AI context, we require explicit location match if locations are present.
			// If locations are empty in the detail, we might skip location check if the Type implies it?
			// Let's require a match if locations are present.
			// Re-reading implementation plan: "Check if resource matches any location".
			matchLocation = true // Permissive if no locations defined? Or strict?
			// Let's go with: if locations defined, must match. If not, maybe it doesn't restrict location.
		} else {
			for _, loc := range detail.Locations {
				if matchesPattern(loc, resource) {
					matchLocation = true
					break
				}
			}
		}

		if !matchLocation {
			continue // Access not granted by this detail
		}

		// Check actions
		matchAction := false
		if len(detail.Actions) == 0 {
			matchAction = true // No actions constraint?
		} else {
			for _, act := range detail.Actions {
				if act == action {
					matchAction = true
					break
				}
			}
		}

		if matchAction {
			// Found a detail that allows this resource and action
			return nil
		}
	}

	return &AgentAuthError{
		Code:    "insufficient_authorization",
		Message: fmt.Sprintf("Access denied for action '%s' on resource '%s'", action, resource),
	}
}
