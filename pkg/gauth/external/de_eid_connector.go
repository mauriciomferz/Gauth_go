package external

import (
	"context"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// =============================================================================
// German eID Constants
// =============================================================================

// EACProtocolVersion represents the Extended Access Control protocol version
type EACProtocolVersion string

const (
	EACProtocolV2_0 EACProtocolVersion = "2.0"
	EACProtocolV2_1 EACProtocolVersion = "2.1"
	EACProtocolV2_2 EACProtocolVersion = "2.2"
)

// EIDASAssuranceLevel represents the eIDAS level of assurance
type EIDASAssuranceLevel string

const (
	EIDASAssuranceLow         EIDASAssuranceLevel = "low"
	EIDASAssuranceSubstantial EIDASAssuranceLevel = "substantial"
	EIDASAssuranceHigh        EIDASAssuranceLevel = "high"
)

// NPAAuthenticationStep represents steps in nPA authentication
type NPAAuthenticationStep string

const (
	NPAStepPACE         NPAAuthenticationStep = "PACE"         // Password Authenticated Connection Establishment
	NPAStepTerminalAuth NPAAuthenticationStep = "TerminalAuth" // Terminal Authentication
	NPAStepChipAuth     NPAAuthenticationStep = "ChipAuth"     // Chip Authentication
	NPAStepRestrictedID NPAAuthenticationStep = "RestrictedID" // Restricted Identification
)

// NPAAccessRight represents data access rights for nPA
type NPAAccessRight string

const (
	NPAAccessRightDocumentType             NPAAccessRight = "DocumentType"
	NPAAccessRightIssuingState             NPAAccessRight = "IssuingState"
	NPAAccessRightDateOfExpiry             NPAAccessRight = "DateOfExpiry"
	NPAAccessRightGivenNames               NPAAccessRight = "GivenNames"
	NPAAccessRightFamilyNames              NPAAccessRight = "FamilyNames"
	NPAAccessRightArtisticName             NPAAccessRight = "ArtisticName"
	NPAAccessRightAcademicTitle            NPAAccessRight = "AcademicTitle"
	NPAAccessRightDateOfBirth              NPAAccessRight = "DateOfBirth"
	NPAAccessRightPlaceOfBirth             NPAAccessRight = "PlaceOfBirth"
	NPAAccessRightNationality              NPAAccessRight = "Nationality"
	NPAAccessRightBirthName                NPAAccessRight = "BirthName"
	NPAAccessRightPlaceOfResidence         NPAAccessRight = "PlaceOfResidence"
	NPAAccessRightCommunityID              NPAAccessRight = "CommunityID"
	NPAAccessRightResidencePermitI         NPAAccessRight = "ResidencePermitI"
	NPAAccessRightResidencePermitII        NPAAccessRight = "ResidencePermitII"
	NPAAccessRightPhoneNumber              NPAAccessRight = "PhoneNumber"
	NPAAccessRightEmailAddress             NPAAccessRight = "EmailAddress"
	NPAAccessRightDocumentValidity         NPAAccessRight = "DocumentValidity"
	NPAAccessRightAgeVerification          NPAAccessRight = "AgeVerification"
	NPAAccessRightWriteAddress             NPAAccessRight = "WriteAddress"
	NPAAccessRightWriteCommunityID         NPAAccessRight = "WriteCommunityID"
	NPAAccessRightRestrictedIdentification NPAAccessRight = "RestrictedIdentification"
	NPAAccessRightPrivilegedTerminal       NPAAccessRight = "PrivilegedTerminal"
	NPAAccessRightCANAllowed               NPAAccessRight = "CANAllowed"
	NPAAccessRightPINManagement            NPAAccessRight = "PINManagement"
)

// =============================================================================
// Request Models
// =============================================================================

// DEEIDVerificationRequest represents a German eID verification request
type DEEIDVerificationRequest struct {
	// Authentication method
	UsePACE bool   `json:"use_pace" validate:"required"`
	PIN     string `json:"pin,omitempty" validate:"omitempty,len=6"`
	CAN     string `json:"can,omitempty" validate:"omitempty,len=6"`  // Card Access Number
	PUK     string `json:"puk,omitempty" validate:"omitempty,len=10"` // Personal Unblocking Key

	// Requested access rights
	RequestedRights []NPAAccessRight `json:"requested_rights" validate:"required,min=1"`

	// eIDAS requirements
	EIDASLevel EIDASAssuranceLevel `json:"eidas_level" validate:"required,oneof=low substantial high"`

	// Age verification (optional)
	MinimumAge int `json:"minimum_age,omitempty" validate:"omitempty,min=0,max=150"`

	// Service provider information
	ServiceProviderID   string `json:"service_provider_id" validate:"required"`
	ServiceProviderName string `json:"service_provider_name" validate:"required"`

	// Certificate chain for terminal authentication
	TerminalCertificate *x509.Certificate `json:"-"` // Not serialized, provided separately

	// Transaction data (optional)
	TransactionInfo string `json:"transaction_info,omitempty"`

	// Metadata
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

// DERestrictedIDRequest represents a restricted identification request
type DERestrictedIDRequest struct {
	SectorPublicKey   []byte `json:"sector_public_key" validate:"required"`
	SectorID          string `json:"sector_id" validate:"required"`
	ServiceProviderID string `json:"service_provider_id" validate:"required"`

	// Metadata
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
}

// =============================================================================
// Response Models
// =============================================================================

// DEEIDVerificationResult represents the result of German eID verification
type DEEIDVerificationResult struct {
	// Verification status
	Verified        bool                `json:"verified"`
	EIDASLevel      EIDASAssuranceLevel `json:"eidas_level"`
	ConfidenceScore float64             `json:"confidence_score"` // 0.0 to 1.0

	// Authentication details
	AuthenticationSteps []NPAAuthenticationStep `json:"authentication_steps"`
	PACESuccessful      bool                    `json:"pace_successful"`
	TASuccessful        bool                    `json:"ta_successful"` // Terminal Authentication
	CASuccessful        bool                    `json:"ca_successful"` // Chip Authentication

	// Card information
	DocumentType  string    `json:"document_type"`
	IssuingState  string    `json:"issuing_state"`
	DateOfExpiry  time.Time `json:"date_of_expiry"`
	DocumentValid bool      `json:"document_valid"`

	// Identity attributes (based on requested rights)
	Attributes *DEIdentityAttributes `json:"attributes,omitempty"`

	// Restricted ID (if requested)
	RestrictedID string `json:"restricted_id,omitempty"`

	// Age verification result (if requested)
	AgeVerified           *bool  `json:"age_verified,omitempty"`
	AgeVerificationResult string `json:"age_verification_result,omitempty"`

	// Certificate verification
	CertificateChainValid bool   `json:"certificate_chain_valid"`
	TrustedCA             string `json:"trusted_ca"`

	// Security checks
	SecurityChecks *DESecurityChecks `json:"security_checks"`

	// Warnings and errors
	Warnings []string            `json:"warnings,omitempty"`
	Errors   []VerificationError `json:"errors,omitempty"`

	// Metadata
	RequestID             string    `json:"request_id"`
	VerificationTimestamp time.Time `json:"verification_timestamp"`
	ProcessingTimeMs      int64     `json:"processing_time_ms"`
}

// DEIdentityAttributes contains identity data from German eID
type DEIdentityAttributes struct {
	GivenNames    string    `json:"given_names,omitempty"`
	FamilyNames   string    `json:"family_names,omitempty"`
	ArtisticName  string    `json:"artistic_name,omitempty"`
	AcademicTitle string    `json:"academic_title,omitempty"`
	DateOfBirth   time.Time `json:"date_of_birth,omitempty"`
	PlaceOfBirth  string    `json:"place_of_birth,omitempty"`
	Nationality   string    `json:"nationality,omitempty"`
	BirthName     string    `json:"birth_name,omitempty"`

	// Address information
	PlaceOfResidence *DEAddress `json:"place_of_residence,omitempty"`

	// Community ID
	CommunityID string `json:"community_id,omitempty"`

	// Residence permit
	ResidencePermitI  string `json:"residence_permit_i,omitempty"`
	ResidencePermitII string `json:"residence_permit_ii,omitempty"`

	// Contact information (if authorized)
	PhoneNumber  string `json:"phone_number,omitempty"`
	EmailAddress string `json:"email_address,omitempty"`
}

// DEAddress represents a German address
type DEAddress struct {
	Street         string `json:"street"`
	City           string `json:"city"`
	ZipCode        string `json:"zip_code"`
	State          string `json:"state"`   // Bundesland
	Country        string `json:"country"` // Should be "DE"
	AddressDetails string `json:"address_details,omitempty"`
}

// DESecurityChecks contains security verification results
type DESecurityChecks struct {
	PACEProtocol            string    `json:"pace_protocol"`
	TAProtocol              string    `json:"ta_protocol"`
	CAProtocol              string    `json:"ca_protocol"`
	ChipAuthenticationValid bool      `json:"chip_authentication_valid"`
	TerminalAuthValid       bool      `json:"terminal_auth_valid"`
	SecureMessagingActive   bool      `json:"secure_messaging_active"`
	AntiCloning             bool      `json:"anti_cloning"`
	CardGenuine             bool      `json:"card_genuine"`
	ChipSoftwareVersion     string    `json:"chip_software_version,omitempty"`
	LastAuthenticationTime  time.Time `json:"last_authentication_time"`
}

// =============================================================================
// Germany eID Connector
// =============================================================================

// DEEIDConnectorConfig contains configuration for German eID connector
type DEEIDConnectorConfig struct {
	// AusweisApp2 SDK configuration
	AusweisApp2Enabled bool          `json:"ausweisapp2_enabled"`
	AusweisApp2URL     string        `json:"ausweisapp2_url"` // e.g., "http://localhost:24727"
	AusweisApp2Timeout time.Duration `json:"ausweisapp2_timeout"`

	// eID service configuration
	EIDServiceURL     string        `json:"eid_service_url"`
	EIDServiceTimeout time.Duration `json:"eid_service_timeout"`

	// Certificate configuration
	TrustedCAPath     string `json:"trusted_ca_path"`     // Path to trusted CA certificates
	CVCertificatePath string `json:"cv_certificate_path"` // Card Verifiable Certificate path

	// Security settings
	RequireEIDAS      bool                `json:"require_eidas"`
	MinimumEIDASLevel EIDASAssuranceLevel `json:"minimum_eidas_level"`
	RequirePACE       bool                `json:"require_pace"`
	RequireTA         bool                `json:"require_ta"`
	RequireCA         bool                `json:"require_ca"`

	// Circuit breaker
	CircuitBreaker *CircuitBreaker `json:"-"`

	// Retry configuration
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`

	// Caching
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`

	// Timeouts
	RequestTimeout time.Duration `json:"request_timeout"`

	// Validation
	StrictValidation bool `json:"strict_validation"`
}

// DEEIDConnector handles German eID verification
type DEEIDConnector struct {
	config     *DEEIDConnectorConfig
	validator  *validator.Validate
	cache      map[string]*cachedDEVerification
	trustedCAs []*x509.Certificate
}

// cachedDEVerification stores a cached German eID verification result
type cachedDEVerification struct {
	Result    *DEEIDVerificationResult
	Timestamp time.Time
}

// NewDEEIDConnector creates a new German eID connector
func NewDEEIDConnector(config *DEEIDConnectorConfig) (*DEEIDConnector, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	// Set defaults
	if config.AusweisApp2URL == "" {
		config.AusweisApp2URL = "http://localhost:24727"
	}
	if config.AusweisApp2Timeout == 0 {
		config.AusweisApp2Timeout = 30 * time.Second
	}
	if config.EIDServiceTimeout == 0 {
		config.EIDServiceTimeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.BackoffMultiplier == 0 {
		config.BackoffMultiplier = 2.0
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 60 * time.Second
	}
	if config.MinimumEIDASLevel == "" {
		config.MinimumEIDASLevel = EIDASAssuranceSubstantial
	}

	connector := &DEEIDConnector{
		config:    config,
		validator: validator.New(),
		cache:     make(map[string]*cachedDEVerification),
	}

	// Load trusted CAs
	if config.TrustedCAPath != "" {
		if err := connector.loadTrustedCAs(); err != nil {
			return nil, fmt.Errorf("failed to load trusted CAs: %w", err)
		}
	}

	return connector, nil
}

// VerifyEID performs German eID verification
func (c *DEEIDConnector) VerifyEID(ctx context.Context, req *DEEIDVerificationRequest) (*DEEIDVerificationResult, error) {
	startTime := time.Now()

	// Validate request
	if err := c.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check minimum eIDAS level
	if !c.isEIDASLevelSufficient(req.EIDASLevel) {
		return nil, fmt.Errorf("requested eIDAS level '%s' is below minimum required level '%s'",
			req.EIDASLevel, c.config.MinimumEIDASLevel)
	}

	// Check cache if enabled
	if c.config.CacheEnabled {
		cacheKey := c.generateCacheKey(req)
		if cached := c.getFromCache(cacheKey); cached != nil {
			return cached, nil
		}
	}

	// Perform verification with circuit breaker
	var result *DEEIDVerificationResult
	var err error

	if c.config.CircuitBreaker != nil {
		err = c.config.CircuitBreaker.Call(func() error {
			result, err = c.performVerification(ctx, req)
			return err
		})
	} else {
		result, err = c.performVerification(ctx, req)
	}

	if err != nil {
		return nil, err
	}

	// Set processing time
	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Cache result if enabled
	if c.config.CacheEnabled && result.Verified {
		c.addToCache(c.generateCacheKey(req), result)
	}

	return result, nil
}

// GetRestrictedID generates a sector-specific identifier
func (c *DEEIDConnector) GetRestrictedID(ctx context.Context, req *DERestrictedIDRequest) (string, error) {
	// Validate request
	if err := c.validator.Struct(req); err != nil {
		return "", fmt.Errorf("invalid request: %w", err)
	}

	// In a real implementation, this would:
	// 1. Use the sector public key to derive a sector-specific identifier
	// 2. Apply the eID chip's restricted identification function
	// 3. Return a pseudonym that is unique per service provider

	// This is a placeholder - real implementation requires eID SDK
	return "", errors.New("restricted ID generation requires eID card reader integration")
}

// VerifyAge performs age verification without revealing actual date of birth
func (c *DEEIDConnector) VerifyAge(ctx context.Context, req *DEEIDVerificationRequest) (bool, error) {
	if req.MinimumAge <= 0 {
		return false, errors.New("minimum age must be greater than 0")
	}

	// Perform verification with age verification right
	result, err := c.VerifyEID(ctx, req)
	if err != nil {
		return false, err
	}

	if result.AgeVerified == nil {
		return false, errors.New("age verification was not performed")
	}

	return *result.AgeVerified, nil
}

// =============================================================================
// Private Methods
// =============================================================================

func (c *DEEIDConnector) performVerification(ctx context.Context, req *DEEIDVerificationRequest) (*DEEIDVerificationResult, error) {
	result := &DEEIDVerificationResult{
		RequestID:             req.RequestID,
		VerificationTimestamp: time.Now(),
		EIDASLevel:            req.EIDASLevel,
		AuthenticationSteps:   []NPAAuthenticationStep{},
		SecurityChecks:        &DESecurityChecks{},
	}

	// Step 1: PACE (Password Authenticated Connection Establishment)
	if req.UsePACE {
		paceSuccess, err := c.performPACE(ctx, req)
		if err != nil {
			result.Errors = append(result.Errors, VerificationError{
				Code:    "PACE_FAILED",
				Message: fmt.Sprintf("PACE authentication failed: %v", err),
			})
			return result, nil
		}
		result.PACESuccessful = paceSuccess
		result.AuthenticationSteps = append(result.AuthenticationSteps, NPAStepPACE)
		result.SecurityChecks.PACEProtocol = "PACE-GM-3DES-CBC-CBC"
	}

	// Step 2: Terminal Authentication (TA)
	if c.config.RequireTA {
		taSuccess, err := c.performTerminalAuth(ctx, req)
		if err != nil {
			result.Errors = append(result.Errors, VerificationError{
				Code:    "TA_FAILED",
				Message: fmt.Sprintf("Terminal authentication failed: %v", err),
			})
			return result, nil
		}
		result.TASuccessful = taSuccess
		result.AuthenticationSteps = append(result.AuthenticationSteps, NPAStepTerminalAuth)
		result.SecurityChecks.TerminalAuthValid = taSuccess
	}

	// Step 3: Chip Authentication (CA)
	if c.config.RequireCA {
		caSuccess, err := c.performChipAuth(ctx, req)
		if err != nil {
			result.Errors = append(result.Errors, VerificationError{
				Code:    "CA_FAILED",
				Message: fmt.Sprintf("Chip authentication failed: %v", err),
			})
			return result, nil
		}
		result.CASuccessful = caSuccess
		result.AuthenticationSteps = append(result.AuthenticationSteps, NPAStepChipAuth)
		result.SecurityChecks.ChipAuthenticationValid = caSuccess
	}

	// Step 4: Read attributes based on access rights
	attributes, err := c.readAttributes(ctx, req.RequestedRights)
	if err != nil {
		result.Errors = append(result.Errors, VerificationError{
			Code:    "ATTRIBUTE_READ_FAILED",
			Message: fmt.Sprintf("Failed to read attributes: %v", err),
		})
		return result, nil
	}
	result.Attributes = attributes

	// Step 5: Perform age verification if requested
	if req.MinimumAge > 0 {
		ageVerified := c.checkAge(attributes.DateOfBirth, req.MinimumAge)
		result.AgeVerified = &ageVerified
		if ageVerified {
			result.AgeVerificationResult = fmt.Sprintf("Age %d or older", req.MinimumAge)
		} else {
			result.AgeVerificationResult = fmt.Sprintf("Under %d years old", req.MinimumAge)
		}
	}

	// Step 6: Generate restricted ID if requested
	for _, right := range req.RequestedRights {
		if right == NPAAccessRightRestrictedIdentification {
			// This would require sector-specific key and eID SDK
			result.Warnings = append(result.Warnings,
				"Restricted ID generation requires additional SDK integration")
			break
		}
	}

	// Calculate confidence score
	result.ConfidenceScore = c.calculateConfidenceScore(result)

	// Determine verification success
	result.Verified = result.PACESuccessful &&
		(!c.config.RequireTA || result.TASuccessful) &&
		(!c.config.RequireCA || result.CASuccessful) &&
		len(result.Errors) == 0

	return result, nil
}

func (c *DEEIDConnector) performPACE(ctx context.Context, req *DEEIDVerificationRequest) (bool, error) {
	// In a real implementation, this would:
	// 1. Establish a secure channel using PACE protocol
	// 2. Use PIN/CAN for authentication
	// 3. Derive session keys

	// This is a placeholder - real implementation requires eID SDK
	if req.PIN == "" && req.CAN == "" {
		return false, errors.New("PIN or CAN required for PACE")
	}

	// Simulate successful PACE
	return true, nil
}

func (c *DEEIDConnector) performTerminalAuth(ctx context.Context, req *DEEIDVerificationRequest) (bool, error) {
	// In a real implementation, this would:
	// 1. Present terminal certificate to the card
	// 2. Verify certificate chain against trusted CAs
	// 3. Prove possession of private key

	if req.TerminalCertificate == nil {
		return false, errors.New("terminal certificate required for TA")
	}

	// Verify certificate chain
	if len(c.trustedCAs) > 0 {
		// Would verify against trusted CAs here
	}

	// Simulate successful TA
	return true, nil
}

func (c *DEEIDConnector) performChipAuth(ctx context.Context, req *DEEIDVerificationRequest) (bool, error) {
	// In a real implementation, this would:
	// 1. Perform Diffie-Hellman key agreement with the chip
	// 2. Verify chip's authenticity
	// 3. Prevent cloning attacks

	// Simulate successful CA
	return true, nil
}

func (c *DEEIDConnector) readAttributes(ctx context.Context, rights []NPAAccessRight) (*DEIdentityAttributes, error) {
	// In a real implementation, this would read data from the eID card
	// based on granted access rights

	attributes := &DEIdentityAttributes{}

	// This is a placeholder - real implementation requires eID SDK
	for _, right := range rights {
		switch right {
		case NPAAccessRightGivenNames:
			// Would read given names from card
		case NPAAccessRightFamilyNames:
			// Would read family names from card
		case NPAAccessRightDateOfBirth:
			// Would read date of birth from card
		case NPAAccessRightPlaceOfResidence:
			// Would read address from card
			// ... handle other rights
		}
	}

	return attributes, nil
}

func (c *DEEIDConnector) checkAge(dob time.Time, minimumAge int) bool {
	if dob.IsZero() {
		return false
	}

	age := int(time.Since(dob).Hours() / 24 / 365.25)
	return age >= minimumAge
}

func (c *DEEIDConnector) calculateConfidenceScore(result *DEEIDVerificationResult) float64 {
	score := 0.0
	maxScore := 0.0

	// PACE authentication
	maxScore += 0.3
	if result.PACESuccessful {
		score += 0.3
	}

	// Terminal authentication
	if c.config.RequireTA {
		maxScore += 0.3
		if result.TASuccessful {
			score += 0.3
		}
	}

	// Chip authentication
	if c.config.RequireCA {
		maxScore += 0.2
		if result.CASuccessful {
			score += 0.2
		}
	}

	// Attribute presence
	maxScore += 0.2
	if result.Attributes != nil {
		score += 0.2
	}

	if maxScore == 0 {
		return 0.0
	}

	return score / maxScore
}

func (c *DEEIDConnector) isEIDASLevelSufficient(requested EIDASAssuranceLevel) bool {
	levels := map[EIDASAssuranceLevel]int{
		EIDASAssuranceLow:         1,
		EIDASAssuranceSubstantial: 2,
		EIDASAssuranceHigh:        3,
	}

	return levels[requested] >= levels[c.config.MinimumEIDASLevel]
}

func (c *DEEIDConnector) loadTrustedCAs() error {
	// In a real implementation, this would load CA certificates from file
	// For now, return nil to indicate success
	return nil
}

func (c *DEEIDConnector) generateCacheKey(req *DEEIDVerificationRequest) string {
	return fmt.Sprintf("de_eid:%s:%s:%v",
		req.ServiceProviderID,
		req.RequestID,
		req.RequestedRights)
}

func (c *DEEIDConnector) getFromCache(key string) *DEEIDVerificationResult {
	cached, exists := c.cache[key]
	if !exists {
		return nil
	}

	// Check if cached result is still valid
	if time.Since(cached.Timestamp) > c.config.CacheTTL {
		delete(c.cache, key)
		return nil
	}

	return cached.Result
}

func (c *DEEIDConnector) addToCache(key string, result *DEEIDVerificationResult) {
	c.cache[key] = &cachedDEVerification{
		Result:    result,
		Timestamp: time.Now(),
	}
}

// =============================================================================
// TR-03110 / TR-03124 Compliance Helpers
// =============================================================================

// TROID represents a TR-03110 Object Identifier
type TROID struct {
	Algorithm string
	OID       asn1.ObjectIdentifier
}

// Common TR-03110 OIDs
var (
	// PACE OIDs
	OID_PACE_ECDH_GM_3DES_CBC_CBC     = asn1.ObjectIdentifier{0, 4, 0, 127, 0, 7, 2, 2, 4, 1, 1}
	OID_PACE_ECDH_GM_AES_CBC_CMAC_128 = asn1.ObjectIdentifier{0, 4, 0, 127, 0, 7, 2, 2, 4, 1, 2}
	OID_PACE_ECDH_GM_AES_CBC_CMAC_192 = asn1.ObjectIdentifier{0, 4, 0, 127, 0, 7, 2, 2, 4, 1, 3}
	OID_PACE_ECDH_GM_AES_CBC_CMAC_256 = asn1.ObjectIdentifier{0, 4, 0, 127, 0, 7, 2, 2, 4, 1, 4}

	// Chip Authentication OIDs
	OID_CA_ECDH_3DES_CBC_CBC     = asn1.ObjectIdentifier{0, 4, 0, 127, 0, 7, 2, 2, 3, 2, 1}
	OID_CA_ECDH_AES_CBC_CMAC_128 = asn1.ObjectIdentifier{0, 4, 0, 127, 0, 7, 2, 2, 3, 2, 2}
	OID_CA_ECDH_AES_CBC_CMAC_192 = asn1.ObjectIdentifier{0, 4, 0, 127, 0, 7, 2, 2, 3, 2, 3}
	OID_CA_ECDH_AES_CBC_CMAC_256 = asn1.ObjectIdentifier{0, 4, 0, 127, 0, 7, 2, 2, 3, 2, 4}
)

// ValidateTR03110Compliance checks TR-03110 compliance
func ValidateTR03110Compliance(result *DEEIDVerificationResult) bool {
	// Check minimum requirements for TR-03110
	return result.PACESuccessful &&
		result.SecurityChecks != nil &&
		result.SecurityChecks.SecureMessagingActive
}

// ValidateTR03124Compliance checks TR-03124 compliance (eID interface)
func ValidateTR03124Compliance(result *DEEIDVerificationResult) bool {
	// Check eID-specific requirements
	return ValidateTR03110Compliance(result) &&
		result.EIDASLevel != "" &&
		result.Attributes != nil
}
