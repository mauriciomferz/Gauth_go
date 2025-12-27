package pdp

// DecisionReason represents standardized denial/approval reasons for observability.
// This taxonomy enables systematic monitoring, alerting, and compliance reporting.
type DecisionReason string

// Authorization Decision Reasons - Comprehensive taxonomy for PDP decisions
const (
	// === APPROVAL REASONS ===
	ReasonApprovedDirect        DecisionReason = "approved_direct"         // Direct permission grant
	ReasonApprovedDelegated     DecisionReason = "approved_delegated"      // Via valid delegation chain
	ReasonApprovedConditional   DecisionReason = "approved_conditional"    // Conditional policy satisfied
	ReasonApprovedABAC          DecisionReason = "approved_abac"           // Attribute-based policy match
	ReasonApprovedCapability    DecisionReason = "approved_capability"     // Capability token valid
	ReasonApprovedEmergencyMode DecisionReason = "approved_emergency_mode" // Emergency bypass active

	// === DENIAL REASONS - Authentication/Identity ===
	ReasonDeniedNoAuth           DecisionReason = "denied_no_auth"           // No authentication provided
	ReasonDeniedExpiredToken     DecisionReason = "denied_expired_token"     // Token/credential expired
	ReasonDeniedInvalidSignature DecisionReason = "denied_invalid_signature" // Cryptographic verification failed
	ReasonDeniedMalformedToken   DecisionReason = "denied_malformed_token"   // Token structure invalid
	ReasonDeniedUnknownSubject   DecisionReason = "denied_unknown_subject"   // Subject not recognized
	ReasonDeniedRevokedToken     DecisionReason = "denied_revoked_token"     // Token explicitly revoked

	// === DENIAL REASONS - Authorization/Policy ===
	ReasonDeniedNoPolicy            DecisionReason = "denied_no_policy"            // No applicable policy found
	ReasonDeniedPolicyMismatch      DecisionReason = "denied_policy_mismatch"      // Policy doesn't match request
	ReasonDeniedInsufficientScope   DecisionReason = "denied_insufficient_scope"   // Scope too narrow
	ReasonDeniedScopeViolation      DecisionReason = "denied_scope_violation"      // Request exceeds granted scope
	ReasonDeniedResourceRestriction DecisionReason = "denied_resource_restriction" // Resource not permitted
	ReasonDeniedActionRestriction   DecisionReason = "denied_action_restriction"   // Action not permitted
	ReasonDeniedConditionalFailed   DecisionReason = "denied_conditional_failed"   // Condition evaluation failed
	ReasonDeniedABACMismatch        DecisionReason = "denied_abac_mismatch"        // Attributes don't satisfy policy

	// === DENIAL REASONS - Delegation ===
	ReasonDeniedNoDelegation      DecisionReason = "denied_no_delegation"      // No delegation chain found
	ReasonDeniedDelegationExpired DecisionReason = "denied_delegation_expired" // Delegation expired
	ReasonDeniedDelegationRevoked DecisionReason = "denied_delegation_revoked" // Delegation revoked
	ReasonDeniedDelegationDepth   DecisionReason = "denied_delegation_depth"   // Max delegation depth exceeded
	ReasonDeniedDelegationScope   DecisionReason = "denied_delegation_scope"   // Delegation scope insufficient
	ReasonDeniedInvalidDelegation DecisionReason = "denied_invalid_delegation" // Delegation chain integrity failed

	// === DENIAL REASONS - Capability/Token ===
	ReasonDeniedCapabilityExpired  DecisionReason = "denied_capability_expired"  // Capability token expired
	ReasonDeniedCapabilityRevoked  DecisionReason = "denied_capability_revoked"  // Capability revoked
	ReasonDeniedCapabilityMismatch DecisionReason = "denied_capability_mismatch" // Capability doesn't match request
	ReasonDeniedCapabilityInvalid  DecisionReason = "denied_capability_invalid"  // Capability validation failed

	// === DENIAL REASONS - Rate Limiting & Quotas ===
	ReasonDeniedRateLimit     DecisionReason = "denied_rate_limit"     // Rate limit exceeded
	ReasonDeniedQuotaExceeded DecisionReason = "denied_quota_exceeded" // Resource quota exceeded
	ReasonDeniedBurstLimit    DecisionReason = "denied_burst_limit"    // Burst limit exceeded
	ReasonDeniedConcurrency   DecisionReason = "denied_concurrency"    // Concurrent request limit hit

	// === DENIAL REASONS - Security/Compliance ===
	ReasonDeniedReplayAttack   DecisionReason = "denied_replay_attack"   // JTI replay detected
	ReasonDeniedTimestampSkew  DecisionReason = "denied_timestamp_skew"  // Time synchronization issue
	ReasonDeniedRiskScore      DecisionReason = "denied_risk_score"      // Risk assessment threshold exceeded
	ReasonDeniedGeoRestriction DecisionReason = "denied_geo_restriction" // Geographic restriction
	ReasonDeniedCompliance     DecisionReason = "denied_compliance"      // Compliance policy violation

	// === DENIAL REASONS - System/Operational ===
	ReasonDeniedSystemError     DecisionReason = "denied_system_error"      // Internal system error (fail-closed)
	ReasonDeniedPolicyLoadFail  DecisionReason = "denied_policy_load_fail"  // Policy loading failed
	ReasonDeniedReplayStoreFail DecisionReason = "denied_replay_store_fail" // Replay store unavailable (fail-closed)
	ReasonDeniedTimeout         DecisionReason = "denied_timeout"           // Decision timeout
	ReasonDeniedMaintenanceMode DecisionReason = "denied_maintenance_mode"  // System in maintenance mode

	// === DENIAL REASONS - Obligations ===
	ReasonDeniedObligationFailed DecisionReason = "denied_obligation_failed" // Mandatory obligation couldn't execute
)

// ReasonCategory groups reasons into high-level categories for dashboards
type ReasonCategory string

const (
	CategoryApproval     ReasonCategory = "approval"
	CategoryAuthN        ReasonCategory = "authentication"
	CategoryAuthZ        ReasonCategory = "authorization"
	CategoryDelegation   ReasonCategory = "delegation"
	CategoryRateLimiting ReasonCategory = "rate_limiting"
	CategorySecurity     ReasonCategory = "security"
	CategorySystem       ReasonCategory = "system"
	CategoryObligation   ReasonCategory = "obligation"
)

// GetCategory returns the high-level category for a decision reason.
func (r DecisionReason) GetCategory() ReasonCategory {
	switch r {
	case ReasonApprovedDirect, ReasonApprovedDelegated, ReasonApprovedConditional,
		ReasonApprovedABAC, ReasonApprovedCapability, ReasonApprovedEmergencyMode:
		return CategoryApproval

	case ReasonDeniedNoAuth, ReasonDeniedExpiredToken, ReasonDeniedInvalidSignature,
		ReasonDeniedMalformedToken, ReasonDeniedUnknownSubject, ReasonDeniedRevokedToken:
		return CategoryAuthN

	case ReasonDeniedNoPolicy, ReasonDeniedPolicyMismatch, ReasonDeniedInsufficientScope,
		ReasonDeniedScopeViolation, ReasonDeniedResourceRestriction, ReasonDeniedActionRestriction,
		ReasonDeniedConditionalFailed, ReasonDeniedABACMismatch:
		return CategoryAuthZ

	case ReasonDeniedNoDelegation, ReasonDeniedDelegationExpired, ReasonDeniedDelegationRevoked,
		ReasonDeniedDelegationDepth, ReasonDeniedDelegationScope, ReasonDeniedInvalidDelegation,
		ReasonDeniedCapabilityExpired, ReasonDeniedCapabilityRevoked, ReasonDeniedCapabilityMismatch,
		ReasonDeniedCapabilityInvalid:
		return CategoryDelegation

	case ReasonDeniedRateLimit, ReasonDeniedQuotaExceeded, ReasonDeniedBurstLimit, ReasonDeniedConcurrency:
		return CategoryRateLimiting

	case ReasonDeniedReplayAttack, ReasonDeniedTimestampSkew, ReasonDeniedRiskScore,
		ReasonDeniedGeoRestriction, ReasonDeniedCompliance:
		return CategorySecurity

	case ReasonDeniedSystemError, ReasonDeniedPolicyLoadFail, ReasonDeniedReplayStoreFail,
		ReasonDeniedTimeout, ReasonDeniedMaintenanceMode:
		return CategorySystem

	case ReasonDeniedObligationFailed:
		return CategoryObligation

	default:
		return CategorySystem
	}
}

// IsApproval returns true if the reason represents an approval.
func (r DecisionReason) IsApproval() bool {
	return r.GetCategory() == CategoryApproval
}

// IsSecurity returns true if the reason is security-related (requires alerting).
func (r DecisionReason) IsSecurity() bool {
	return r.GetCategory() == CategorySecurity
}
