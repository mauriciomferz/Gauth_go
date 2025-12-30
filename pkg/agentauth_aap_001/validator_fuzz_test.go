//go:build go1.18

package agentauth_aap_001

import (
	"testing"
	"time"
)

// FuzzPoAValidator tests the Basic and Advanced PoA validators for robustness.
func FuzzPoAValidator(f *testing.F) {
	// Seed with some valid/interesting cases
	f.Add("alice", "bob", "read", "jurisdiction=US", 0, 3600)
	f.Add("grantor", "grantor", "*", "", 0, 3600)
	f.Add("g1", "g2", "regulatory:tax", "jurisdiction=EU", 0, 3600)
	f.Add("j1", "j2", "joint:account", "signatures=2", 0, 3600)

	f.Fuzz(func(t *testing.T, grantor, grantee, scopeCSV, restrCSV string, fromOff, untilOff int) {
		p := &PowerOfAttorney{
			Grantor:      grantor,
			Grantee:      grantee,
			Scope:        splitSemi(scopeCSV),
			Restrictions: restrMap(restrCSV),
			ValidFrom:    time.Unix(int64(fromOff), 0).UTC(),
			ValidUntil:   time.Unix(int64(fromOff+untilOff), 0).UTC(),
		}

		// Test Basic Validator
		bv := BasicPoAValidator{}
		_ = bv.Validate(p)

		// Test Advanced Validator
		av := AdvancedPoAValidator{}
		_ = av.Validate(p)

		// Test Enhanced Validator if possible (requires more setup, but let's try basic call)
		// ev := NewEnhancedPoAValidator()
		// _ = ev.Validate(p)
	})
}
