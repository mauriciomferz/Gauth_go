// Package agentauth - Formal Requirements Service Benchmarks
// Task 8: Performance testing and optimization for formal requirements validation
package agentauth

import (
	"testing"
	"time"
)

// BenchmarkJurisdictionLookup measures jurisdiction requirement lookup performance
func BenchmarkJurisdictionLookup(b *testing.B) {
	// Create service with mock validator
	config := FormalRequirementsServiceConfig{
		EnableJurisdictionChecks: true,
		CacheDuration:            5 * time.Minute,
	}
	validator := NewFormalRequirementsValidator(nil, nil, nil, false)
	service := NewFormalRequirementsService(validator, config)

	jurisdictions := []string{"DE", "US", "UK", "FR", "IT", "ES", "PT", "NL", "BE"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		jurisdiction := jurisdictions[i%len(jurisdictions)]
		service.mu.RLock()
		_ = service.jurisdictionReqs[jurisdiction]
		service.mu.RUnlock()
	}
}

// BenchmarkEU27JurisdictionCoverage measures map access for all EU-27 jurisdictions
func BenchmarkEU27JurisdictionCoverage(b *testing.B) {
	config := FormalRequirementsServiceConfig{
		EnableJurisdictionChecks: true,
		CacheDuration:            5 * time.Minute,
	}
	validator := NewFormalRequirementsValidator(nil, nil, nil, false)
	service := NewFormalRequirementsService(validator, config)

	eu27 := []string{
		"DE", "FR", "IT", "ES", "PT", "NL", "BE", "LU", "AT", "IE",
		"FI", "SE", "DK", "EE", "LV", "LT", "PL", "CZ", "SK", "HU",
		"RO", "BG", "HR", "SI", "GR", "CY", "MT",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		jurisdiction := eu27[i%len(eu27)]
		service.mu.RLock()
		_ = service.jurisdictionReqs[jurisdiction]
		service.mu.RUnlock()
	}
}

// BenchmarkJurisdictionMapAccessSingleKey measures single map access performance
func BenchmarkJurisdictionMapAccessSingleKey(b *testing.B) {
	config := FormalRequirementsServiceConfig{
		EnableJurisdictionChecks: true,
		CacheDuration:            5 * time.Minute,
	}
	validator := NewFormalRequirementsValidator(nil, nil, nil, false)
	service := NewFormalRequirementsService(validator, config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service.mu.RLock()
		_ = service.jurisdictionReqs["DE"]
		service.mu.RUnlock()
	}
}

// BenchmarkJurisdictionMapAccessWithoutLock measures map access without lock (baseline)
func BenchmarkJurisdictionMapAccessWithoutLock(b *testing.B) {
	config := FormalRequirementsServiceConfig{
		EnableJurisdictionChecks: true,
		CacheDuration:            5 * time.Minute,
	}
	validator := NewFormalRequirementsValidator(nil, nil, nil, false)
	service := NewFormalRequirementsService(validator, config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = service.jurisdictionReqs["DE"]
	}
}

// BenchmarkServiceInitialization measures service initialization performance
func BenchmarkServiceInitialization(b *testing.B) {
	config := FormalRequirementsServiceConfig{
		EnableJurisdictionChecks: true,
		EnableDocumentChecks:     true,
		EnableNotaryVerification: true,
		EnableLegalCompliance:    true,
		CacheDuration:            5 * time.Minute,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		validator := NewFormalRequirementsValidator(nil, nil, nil, false)
		_ = NewFormalRequirementsService(validator, config)
	}
}

// BenchmarkServiceStatistics measures statistics tracking overhead
func BenchmarkServiceStatistics(b *testing.B) {
	config := FormalRequirementsServiceConfig{
		EnableJurisdictionChecks: true,
		CacheDuration:            5 * time.Minute,
	}
	validator := NewFormalRequirementsValidator(nil, nil, nil, false)
	service := NewFormalRequirementsService(validator, config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service.mu.Lock()
		service.validationAttempts++
		service.validationSuccesses++
		service.jurisdictionChecks["DE"]++
		service.mu.Unlock()
	}
}

// BenchmarkCacheClearing measures cache clearing performance
func BenchmarkCacheClearing(b *testing.B) {
	config := FormalRequirementsServiceConfig{
		EnableJurisdictionChecks: true,
		EnableDocumentChecks:     true,
		CacheDuration:            5 * time.Minute,
	}
	validator := NewFormalRequirementsValidator(nil, nil, nil, false)
	service := NewFormalRequirementsService(validator, config)

	// Populate caches
	for i := 0; i < 100; i++ {
		key := string(rune('A' + i%26))
		service.documentReqsCache[key] = &DocumentRequirementCheck{
			RequirementType: "test",
			Satisfied:       true,
		}
		service.complianceCheckCache[key] = &LegalComplianceCheck{
			Framework: "test",
			Compliant: true,
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service.mu.Lock()
		service.documentReqsCache = make(map[string]*DocumentRequirementCheck)
		service.complianceCheckCache = make(map[string]*LegalComplianceCheck)
		service.lastCacheClear = time.Now()
		service.mu.Unlock()
	}
}
