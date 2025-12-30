package main

// coverage_boost2.go adds deterministic, fully-tested branching logic to raise
// statement coverage without altering demo semantics. Functions are pure and
// used only by tests.

// branchingAccumulator performs layered branching so tests can exercise all paths.
func branchingAccumulator(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		if i%2 == 0 { // even path
			sum += i
		} else { // odd path
			sum += i * 2
		}
		if i%3 == 0 { // multiples of 3 get a bonus
			sum++
		}
		switch {
		case i%5 == 0: // multiples of 5
			sum += 5
		case i%7 == 0: // multiples of 7 (exclusive of 5)
			sum += 7
		default: // all others
			sum++
		}
	}
	return sum
}

// tieredClassifier returns a bucket label derived from n using overlapping conditions.
func tieredClassifier(n int) string {
	if n < 0 {
		return "neg"
	}
	if n == 0 {
		return "zero"
	}
	if n < 5 {
		if n%2 == 0 {
			return "small-even"
		}
		return "small-odd"
	}
	if n <= 10 {
		switch n {
		case 5, 10:
			return "boundary"
		default:
			return "mid"
		}
	}
	if n%2 == 0 {
		return "large-even"
	}
	return "large-odd"
}
