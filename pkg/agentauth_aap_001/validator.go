package agentauth_aap_001

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/aap"
)

// PoAValidator defines semantic validation beyond syntactic request checks.
// It runs after the DelegationRequest passes basic field & limit validation and
// before signing / persistence. Implementations SHOULD enforce cross-field
// invariants (e.g., grantor != grantee for certain scopes, duration alignment,
// mandatory restriction presence for scoped financial actions, etc.).
// Return nil if validation passes or an RFC-wrapped error (ErrInvalidRequest) otherwise.
// Future: extend with warning collection.
type PoAValidator interface {
	Validate(*PowerOfAttorney) error
}

// NoopPoAValidator is the default when no semantic validator supplied.
type NoopPoAValidator struct{}

func (NoopPoAValidator) Validate(*PowerOfAttorney) error { return nil }

// BasicPoAValidator provides minimal semantic rules intended to preserve backwards
// compatibility with earlier AAP001 test fixtures. Stricter governance rules
// (currency requirement for financial scopes, duration caps, wildcard gating, business
// hour / weekday gating, aggregate scope length limits, etc.) are reserved for the
// AdvancedPoAValidator to avoid breaking existing integrations when the basic mode
// is selected (default).
// Basic Rules:
// 1. Grantor != Grantee (avoid self-delegation unless scope is exactly ["*"]).
// 2. valid_from < valid_until.
// 3. Normalize nil restrictions map.
// 4. Regulatory scopes require a jurisdiction restriction.
// 5. Joint scopes (prefix "joint:") require a signatures >=2 restriction.
// 6. Numeric / currency field format sanity if provided (but not mandatory).
type BasicPoAValidator struct{}

//nolint:gocyclo // Basic PoA validation with field checks
func (BasicPoAValidator) Validate(p *PowerOfAttorney) error {
	if p == nil {
		return aap.New(aap.ErrInvalidRequest, "nil poa")
	}
	// Rule 1: Prevent trivial self-delegation unless scope is exactly ["*"]
	if p.Grantor == p.Grantee {
		if !(len(p.Scope) == 1 && p.Scope[0] == "*") {
			return aap.New(aap.ErrInvalidRequest, "grantor and grantee must differ for non-wildcard delegation")
		}
	}
	// Temporal invariant
	if !p.ValidFrom.Before(p.ValidUntil) {
		return aap.New(aap.ErrInvalidRequest, "valid_from must be before valid_until")
	}
	// Normalize restrictions map if nil
	if p.Restrictions == nil {
		p.Restrictions = map[string]string{}
	}

	// Prototype Rule 6: Jurisdiction presence for regulatory scopes.
	// If any scope starts with "regulatory:" require restriction key "jurisdiction".
	for _, sc := range p.Scope {
		if len(sc) >= 11 && sc[:11] == "regulatory:" {
			if _, ok := p.Restrictions["jurisdiction"]; !ok {
				return aap.New(aap.ErrInvalidRequest, "jurisdiction restriction required for regulatory scopes")
			}
		}
	}

	// Prototype Rule 7: Joint delegation placeholder enforcement.
	// If scope contains "joint:" require restriction key "signatures" with integer count >=2.
	for _, sc := range p.Scope {
		if strings.HasPrefix(sc, "joint:") {
			val, ok := p.Restrictions["signatures"]
			if !ok || val == "" {
				return aap.New(aap.ErrInvalidRequest, "signatures restriction required for joint delegation")
			}
			n, err := strconv.Atoi(val)
			if err != nil || n < 2 {
				return aap.New(aap.ErrInvalidRequest, "signatures count must be integer >=2 for joint delegation")
			}
		}
	}

	// Numeric restriction parsing & consistency checks.
	// max_amount, max_daily_amount must be positive decimals; if both present max_daily_amount >= max_amount.
	// currency must be ISO-like (3 uppercase letters) when present.
	if amtStr, ok := p.Restrictions["max_amount"]; ok {
		if err := validatePositiveDecimal(amtStr); err != nil {
			return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("invalid max_amount: %v", err))
		}
	}
	if damtStr, ok := p.Restrictions["max_daily_amount"]; ok {
		if err := validatePositiveDecimal(damtStr); err != nil {
			return aap.New(aap.ErrInvalidRequest, fmt.Sprintf("invalid max_daily_amount: %v", err))
		}
		if amtStr, ok2 := p.Restrictions["max_amount"]; ok2 {
			if greaterThan(amtStr, damtStr) {
				return aap.New(aap.ErrInvalidRequest, "max_daily_amount must be >= max_amount")
			}
		}
	}
	if cur, ok := p.Restrictions["currency"]; ok {
		if len(cur) != 3 || cur != strings.ToUpper(cur) || !isAlpha(cur) {
			return aap.New(aap.ErrInvalidRequest, "currency must be 3 uppercase letters")
		}
	}
	return nil
}

// AdvancedPoAValidator extends BasicPoAValidator with additional governance semantics:
// - Financial (transaction:) scopes must include a currency restriction.
// - Transaction delegation duration hard-capped at 30 days even if global limit higher.
// - Optional business-hour and weekday gating via restrictions:
//   - valid_hours: "HH-HH" (24h clock, start inclusive, end exclusive). Example: "09-17".
//   - valid_weekdays: comma-separated integers 0-6 (0=Sunday). Example: "1,2,3,4,5".
//     These are validated syntactically at issuance; runtime enforcement occurs in VerifyToken.
//
// - If threshold >1 then inline MultiSignatures forbidden unless AGENTAUTH_ALLOW_INLINE_MULTISIG=1.
// - Optional maximum aggregate scope length (AGENTAUTH_MAX_SCOPE_AGG_LEN env).
// - Wildcard scope disabled unless AGENTAUTH_ALLOW_WILDCARD=1.
type AdvancedPoAValidator struct{ BasicPoAValidator }

//nolint:gocyclo // Advanced PoA validation with comprehensive checks

//nolint:gocyclo // Advanced PoA validation with comprehensive checks
func (AdvancedPoAValidator) Validate(p *PowerOfAttorney) error {
	if err := (BasicPoAValidator{}).Validate(p); err != nil {
		return err
	}
	if p == nil {
		return aap.New(aap.ErrInvalidRequest, "nil poa")
	}
	// Enforce currency + 30d duration cap for transaction scopes
	hasTxn := false
	for _, sc := range p.Scope {
		if strings.HasPrefix(sc, "transaction:") {
			hasTxn = true
			break
		}
	}
	if hasTxn {
		if _, ok := p.Restrictions["currency"]; !ok {
			return aap.New(aap.ErrInvalidRequest, "currency restriction required for transaction scopes")
		}
		if p.ValidUntil.Sub(p.ValidFrom) > (30 * 24 * time.Hour) {
			return aap.New(aap.ErrInvalidRequest, "transaction delegation duration exceeds 30d cap")
		}
	}
	// valid_hours format check
	if vh, ok := p.Restrictions["valid_hours"]; ok {
		parts := strings.Split(vh, "-")
		if len(parts) != 2 {
			return aap.New(aap.ErrInvalidRequest, "valid_hours must be HH-HH")
		}
		sh, eh := parts[0], parts[1]
		if len(sh) != 2 || len(eh) != 2 {
			return aap.New(aap.ErrInvalidRequest, "valid_hours hours must be 2 digits")
		}
		sHi, err1 := strconv.Atoi(sh)
		eHi, err2 := strconv.Atoi(eh)
		if err1 != nil || err2 != nil || sHi < 0 || sHi > 23 || eHi < 0 || eHi > 23 {
			return aap.New(aap.ErrInvalidRequest, "valid_hours invalid hour range")
		}
	}
	if vwd, ok := p.Restrictions["valid_weekdays"]; ok {
		items := strings.Split(vwd, ",")
		if len(items) == 0 {
			return aap.New(aap.ErrInvalidRequest, "valid_weekdays empty")
		}
		seen := map[int]struct{}{}
		for _, it := range items {
			it = strings.TrimSpace(it)
			if it == "" {
				return aap.New(aap.ErrInvalidRequest, "valid_weekdays contains empty entry")
			}
			v, err := strconv.Atoi(it)
			if err != nil || v < 0 || v > 6 {
				return aap.New(aap.ErrInvalidRequest, "valid_weekdays out of range 0-6")
			}
			if _, dup := seen[v]; dup {
				return aap.New(aap.ErrInvalidRequest, "valid_weekdays duplicate value")
			}
			seen[v] = struct{}{}
		}
	}
	// Multi-signature inline constraint
	if p.Threshold > 1 && p.MultiSignatures != nil && len(p.MultiSignatures) > 0 {
		if os.Getenv("AGENTAUTH_ALLOW_INLINE_MULTISIG") != "1" {
			return aap.New(aap.ErrInvalidRequest, "inline multi_signatures not allowed (enable AGENTAUTH_ALLOW_INLINE_MULTISIG=1)")
		}
	}
	// Aggregate scope length constraint (optional)
	if limStr := os.Getenv("AGENTAUTH_MAX_SCOPE_AGG_LEN"); limStr != "" {
		if l, err := strconv.Atoi(limStr); err == nil && l > 0 {
			total := 0
			for _, sc := range p.Scope {
				total += len(sc)
			}
			if total > l {
				return aap.New(aap.ErrInvalidRequest, "aggregate scope length exceeds limit")
			}
		}
	}
	// Wildcard control
	if os.Getenv("AGENTAUTH_ALLOW_WILDCARD") != "1" {
		for _, sc := range p.Scope {
			if sc == "*" {
				return aap.New(aap.ErrInvalidRequest, "wildcard scope disabled")
			}
		}
	}
	return nil
}

// selectPoAValidator returns a PoAValidator based on environment (semantic|advanced|basic|none).
func selectPoAValidator() PoAValidator {
	switch strings.ToLower(os.Getenv("AGENTAUTH_POA_VALIDATOR")) {
	case "semantic":
		// Full AAP002 semantic validation with enhanced checks
		return NewEnhancedPoAValidator()
	case "advanced":
		return AdvancedPoAValidator{}
	case "basic":
		return BasicPoAValidator{}
	case "none":
		return NoopPoAValidator{}
	default:
		// Default to basic for backward compatibility
		return BasicPoAValidator{}
	}
}

// validatePositiveDecimal ensures the string parses to a float64 >=0.
func validatePositiveDecimal(s string) error {
	if s == "" {
		return fmt.Errorf("empty")
	}
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return err
	}
	// We rely on ParseFloat to ensure numeric; sign check handled implicitly.
	if strings.HasPrefix(s, "-") {
		return fmt.Errorf("negative")
	}
	return nil
}

// greaterThan compares two decimal strings (prototype: ParseFloat for simplicity).
func greaterThan(a, b string) bool {
	af, err1 := strconv.ParseFloat(a, 64)
	bf, err2 := strconv.ParseFloat(b, 64)
	if err1 != nil || err2 != nil {
		return false
	}
	return af > bf
}

// isAlpha returns true if all characters are A-Z.
func isAlpha(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
