package agentauth_aap_001

import (
	"testing"
	"time"
)

// TestCanonicalDigestPermutations strengthens AAP002 §9 evidence: digest MUST be stable
// under field reordering (scope, restrictions, weights) and control characters are escaped.
func TestCanonicalDigestPermutations(t *testing.T) {
	base := &PowerOfAttorney{
		ID:           "poa-1",
		Version:      1,
		Grantor:      "alice",
		Grantee:      "bob",
		Scope:        []string{"write", "read"}, // unsorted intentionally
		Restrictions: map[string]string{"currency": "USD", "env": "prod"},
		ValidFrom:    time.Unix(1000, 0).UTC(),
		ValidUntil:   time.Unix(1000, 0).UTC().Add(time.Hour),
		CreatedAt:    time.Unix(1000, 0).UTC(),
		Threshold:    1,
	}
	dig1, canon1, err := CanonicalPOADigest(base)
	if err != nil {
		t.Fatalf("digest err: %v", err)
	}
	perm := *base
	perm.Scope = []string{"read", "write"}
	perm.Restrictions = map[string]string{"env": "prod", "currency": "USD"}
	dig2, canon2, err2 := CanonicalPOADigest(&perm)
	if err2 != nil {
		t.Fatalf("digest perm err: %v", err2)
	}
	if dig1 != dig2 {
		t.Fatalf("expected stable digest across permutations got %s != %s\ncanon1=%s\ncanon2=%s", dig1, dig2, string(canon1), string(canon2))
	}
	// Control characters cause digest change & are escaped.
	ctrl := *base
	ctrl.Scope = []string{"read\n", "write\t"}
	dig3, canon3, err3 := CanonicalPOADigest(&ctrl)
	if err3 != nil {
		t.Fatalf("digest ctrl err: %v", err3)
	}
	if dig3 == dig1 {
		t.Fatalf("expected different digest when scope includes control chars")
	}
	if string(canon3) == string(canon1) {
		t.Fatalf("expected canonical JSON to differ with control chars")
	}
	if !containsEscapes(string(canon3)) {
		t.Fatalf("expected control character escapes in canonical JSON: %s", string(canon3))
	}
}

func containsEscapes(s string) bool { return containsAny(s, []string{"\\n", "\\t"}) }
func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && stringContains(s, sub) {
			return true
		}
	}
	return false
}
func stringContains(s, sub string) bool {
	return len(sub) <= len(s) && (len(sub) == 0 || indexOfStr(s, sub) >= 0)
}
func indexOfStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
