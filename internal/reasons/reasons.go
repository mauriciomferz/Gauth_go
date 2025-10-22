// Package reasons enumerates canonical lifecycle / decision reason codes to
// ensure consistent audit log vocabulary across subsystems.
package reasons

// CanonicalReasons defines the authoritative set of lifecycle / decision reason codes.
// Update this slice (and corresponding JSON Schemas) together to maintain consistency.
var CanonicalReasons = []string{
	"init",
	"status_change",
	"noop",
	"maintenance",
	"rate_limited",
	"policy_violation",
	"invalid_transition",
	"unsupported_status",
	"invalid_payload",
	"not_found",
}

// Set used for quick membership checks.
var canonicalSet map[string]struct{}

func init() {
	canonicalSet = make(map[string]struct{}, len(CanonicalReasons))
	for _, r := range CanonicalReasons {
		canonicalSet[r] = struct{}{}
	}
}

// IsValid returns true if the provided reason is in the canonical taxonomy.
func IsValid(reason string) bool { _, ok := canonicalSet[reason]; return ok }

// All returns a copy of the canonical reasons slice to prevent accidental modification.
func All() []string {
	out := make([]string, len(CanonicalReasons))
	copy(out, CanonicalReasons)
	return out
}
