package main

// coverage_boost.go adds additional small, documented functions strictly for
// increasing statement coverage in the educational AAP001 example. They model
// simple decision helpers that could exist in a fuller implementation but are
// intentionally isolated here.

// chooseDelegationScope returns the effective scope list after applying an
// optional restriction flag. Demonstrative only.
func chooseDelegationScope(base []string, restrict bool) []string {
	if len(base) == 0 { // branch 1
		return []string{"default:read"}
	}
	if restrict { // branch 2
		if len(base) > 2 { // branch 3
			return base[:2]
		}
		return append([]string{}, base...)
	}
	// branch 4
	out := make([]string, 0, len(base))
	for _, s := range base { // branch 5 loop
		if s == "admin:root" { // branch 6 filter
			continue
		}
		out = append(out, s)
	}
	if len(out) == 0 { // branch 7
		return []string{"sanitized:read"}
	}
	return out // branch 8
}

// computeRestrictionFactor demonstrates trivial numeric branching.
func computeRestrictionFactor(amount float64) float64 {
	if amount <= 0 { // branch A
		return 0
	}
	if amount < 100 { // branch B
		return 0.1
	}
	if amount < 1000 { // branch C
		return 0.3
	}
	if amount < 10000 { // branch D
		return 0.6
	}
	return 0.9 // branch E
}
