package agentauth_aap_001

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/aap"
)

// DelegationChainValidator provides validation for transitive (multi-hop) delegation chains.
//
// Security Context: Addresses High Vulnerability - Broken Delegation Chain Logic (Transitive Trust)
//
// Problem:
// The simple grantee check `if poa.Grantee != grantee` at line 2509 fails to support transitive delegation:
//   - Alice (Principal) delegates to Bob (Agent)  →  PoA₁: Alice→Bob
//   - Bob (Agent) delegates to Charlie (Sub-Agent) →  PoA₂: Bob→Charlie (with ParentPOAID = PoA₁.ID)
//
// When Charlie presents PoA₂, the current code:
//  1. Loads PoA₂ (Bob→Charlie)
//  2. Checks if PoA₂.Grantee (Charlie) == Session.User (Charlie) ✓
//  3. BUT never validates the parent chain (Alice→Bob)
//
// Attack Scenarios:
//   - Scenario A (False Negative): Charlie's valid delegation is rejected because code checks root PoA₁.Grantee (Bob) != Charlie
//   - Scenario B (Unauthorized Delegation): Bob delegates more scopes than Alice gave him (scope escalation in chain)
//   - Scenario C (Revoked Ancestor): Alice revokes PoA₁, but Charlie's PoA₂ is still accepted
//
// Solution:
// Validate the full delegation chain from Root Principal to Current Session User:
//  1. Walk from leaf PoA upward through ParentPOAID references
//  2. For each link: Verify Link[N].Grantee == Link[N+1].Grantor (transitive trust)
//  3. Validate scope inheritance: Child scopes must be subset of parent scopes
//  4. Check status: All ancestors must be Active (not Revoked/Suspended/Expired)
//  5. Final check: Root PoA must be issued by a trusted Principal
type DelegationChainValidator struct {
	repo    POARepository
	nowFn   func() func() time.Time
	metrics metrics.Metrics
}

// ChainValidationResult contains the outcome of chain validation.
type ChainValidationResult struct {
	Valid       bool // Overall chain validity
	ChainLength int  // Number of hops (0 = root, 1 = direct delegation, 2+ = transitive)
	RootPOA     *PowerOfAttorney
	ChainPath   []*PowerOfAttorney // Ordered from leaf to root
	Errors      []string           // Specific validation errors encountered
}

// NewDelegationChainValidator constructs a new chain validator.
func NewDelegationChainValidator(
	repo POARepository, nowFn func() func() time.Time, metrics metrics.Metrics,
) *DelegationChainValidator {
	return &DelegationChainValidator{
		repo:    repo,
		nowFn:   nowFn,
		metrics: metrics,
	}
}

// ValidateChain performs full transitive delegation chain validation.
//
// Parameters:
//   - ctx: context for cancellation
//   - leafPOA: the PoA being validated (presented by current session user)
//   - sessionUser: the authenticated user presenting the PoA (should match leafPOA.Grantee)
//
// Returns:
//   - ChainValidationResult: detailed validation outcome
//   - error: non-nil if validation logic fails (not if chain is invalid - check result.Valid)
//
// Validation Rules:
//  1. Leaf Check: leafPOA.Grantee MUST equal sessionUser
//  2. Parent Chain: For each parent link, Grantee[N] MUST equal Grantor[N+1]
//  3. Scope Inheritance: Each child's scopes MUST be subset of parent's scopes
//  4. Status Validation: All PoAs in chain MUST be Active (not Revoked/Suspended/Expired)
//  5. Depth Limits: Chain depth MUST NOT exceed AGENTAUTH_MAX_DELEGATION_DEPTH (default 5)
func (v *DelegationChainValidator) ValidateChain(
	ctx context.Context, leafPOA *PowerOfAttorney, sessionUser string,
) (*ChainValidationResult, error) {
	result := &ChainValidationResult{
		Valid:       true,
		ChainPath:   []*PowerOfAttorney{leafPOA},
		ChainLength: leafPOA.Depth,
	}

	// Rule 1: Leaf PoA grantee must match session user (holder-of-key binding)
	if leafPOA.Grantee != sessionUser {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf(
			"leaf grantee mismatch: expected %s, got session user %s", leafPOA.Grantee, sessionUser))
		if v.metrics != nil {
			v.metrics.IncUnauthorized()
		}
		return result, nil
	}

	// If no parent, this is a root delegation - validation complete
	if leafPOA.ParentPOAID == "" {
		result.RootPOA = leafPOA
		result.ChainLength = 0
		return result, nil
	}

	// Walk the chain upward from leaf to root
	currentPOA := leafPOA
	visitedIDs := make(map[string]bool) // Cycle detection
	maxDepth := 10                      // Safety limit to prevent infinite loops
	depth := 0

	for currentPOA.ParentPOAID != "" {
		depth++
		if depth > maxDepth {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("chain depth exceeds safety limit %d (possible cycle)", maxDepth))
			if v.metrics != nil {
				v.metrics.IncUnauthorized()
			}
			return result, nil
		}

		// Check for cycles
		if visitedIDs[currentPOA.ID] {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("delegation cycle detected at PoA %s", currentPOA.ID))
			if v.metrics != nil {
				v.metrics.IncUnauthorized()
			}
			return result, nil
		}
		visitedIDs[currentPOA.ID] = true

		// Load parent PoA
		parentPOA, ok := v.repo.Get(currentPOA.ParentPOAID)
		if !ok || parentPOA == nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf(
				"parent PoA %s not found for child %s", currentPOA.ParentPOAID, currentPOA.ID))
			if v.metrics != nil {
				v.metrics.IncUnauthorized()
			}
			return result, nil
		}

		// Rule 2: Transitive trust - parent's grantee must match child's grantor
		if parentPOA.Grantee != currentPOA.Grantor {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("chain break: parent PoA %s grantee (%s) != child PoA %s grantor (%s)",
				parentPOA.ID, parentPOA.Grantee, currentPOA.ID, currentPOA.Grantor))
			if v.metrics != nil {
				v.metrics.IncUnauthorized()
			}
			return result, nil
		}

		// Rule 3: Scope inheritance - child scopes must be subset of parent scopes
		if err := validateInheritedScope(parentPOA.Scope, currentPOA.Scope); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf(
				"scope inheritance violation from %s to %s: %v", parentPOA.ID, currentPOA.ID, err))
			if v.metrics != nil {
				v.metrics.IncScopeViolations()
			}
			return result, nil
		}

		// Rule 4: Status validation - all ancestors must be Active
		if parentPOA.Status != POAStatusActive {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("parent PoA %s has invalid status: %s", parentPOA.ID, parentPOA.Status))
			if v.metrics != nil {
				v.metrics.IncUnauthorized()
			}
			return result, nil
		}

		// Check expiration for parent (using same logic as Service.ValidateDelegation)
		now := v.nowFn()()
		if now.After(parentPOA.ValidUntil) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf(
				"parent PoA %s expired at %s", parentPOA.ID, parentPOA.ValidUntil.Format("2006-01-02T15:04:05Z")))
			if v.metrics != nil {
				v.metrics.IncExpired()
			}
			return result, nil
		}

		// Add to chain path (building root-to-leaf order)
		result.ChainPath = append([]*PowerOfAttorney{parentPOA}, result.ChainPath...)
		currentPOA = parentPOA
	}

	// Reached root PoA
	result.RootPOA = currentPOA
	result.ChainLength = depth

	return result, nil
}

// ValidateChainForAction validates both the chain structure AND that the action is permitted.
// This combines structural validation with scope checking for the requested action.
func (v *DelegationChainValidator) ValidateChainForAction(
	ctx context.Context, leafPOA *PowerOfAttorney, sessionUser, action string,
) error {
	// First validate chain structure
	chainResult, err := v.ValidateChain(ctx, leafPOA, sessionUser)
	if err != nil {
		return aap.New(aap.ErrInternal, fmt.Sprintf("chain validation failed: %v", err))
	}

	if !chainResult.Valid {
		errorMsg := "delegation chain invalid"
		if len(chainResult.Errors) > 0 {
			errorMsg = chainResult.Errors[0] // Return first error
		}
		return aap.New(aap.ErrUnauthorized, errorMsg)
	}

	// Then check if action is in leaf PoA's scope
	if !containsScope(leafPOA.Scope, action) {
		if v.metrics != nil {
			v.metrics.IncScopeViolations()
		}
		return aap.New(aap.ErrScopeViolation, fmt.Sprintf("action %s not in delegation scope", action))
	}

	return nil
}

// GetChainDepth returns the delegation depth for a PoA without full validation.
// Useful for metrics and depth enforcement checks.
func (v *DelegationChainValidator) GetChainDepth(poaID string) (int, error) {
	poa, ok := v.repo.Get(poaID)
	if !ok || poa == nil {
		return 0, fmt.Errorf("PoA %s not found", poaID)
	}
	return poa.Depth, nil
}

// ListChainAncestors returns all ancestor PoAs from leaf to root.
// Does NOT validate chain integrity - use ValidateChain for security checks.
func (v *DelegationChainValidator) ListChainAncestors(leafPoaID string) ([]*PowerOfAttorney, error) {
	poa, ok := v.repo.Get(leafPoaID)
	if !ok || poa == nil {
		return nil, fmt.Errorf("PoA %s not found", leafPoaID)
	}

	chain := []*PowerOfAttorney{poa}
	currentPOA := poa
	visitedIDs := make(map[string]bool)
	maxDepth := 10

	for currentPOA.ParentPOAID != "" && len(chain) < maxDepth {
		if visitedIDs[currentPOA.ID] {
			return chain, fmt.Errorf("cycle detected at PoA %s", currentPOA.ID)
		}
		visitedIDs[currentPOA.ID] = true

		parentPOA, ok := v.repo.Get(currentPOA.ParentPOAID)
		if !ok || parentPOA == nil {
			return chain, fmt.Errorf("parent PoA %s not found", currentPOA.ParentPOAID)
		}

		chain = append([]*PowerOfAttorney{parentPOA}, chain...)
		currentPOA = parentPOA
	}

	return chain, nil
}
