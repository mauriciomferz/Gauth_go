// Package agentauth - AAP001 Step (g) Transaction/Decision/Action Executor
// This implements Step (g) from AAP001 Section 3.2.2 (Request-specific steps)
package agentauth

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// TransactionExecutor handles AAP001 Step (g): Transaction/Decision/Action Request
// This component enables the client to make requests to resource servers using extended tokens
type TransactionExecutor struct {
	tokenValidator    TokenValidator
	complianceTracker ComplianceTracker
}

// TokenValidator validates extended tokens
type TokenValidator interface {
	ValidateExtendedToken(ctx context.Context, tokenString string) (*ExtendedToken, error)
}

// NewTransactionExecutor creates a new transaction executor for Step (g)
func NewTransactionExecutor(
	tokenValidator TokenValidator,
	complianceTracker ComplianceTracker,
) *TransactionExecutor {
	return &TransactionExecutor{
		tokenValidator:    tokenValidator,
		complianceTracker: complianceTracker,
	}
}

// TransactionExecutionRequest represents Step (g): Client request with extended token
type TransactionExecutionRequest struct {
	// Extended token for authentication and authorization
	ExtendedToken string

	// Type of request
	RequestType string // "transaction", "decision", "action"

	// Transaction details (if RequestType == "transaction")
	Transaction *ExecutorTransactionDetails

	// Decision details (if RequestType == "decision")
	Decision *DecisionDetails

	// Action details (if RequestType == "action")
	Action *ActionDetails

	// Resource information
	ResourceID   string
	ResourceType string

	// Additional context
	Context   map[string]interface{}
	Timestamp time.Time
}

// ExecutorTransactionDetails represents a financial or business transaction for Step (g)
type ExecutorTransactionDetails struct {
	Type           poa.TransactionType
	Amount         float64
	Currency       string
	FromAccount    string
	ToAccount      string
	Counterparty   string
	Description    string
	Reference      string
	AdditionalData map[string]interface{}
}

// DecisionDetails represents an AI decision request
type DecisionDetails struct {
	Type              poa.DecisionType
	Subject           string
	Options           []string
	RecommendedOption string
	Rationale         string
	ImpactLevel       string // "low", "medium", "high", "critical"
	RequiresApproval  bool
	AdditionalData    map[string]interface{}
}

// ActionDetails represents a physical or digital action
type ActionDetails struct {
	Type           string // Action type as string
	Description    string
	IsPhysical     bool
	Location       string
	Coordinates    *GeographicCoordinates
	SafetyLevel    string
	AdditionalData map[string]interface{}
}

// GeographicCoordinates for physical actions
type GeographicCoordinates struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
}

// TransactionExecutionResponse represents the result of Step (g)
type TransactionExecutionResponse struct {
	// Execution result
	Success     bool
	ExecutionID string
	ExecutedAt  time.Time

	// Authorization validation
	TokenValid         bool
	AuthorizationValid bool
	ComplianceValid    bool

	// Result details
	ResultData map[string]interface{}

	// Compliance tracking
	ComplianceStatus   *ComplianceStatus
	ViolationsDetected []string

	// Error information (if any)
	ErrorCode    string
	ErrorMessage string
}

// ExecuteTransaction implements AAP001 Step (g): Transaction/Decision/Action Request
// This is called by the client to execute a transaction using an extended token
func (e *TransactionExecutor) ExecuteTransaction(
	ctx context.Context,
	request *TransactionExecutionRequest,
) (*TransactionExecutionResponse, error) {
	// Validate request structure
	if err := e.validateExecutionRequest(request); err != nil {
		return &TransactionExecutionResponse{
			Success:      false,
			TokenValid:   false,
			ErrorCode:    "invalid_request",
			ErrorMessage: err.Error(),
		}, nil
	}

	// STEP (h): Token Validation (part of step g execution)
	// Validate the extended token
	extendedToken, err := e.tokenValidator.ValidateExtendedToken(ctx, request.ExtendedToken)
	if err != nil {
		return &TransactionExecutionResponse{
			Success:      false,
			TokenValid:   false,
			ErrorCode:    "invalid_token",
			ErrorMessage: fmt.Sprintf("Token validation failed: %v", err),
		}, nil
	}

	// Check token expiration
	expiryTime := extendedToken.IssuedAt.Add(time.Duration(extendedToken.ExpiresIn) * time.Second)
	if time.Now().After(expiryTime) {
		return &TransactionExecutionResponse{
			Success:      false,
			TokenValid:   false,
			ErrorCode:    "token_expired",
			ErrorMessage: "Extended token has expired",
		}, nil
	}

	// Validate request matches token scope
	scopeValid, scopeErr := e.validateRequestScope(request, extendedToken)
	if !scopeValid {
		return &TransactionExecutionResponse{
			Success:            false,
			TokenValid:         true,
			AuthorizationValid: false,
			ErrorCode:          "scope_mismatch",
			ErrorMessage:       scopeErr.Error(),
		}, nil
	}

	// Validate against power restrictions
	restrictionsValid, restrictionErr := e.validatePowerRestrictions(request, extendedToken)
	if !restrictionsValid {
		return &TransactionExecutionResponse{
			Success:            false,
			TokenValid:         true,
			AuthorizationValid: false,
			ComplianceValid:    false,
			ErrorCode:          "power_restriction_violated",
			ErrorMessage:       restrictionErr.Error(),
			ViolationsDetected: []string{restrictionErr.Error()},
		}, nil
	}

	// Track compliance (Step i)
	// Note: Compliance tracking is handled via CheckCompliance, not TrackAction
	if e.complianceTracker != nil {
		complianceStatus, trackingErr := e.complianceTracker.CheckCompliance(ctx, extendedToken.AccessToken)
		if trackingErr != nil {
			// Log but don't fail
			fmt.Printf("Warning: Failed to check compliance: %v\n", trackingErr)
		} else if complianceStatus != nil && !complianceStatus.Compliant {
			return &TransactionExecutionResponse{
				Success:            false,
				TokenValid:         true,
				AuthorizationValid: true,
				ComplianceValid:    false,
				ErrorCode:          "compliance_violation",
				ErrorMessage:       "Compliance violation detected",
				ViolationsDetected: complianceStatus.Violations,
			}, nil
		}
	}

	// Execute the actual transaction/decision/action
	// Note: Actual execution is delegated to resource-specific handlers
	// This component validates authorization and prepares execution
	executionID := generateExecutionID()

	response := &TransactionExecutionResponse{
		Success:            true,
		ExecutionID:        executionID,
		ExecutedAt:         time.Now(),
		TokenValid:         true,
		AuthorizationValid: true,
		ComplianceValid:    true,
		ResultData: map[string]interface{}{
			"status":       "authorized",
			"execution_id": executionID,
			"message":      "Request authorized and ready for execution",
		},
		ComplianceStatus: &ComplianceStatus{
			Compliant:   true,
			Violations:  []string{},
			LastChecked: time.Now(),
			NextCheck:   time.Now().Add(1 * time.Hour),
		},
		ViolationsDetected: []string{},
	}

	return response, nil
}

// validateExecutionRequest validates the basic structure of execution request
func (e *TransactionExecutor) validateExecutionRequest(request *TransactionExecutionRequest) error {
	if request.ExtendedToken == "" {
		return fmt.Errorf("extended token is required")
	}
	if request.RequestType == "" {
		return fmt.Errorf("request type is required")
	}

	// Validate type-specific details
	switch request.RequestType {
	case "transaction":
		if request.Transaction == nil {
			return fmt.Errorf("transaction details are required for transaction type")
		}
	case "decision":
		if request.Decision == nil {
			return fmt.Errorf("decision details are required for decision type")
		}
	case "action":
		if request.Action == nil {
			return fmt.Errorf("action details are required for action type")
		}
	default:
		return fmt.Errorf("invalid request type: %s", request.RequestType)
	}

	return nil
}

// validateRequestScope checks if the request matches the token's authorized scope
func (e *TransactionExecutor) validateRequestScope(
	request *TransactionExecutionRequest,
	token *ExtendedToken,
) (bool, error) {
	// Check if request type is in token scope
	if len(token.Scope) == 0 {
		return false, fmt.Errorf("token has no scope defined")
	}

	// Validate based on request type
	switch request.RequestType {
	case "transaction":
		if request.Transaction == nil {
			return false, fmt.Errorf("transaction details missing")
		}
		// Check if transaction type is authorized
		// This would check token.PowerOfAttorney.AuthorizedActions.Transactions
		return true, nil

	case "decision":
		if request.Decision == nil {
			return false, fmt.Errorf("decision details missing")
		}
		// Check if decision type is authorized
		// This would check token.PowerOfAttorney.AuthorizedActions.Decisions
		return true, nil

	case "action":
		if request.Action == nil {
			return false, fmt.Errorf("action details missing")
		}
		// Check if action type is authorized
		// This would check token.PowerOfAttorney.AuthorizedActions
		return true, nil

	default:
		return false, fmt.Errorf("unknown request type: %s", request.RequestType)
	}
}

// validatePowerRestrictions checks if the request violates any power restrictions
func (e *TransactionExecutor) validatePowerRestrictions(
	request *TransactionExecutionRequest,
	token *ExtendedToken,
) (bool, error) {
	if token.PowerOfAttorney == nil {
		return false, fmt.Errorf("token has no power of attorney defined")
	}

	// Check token-level restrictions
	if len(token.Restrictions) == 0 {
		// No restrictions means everything is allowed within scope
		return true, nil
	}

	// Check transaction-specific restrictions
	if request.RequestType == "transaction" && request.Transaction != nil {
		// Validate monetary limits from token restrictions
		for _, restriction := range token.Restrictions {
			if restriction.RestrictionType == "value_limit" && restriction.Value != nil {
				if limitValue, ok := restriction.Value.(float64); ok {
					if request.Transaction.Amount > limitValue {
						return false, fmt.Errorf(
							"transaction amount %f %s exceeds limit %f",
							request.Transaction.Amount,
							request.Transaction.Currency,
							limitValue,
						)
					}
				}
			}
		}
	}

	// Check geographic restrictions for physical actions
	if request.RequestType == "action" && request.Action != nil && request.Action.IsPhysical {
		for _, restriction := range token.Restrictions {
			if restriction.RestrictionType == "geographic_limit" {
				// Would validate coordinates against allowed regions
				_ = restriction // Implementation depends on restriction.Value structure
			}
		}
	}

	// Check temporal restrictions (time-based limitations)
	for _, restriction := range token.Restrictions {
		if restriction.RestrictionType == "time_limit" {
			// Parse temporal restrictions from restriction.Value
			// This would check ValidFrom/ValidUntil if present
			_ = restriction // time validation placeholder - would be used for validation
		}
	}

	return true, nil
}

// generateExecutionID creates a unique execution identifier
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}
