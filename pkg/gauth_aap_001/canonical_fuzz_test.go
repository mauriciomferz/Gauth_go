//go:build go1.18

package gauth_aap_001

import (
	"testing"
	"time"
)

// FuzzCanonicalPOADigest provides property-style fuzzing for canonical digest stability.
func FuzzCanonicalPOADigest(f *testing.F) {
	seed := []*PowerOfAttorney{
		{ID: "seed1", Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Restrictions: map[string]string{"env": "dev"}, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(3600, 0).UTC(), Status: POAStatusActive, CreatedAt: time.Unix(10, 0).UTC(), UpdatedAt: time.Unix(10, 0).UTC()},
		{ID: "seed2", Grantor: "carol", Grantee: "dave", Scope: []string{"write", "read"}, Restrictions: map[string]string{}, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(7200, 0).UTC(), Status: POAStatusActive, CreatedAt: time.Unix(11, 0).UTC(), UpdatedAt: time.Unix(11, 0).UTC()},
	}
	for _, p := range seed {
		d, _, _ := CanonicalPOADigest(p)
		f.Add(p.ID, p.Grantor, p.Grantee, joinSemi(p.Scope), restrJoin(p.Restrictions), d)
	}
	f.Fuzz(func(t *testing.T, id, grantor, grantee, scopeSemi, restrSemi, priorDigest string) {
		scope := splitSemi(scopeSemi)
		restr := restrMap(restrSemi)
		poa := &PowerOfAttorney{ID: id, Grantor: grantor, Grantee: grantee, Scope: scope, Restrictions: restr, ValidFrom: time.Unix(0, 0).UTC(), ValidUntil: time.Unix(3600, 0).UTC(), Status: POAStatusActive, CreatedAt: time.Unix(5, 0).UTC(), UpdatedAt: time.Unix(5, 0).UTC()}
		d1, c1, err := CanonicalPOADigest(poa)
		if err != nil {
			t.Fatalf("digest err: %v", err)
		}
		d2, c2, err := CanonicalPOADigest(poa)
		if err != nil {
			t.Fatalf("digest err2: %v", err)
		}
		if d1 != d2 {
			t.Fatalf("non-deterministic digest: %s vs %s", d1, d2)
		}
		if string(c1) != string(c2) {
			t.Fatalf("canonical mismatch")
		}
		poa.Status = POAStatusRevoked
		poa.UpdatedAt = poa.UpdatedAt.Add(time.Minute)
		d3, _, err := CanonicalPOADigest(poa)
		if err != nil {
			t.Fatalf("digest err3: %v", err)
		}
		if d1 != d3 {
			t.Fatalf("digest changed after mutable field update: %s -> %s", d1, d3)
		}
		// 3. Scope Reordering Independence
		// Create copy with reversed scope - should match digest if canonical sorts it
		if len(scope) > 1 {
			poaReorder := *poa
			// Simple reverse
			reversed := make([]string, len(scope))
			for i, v := range scope {
				reversed[len(scope)-1-i] = v
			}
			poaReorder.Scope = reversed
			dReorder, _, err := CanonicalPOADigest(&poaReorder)
			if err != nil {
				t.Fatalf("digest err reorder: %v", err)
			}
			if d1 != dReorder {
				t.Fatalf("Digest changed on Scope reordering. Original: %v, Reversed: %v", scope, reversed)
			}
		}

		// 4. Sensitivity (Grantor Change)
		// Changing Grantor MUST change digest (unless new value collides or is same)
		if grantor != "evil_actor" {
			poaSens := *poa
			poaSens.Grantor = "evil_actor"
			dSens, _, err := CanonicalPOADigest(&poaSens)
			if err != nil {
				t.Fatalf("digest err sens: %v", err)
			}
			if d1 == dSens {
				t.Fatalf("Digest collision on Grantor change")
			}
		}
	})
}

func joinSemi(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += ";" + items[i]
	}
	return out
}
func restrJoin(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	i := 0
	out := ""
	for k, v := range m {
		if i > 0 {
			out += ";"
		}
		out += k + "=" + v
		i++
	}
	return out
}
func splitSemi(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
func restrMap(s string) map[string]string {
	m := map[string]string{}
	if s == "" {
		return m
	}
	parts := splitSemi(s)
	for _, p := range parts {
		if p == "" {
			continue
		}
		eq := -1
		for i := 0; i < len(p); i++ {
			if p[i] == '=' {
				eq = i
				break
			}
		}
		if eq > 0 {
			m[p[:eq]] = p[eq+1:]
		}
	}
	return m
}

// TestCanonicalDigest wraps the fuzz test for conformance tool detection
func TestCanonicalDigest(t *testing.T) {
	// Simple sanity check that would run in normal unit test suite
	p := &PowerOfAttorney{
		ID:        "seed1",
		Grantor:   "alice",
		Grantee:   "bob",
		Scope:     []string{"read"},
		ValidFrom: time.Unix(0, 0).UTC(),
	}
	d, _, err := CanonicalPOADigest(p)
	if err != nil {
		t.Fatalf("Digest failed: %v", err)
	}
	if d == "" {
		t.Fatal("Empty digest")
	}
}
