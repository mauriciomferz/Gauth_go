//go:build test

package gauth_aap_001

// TestInjectPOA allows tests in external packages to inject a PowerOfAttorney directly
// into the service repository bypassing authorization/policy checks. Not for production use.
// It is compiled only in test builds via the "test" build tag to avoid exposure in production binaries.
func (s *Service) TestInjectPOA(p *PowerOfAttorney) {
	if s == nil || p == nil {
		return
	}
	_ = s.repo.Create(p)
}
