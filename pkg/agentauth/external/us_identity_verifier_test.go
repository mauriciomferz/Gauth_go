package external

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Setup
// =============================================================================

func setupTestVerifier(t *testing.T) *USIdentityVerifier {
	t.Helper()

	mockProvider := NewMockUSIdentityProvider("test-provider")

	config := &USIdentityVerifierConfig{
		PrimaryProvider:   mockProvider,
		MaxRetries:        3,
		RetryDelay:        1 * time.Millisecond, // Short for testing
		BackoffMultiplier: 2.0,
		RequestTimeout:    5 * time.Second,
		StrictValidation:  true,
		CacheEnabled:      false, // Disable cache for most tests
		CacheTTL:          1 * time.Minute,
	}

	return NewUSIdentityVerifier(config)
}

// =============================================================================
// Passport Verification Tests
// =============================================================================

func TestVerifyPassport_Valid(t *testing.T) {
	verifier := setupTestVerifier(t)

	req := &PassportVerificationRequest{
		PassportNumber: "123456789",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Nationality:    "US",
		RequestID:      "test-001",
		Timestamp:      time.Now(),
	}

	result, err := verifier.VerifyPassport(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, result.Verified)
	assert.Equal(t, DocumentTypePassport, result.DocumentType)
	assert.GreaterOrEqual(t, result.ConfidenceScore, 0.8)
	assert.NotNil(t, result.Checks)
	assert.Equal(t, CheckStatusPassed, result.Checks.DocumentAuthenticity.Status)
	assert.GreaterOrEqual(t, result.ProcessingTimeMs, int64(0) // Can be 0 in tests
}

func TestVerifyPassport_InvalidFormat(t *testing.T) {
	verifier := setupTestVerifier(t)

	tests := []struct {
		name           string
		passportNumber string
		expectedError  string
	}{
		{
			name:           "Too Short",
			passportNumber: "12345678",
			expectedError:  "passport number must be 9 digits",
		},
		{
			name:           "Too Long",
			passportNumber: "1234567890",
			expectedError:  "passport number must be 9 digits",
		},
		{
			name:           "Contains Letters",
			passportNumber: "12345678A",
			expectedError:  "passport number must be 9 digits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &PassportVerificationRequest{
				PassportNumber: tt.passportNumber,
				FirstName:      "John",
				LastName:       "Doe",
				DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				ExpirationDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
				Nationality:    "US",
				RequestID:      "test-002",
				Timestamp:      time.Now(),
			}

			result, err := verifier.VerifyPassport(context.Background(), req)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestVerifyPassport_Expired(t *testing.T) {
	verifier := setupTestVerifier(t)

	req := &PassportVerificationRequest{
		PassportNumber: "123456789",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), // Expired
		Nationality:    "US",
		RequestID:      "test-003",
		Timestamp:      time.Now(),
	}

	result, err := verifier.VerifyPassport(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), ErrDocumentExpired)
}

// =============================================================================
// Driver's License Verification Tests
// =============================================================================

func TestVerifyDriverLicense_StateVariations(t *testing.T) {
	verifier := setupTestVerifier(t)

	tests := []struct {
		name          string
		state         string
		licenseNumber string
		valid         bool
	}{
		// California
		{"CA Valid", "CA", "A1234567", true},
		{"CA Invalid - No Letter", "CA", "1234567", false},
		{"CA Invalid - Too Short", "CA", "A123456", false},

		// Texas (7-8 digits)
		{"TX Valid - 8 digits", "TX", "12345678", true},
		{"TX Valid - 7 digits", "TX", "1234567", true},
		{"TX Invalid - Too Short", "TX", "123456", false},
		{"TX Invalid - Contains Letter", "TX", "A1234567", false},

		// Florida
		{"FL Valid", "FL", "A123456789012", true},
		{"FL Invalid - No Letter", "FL", "123456789012", false},

		// New York (multiple formats)
		{"NY Valid - 9 digits", "NY", "123456789", true},
		{"NY Valid - 16 digits", "NY", "1234567890123456", true},
		{"NY Invalid - 10 digits", "NY", "1234567890", false},

		// Pennsylvania
		{"PA Valid", "PA", "12345678", true},
		{"PA Invalid - Too Short", "PA", "1234567", false},

		// Illinois
		{"IL Valid", "IL", "A12345678901", true},
		{"IL Invalid - No Letter", "IL", "12345678901", false},

		// Ohio
		{"OH Valid", "OH", "AB123456", true},
		{"OH Invalid - 3 Letters", "OH", "ABC123456", false},

		// Georgia (7-9 digits)
		{"GA Valid - 9 digits", "GA", "123456789", true},
		{"GA Valid - 7 digits", "GA", "1234567", true},
		{"GA Invalid - Too Short", "GA", "123456", false},

		// Additional states
		{"AZ Valid - Format 1", "AZ", "A12345678", true},
		{"AZ Valid - Format 2", "AZ", "AB12345", true},
		{"MI Valid", "MI", "A123456789012", true},
		{"WA Valid", "WA", "ABC1234DEF56", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &DLVerificationRequest{
				LicenseNumber:  tt.licenseNumber,
				State:          tt.state,
				FirstName:      "John",
				LastName:       "Doe",
				DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				ExpirationDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				RequestID:      "test-dl-" + tt.state,
				Timestamp:      time.Now(),
			}

			result, err := verifier.VerifyDriverLicense(context.Background(), req)

			if tt.valid {
				assert.NoError(t, err, "Expected valid license for %s", tt.name)
				assert.NotNil(t, result)
				assert.True(t, result.Verified)
				assert.Equal(t, tt.state, result.DocumentState)
			} else {
				assert.Error(t, err, "Expected invalid license for %s", tt.name)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), ErrDocumentFormatInvalid)
			}
		})
	}
}

func TestVerifyDriverLicense_EnhancedVerificationWarning(t *testing.T) {
	verifier := setupTestVerifier(t)

	// California supports enhanced verification
	req := &DLVerificationRequest{
		LicenseNumber:  "A1234567",
		State:          "CA",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		RequestID:      "test-004",
		Timestamp:      time.Now(),
		// No image provided
	}

	result, err := verifier.VerifyDriverLicense(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Warnings[0], "enhanced verification")
}

// =============================================================================
// SSN Validation Tests
// =============================================================================

func TestValidateSSN_Format(t *testing.T) {
	verifier := setupTestVerifier(t)

	tests := []struct {
		name          string
		ssn           string
		expectedValid bool
		expectedError string
	}{
		{"Valid SSN", "123456789", true, ""},
		{"Too Short", "12345678", false, "len"},          // Validator error
		{"Too Long", "1234567890", false, "len"},         // Validator error
		{"Contains Dashes", "123-45-6789", false, "len"}, // Validator error
		{"Contains Letters", "12345678A", false, "SSN must contain only digits"},
		{"All Zeros Area", "000123456", false, "SSN cannot contain all zeros"},
		{"All Zeros Group", "123001234", false, "SSN cannot contain all zeros"},
		{"All Zeros Serial", "123450000", false, "SSN cannot contain all zeros"},
		{"Area 666", "666123456", false, "SSN area number 666 is never issued"},
		{"Area 900", "900123456", false, "SSN area numbers 900-999 are not valid"},
		{"Area 950", "950123456", false, "SSN area numbers 900-999 are not valid"},
		{"Area 999", "999123456", false, "SSN area numbers 900-999 are not valid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &SSNValidationRequest{
				SSN:             tt.ssn,
				FirstName:       "John",
				LastName:        "Doe",
				DateOfBirth:     time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				ValidationLevel: SSNValidationLevelBasic,
				RequestID:       "test-ssn-" + tt.name,
				Timestamp:       time.Now(),
			}

			result, err := verifier.ValidateSSN(context.Background(), req)

			if tt.expectedValid {
				require.NoError(t, err)
				assert.True(t, result.Valid)
				assert.True(t, result.FormatValid)
				// SSN should be masked in response
				assert.Contains(t, result.SSN, "XXX-XX-")
			} else {
				// Invalid SSN may return error from validator or result with errors
				if err != nil {
					assert.Error(t, err)
					if tt.expectedError != "" {
						assert.Contains(t, err.Error(), tt.expectedError)
					}
				} else {
					require.NotNil(t, result)
					assert.False(t, result.Valid)
					assert.False(t, result.FormatValid)
					assert.NotEmpty(t, result.Errors)
					if tt.expectedError != "" {
						assert.Contains(t, result.Errors[0].Message, tt.expectedError)
					}
				}
			}
		})
	}
}

func TestValidateSSN_Masking(t *testing.T) {
	tests := []struct {
		name     string
		ssn      string
		expected string
	}{
		{"Valid SSN", "123456789", "XXX-XX-6789"},
		{"Invalid Length", "12345", "XXX-XX-XXXX"},
		{"Empty", "", "XXX-XX-XXXX"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked := maskSSN(tt.ssn)
			assert.Equal(t, tt.expected, masked)
		})
	}
}

// =============================================================================
// State ID Verification Tests
// =============================================================================

func TestVerifyStateID_Valid(t *testing.T) {
	verifier := setupTestVerifier(t)

	req := &StateIDVerificationRequest{
		IDNumber:       "A1234567",
		State:          "CA",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		RequestID:      "test-005",
		Timestamp:      time.Now(),
	}

	result, err := verifier.VerifyStateID(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, result.Verified)
	assert.Equal(t, "CA", result.DocumentState)
	assert.NotNil(t, result.Checks)
}

// =============================================================================
// Caching Tests
// =============================================================================

func TestCaching_VerifyPassport(t *testing.T) {
	mockProvider := NewMockUSIdentityProvider("test-provider")

	config := &USIdentityVerifierConfig{
		PrimaryProvider:   mockProvider,
		MaxRetries:        3,
		RetryDelay:        1 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RequestTimeout:    5 * time.Second,
		StrictValidation:  true,
		CacheEnabled:      true,
		CacheTTL:          1 * time.Minute,
	}

	verifier := NewUSIdentityVerifier(config)

	req := &PassportVerificationRequest{
		PassportNumber: "123456789",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Nationality:    "US",
		RequestID:      "test-006",
		Timestamp:      time.Now(),
	}

	// First call - should hit provider
	result1, err := verifier.VerifyPassport(context.Background(), req)
	require.NoError(t, err)
	assert.NotContains(t, result1.Warnings, "cache")

	// Second call - should hit cache
	result2, err := verifier.VerifyPassport(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, result2.Warnings, "Result retrieved from cache")

	// Results should be identical (except warnings)
	assert.Equal(t, result1.Verified, result2.Verified)
	assert.Equal(t, result1.ConfidenceScore, result2.ConfidenceScore)
}

func TestCaching_Expiration(t *testing.T) {
	mockProvider := NewMockUSIdentityProvider("test-provider")

	config := &USIdentityVerifierConfig{
		PrimaryProvider:   mockProvider,
		MaxRetries:        3,
		RetryDelay:        1 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RequestTimeout:    5 * time.Second,
		StrictValidation:  true,
		CacheEnabled:      true,
		CacheTTL:          100 * time.Millisecond, // Very short TTL for testing
	}

	verifier := NewUSIdentityVerifier(config)

	req := &PassportVerificationRequest{
		PassportNumber: "123456789",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Nationality:    "US",
		RequestID:      "test-007",
		Timestamp:      time.Now(),
	}

	// First call - should hit provider
	result1, err := verifier.VerifyPassport(context.Background(), req)
	require.NoError(t, err)
	assert.NotContains(t, result1.Warnings, "cache")

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Second call - cache expired, should hit provider again
	result2, err := verifier.VerifyPassport(context.Background(), req)
	require.NoError(t, err)
	assert.NotContains(t, result2.Warnings, "Result retrieved from cache")
}

// =============================================================================
// Circuit Breaker Tests
// =============================================================================

func TestCircuitBreaker_FailureHandling(t *testing.T) {
	// Create a mock provider that always fails
	failingProvider := &failingMockProvider{name: "failing-provider"}

	circuitBreaker := NewCircuitBreaker(2, 5*time.Second, 100*time.Millisecond)

	config := &USIdentityVerifierConfig{
		PrimaryProvider:  failingProvider,
		MaxRetries:       0, // No retries to speed up test
		CircuitBreaker:   circuitBreaker,
		StrictValidation: false, // Don't fail on format validation
	}

	verifier := NewUSIdentityVerifier(config)

	req := &PassportVerificationRequest{
		PassportNumber: "123456789",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Nationality:    "US",
		RequestID:      "test-008",
		Timestamp:      time.Now(),
	}

	// First two calls should fail and open the circuit breaker
	for i := 0; i < 2; i++ {
		_, err := verifier.VerifyPassport(context.Background(), req)
		assert.Error(t, err)
	}

	// Circuit breaker should now be open
	_, err := verifier.VerifyPassport(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit breaker should be in half-open state, but will fail again
	_, err = verifier.VerifyPassport(context.Background(), req)
	assert.Error(t, err)
}

// =============================================================================
// Retry Logic Tests
// =============================================================================

func TestRetryLogic_TransientErrors(t *testing.T) {
	// Create a provider that fails first 2 times, then succeeds
	intermittentProvider := &intermittentMockProvider{
		name:         "intermittent-provider",
		failuresLeft: 2,
	}

	config := &USIdentityVerifierConfig{
		PrimaryProvider:   intermittentProvider,
		MaxRetries:        3,
		RetryDelay:        1 * time.Millisecond,
		BackoffMultiplier: 1.5,
		RequestTimeout:    5 * time.Second,
		StrictValidation:  false,
	}

	verifier := NewUSIdentityVerifier(config)

	req := &PassportVerificationRequest{
		PassportNumber: "123456789",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Nationality:    "US",
		RequestID:      "test-009",
		Timestamp:      time.Now(),
	}

	// Should succeed after 2 retries
	result, err := verifier.VerifyPassport(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.Verified)
	assert.Equal(t, 0, intermittentProvider.failuresLeft)
}

// =============================================================================
// Fallback Provider Tests
// =============================================================================

func TestFallbackProvider_Success(t *testing.T) {
	// Primary provider always fails
	failingProvider := &failingMockProvider{name: "primary-failing"}

	// Fallback provider always succeeds
	fallbackProvider := NewMockUSIdentityProvider("fallback-success")

	config := &USIdentityVerifierConfig{
		PrimaryProvider:  failingProvider,
		FallbackProvider: fallbackProvider,
		MaxRetries:       0,
		StrictValidation: false,
	}

	verifier := NewUSIdentityVerifier(config)

	req := &PassportVerificationRequest{
		PassportNumber: "123456789",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Nationality:    "US",
		RequestID:      "test-010",
		Timestamp:      time.Now(),
	}

	// Should succeed using fallback provider
	result, err := verifier.VerifyPassport(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.Verified)
	assert.Contains(t, result.Warnings[0], "fallback provider")
	assert.Equal(t, "fallback-success", result.ProviderName)
}

// =============================================================================
// Test Helper Providers
// =============================================================================

// failingMockProvider always returns errors
type failingMockProvider struct {
	name string
}

func (f *failingMockProvider) VerifyDocument(ctx context.Context, req interface{}) (*IdentityVerificationResult, error) {
	return nil, errors.New("provider unavailable")
}

func (f *failingMockProvider) ValidateSSN(ctx context.Context, req *SSNValidationRequest) (*SSNValidationResult, error) {
	return nil, errors.New("provider unavailable")
}

func (f *failingMockProvider) GetProviderName() string {
	return f.name
}

func (f *failingMockProvider) GetSupportedDocumentTypes() []DocumentType {
	return []DocumentType{DocumentTypePassport}
}

// intermittentMockProvider fails N times then succeeds
type intermittentMockProvider struct {
	name         string
	failuresLeft int
}

func (i *intermittentMockProvider) VerifyDocument(ctx context.Context, req interface{}) (*IdentityVerificationResult, error) {
	if i.failuresLeft > 0 {
		i.failuresLeft--
		return nil, errors.New("transient error")
	}

	return &IdentityVerificationResult{
		Verified:          true,
		VerificationLevel: VerificationLevelStandard,
		ConfidenceScore:   0.95,
		DocumentType:      DocumentTypePassport,
		ProviderName:      i.name,
		Checks: &VerificationChecks{
			DocumentAuthenticity: &CheckResult{Status: CheckStatusPassed, Score: 1.0},
		},
		VerificationTimestamp: time.Now(),
	}, nil
}

func (i *intermittentMockProvider) ValidateSSN(ctx context.Context, req *SSNValidationRequest) (*SSNValidationResult, error) {
	if i.failuresLeft > 0 {
		i.failuresLeft--
		return nil, errors.New("transient error")
	}

	return &SSNValidationResult{
		Valid:               true,
		ValidationLevel:     req.ValidationLevel,
		ConfidenceScore:     0.95,
		FormatValid:         true,
		NotDeceased:         true,
		ProviderName:        i.name,
		ValidationTimestamp: time.Now(),
	}, nil
}

func (i *intermittentMockProvider) GetProviderName() string {
	return i.name
}

func (i *intermittentMockProvider) GetSupportedDocumentTypes() []DocumentType {
	return []DocumentType{DocumentTypePassport}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkVerifyPassport(b *testing.B) {
	mockProvider := NewMockUSIdentityProvider("test-provider")
	config := &USIdentityVerifierConfig{
		PrimaryProvider:   mockProvider,
		MaxRetries:        3,
		RetryDelay:        1 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RequestTimeout:    5 * time.Second,
		StrictValidation:  false,
	}
	verifier := NewUSIdentityVerifier(config)

	req := &PassportVerificationRequest{
		PassportNumber: "123456789",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Nationality:    "US",
		RequestID:      "bench-001",
		Timestamp:      time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.VerifyPassport(context.Background(), req)
	}
}

func BenchmarkVerifyDriverLicense(b *testing.B) {
	mockProvider := NewMockUSIdentityProvider("test-provider")
	config := &USIdentityVerifierConfig{
		PrimaryProvider:   mockProvider,
		MaxRetries:        3,
		RetryDelay:        1 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RequestTimeout:    5 * time.Second,
		StrictValidation:  false,
	}
	verifier := NewUSIdentityVerifier(config)

	req := &DLVerificationRequest{
		LicenseNumber:  "A1234567",
		State:          "CA",
		FirstName:      "John",
		LastName:       "Doe",
		DateOfBirth:    time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpirationDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		RequestID:      "bench-002",
		Timestamp:      time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.VerifyDriverLicense(context.Background(), req)
	}
}

func BenchmarkValidateSSN(b *testing.B) {
	mockProvider := NewMockUSIdentityProvider("test-provider")
	config := &USIdentityVerifierConfig{
		PrimaryProvider:   mockProvider,
		MaxRetries:        3,
		RetryDelay:        1 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RequestTimeout:    5 * time.Second,
		StrictValidation:  false,
	}
	verifier := NewUSIdentityVerifier(config)

	req := &SSNValidationRequest{
		SSN:             "123456789",
		FirstName:       "John",
		LastName:        "Doe",
		DateOfBirth:     time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		ValidationLevel: SSNValidationLevelBasic,
		RequestID:       "bench-003",
		Timestamp:       time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.ValidateSSN(context.Background(), req)
	}
}
