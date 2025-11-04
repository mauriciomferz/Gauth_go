package authorization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

// CompositeAuthorizationArtifact models hierarchical delegation and guardrails.
type CompositeAuthorizationArtifact struct {
	AISystemID           string                `json:"ai_system_id"`
	AuthorizingParty     *AuthorizingParty     `json:"authorizing_party"`
	AuthorizationGrant   *AuthorizationGrant   `json:"authorization_grant"`
	PowersGranted        *PowersGranted        `json:"powers_granted"`
	DecisionAuthority    *DecisionAuthority    `json:"decision_authority"`
	TransactionRights    *TransactionRights    `json:"transaction_rights"`
	ActionPermissions    *ActionPermissions    `json:"action_permissions"`
	DualControlPrinciple *DualControlPrinciple `json:"dual_control_principle"`
	AuthorizationCascade *AuthorizationCascade `json:"authorization_cascade"`
	ExpiresAt            time.Time             `json:"expires_at"`
}

type AuthorizingParty struct {
	ID                       string                    `json:"id"`
	Name                     string                    `json:"name"`
	Type                     string                    `json:"type"`
	AuthorizedRepresentative *AuthorizedRepresentative `json:"authorized_representative,omitempty"`
	LegalCapacity            *LegalCapacity            `json:"legal_capacity,omitempty"`
}

type AuthorizedRepresentative struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Position       string     `json:"position"`
	AuthorityScope []string   `json:"authority_scope,omitempty"`
	ValidFrom      *time.Time `json:"valid_from,omitempty"`
	ValidUntil     *time.Time `json:"valid_until,omitempty"`
	VerifiedBy     string     `json:"verified_by,omitempty"`
}

type LegalCapacity struct {
	Verified         bool       `json:"verified"`
	VerificationDate *time.Time `json:"verification_date,omitempty"`
	VerifiedBy       string     `json:"verified_by,omitempty"`
	Jurisdiction     string     `json:"jurisdiction,omitempty"`
	LegalFramework   string     `json:"legal_framework,omitempty"`
}

type AuthorizationGrant struct {
	Type        string     `json:"type"`
	Scope       []string   `json:"scope"`
	Limitations []string   `json:"limitations,omitempty"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	Revocable   bool       `json:"revocable"`
}

type PowersGranted struct {
	BasicPowers     []string            `json:"basic_powers"`
	DerivedPowers   []string            `json:"derived_powers,omitempty"`
	PowerDerivation map[string][]string `json:"power_derivation,omitempty"`
}

type DecisionAuthority struct {
	AutonomousDecisions []string          `json:"autonomous_decisions,omitempty"`
	ApprovalRequired    []string          `json:"approval_required,omitempty"`
	DecisionMatrix      map[string]string `json:"decision_matrix,omitempty"`
	EscalationRules     *EscalationRules  `json:"escalation_rules,omitempty"`
}

type EscalationRules struct {
	ThresholdTriggers        map[string]interface{} `json:"threshold_triggers,omitempty"`
	EscalationPath           []string               `json:"escalation_path,omitempty"`
	ResponseTimeRequirements map[string]string      `json:"response_time_requirements,omitempty"`
	OverrideAuthority        []string               `json:"override_authority,omitempty"`
}

type TransactionRights struct {
	AllowedTransactionTypes []string `json:"allowed_transaction_types,omitempty"`
	ProhibitedTransactions  []string `json:"prohibited_transactions,omitempty"`
}

type ActionPermissions struct {
	SystemActions     []string               `json:"system_actions,omitempty"`
	ResourceActions   map[string][]string    `json:"resource_actions,omitempty"`
	HumanInteractions map[string]interface{} `json:"human_interactions,omitempty"`
	AgentInteractions map[string]interface{} `json:"agent_interactions,omitempty"`
}

type DualControlPrinciple struct {
	Enabled             bool                `json:"enabled"`
	RequiresDualControl []string            `json:"requires_dual_control,omitempty"`
	ApprovalMatrix      map[string][]string `json:"approval_matrix,omitempty"`
}

type AuthorizationCascade struct {
	AccountabilityChain []string                 `json:"accountability_chain"`
	CascadeChain        []map[string]interface{} `json:"cascade_chain,omitempty"`
}

// CompositeAuthorizationState holds activation metadata.
type CompositeAuthorizationState struct {
	Artifact             *CompositeAuthorizationArtifact `json:"artifact"`
	ActivatedAt          time.Time                       `json:"activated_at"`
	Version              string                          `json:"version"`
	PreviousArtifactHash string                          `json:"previous_artifact_hash,omitempty"`
	CanonicalHash        string                          `json:"canonical_hash"`
}

// CanonicalHash computes a stable hash over selected fields ensuring consistent ordering.
func CanonicalHash(a *CompositeAuthorizationArtifact) (string, error) {
	if a == nil {
		return "", nil
	}
	// Build a canonical view (sorted slices where order not semantically significant)
	canon := map[string]interface{}{}
	canon["ai_system_id"] = a.AISystemID
	canon["expires_at"] = a.ExpiresAt.UTC().Format(time.RFC3339)
	if a.AuthorizationGrant != nil {
		sort.Strings(a.AuthorizationGrant.Scope)
		sort.Strings(a.AuthorizationGrant.Limitations)
	}
	if a.PowersGranted != nil {
		sort.Strings(a.PowersGranted.BasicPowers)
		sort.Strings(a.PowersGranted.DerivedPowers)
	}
	if a.AuthorizationCascade != nil {
		// AccountabilityChain order is meaningful; retain as-is
		canon["accountability_chain"] = a.AuthorizationCascade.AccountabilityChain
	}
	// Minimal inclusion of decision matrix keys sorted for determinism
	if a.DecisionAuthority != nil && len(a.DecisionAuthority.DecisionMatrix) > 0 {
		keys := make([]string, 0, len(a.DecisionAuthority.DecisionMatrix))
		for k := range a.DecisionAuthority.DecisionMatrix {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ordered := make([][2]string, 0, len(keys))
		for _, k := range keys {
			ordered = append(ordered, [2]string{k, a.DecisionAuthority.DecisionMatrix[k]})
		}
		canon["decision_matrix"] = ordered
	}
	b, err := json.Marshal(canon)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
