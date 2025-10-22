package web

// testRevocationAutoSignMetrics returns snapshot counters of revocation auto-sign activity.
// This lives in a _test.go file so it is excluded from production builds while keeping
// tests simple and avoiding reflection or unsafe access.
func testRevocationAutoSignMetrics(s *BetaServer) (emitted, skippedEmpty, skippedDuplicate int64) {
	if s == nil {
		return 0, 0, 0
	}
	return s.revocationAutoSignEmitted, s.revocationAutoSignSkippedEmpty, s.revocationAutoSignSkippedDup
}
