package legalframework

import "errors"

// ErrNoPolicyDecisionPoint is returned if the PDP is missing
var ErrNoPolicyDecisionPoint = errors.New("no policy decision point configured")

// Decision represents an authorization decision

// AppliesTo returns true if the policy applies to the given subject/resource/action (stub)
func (p Policy) AppliesTo(subject, resource, action string) bool {
	// STUB: Always applies
	return true
}

// Evaluate returns a Decision for the given subject/resource/action (stub)
func (p Policy) Evaluate(subject, resource, action string) Decision {
	// STUB: Always permit
	return Decision{Effect: "permit"}
}

// DenyOverridesCombiner combines decisions using deny-overrides algorithm
func DenyOverridesCombiner(decisions []Decision) string {
	for _, decision := range decisions {
		if decision.Effect == "deny" {
			return "deny"
		}
	}
	for _, decision := range decisions {
		if decision.Effect == "permit" {
			return "permit"
		}
	}
	return ""
}


// PermitOverridesCombiner combines decisions using permit-overrides algorithm
func PermitOverridesCombiner(decisions []Decision) string {
	for _, decision := range decisions {
		if decision.Effect == "permit" {
			return "permit"
		}
	}
	for _, decision := range decisions {
		if decision.Effect == "deny" {
			return "deny"
		}
	}
	return ""
}

// FirstApplicableCombiner returns the effect of the first applicable rule
func FirstApplicableCombiner(decisions []Decision) string {
	if len(decisions) > 0 {
		return decisions[0].Effect
	}
	return ""
}

// Authorize is a stub method for LegalFramework (to be implemented)
func (lf *LegalFramework) Authorize(ctx interface{}, subject, resource, action string) (*Decision, error) {
	// Minimal implementation: evaluate all policies, combine results, return a Decision
	if lf == nil || lf.PDP == nil {
		return nil, ErrNoPolicyDecisionPoint
	}

	var decisions []Decision
	for _, policy := range lf.PDP.Policies {
		if policy.AppliesTo(subject, resource, action) {
			decision := policy.Evaluate(subject, resource, action)
			decisions = append(decisions, decision)
		}
	}

	var effect string
	if lf.PDP.RuleCombiner != nil {
		effect = lf.PDP.RuleCombiner(decisions)
	} else {
		effect = FirstApplicableCombiner(decisions)
	}

	if effect == "" {
		effect = lf.PDP.DefaultEffect
	}

	return &Decision{Effect: effect}, nil
}

type LegalFramework struct {
	ID                     string
	Name                   string
	Version                string
	PIP                    PowerInformationPoint
	PAP                    PowerAdministrationPoint
	PVP                    PowerVerificationPoint
	PDP                    *PolicyDecisionPoint
	PEP                    EnforcementPoint
	CentralAuthorities     []Authority
	DelegatedAuthorities   []Authority
	Jurisdictions          []string
	ValidationRules        []ValidationRule
	ContextEnrichmentRules []ContextEnrichmentRule
}

type PolicyDecisionPoint struct {
	Policies      []Policy
	RuleCombiner  RuleCombiner
	DefaultEffect string
}

type RuleCombiner func([]Decision) string

func NewLegalFramework(id, name, version string) *LegalFramework {
	return &LegalFramework{
		ID:      id,
		Name:    name,
		Version: version,
		PIP: PowerInformationPoint{
			DataSources: []DataSource{},
			Cache:       &DataCache{Entries: make(map[string]DataCacheEntry)},
			Updates:     make(chan DataUpdate, 100),
		},
		PAP: PowerAdministrationPoint{
			Policies:       []Policy{},
			Administrators: []string{},
			AuditLog:       []AdminAction{},
		},
		PVP: PowerVerificationPoint{
			TrustAnchors: []TrustAnchor{},
			CertStore:    &CertificateStore{Certificates: make(map[string]Certificate)},
			Validators:   []LegalValidator{},
		},
		PDP: &PolicyDecisionPoint{
			Policies:      []Policy{},
			DefaultEffect: "deny",
		},
		PEP: EnforcementPoint{
			Rules:    []EnforcementRule{},
			Handlers: []Handler{},
			Audit:    &AuditLog{Entries: []AuditEntry{}, MaxSize: 1000},
		},
		CentralAuthorities:     []Authority{},
		DelegatedAuthorities:   []Authority{},
		Jurisdictions:          []string{},
		ValidationRules:        []ValidationRule{},
		ContextEnrichmentRules: []ContextEnrichmentRule{},
	}
}
