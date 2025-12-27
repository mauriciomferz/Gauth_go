package metrics

// Standard Decision Reasons
const (
	ReasonPolicyAllow            = "policy_allow"
	ReasonPolicyDeny             = "policy_deny"
	ReasonScopeViolation         = "scope_violation"
	ReasonTemporalViolation      = "temporal_violation"
	ReasonDelegationExceeded     = "delegation_exceeded"
	ReasonSignatureInvalid       = "signature_invalid"
	ReasonReplayDetected         = "replay_detected"
	ReasonRevocation             = "revocation"
	ReasonCapabilityInsufficient = "capability_insufficient"
	ReasonJurisdictionViolation  = "jurisdiction_violation"
	// Additional granular reasons
	ReasonTokenExpired    = "token_expired" // Sub-category of temporal
	ReasonNotYetValid     = "not_yet_valid" // Sub-category of temporal
	ReasonKeyNotFound     = "key_not_found"
	ReasonDomainConflict  = "domain_conflict"
	ReasonTamperSuspected = "tamper_suspected"
)

// Standard Outcomes
const (
	OutcomeAllow = "allow"
	OutcomeDeny  = "deny"
	OutcomeError = "error"
)
