// Package observability provides lightweight in-memory metrics primitives used by AgentAuth.
// It purposefully avoids external dependencies so core validation logic remains portable.
// Adapters can export these counters to Prometheus, OpenTelemetry or custom collectors.
package observability

import "sync/atomic"

// ViolationCategory enumerates the classes of validation rejection reasons we track.
// These map to token authenticity, temporal validity, replay protection, and structural issues.
type ViolationCategory int32

const (
	// SigInvalid covers signature or MAC failures and malformed header/payload structure.
	SigInvalid ViolationCategory = iota
	// Expired covers tokens rejected due to exp claim being in the past.
	Expired
	// NotYetValid covers tokens rejected because nbf is in the future.
	NotYetValid
	// IssuerMismatch covers tokens whose iss does not match configured authority.
	IssuerMismatch
	// ReplayDetected covers duplicate JTI or replay store rejection.
	ReplayDetected
	// AudienceMismatch covers aud claim not intersecting accepted audiences.
	AudienceMismatch
	// MissingClaim covers required structural claims absent (jti, exp, etc.).
	MissingClaim
	// Unknown is a fallback bucket for uncategorized failures.
	Unknown
	// ScopeUTF8Invalid indicates a scope entry failed UTF-8 validation.
	ScopeUTF8Invalid
	// ScopeControlChar indicates a scope entry contained an ASCII control character.
	ScopeControlChar
	// RestrictionUTF8Invalid indicates a restriction key/value failed UTF-8 validation.
	RestrictionUTF8Invalid
	// RestrictionControlChar indicates a restriction key/value contained an ASCII control character.
	RestrictionControlChar
	// numCategories is internal sentinel for array sizing; must be last.
	numCategories
)

// ViolationCounters maintains atomic counters per category.
type ViolationCounters struct {
	counts [numCategories]atomic.Uint64
}

// NewViolationCounters constructs a zeroed counter set.
func NewViolationCounters() *ViolationCounters { return &ViolationCounters{} }

// Inc increments the counter for the given category.
func (v *ViolationCounters) Inc(cat ViolationCategory) {
	if cat < 0 || cat >= numCategories {
		cat = Unknown
	}
	v.counts[cat].Add(1)
}

// Snapshot returns a copy of all counters as a map for easy JSON marshaling/export.
func (v *ViolationCounters) Snapshot() map[string]uint64 {
	m := map[string]uint64{
		"sig_invalid":              v.counts[SigInvalid].Load(),
		"expired":                  v.counts[Expired].Load(),
		"not_yet_valid":            v.counts[NotYetValid].Load(),
		"issuer_mismatch":          v.counts[IssuerMismatch].Load(),
		"replay_detected":          v.counts[ReplayDetected].Load(),
		"audience_mismatch":        v.counts[AudienceMismatch].Load(),
		"missing_claim":            v.counts[MissingClaim].Load(),
		"unknown":                  v.counts[Unknown].Load(),
		"scope_utf8_invalid":       v.counts[ScopeUTF8Invalid].Load(),
		"scope_control_char":       v.counts[ScopeControlChar].Load(),
		"restriction_utf8_invalid": v.counts[RestrictionUTF8Invalid].Load(),
		"restriction_control_char": v.counts[RestrictionControlChar].Load(),
	}
	return m
}

// Reset sets all counters back to zero.
func (v *ViolationCounters) Reset() {
	for i := ViolationCategory(0); i < numCategories; i++ {
		v.counts[i].Store(0)
	}
}

// SetFromSnapshot initializes counters from a provided snapshot map. Missing keys are treated as zero.
// This is primarily used for persistence restoration. It overwrites existing values atomically.
func (v *ViolationCounters) SetFromSnapshot(m map[string]uint64) {
	if m == nil {
		return
	}
	// Explicit mapping to avoid accidental key mismatch
	v.counts[SigInvalid].Store(m["sig_invalid"])
	v.counts[Expired].Store(m["expired"])
	v.counts[NotYetValid].Store(m["not_yet_valid"])
	v.counts[IssuerMismatch].Store(m["issuer_mismatch"])
	v.counts[ReplayDetected].Store(m["replay_detected"])
	v.counts[AudienceMismatch].Store(m["audience_mismatch"])
	v.counts[MissingClaim].Store(m["missing_claim"])
	v.counts[Unknown].Store(m["unknown"])
	// Hygiene categories (treat missing keys as zero)
	v.counts[ScopeUTF8Invalid].Store(m["scope_utf8_invalid"])
	v.counts[ScopeControlChar].Store(m["scope_control_char"])
	v.counts[RestrictionUTF8Invalid].Store(m["restriction_utf8_invalid"])
	v.counts[RestrictionControlChar].Store(m["restriction_control_char"])
}
