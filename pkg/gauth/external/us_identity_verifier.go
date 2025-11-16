// Package external provides external service integrations for identity verification,
// policy retrieval, and trust service provider interactions.
//
// This file implements US-specific identity verification supporting:
// - US Passport verification
// - State driver's license verification (all 50 states + DC)
// - Social Security Number (SSN) validation
// - State-issued ID card verification
package external

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// =============================================================================
// US Identity Verification Enums and Constants
// =============================================================================

// DocumentType represents the type of identity document being verified
type DocumentType string

const (
	DocumentTypePassport      DocumentType = "passport"
	DocumentTypeDriverLicense DocumentType = "driver_license"
	DocumentTypeStateID       DocumentType = "state_id"
)

// VerificationLevel represents the depth of identity verification performed
type VerificationLevel string

const (
	VerificationLevelBasic    VerificationLevel = "basic"    // Format validation only
	VerificationLevelStandard VerificationLevel = "standard" // API verification
	VerificationLevelEnhanced VerificationLevel = "enhanced" // API + biometric
)

// SSNValidationLevel represents the level of SSN validation
type SSNValidationLevel string

const (
	SSNValidationLevelBasic        SSNValidationLevel = "basic"        // Format only
	SSNValidationLevelStandard     SSNValidationLevel = "standard"     // SSN validity
	SSNValidationLevelComprehensive SSNValidationLevel = "comprehensive" // Full match
)

// CheckStatus represents the status of an individual verification check
type CheckStatus string

const (
	CheckStatusPassed       CheckStatus = "passed"
	CheckStatusFailed       CheckStatus = "failed"
	CheckStatusWarning      CheckStatus = "warning"
	CheckStatusNotPerformed CheckStatus = "not_performed"
)

// Error codes for US identity verification
const (
	// Document errors
	ErrDocumentExpired       = "DOCUMENT_EXPIRED"
	ErrDocumentInvalid       = "DOCUMENT_INVALID"
	ErrDocumentNotFound      = "DOCUMENT_NOT_FOUND"
	ErrDocumentFormatInvalid = "DOCUMENT_FORMAT_INVALID"

	// Identity mismatch errors
	ErrNameMismatch    = "NAME_MISMATCH"
	ErrDOBMismatch     = "DOB_MISMATCH"
	ErrAddressMismatch = "ADDRESS_MISMATCH"

	// API provider errors
	ErrProviderUnavailable = "PROVIDER_UNAVAILABLE"
	ErrProviderTimeout     = "PROVIDER_TIMEOUT"
	ErrProviderAuthFailure = "PROVIDER_AUTH_FAILURE"
	ErrProviderRateLimited = "PROVIDER_RATE_LIMITED"

	// SSN-specific errors
	ErrSSNInvalid  = "SSN_INVALID"
	ErrSSNDeceased = "SSN_DECEASED"

	// System errors
	ErrCodeCircuitBreakerOpen = "CIRCUIT_BREAKER_OPEN"
	ErrMaxRetriesExceeded     = "MAX_RETRIES_EXCEEDED"
)

// State driver's license number formats (regex patterns)
var stateDLFormats = map[string]*regexp.Regexp{
	"AL": regexp.MustCompile(`^\d{1,8}$`),                         // 1-8 digits
	"AK": regexp.MustCompile(`^\d{1,7}$`),                         // 1-7 digits
	"AZ": regexp.MustCompile(`^([A-Z]\d{8}|[A-Z]{2}\d{2,5})$`),    // A12345678 OR AB12345
	"AR": regexp.MustCompile(`^\d{4,9}$`),                         // 4-9 digits
	"CA": regexp.MustCompile(`^[A-Z]\d{7}$`),                      // A1234567
	"CO": regexp.MustCompile(`^(\d{9}|[A-Z]{1,2}\d{3,6})$`),       // 123456789 OR AB12345
	"CT": regexp.MustCompile(`^\d{9}$`),                           // 9 digits
	"DE": regexp.MustCompile(`^\d{1,7}$`),                         // 1-7 digits
	"DC": regexp.MustCompile(`^(\d{7}|\d{9})$`),                   // 7 or 9 digits
	"FL": regexp.MustCompile(`^[A-Z]\d{12}$`),                     // A123456789012
	"GA": regexp.MustCompile(`^\d{7,9}$`),                         // 7-9 digits
	"HI": regexp.MustCompile(`^([A-Z]\d{8}|\d{9})$`),              // H12345678 OR 123456789
	"ID": regexp.MustCompile(`^([A-Z]{2}\d{6}[A-Z]|\d{9})$`),      // AB123456C OR 123456789
	"IL": regexp.MustCompile(`^[A-Z]\d{11}$`),                     // A12345678901
	"IN": regexp.MustCompile(`^([A-Z]\d{9}|\d{10})$`),             // A123456789 OR 1234567890
	"IA": regexp.MustCompile(`^(\d{9}|\d{3}[A-Z]{2}\d{4})$`),      // 123456789 OR 123AB4567
	"KS": regexp.MustCompile(`^([A-Z]\d{8}|\d{9})$`),              // K12345678 OR 123456789
	"KY": regexp.MustCompile(`^([A-Z]\d{8}|\d{9})$`),              // A12345678 OR 123456789
	"LA": regexp.MustCompile(`^\d{1,9}$`),                         // 1-9 digits
	"ME": regexp.MustCompile(`^\d{7}X?$`),                         // 7 digits + optional X
	"MD": regexp.MustCompile(`^[A-Z]\d{12}$`),                     // A123456789012
	"MA": regexp.MustCompile(`^([A-Z]\d{8}|\d{9})$`),              // S12345678 OR 123456789
	"MI": regexp.MustCompile(`^[A-Z]\d{12}$`),                     // A123456789012
	"MN": regexp.MustCompile(`^[A-Z]\d{12}$`),                     // A123456789012
	"MS": regexp.MustCompile(`^\d{9}$`),                           // 9 digits
	"MO": regexp.MustCompile(`^([A-Z]\d{5,9}|\d{9})$`),            // A123456789 OR 123456789
	"MT": regexp.MustCompile(`^(\d{9}|\d{13})$`),                  // 9 or 13 digits
	"NE": regexp.MustCompile(`^[A-Z]\d{3,8}$`),                    // A12345678
	"NV": regexp.MustCompile(`^(\d{9,10}|X\d{8})$`),               // 123456789 OR X12345678
	"NH": regexp.MustCompile(`^[A-Z]{3}\d{5}$`),                   // ABC12345
	"NJ": regexp.MustCompile(`^[A-Z]\d{14}$`),                     // A12345678901234
	"NM": regexp.MustCompile(`^\d{8,9}$`),                         // 8-9 digits
	"NY": regexp.MustCompile(`^([A-Z]\d{7}|[A-Z]\d{18}|\d{8,9}|\d{16})$`), // Multiple formats
	"NC": regexp.MustCompile(`^\d{1,12}$`),                        // 1-12 digits
	"ND": regexp.MustCompile(`^([A-Z]{3}\d{6}|\d{9})$`),           // ABC123456 OR 123456789
	"OH": regexp.MustCompile(`^([A-Z]{2}\d{6}|\d{8})$`),           // AB123456 OR 12345678
	"OK": regexp.MustCompile(`^([A-Z]\d{9}|\d{9})$`),              // A123456789 OR 123456789
	"OR": regexp.MustCompile(`^\d{1,9}$`),                         // 1-9 digits
	"PA": regexp.MustCompile(`^\d{8}$`),                           // 8 digits
	"RI": regexp.MustCompile(`^(\d{7}|[A-Z]\d{6})$`),              // 1234567 OR A123456
	"SC": regexp.MustCompile(`^\d{5,11}$`),                        // 5-11 digits
	"SD": regexp.MustCompile(`^\d{6,10}$`),                        // 6-10 digits
	"TN": regexp.MustCompile(`^\d{7,9}$`),                         // 7-9 digits
	"TX": regexp.MustCompile(`^\d{7,8}$`),                         // 7-8 digits
	"UT": regexp.MustCompile(`^\d{4,10}$`),                        // 4-10 digits
	"VT": regexp.MustCompile(`^\d{8}|\d{7}A$`),                    // 8 digits OR 7 digits + A
	"VA": regexp.MustCompile(`^([A-Z]\d{8}|\d{9})$`),              // A12345678 OR 123456789
	"WA": regexp.MustCompile(`^[A-Z0-9]{12}$`),                    // 12 alphanumeric
	"WV": regexp.MustCompile(`^[A-Z]{1,2}\d{5,6}$`),               // AB12345
	"WI": regexp.MustCompile(`^[A-Z]\d{13}$`),                     // A1234567890123
	"WY": regexp.MustCompile(`^\d{9,10}$`),                        // 9-10 digits
}

// States with enhanced verification capabilities (REAL ID Act compliant)
var enhancedVerificationStates = map[string]bool{
	"CA": true, "TX": true, "FL": true, "NY": true,
	"WA": true, "MI": true, "VT": true, "MN": true,
}

// =============================================================================
// Request Models
// =============================================================================

// Address represents a US mailing address
type Address struct {
	StreetAddress1 string `json:"street_address_1" validate:"required"`
	StreetAddress2 string `json:"street_address_2,omitempty"`
	City           string `json:"city" validate:"required"`
	State          string `json:"state" validate:"required,len=2"`
	ZipCode        string `json:"zip_code" validate:"required"`
	Country        string `json:"country" validate:"required"`
}

// PassportVerificationRequest represents a US passport verification request
type PassportVerificationRequest struct {
	PassportNumber string    `json:"passport_number" validate:"required"`
	FirstName      string    `json:"first_name" validate:"required"`
	LastName       string    `json:"last_name" validate:"required"`
	DateOfBirth    time.Time `json:"date_of_birth" validate:"required"`
	IssueDate      time.Time `json:"issue_date" validate:"required"`
	ExpirationDate time.Time `json:"expiration_date" validate:"required"`
	Nationality    string    `json:"nationality" validate:"required,eq=US"`

	// Optional fields for enhanced verification
	PlaceOfBirth       string `json:"place_of_birth,omitempty"`
	PassportImageFront []byte `json:"passport_image_front,omitempty"`
	PassportImageBack  []byte `json:"passport_image_back,omitempty"`
	FaceImage          []byte `json:"face_image,omitempty"`

	// Metadata
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

// DLVerificationRequest represents a driver's license verification request
type DLVerificationRequest struct {
	LicenseNumber  string    `json:"license_number" validate:"required"`
	State          string    `json:"state" validate:"required,len=2"`
	FirstName      string    `json:"first_name" validate:"required"`
	LastName       string    `json:"last_name" validate:"required"`
	DateOfBirth    time.Time `json:"date_of_birth" validate:"required"`
	IssueDate      time.Time `json:"issue_date" validate:"required"`
	ExpirationDate time.Time `json:"expiration_date" validate:"required"`

	// Optional fields
	Address           *Address `json:"address,omitempty"`
	LicenseClass      string   `json:"license_class,omitempty"` // e.g., "C", "M"
	Endorsements      []string `json:"endorsements,omitempty"`
	Restrictions      []string `json:"restrictions,omitempty"`
	LicenseImageFront []byte   `json:"license_image_front,omitempty"`
	LicenseImageBack  []byte   `json:"license_image_back,omitempty"`

	// Metadata
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

// SSNValidationRequest represents an SSN validation request
type SSNValidationRequest struct {
	SSN         string    `json:"ssn" validate:"required,len=9"` // 9 digits without dashes
	FirstName   string    `json:"first_name" validate:"required"`
	LastName    string    `json:"last_name" validate:"required"`
	DateOfBirth time.Time `json:"date_of_birth" validate:"required"`

	// Optional fields
	MiddleName string   `json:"middle_name,omitempty"`
	Address    *Address `json:"address,omitempty"`

	// Validation level
	ValidationLevel SSNValidationLevel `json:"validation_level"`

	// Metadata
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

// StateIDVerificationRequest represents a state-issued ID card verification request
type StateIDVerificationRequest struct {
	IDNumber       string    `json:"id_number" validate:"required"`
	State          string    `json:"state" validate:"required,len=2"`
	FirstName      string    `json:"first_name" validate:"required"`
	LastName       string    `json:"last_name" validate:"required"`
	DateOfBirth    time.Time `json:"date_of_birth" validate:"required"`
	IssueDate      time.Time `json:"issue_date" validate:"required"`
	ExpirationDate time.Time `json:"expiration_date" validate:"required"`

	// Optional fields
	Address      *Address `json:"address,omitempty"`
	IDImageFront []byte   `json:"id_image_front,omitempty"`
	IDImageBack  []byte   `json:"id_image_back,omitempty"`

	// Metadata
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

// =============================================================================
// Response Models
// =============================================================================

// VerifiedIdentity contains verified identity information
type VerifiedIdentity struct {
	FirstName   string    `json:"first_name"`
	MiddleName  string    `json:"middle_name,omitempty"`
	LastName    string    `json:"last_name"`
	DateOfBirth time.Time `json:"date_of_birth"`
	Address     *Address  `json:"address,omitempty"`
	Nationality string    `json:"nationality,omitempty"`
}

// CheckResult represents the result of an individual verification check
type CheckResult struct {
	Status  CheckStatus `json:"status"`
	Details string      `json:"details,omitempty"`
	Score   float64     `json:"score,omitempty"` // 0.0 to 1.0
}

// VerificationChecks contains detailed verification check results
type VerificationChecks struct {
	DocumentAuthenticity *CheckResult `json:"document_authenticity"` // Is document genuine?
	DocumentExpiration   *CheckResult `json:"document_expiration"`   // Is document valid (not expired)?
	NameMatch            *CheckResult `json:"name_match"`            // Does name match?
	DOBMatch             *CheckResult `json:"dob_match"`             // Does DOB match?
	AddressMatch         *CheckResult `json:"address_match,omitempty"`
	FaceMatch            *CheckResult `json:"face_match,omitempty"`   // If biometric provided
	LivenessCheck        *CheckResult `json:"liveness_check,omitempty"`
}

// VerificationError represents an error encountered during verification
type VerificationError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Severity    string `json:"severity"` // error, warning, info
	Recoverable bool   `json:"recoverable"`
}

// IdentityVerificationResult represents the result of identity verification
type IdentityVerificationResult struct {
	// Verification status
	Verified          bool              `json:"verified"`
	VerificationLevel VerificationLevel `json:"verification_level"`
	ConfidenceScore   float64           `json:"confidence_score"` // 0.0 to 1.0

	// Document details
	DocumentType     DocumentType `json:"document_type"`
	DocumentNumber   string       `json:"document_number"`
	DocumentState    string       `json:"document_state,omitempty"`
	IssuingAuthority string       `json:"issuing_authority"`

	// Identity details
	VerifiedIdentity *VerifiedIdentity `json:"verified_identity"`

	// Verification checks
	Checks *VerificationChecks `json:"checks"`

	// Warnings and errors
	Warnings []string            `json:"warnings,omitempty"`
	Errors   []VerificationError `json:"errors,omitempty"`

	// Provider information
	ProviderName          string `json:"provider_name"`
	ProviderTransactionID string `json:"provider_transaction_id"`

	// Metadata
	RequestID            string    `json:"request_id"`
	VerificationTimestamp time.Time `json:"verification_timestamp"`
	ProcessingTimeMs     int64     `json:"processing_time_ms"`
}

// NameMatchResult represents the result of a name match check
type NameMatchResult struct {
	Matched     bool    `json:"matched"`
	Score       float64 `json:"score"`       // 0.0 to 1.0
	Details     string  `json:"details,omitempty"`
}

// DOBMatchResult represents the result of a date of birth match check
type DOBMatchResult struct {
	Matched     bool    `json:"matched"`
	Details     string  `json:"details,omitempty"`
}

// SSNValidationResult represents the result of SSN validation
type SSNValidationResult struct {
	// Validation status
	Valid           bool               `json:"valid"`
	ValidationLevel SSNValidationLevel `json:"validation_level"`
	ConfidenceScore float64            `json:"confidence_score"`

	// SSN details
	SSN               string `json:"ssn"` // Masked: XXX-XX-1234
	IssuanceState     string `json:"issuance_state,omitempty"`
	IssuanceYearRange string `json:"issuance_year_range,omitempty"`

	// Validation checks
	FormatValid  bool            `json:"format_valid"`
	NotDeceased  bool            `json:"not_deceased"`
	NameMatch    *NameMatchResult `json:"name_match,omitempty"`
	DOBMatch     *DOBMatchResult  `json:"dob_match,omitempty"`

	// Warnings and errors
	Warnings []string            `json:"warnings,omitempty"`
	Errors   []VerificationError `json:"errors,omitempty"`

	// Provider information
	ProviderName          string `json:"provider_name"`
	ProviderTransactionID string `json:"provider_transaction_id"`

	// Metadata
	RequestID           string    `json:"request_id"`
	ValidationTimestamp time.Time `json:"validation_timestamp"`
	ProcessingTimeMs    int64     `json:"processing_time_ms"`
}

// =============================================================================
// API Provider Interface
// =============================================================================

// USIdentityAPIProvider defines the interface for US identity verification providers
type USIdentityAPIProvider interface {
	// VerifyDocument verifies an identity document (passport, DL, state ID)
	VerifyDocument(ctx context.Context, req interface{}) (*IdentityVerificationResult, error)

	// ValidateSSN validates a Social Security Number
	ValidateSSN(ctx context.Context, req *SSNValidationRequest) (*SSNValidationResult, error)

	// GetProviderName returns the name of the provider
	GetProviderName() string

	// GetSupportedDocumentTypes returns the document types supported by this provider
	GetSupportedDocumentTypes() []DocumentType
}

// =============================================================================
// US Identity Verifier
// =============================================================================

// USIdentityVerifierConfig contains configuration for US identity verification
type USIdentityVerifierConfig struct {
	// API providers
	PrimaryProvider  USIdentityAPIProvider
	FallbackProvider USIdentityAPIProvider

	// Circuit breaker
	CircuitBreaker *CircuitBreaker

	// Retry configuration
	MaxRetries        int
	RetryDelay        time.Duration
	BackoffMultiplier float64

	// Caching
	CacheEnabled bool
	CacheTTL     time.Duration

	// Timeouts
	RequestTimeout time.Duration

	// Validation
	StrictValidation bool // If true, enforce strict format validation
}

// USIdentityVerifier handles US-specific identity verification
type USIdentityVerifier struct {
	config    *USIdentityVerifierConfig
	validator *validator.Validate
	cache     map[string]*cachedVerification
}

// cachedVerification stores a cached verification result
type cachedVerification struct {
	Result    interface{}
	Timestamp time.Time
}

// NewUSIdentityVerifier creates a new US identity verifier
func NewUSIdentityVerifier(config *USIdentityVerifierConfig) *USIdentityVerifier {
	if config == nil {
		config = &USIdentityVerifierConfig{
			MaxRetries:        3,
			RetryDelay:        1 * time.Second,
			BackoffMultiplier: 2.0,
			RequestTimeout:    30 * time.Second,
			StrictValidation:  true,
		}
	}

	return &USIdentityVerifier{
		config:    config,
		validator: validator.New(),
		cache:     make(map[string]*cachedVerification),
	}
}

// =============================================================================
// Public Methods
// =============================================================================

// VerifyPassport verifies a US passport
func (v *USIdentityVerifier) VerifyPassport(
	ctx context.Context,
	req *PassportVerificationRequest,
) (*IdentityVerificationResult, error) {
	startTime := time.Now()

	// Validate request
	if err := v.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid passport verification request: %w", err)
	}

	// Check cache
	if v.config.CacheEnabled {
		if cached := v.getFromCache(req); cached != nil {
			result := cached.(*IdentityVerificationResult)
			result.Warnings = append(result.Warnings, "Result retrieved from cache")
			return result, nil
		}
	}

	// Format validation
	if v.config.StrictValidation {
		if err := v.validatePassportFormat(req); err != nil {
			return nil, err
		}
	}

	// Execute with circuit breaker and retry logic
	result, err := v.executeWithRetry(ctx, func() (interface{}, error) {
		return v.config.PrimaryProvider.VerifyDocument(ctx, req)
	})

	if err != nil {
		// Try fallback provider
		if v.config.FallbackProvider != nil {
			result, err = v.config.FallbackProvider.VerifyDocument(ctx, req)
			if err == nil {
				verificationResult := result.(*IdentityVerificationResult)
				verificationResult.Warnings = append(verificationResult.Warnings, 
					"Verified using fallback provider")
				v.cacheResult(req, verificationResult)
				return verificationResult, nil
			}
		}
		return nil, err
	}

	verificationResult := result.(*IdentityVerificationResult)
	verificationResult.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Cache result
	if v.config.CacheEnabled {
		v.cacheResult(req, verificationResult)
	}

	return verificationResult, nil
}

// VerifyDriverLicense verifies a state driver's license
func (v *USIdentityVerifier) VerifyDriverLicense(
	ctx context.Context,
	req *DLVerificationRequest,
) (*IdentityVerificationResult, error) {
	startTime := time.Now()

	// Validate request
	if err := v.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid driver license verification request: %w", err)
	}

	// Check cache
	if v.config.CacheEnabled {
		if cached := v.getFromCache(req); cached != nil {
			result := cached.(*IdentityVerificationResult)
			result.Warnings = append(result.Warnings, "Result retrieved from cache")
			return result, nil
		}
	}

	// Format validation
	if v.config.StrictValidation {
		if !v.validateDLFormat(req.State, req.LicenseNumber) {
			return nil, fmt.Errorf("%s: invalid driver license format for state %s",
				ErrDocumentFormatInvalid, req.State)
		}
	}

	// Execute with circuit breaker and retry logic
	result, err := v.executeWithRetry(ctx, func() (interface{}, error) {
		return v.config.PrimaryProvider.VerifyDocument(ctx, req)
	})

	if err != nil {
		// Try fallback provider
		if v.config.FallbackProvider != nil {
			result, err = v.config.FallbackProvider.VerifyDocument(ctx, req)
			if err == nil {
				verificationResult := result.(*IdentityVerificationResult)
				verificationResult.Warnings = append(verificationResult.Warnings,
					"Verified using fallback provider")
				v.cacheResult(req, verificationResult)
				return verificationResult, nil
			}
		}
		return nil, err
	}

	verificationResult := result.(*IdentityVerificationResult)
	verificationResult.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	verificationResult.DocumentState = req.State

	// Add warning if enhanced verification is available but not used
	if enhancedVerificationStates[req.State] && len(req.LicenseImageFront) == 0 {
		verificationResult.Warnings = append(verificationResult.Warnings,
			fmt.Sprintf("State %s supports enhanced verification with document images", req.State))
	}

	// Cache result
	if v.config.CacheEnabled {
		v.cacheResult(req, verificationResult)
	}

	return verificationResult, nil
}

// ValidateSSN validates a Social Security Number
func (v *USIdentityVerifier) ValidateSSN(
	ctx context.Context,
	req *SSNValidationRequest,
) (*SSNValidationResult, error) {
	startTime := time.Now()

	// Validate request
	if err := v.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid SSN validation request: %w", err)
	}

	// Check cache
	if v.config.CacheEnabled {
		if cached := v.getFromCache(req); cached != nil {
			result := cached.(*SSNValidationResult)
			result.Warnings = append(result.Warnings, "Result retrieved from cache")
			return result, nil
		}
	}

	// Format validation
	formatValid, formatErr := v.validateSSNFormat(req.SSN)
	if !formatValid {
		return &SSNValidationResult{
			Valid:           false,
			ValidationLevel: SSNValidationLevelBasic,
			ConfidenceScore: 0.0,
			SSN:             maskSSN(req.SSN),
			FormatValid:     false,
			Errors: []VerificationError{{
				Code:        ErrSSNInvalid,
				Message:     formatErr.Error(),
				Severity:    "error",
				Recoverable: false,
			}},
			RequestID:           req.RequestID,
			ValidationTimestamp: time.Now(),
			ProcessingTimeMs:    time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Execute with circuit breaker and retry logic
	result, err := v.executeWithRetry(ctx, func() (interface{}, error) {
		return v.config.PrimaryProvider.ValidateSSN(ctx, req)
	})

	if err != nil {
		// Try fallback provider
		if v.config.FallbackProvider != nil {
			result, err = v.config.FallbackProvider.ValidateSSN(ctx, req)
			if err == nil {
				validationResult := result.(*SSNValidationResult)
				validationResult.Warnings = append(validationResult.Warnings,
					"Validated using fallback provider")
				validationResult.SSN = maskSSN(req.SSN)
				v.cacheResult(req, validationResult)
				return validationResult, nil
			}
		}
		return nil, err
	}

	validationResult := result.(*SSNValidationResult)
	validationResult.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	validationResult.SSN = maskSSN(req.SSN)

	// Cache result
	if v.config.CacheEnabled {
		v.cacheResult(req, validationResult)
	}

	return validationResult, nil
}

// VerifyStateID verifies a state-issued ID card
func (v *USIdentityVerifier) VerifyStateID(
	ctx context.Context,
	req *StateIDVerificationRequest,
) (*IdentityVerificationResult, error) {
	startTime := time.Now()

	// Validate request
	if err := v.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid state ID verification request: %w", err)
	}

	// Check cache
	if v.config.CacheEnabled {
		if cached := v.getFromCache(req); cached != nil {
			result := cached.(*IdentityVerificationResult)
			result.Warnings = append(result.Warnings, "Result retrieved from cache")
			return result, nil
		}
	}

	// Format validation (similar to driver's license for many states)
	if v.config.StrictValidation {
		if !v.validateDLFormat(req.State, req.IDNumber) {
			return nil, fmt.Errorf("%s: invalid state ID format for state %s",
				ErrDocumentFormatInvalid, req.State)
		}
	}

	// Execute with circuit breaker and retry logic
	result, err := v.executeWithRetry(ctx, func() (interface{}, error) {
		return v.config.PrimaryProvider.VerifyDocument(ctx, req)
	})

	if err != nil {
		// Try fallback provider
		if v.config.FallbackProvider != nil {
			result, err = v.config.FallbackProvider.VerifyDocument(ctx, req)
			if err == nil {
				verificationResult := result.(*IdentityVerificationResult)
				verificationResult.Warnings = append(verificationResult.Warnings,
					"Verified using fallback provider")
				v.cacheResult(req, verificationResult)
				return verificationResult, nil
			}
		}
		return nil, err
	}

	verificationResult := result.(*IdentityVerificationResult)
	verificationResult.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	verificationResult.DocumentState = req.State

	// Cache result
	if v.config.CacheEnabled {
		v.cacheResult(req, verificationResult)
	}

	return verificationResult, nil
}

// =============================================================================
// Private Helper Methods
// =============================================================================

// validatePassportFormat validates US passport format
func (v *USIdentityVerifier) validatePassportFormat(req *PassportVerificationRequest) error {
	// US passport numbers are 9 digits
	passportPattern := regexp.MustCompile(`^\d{9}$`)
	if !passportPattern.MatchString(req.PassportNumber) {
		return fmt.Errorf("%s: passport number must be 9 digits", ErrDocumentFormatInvalid)
	}

	// Check expiration
	if req.ExpirationDate.Before(time.Now()) {
		return fmt.Errorf("%s: passport expired on %s", ErrDocumentExpired,
			req.ExpirationDate.Format("2006-01-02"))
	}

	return nil
}

// validateDLFormat validates driver's license format for a specific state
func (v *USIdentityVerifier) validateDLFormat(state, licenseNumber string) bool {
	format, exists := stateDLFormats[strings.ToUpper(state)]
	if !exists {
		// Unknown state format, rely on API provider validation
		return true
	}
	return format.MatchString(licenseNumber)
}

// validateSSNFormat validates SSN format according to SSA rules
func (v *USIdentityVerifier) validateSSNFormat(ssn string) (bool, error) {
	// Must be exactly 9 digits
	if len(ssn) != 9 {
		return false, errors.New("SSN must be exactly 9 digits")
	}

	// Check if all digits
	ssnPattern := regexp.MustCompile(`^\d{9}$`)
	if !ssnPattern.MatchString(ssn) {
		return false, errors.New("SSN must contain only digits")
	}

	// Extract area, group, serial
	area := ssn[0:3]
	group := ssn[3:5]
	serial := ssn[5:9]

	// Cannot be all zeros
	if area == "000" || group == "00" || serial == "0000" {
		return false, errors.New("SSN cannot contain all zeros in any section")
	}

	// Area cannot be 666
	if area == "666" {
		return false, errors.New("SSN area number 666 is never issued")
	}

	// Area cannot be 900-999 (except for ITIN)
	if area[0] == '9' {
		return false, errors.New("SSN area numbers 900-999 are not valid for individuals")
	}

	return true, nil
}

// maskSSN masks SSN for display (XXX-XX-1234)
func maskSSN(ssn string) string {
	if len(ssn) != 9 {
		return "XXX-XX-XXXX"
	}
	return fmt.Sprintf("XXX-XX-%s", ssn[5:])
}

// generateCacheKey generates a cache key from a request
func (v *USIdentityVerifier) generateCacheKey(req interface{}) string {
	// Serialize request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		return ""
	}

	// Hash the serialized data
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// getFromCache retrieves a result from cache
func (v *USIdentityVerifier) getFromCache(req interface{}) interface{} {
	key := v.generateCacheKey(req)
	if key == "" {
		return nil
	}

	cached, exists := v.cache[key]
	if !exists {
		return nil
	}

	// Check if cached result is still valid
	if time.Since(cached.Timestamp) > v.config.CacheTTL {
		delete(v.cache, key)
		return nil
	}

	return cached.Result
}

// cacheResult stores a result in cache
func (v *USIdentityVerifier) cacheResult(req interface{}, result interface{}) {
	if !v.config.CacheEnabled {
		return
	}

	key := v.generateCacheKey(req)
	if key == "" {
		return
	}

	v.cache[key] = &cachedVerification{
		Result:    result,
		Timestamp: time.Now(),
	}
}

// executeWithRetry executes a function with retry logic and circuit breaker
func (v *USIdentityVerifier) executeWithRetry(
	ctx context.Context,
	fn func() (interface{}, error),
) (interface{}, error) {
	var lastErr error
	var result interface{}

	// Wrap the function for circuit breaker compatibility
	executeFn := func() error {
		var err error
		result, err = fn()
		return err
	}

	for attempt := 0; attempt <= v.config.MaxRetries; attempt++ {
		// Check circuit breaker
		if v.config.CircuitBreaker != nil {
			err := v.config.CircuitBreaker.Execute(executeFn)
			if err == nil {
				return result, nil
			}
			lastErr = err

			// Check if circuit breaker is open
			if err.Error() == "circuit breaker is open" {
				return nil, fmt.Errorf("%s: %w", ErrCodeCircuitBreakerOpen, err)
			}
		} else {
			// Execute without circuit breaker
			result, err := fn()
			if err == nil {
				return result, nil
			}
			lastErr = err
		}

		// Don't retry if this was the last attempt
		if attempt == v.config.MaxRetries {
			break
		}

		// Calculate backoff delay
		delay := time.Duration(float64(v.config.RetryDelay) *
			float64(attempt+1) * v.config.BackoffMultiplier)

		// Wait before retrying
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("%s: %w", ErrMaxRetriesExceeded, lastErr)
}

// shouldFallback determines if fallback should be attempted for an error
func shouldFallback(err error) bool {
	if err == nil {
		return false
	}

	errorString := err.Error()
	return strings.Contains(errorString, ErrProviderUnavailable) ||
		strings.Contains(errorString, ErrProviderTimeout) ||
		strings.Contains(errorString, ErrCodeCircuitBreakerOpen)
}

// =============================================================================
// Mock Provider for Testing
// =============================================================================

// MockUSIdentityProvider is a mock implementation for testing
type MockUSIdentityProvider struct {
	name               string
	supportedDocuments []DocumentType
}

// NewMockUSIdentityProvider creates a new mock provider
func NewMockUSIdentityProvider(name string) *MockUSIdentityProvider {
	return &MockUSIdentityProvider{
		name: name,
		supportedDocuments: []DocumentType{
			DocumentTypePassport,
			DocumentTypeDriverLicense,
			DocumentTypeStateID,
		},
	}
}

// VerifyDocument implements USIdentityAPIProvider
func (m *MockUSIdentityProvider) VerifyDocument(ctx context.Context, req interface{}) (*IdentityVerificationResult, error) {
	// Mock implementation - always returns success
	return &IdentityVerificationResult{
		Verified:          true,
		VerificationLevel: VerificationLevelStandard,
		ConfidenceScore:   0.95,
		DocumentType:      DocumentTypePassport,
		ProviderName:      m.name,
		Checks: &VerificationChecks{
			DocumentAuthenticity: &CheckResult{Status: CheckStatusPassed, Score: 1.0},
			DocumentExpiration:   &CheckResult{Status: CheckStatusPassed, Score: 1.0},
			NameMatch:            &CheckResult{Status: CheckStatusPassed, Score: 0.95},
			DOBMatch:             &CheckResult{Status: CheckStatusPassed, Score: 1.0},
		},
		VerificationTimestamp: time.Now(),
	}, nil
}

// ValidateSSN implements USIdentityAPIProvider
func (m *MockUSIdentityProvider) ValidateSSN(ctx context.Context, req *SSNValidationRequest) (*SSNValidationResult, error) {
	// Mock implementation - always returns valid
	return &SSNValidationResult{
		Valid:           true,
		ValidationLevel: req.ValidationLevel,
		ConfidenceScore: 0.95,
		FormatValid:     true,
		NotDeceased:     true,
		NameMatch: &NameMatchResult{
			Matched: true,
			Score:   0.95,
		},
		DOBMatch: &DOBMatchResult{
			Matched: true,
		},
		ProviderName:        m.name,
		ValidationTimestamp: time.Now(),
	}, nil
}

// GetProviderName implements USIdentityAPIProvider
func (m *MockUSIdentityProvider) GetProviderName() string {
	return m.name
}

// GetSupportedDocumentTypes implements USIdentityAPIProvider
func (m *MockUSIdentityProvider) GetSupportedDocumentTypes() []DocumentType {
	return m.supportedDocuments
}
