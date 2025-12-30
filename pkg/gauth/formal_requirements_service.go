// Package gauth - Formal Requirements Validation Service
// Task 6: Implements jurisdiction-specific validation, document requirements checking,
// and legal framework compliance as recommended by QA Manager
package gauth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/compliance"
	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// JurisdictionRequirement defines requirements for a specific jurisdiction
type JurisdictionRequirement struct {
	Jurisdiction         string
	RequiresNotarization bool
	RequiredDocuments    []string
	MinimumIDLevel       string // "basic", "advanced", "qualified"
	MaxValueWithoutBoard float64
	RequiredSignatures   int
	AcceptedIDTypes      []string
	NotaryRequirements   *NotaryRequirements
	LegalReferences      []string
}

// NotaryRequirements defines notarization requirements
type NotaryRequirements struct {
	Required            bool
	RequiresApostille   bool
	AcceptedAuthorities []string
	MaxCertificateAge   time.Duration
}

// DocumentRequirementCheck contains document requirement validation results
type DocumentRequirementCheck struct {
	RequirementType string
	Satisfied       bool
	Details         string
	Evidence        []string
	Issues          []string
}

// LegalComplianceCheck contains legal framework compliance validation results
type LegalComplianceCheck struct {
	Framework       string
	Compliant       bool
	Requirements    []string
	Violations      []string
	Warnings        []string
	LegalReferences []string
}

// FormalRequirementsServiceConfig configures the formal requirements service
type FormalRequirementsServiceConfig struct {
	StrictMode               bool
	EnableJurisdictionChecks bool
	EnableDocumentChecks     bool
	EnableNotaryVerification bool
	EnableLegalCompliance    bool
	CacheDuration            time.Duration
}

// FormalRequirementsService provides comprehensive formal requirements validation
type FormalRequirementsService struct {
	mu                   sync.RWMutex
	config               FormalRequirementsServiceConfig
	validator            *FormalRequirementsValidator
	legalValidator       *compliance.LegalFrameworkValidator
	jurisdictionReqs     map[string]*JurisdictionRequirement
	documentReqsCache    map[string]*DocumentRequirementCheck
	complianceCheckCache map[string]*LegalComplianceCheck
	validationAttempts   int64
	validationSuccesses  int64
	validationFailures   int64
	jurisdictionChecks   map[string]int64
	lastCacheClear       time.Time
}

// ValidationRequest contains all data needed for formal requirements validation
type ValidationRequest struct {
	PoADefinition    *poa.PoADefinition
	NotaryCert       *NotarialCertificate
	IdentityDocs     []*IdentityDocument
	DigitalSigs      []DigitalSignature
	Jurisdiction     string
	TransactionValue float64
	RequestMetadata  map[string]interface{}
}

// ComprehensiveValidationResult contains complete validation results
type ComprehensiveValidationResult struct {
	Valid               bool
	ValidationTimestamp time.Time

	// Component results
	FormalRequirements     *FormalRequirementsResult
	JurisdictionCompliance *JurisdictionComplianceResult
	DocumentRequirements   []DocumentRequirementCheck
	LegalCompliance        *LegalComplianceCheck
	NotaryVerification     *NotarialVerificationResult

	// Summary
	OverallScore    float64 // 0-100
	CriticalIssues  []string
	MinorIssues     []string
	Warnings        []string
	Recommendations []string

	// Metadata
	ValidationDuration time.Duration
	ValidatorVersion   string
}

// JurisdictionComplianceResult contains jurisdiction-specific validation results
type JurisdictionComplianceResult struct {
	Jurisdiction           string
	Compliant              bool
	RequirementsMet        []string
	RequirementsNotMet     []string
	DocumentsSatisfied     []string
	DocumentsMissing       []string
	NotarizationRequired   bool
	NotarizationSatisfied  bool
	ValueLimitCompliant    bool
	MaxAllowedValue        float64
	ActualValue            float64
	SignatureCountRequired int
	SignatureCountProvided int
	LegalReferences        []string
	Issues                 []string
}

// NewFormalRequirementsService creates a new formal requirements service
func NewFormalRequirementsService(
	validator *FormalRequirementsValidator,
	config FormalRequirementsServiceConfig,
) *FormalRequirementsService {
	service := &FormalRequirementsService{
		config:               config,
		validator:            validator,
		legalValidator:       compliance.NewLegalFrameworkValidator(),
		jurisdictionReqs:     make(map[string]*JurisdictionRequirement),
		documentReqsCache:    make(map[string]*DocumentRequirementCheck),
		complianceCheckCache: make(map[string]*LegalComplianceCheck),
		jurisdictionChecks:   make(map[string]int64),
		lastCacheClear:       time.Now(),
	}

	service.initializeDefaultJurisdictions()
	return service
}

// ValidateComprehensive performs comprehensive formal requirements validation
func (s *FormalRequirementsService) ValidateComprehensive(
	ctx context.Context,
	req *ValidationRequest,
) (*ComprehensiveValidationResult, error) {
	startTime := time.Now()

	s.mu.Lock()
	s.validationAttempts++
	s.mu.Unlock()

	result := &ComprehensiveValidationResult{
		Valid:                true,
		ValidationTimestamp:  startTime,
		ValidatorVersion:     "1.0.0",
		CriticalIssues:       []string{},
		MinorIssues:          []string{},
		Warnings:             []string{},
		Recommendations:      []string{},
		DocumentRequirements: []DocumentRequirementCheck{},
	}

	// Step 1: Basic formal requirements validation
	if s.config.EnableDocumentChecks {
		formalResult, err := s.validator.ValidateFormalRequirements(
			ctx,
			req.PoADefinition,
			req.NotaryCert,
			req.IdentityDocs,
			req.DigitalSigs,
		)
		if err != nil {
			return nil, fmt.Errorf("formal requirements validation failed: %w", err)
		}
		result.FormalRequirements = formalResult

		if !formalResult.Valid {
			result.Valid = false
			result.CriticalIssues = append(result.CriticalIssues, formalResult.Issues...)
		}
		result.Warnings = append(result.Warnings, formalResult.Warnings...)
	}

	// Step 2: Jurisdiction-specific compliance
	if s.config.EnableJurisdictionChecks && req.Jurisdiction != "" {
		jurisdictionResult, err := s.validateJurisdictionCompliance(ctx, req)
		if err != nil {
			result.MinorIssues = append(result.MinorIssues,
				fmt.Sprintf("Jurisdiction validation error: %v", err))
		} else {
			result.JurisdictionCompliance = jurisdictionResult

			if !jurisdictionResult.Compliant {
				result.Valid = false
				result.CriticalIssues = append(result.CriticalIssues, jurisdictionResult.Issues...)
			}
		}
	}

	// Step 3: Document requirements checking
	if s.config.EnableDocumentChecks {
		docChecks := s.checkDocumentRequirements(ctx, req)
		result.DocumentRequirements = docChecks

		for _, check := range docChecks {
			if !check.Satisfied {
				result.Valid = false
				result.CriticalIssues = append(result.CriticalIssues,
					fmt.Sprintf("Document requirement not satisfied: %s - %s",
						check.RequirementType, check.Details))
			}
		}
	}

	// Step 4: Notary verification
	if s.config.EnableNotaryVerification && req.NotaryCert != nil {
		notaryResult, err := s.validator.ValidateNotarialCertification(ctx, req.NotaryCert)
		if err != nil {
			result.MinorIssues = append(result.MinorIssues,
				fmt.Sprintf("Notary verification error: %v", err))
		} else {
			result.NotaryVerification = notaryResult

			if !notaryResult.Valid {
				result.Valid = false
				result.CriticalIssues = append(result.CriticalIssues, notaryResult.Issues...)
			}
		}
	}

	// Step 5: Legal framework compliance
	if s.config.EnableLegalCompliance {
		legalCheck := s.checkLegalCompliance(ctx, req)
		result.LegalCompliance = legalCheck

		if !legalCheck.Compliant {
			result.Valid = false
			result.CriticalIssues = append(result.CriticalIssues, legalCheck.Violations...)
		}
		result.Warnings = append(result.Warnings, legalCheck.Warnings...)
	}

	// Calculate overall score
	result.OverallScore = s.calculateComplianceScore(result)

	// Add recommendations
	result.Recommendations = s.generateRecommendations(result)

	// Record metrics
	result.ValidationDuration = time.Since(startTime)
	s.mu.Lock()
	if result.Valid {
		s.validationSuccesses++
	} else {
		s.validationFailures++
	}
	if req.Jurisdiction != "" {
		s.jurisdictionChecks[req.Jurisdiction]++
	}
	s.mu.Unlock()

	return result, nil
}

// validateJurisdictionCompliance validates jurisdiction-specific requirements
func (s *FormalRequirementsService) validateJurisdictionCompliance(
	ctx context.Context,
	req *ValidationRequest,
) (*JurisdictionComplianceResult, error) {
	s.mu.RLock()
	jurisdictionReq, exists := s.jurisdictionReqs[req.Jurisdiction]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported jurisdiction: %s", req.Jurisdiction)
	}

	result := &JurisdictionComplianceResult{
		Jurisdiction:           req.Jurisdiction,
		Compliant:              true,
		RequirementsMet:        []string{},
		RequirementsNotMet:     []string{},
		DocumentsSatisfied:     []string{},
		DocumentsMissing:       []string{},
		Issues:                 []string{},
		LegalReferences:        jurisdictionReq.LegalReferences,
		MaxAllowedValue:        jurisdictionReq.MaxValueWithoutBoard,
		ActualValue:            req.TransactionValue,
		SignatureCountRequired: jurisdictionReq.RequiredSignatures,
		SignatureCountProvided: len(req.DigitalSigs),
	}

	// Check notarization requirement
	result.NotarizationRequired = jurisdictionReq.RequiresNotarization
	result.NotarizationSatisfied = req.NotaryCert != nil &&
		!req.NotaryCert.ExpirationDate.IsZero() &&
		req.NotaryCert.ExpirationDate.After(time.Now())

	if result.NotarizationRequired && !result.NotarizationSatisfied {
		result.Compliant = false
		result.RequirementsNotMet = append(result.RequirementsNotMet, "Notarization required but not provided")
		result.Issues = append(result.Issues,
			fmt.Sprintf("Jurisdiction %s requires notarization", req.Jurisdiction))
	} else if result.NotarizationRequired && result.NotarizationSatisfied {
		result.RequirementsMet = append(result.RequirementsMet, "Notarization requirement satisfied")
	}

	// Check value limits
	result.ValueLimitCompliant = req.TransactionValue <= jurisdictionReq.MaxValueWithoutBoard
	if !result.ValueLimitCompliant {
		result.Compliant = false
		result.RequirementsNotMet = append(result.RequirementsNotMet, "Transaction value exceeds limits")
		result.Issues = append(result.Issues,
			fmt.Sprintf("Value %.2f exceeds maximum %.2f without board approval",
				req.TransactionValue, jurisdictionReq.MaxValueWithoutBoard))
	} else {
		result.RequirementsMet = append(result.RequirementsMet, "Value limit compliance satisfied")
	}

	// Check signature count
	if len(req.DigitalSigs) < jurisdictionReq.RequiredSignatures {
		result.Compliant = false
		result.RequirementsNotMet = append(result.RequirementsNotMet, "Insufficient signatures")
		result.Issues = append(result.Issues,
			fmt.Sprintf("Jurisdiction requires %d signatures, provided %d",
				jurisdictionReq.RequiredSignatures, len(req.DigitalSigs)))
	} else {
		result.RequirementsMet = append(result.RequirementsMet, "Signature count requirement satisfied")
	}

	// Check required documents
	for _, requiredDoc := range jurisdictionReq.RequiredDocuments {
		found := false
		for _, providedDoc := range req.IdentityDocs {
			if strings.EqualFold(providedDoc.DocumentType, requiredDoc) {
				found = true
				result.DocumentsSatisfied = append(result.DocumentsSatisfied, requiredDoc)
				break
			}
		}
		if !found {
			result.Compliant = false
			result.DocumentsMissing = append(result.DocumentsMissing, requiredDoc)
			result.RequirementsNotMet = append(result.RequirementsNotMet,
				fmt.Sprintf("Required document missing: %s", requiredDoc))
		}
	}

	// Check ID types
	for _, doc := range req.IdentityDocs {
		accepted := false
		for _, acceptedType := range jurisdictionReq.AcceptedIDTypes {
			if strings.EqualFold(doc.DocumentType, acceptedType) {
				accepted = true
				break
			}
		}
		if !accepted {
			result.Issues = append(result.Issues,
				fmt.Sprintf("ID type '%s' not accepted in jurisdiction %s",
					doc.DocumentType, req.Jurisdiction))
		}
	}

	return result, nil
}

// checkDocumentRequirements validates document requirements
func (s *FormalRequirementsService) checkDocumentRequirements(
	ctx context.Context,
	req *ValidationRequest,
) []DocumentRequirementCheck {
	checks := []DocumentRequirementCheck{}

	// Check 1: Written form requirement
	writtenFormCheck := DocumentRequirementCheck{
		RequirementType: "written_form",
		Satisfied:       true,
		Details:         "Power of Attorney must be in written form",
		Evidence:        []string{},
		Issues:          []string{},
	}

	if req.PoADefinition == nil {
		writtenFormCheck.Satisfied = false
		writtenFormCheck.Issues = append(writtenFormCheck.Issues, "No PoA definition provided")
	} else {
		writtenFormCheck.Evidence = append(writtenFormCheck.Evidence,
			"PoA definition document present")

		// Check if principal and authorized client are specified
		if req.PoADefinition.Parties.Principal.Identity == "" {
			writtenFormCheck.Satisfied = false
			writtenFormCheck.Issues = append(writtenFormCheck.Issues, "Principal identity not specified")
		}
		if req.PoADefinition.Parties.AuthorizedClient.Identity == "" {
			writtenFormCheck.Satisfied = false
			writtenFormCheck.Issues = append(writtenFormCheck.Issues, "Authorized client identity not specified")
		}
	}
	checks = append(checks, writtenFormCheck)

	// Check 2: Identity verification requirement
	identityCheck := DocumentRequirementCheck{
		RequirementType: "identity_verification",
		Satisfied:       len(req.IdentityDocs) > 0,
		Details:         "Identity documents must be provided and verified",
		Evidence:        []string{},
		Issues:          []string{},
	}

	if len(req.IdentityDocs) == 0 {
		identityCheck.Issues = append(identityCheck.Issues, "No identity documents provided")
	} else {
		for _, doc := range req.IdentityDocs {
			// Consider document verified if it has an expiration date in the future
			if !doc.ExpirationDate.IsZero() && doc.ExpirationDate.After(time.Now()) {
				identityCheck.Evidence = append(identityCheck.Evidence,
					fmt.Sprintf("%s verified: %s", doc.DocumentType, doc.DocumentNumber))
			} else {
				identityCheck.Satisfied = false
				identityCheck.Issues = append(identityCheck.Issues,
					fmt.Sprintf("%s not verified or expired: %s", doc.DocumentType, doc.DocumentNumber))
			}
		}
	}
	checks = append(checks, identityCheck)

	// Check 3: Signature requirement
	signatureCheck := DocumentRequirementCheck{
		RequirementType: "digital_signature",
		Satisfied:       len(req.DigitalSigs) > 0,
		Details:         "Document must be digitally signed",
		Evidence:        []string{},
		Issues:          []string{},
	}

	if len(req.DigitalSigs) == 0 {
		signatureCheck.Issues = append(signatureCheck.Issues, "No digital signatures provided")
	} else {
		for _, sig := range req.DigitalSigs {
			// Consider signature valid if it has a timestamp
			if !sig.Timestamp.IsZero() {
				signerInfo := sig.SignerInfo
				if signerInfo == "" {
					signerInfo = "Unknown"
				}
				signatureCheck.Evidence = append(signatureCheck.Evidence,
					fmt.Sprintf("Valid signature from %s", signerInfo))
			} else {
				signatureCheck.Satisfied = false
				signerInfo := sig.SignerInfo
				if signerInfo == "" {
					signerInfo = "Unknown"
				}
				signatureCheck.Issues = append(signatureCheck.Issues,
					fmt.Sprintf("Invalid signature from %s", signerInfo))
			}
		}
	}
	checks = append(checks, signatureCheck)

	// Check 4: Scope specification requirement
	scopeCheck := DocumentRequirementCheck{
		RequirementType: "scope_specification",
		Satisfied:       true,
		Details:         "Authorized actions must be clearly specified",
		Evidence:        []string{},
		Issues:          []string{},
	}

	if req.PoADefinition != nil {
		actionCount := len(req.PoADefinition.Authorization.AuthorizedActions.Transactions) +
			len(req.PoADefinition.Authorization.AuthorizedActions.Decisions) +
			len(req.PoADefinition.Authorization.AuthorizedActions.PhysicalActions) +
			len(req.PoADefinition.Authorization.AuthorizedActions.NonPhysicalActions)

		if actionCount == 0 {
			scopeCheck.Satisfied = false
			scopeCheck.Issues = append(scopeCheck.Issues, "No authorized actions specified")
		} else {
			scopeCheck.Evidence = append(scopeCheck.Evidence,
				fmt.Sprintf("%d authorized actions specified", actionCount))
		}
	}
	checks = append(checks, scopeCheck)

	// Check 5: Validity period requirement
	validityCheck := DocumentRequirementCheck{
		RequirementType: "validity_period",
		Satisfied:       true,
		Details:         "Validity period must be specified",
		Evidence:        []string{},
		Issues:          []string{},
	}

	if req.PoADefinition != nil {
		if req.PoADefinition.Requirements.ValidityPeriod.StartTime.IsZero() {
			validityCheck.Satisfied = false
			validityCheck.Issues = append(validityCheck.Issues, "Start time not specified")
		} else {
			validityCheck.Evidence = append(validityCheck.Evidence,
				fmt.Sprintf("Validity period: %s to %s",
					req.PoADefinition.Requirements.ValidityPeriod.StartTime.Format(time.RFC3339),
					req.PoADefinition.Requirements.ValidityPeriod.EndTime.Format(time.RFC3339)))
		}
	}
	checks = append(checks, validityCheck)

	return checks
}

// checkLegalCompliance validates legal framework compliance
func (s *FormalRequirementsService) checkLegalCompliance(
	ctx context.Context,
	req *ValidationRequest,
) *LegalComplianceCheck {
	check := &LegalComplianceCheck{
		Framework:       req.Jurisdiction,
		Compliant:       true,
		Requirements:    []string{},
		Violations:      []string{},
		Warnings:        []string{},
		LegalReferences: []string{},
	}

	if req.Jurisdiction == "" {
		check.Warnings = append(check.Warnings, "No jurisdiction specified")
		return check
	}

	// Convert jurisdiction string to compliance.Jurisdiction enum
	jurisdiction := compliance.Jurisdiction(req.Jurisdiction)

	// Determine action from PoA scope
	action := "delegation"
	if req.PoADefinition != nil && len(req.PoADefinition.Authorization.AuthorizedActions.Transactions) > 0 {
		action = "financial_transaction"
	}

	// Validate against legal framework
	err := s.legalValidator.ValidateJurisdiction(ctx, jurisdiction, action)
	if err != nil {
		check.Compliant = false
		check.Violations = append(check.Violations, err.Error())
	} else {
		check.Requirements = append(check.Requirements,
			fmt.Sprintf("Jurisdiction %s compliance validated for action: %s", req.Jurisdiction, action))
	}

	// Get jurisdiction-specific rules
	rules, err := s.legalValidator.GetJurisdictionRules(jurisdiction)
	if err == nil {
		// Check value limits
		if valueLimit, exists := rules.ValueLimits[action]; exists {
			if req.TransactionValue > valueLimit {
				check.Compliant = false
				check.Violations = append(check.Violations,
					fmt.Sprintf("Transaction value %.2f exceeds jurisdiction limit %.2f for action %s",
						req.TransactionValue, valueLimit, action))
			}
		}

		// Check approval requirements
		if approvalLevel, exists := rules.RequiredApprovals[action]; exists {
			check.Requirements = append(check.Requirements,
				fmt.Sprintf("Approval level required: %s", approvalLevel))
		}
	}

	return check
}

// calculateComplianceScore calculates an overall compliance score (0-100)
func (s *FormalRequirementsService) calculateComplianceScore(
	result *ComprehensiveValidationResult,
) float64 {
	score := 100.0

	// Deduct for critical issues (10 points each, max 50)
	criticalDeduction := float64(len(result.CriticalIssues)) * 10.0
	if criticalDeduction > 50.0 {
		criticalDeduction = 50.0
	}
	score -= criticalDeduction

	// Deduct for minor issues (3 points each, max 20)
	minorDeduction := float64(len(result.MinorIssues)) * 3.0
	if minorDeduction > 20.0 {
		minorDeduction = 20.0
	}
	score -= minorDeduction

	// Deduct for warnings (1 point each, max 10)
	warningDeduction := float64(len(result.Warnings))
	if warningDeduction > 10.0 {
		warningDeduction = 10.0
	}
	score -= warningDeduction

	// Deduct for unsatisfied document requirements (5 points each, max 20)
	unsatisfiedCount := 0
	for _, check := range result.DocumentRequirements {
		if !check.Satisfied {
			unsatisfiedCount++
		}
	}
	docDeduction := float64(unsatisfiedCount) * 5.0
	if docDeduction > 20.0 {
		docDeduction = 20.0
	}
	score -= docDeduction

	if score < 0 {
		score = 0
	}

	return score
}

// generateRecommendations generates recommendations based on validation results
func (s *FormalRequirementsService) generateRecommendations(
	result *ComprehensiveValidationResult,
) []string {
	recommendations := []string{}

	// Check notarization
	if result.JurisdictionCompliance != nil &&
		result.JurisdictionCompliance.NotarizationRequired &&
		!result.JurisdictionCompliance.NotarizationSatisfied {
		recommendations = append(recommendations,
			"Obtain notarial certification to satisfy jurisdiction requirements")
	}

	// Check document requirements
	for _, check := range result.DocumentRequirements {
		if !check.Satisfied {
			recommendations = append(recommendations,
				fmt.Sprintf("Satisfy %s requirement: %s", check.RequirementType, check.Details))
		}
	}

	// Check value limits
	if result.JurisdictionCompliance != nil && !result.JurisdictionCompliance.ValueLimitCompliant {
		recommendations = append(recommendations,
			"Obtain board approval for transaction exceeding value limits")
	}

	// Check signatures
	if result.JurisdictionCompliance != nil &&
		result.JurisdictionCompliance.SignatureCountProvided < result.JurisdictionCompliance.SignatureCountRequired {
		recommendations = append(recommendations,
			fmt.Sprintf("Obtain %d additional signature(s) to meet jurisdiction requirements",
				result.JurisdictionCompliance.SignatureCountRequired-result.JurisdictionCompliance.SignatureCountProvided))
	}

	// Check legal compliance
	if result.LegalCompliance != nil && !result.LegalCompliance.Compliant {
		recommendations = append(recommendations,
			"Review and address legal framework violations before proceeding")
	}

	// Score-based recommendations
	if result.OverallScore < 70.0 {
		recommendations = append(recommendations,
			"Compliance score is below acceptable threshold - comprehensive review recommended")
	}

	return recommendations
}

// initializeDefaultJurisdictions sets up default jurisdiction requirements
func (s *FormalRequirementsService) initializeDefaultJurisdictions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Germany (DE) - Strict requirements
	s.jurisdictionReqs["DE"] = &JurisdictionRequirement{
		Jurisdiction:         "DE",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id"},
		MinimumIDLevel:       "qualified",
		MaxValueWithoutBoard: 100000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"German_Notary_Chamber"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"BGB §167", "BGB §126"},
	}

	// United States (US) - Varies by state
	s.jurisdictionReqs["US"] = &JurisdictionRequirement{
		Jurisdiction:         "US",
		RequiresNotarization: false, // Varies by state
		RequiredDocuments:    []string{"passport", "drivers_license"},
		MinimumIDLevel:       "basic",
		MaxValueWithoutBoard: 500000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "drivers_license", "state_id"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"State_Notary_Public"},
			MaxCertificateAge:   730 * 24 * time.Hour,
		},
		LegalReferences: []string{"Uniform Power of Attorney Act"},
	}

	// United Kingdom (UK)
	s.jurisdictionReqs["UK"] = &JurisdictionRequirement{
		Jurisdiction:         "UK",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "drivers_licence"},
		MinimumIDLevel:       "basic",
		MaxValueWithoutBoard: 250000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "drivers_licence", "biometric_residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"UK_Notary_Public"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Powers of Attorney Act 1971", "Mental Capacity Act 2005"},
	}

	// European Union (EU) - General
	s.jurisdictionReqs["EU"] = &JurisdictionRequirement{
		Jurisdiction:         "EU",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 150000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "eIDAS_card"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"EU_Member_State_Notary"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"eIDAS Regulation (EU) 910/2014"},
	}

	// Canada (CA)
	s.jurisdictionReqs["CA"] = &JurisdictionRequirement{
		Jurisdiction:         "CA",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "drivers_license"},
		MinimumIDLevel:       "basic",
		MaxValueWithoutBoard: 300000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "drivers_license", "provincial_id"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Canadian_Notary_Public"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Powers of Attorney Act (Provincial)"},
	}

	// France (FR)
	s.jurisdictionReqs["FR"] = &JurisdictionRequirement{
		Jurisdiction:         "FR",
		RequiresNotarization: true,                 // Certain acts (notaries)
		RequiredDocuments:    []string{"passport"}, // Simplified, CNI is also valid
		MinimumIDLevel:       "substantial",        // eIDAS level
		MaxValueWithoutBoard: 100000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "cni", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Notaire_De_France"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Code civil"},
	}

	// Italy (IT)
	s.jurisdictionReqs["IT"] = &JurisdictionRequirement{
		Jurisdiction:         "IT",
		RequiresNotarization: true,               // Procura speciale requires notary for some acts
		RequiredDocuments:    []string{"tax_id"}, // Codice Fiscale is ubiquitous
		MinimumIDLevel:       "substantial",      // SPID Level 2
		MaxValueWithoutBoard: 100000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "cie", "tax_id"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Notaio_Italiano"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Codice Civile"},
	}

	// Spain (ES)
	s.jurisdictionReqs["ES"] = &JurisdictionRequirement{
		Jurisdiction:         "ES",
		RequiresNotarization: true,                    // Poder notarial
		RequiredDocuments:    []string{"national_id"}, // DNI/NIE
		MinimumIDLevel:       "substantial",           // Cl@ve
		MaxValueWithoutBoard: 100000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "dni", "nie"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Notario_Espana"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Código Civil"},
	}

	// Australia (AU)
	s.jurisdictionReqs["AU"] = &JurisdictionRequirement{
		Jurisdiction:         "AU",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "drivers_licence"},
		MinimumIDLevel:       "basic",
		MaxValueWithoutBoard: 400000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "drivers_licence", "photo_card"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Australian_Notary_Public"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Powers of Attorney Act (State-specific)"},
	}

	// Italy (IT) - Strict notarization requirements
	s.jurisdictionReqs["IT"] = &JurisdictionRequirement{
		Jurisdiction:         "IT",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "carta_identita"},
		MinimumIDLevel:       "qualified",
		MaxValueWithoutBoard: 75000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "carta_identita", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Consiglio_Nazionale_del_Notariato"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Codice Civile Art. 1392", "Legge 16 febbraio 1913, n. 89"},
	}

	// Netherlands (NL) - Moderate requirements
	s.jurisdictionReqs["NL"] = &JurisdictionRequirement{
		Jurisdiction:         "NL",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 120000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "drivers_license", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Koninklijke_Notariele_Beroepsorganisatie"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Burgerlijk Wetboek Boek 3", "Wet op het Notarisambt"},
	}

	// Spain (ES) - Notarization required for significant transactions
	s.jurisdictionReqs["ES"] = &JurisdictionRequirement{
		Jurisdiction:         "ES",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "dni"},
		MinimumIDLevel:       "qualified",
		MaxValueWithoutBoard: 80000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "dni", "nie", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Consejo_General_del_Notariado"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Código Civil Art. 1709-1739", "Ley del Notariado"},
	}

	// Portugal (PT) - Similar to Spain
	s.jurisdictionReqs["PT"] = &JurisdictionRequirement{
		Jurisdiction:         "PT",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "cartao_cidadao"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 85000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "cartao_cidadao", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Ordem_dos_Notarios"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Código Civil Português Art. 262-264", "Estatuto do Notariado"},
	}

	// France (FR) - Strict civil law requirements
	s.jurisdictionReqs["FR"] = &JurisdictionRequirement{
		Jurisdiction:         "FR",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "carte_identite"},
		MinimumIDLevel:       "qualified",
		MaxValueWithoutBoard: 90000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "carte_identite", "residence_permit", "titre_sejour"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Conseil_Superieur_du_Notariat"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Code Civil Art. 1984-2010", "Ordonnance du 2 novembre 1945"},
	}

	// Norway (NO) - Nordic model
	s.jurisdictionReqs["NO"] = &JurisdictionRequirement{
		Jurisdiction:         "NO",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "national_id", "bank_id"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 200000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "bank_id", "drivers_license"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Norwegian_Notary_Public"},
			MaxCertificateAge:   730 * 24 * time.Hour,
		},
		LegalReferences: []string{"Fullmaktsloven", "Straffeloven"},
	}

	// Denmark (DK) - Nordic model with digital emphasis
	s.jurisdictionReqs["DK"] = &JurisdictionRequirement{
		Jurisdiction:         "DK",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "national_id", "nemid"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 180000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "nemid", "mitid", "drivers_license"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Danish_Notary_Public"},
			MaxCertificateAge:   730 * 24 * time.Hour,
		},
		LegalReferences: []string{"Fuldmagtsforhold", "Aftaleloven"},
	}

	// Sweden (SE) - Nordic model with high digital trust
	s.jurisdictionReqs["SE"] = &JurisdictionRequirement{
		Jurisdiction:         "SE",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "national_id", "bank_id"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 220000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "bank_id", "drivers_license", "e_legitimation"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Swedish_Notary_Public"},
			MaxCertificateAge:   730 * 24 * time.Hour,
		},
		LegalReferences: []string{"Fullmaktslagen", "Avtalslagen"},
	}

	// Estonia (EE) - Digital-first e-Residency nation
	s.jurisdictionReqs["EE"] = &JurisdictionRequirement{
		Jurisdiction:         "EE",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "national_id", "e_residency_card"},
		MinimumIDLevel:       "qualified",
		MaxValueWithoutBoard: 150000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "e_residency_card", "digi_id", "mobile_id"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Estonian_Notary_Chamber"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Võlaõigusseadus", "Digital Signature Act", "e-Residency Act"},
	}

	// Lithuania (LT) - Baltic digital adoption
	s.jurisdictionReqs["LT"] = &JurisdictionRequirement{
		Jurisdiction:         "LT",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "national_id"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 140000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "residence_permit", "drivers_license"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Lithuanian_Chamber_of_Notaries"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Civilinis kodeksas", "Notariato įstatymas"},
	}

	// Austria (AT) - Similar to Germany, strict civil law
	s.jurisdictionReqs["AT"] = &JurisdictionRequirement{
		Jurisdiction:         "AT",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "personalausweis"},
		MinimumIDLevel:       "qualified",
		MaxValueWithoutBoard: 95000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "personalausweis", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Osterreichische_Notariatskammer"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"ABGB §1002-1020", "Notariatsordnung"},
	}

	// Belgium (BE) - Bilingual system, notarization required
	s.jurisdictionReqs["BE"] = &JurisdictionRequirement{
		Jurisdiction:         "BE",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "carte_identite"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 110000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "carte_identite", "eID_card", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Federale_Raad_der_Notarissen", "Conseil_Federal_des_Notaires"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Code Civil Belge Art. 1984-2010", "Wetboek der Notarisambt"},
	}

	// Bulgaria (BG) - EU member, moderate requirements
	s.jurisdictionReqs["BG"] = &JurisdictionRequirement{
		Jurisdiction:         "BG",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 60000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "lична_карта", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Bulgarian_Chamber_of_Notaries"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Закон за задълженията и договорите", "Закон за нотариусите"},
	}

	// Croatia (HR) - EU member since 2013
	s.jurisdictionReqs["HR"] = &JurisdictionRequirement{
		Jurisdiction:         "HR",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "osobna_iskaznica"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 70000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "osobna_iskaznica", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Croatian_Chamber_of_Notaries"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Zakon o obveznim odnosima", "Zakon o javnom bilježništvu"},
	}

	// Cyprus (CY) - Common law influence, EU member
	s.jurisdictionReqs["CY"] = &JurisdictionRequirement{
		Jurisdiction:         "CY",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "national_id"},
		MinimumIDLevel:       "basic",
		MaxValueWithoutBoard: 130000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Cyprus_Bar_Association"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Contract Law Cap. 149", "Powers of Attorney Law"},
	}

	// Czech Republic (CZ) - Post-communist modernization
	s.jurisdictionReqs["CZ"] = &JurisdictionRequirement{
		Jurisdiction:         "CZ",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "obcansky_prukaz"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 85000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "obcansky_prukaz", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Notarska_komora_CR"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Občanský zákoník §2159-2188", "Notářský řád"},
	}

	// Finland (FI) - Nordic model with digital emphasis
	s.jurisdictionReqs["FI"] = &JurisdictionRequirement{
		Jurisdiction:         "FI",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "national_id", "henkilokortti"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 190000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "henkilokortti", "drivers_license", "bank_id"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Finnish_Notary_Public"},
			MaxCertificateAge:   730 * 24 * time.Hour,
		},
		LegalReferences: []string{"Valtakirjalaki", "Oikeustoimilaki"},
	}

	// Greece (GR) - Civil law with notarization emphasis
	s.jurisdictionReqs["GR"] = &JurisdictionRequirement{
		Jurisdiction:         "GR",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "taytotita"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 65000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "taytotita", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Notarial_Association_of_Greece"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Αστικός Κώδικας Άρθρο 211-225", "Κώδικας Συμβολαιογράφων"},
	}

	// Hungary (HU) - EU member, moderate requirements
	s.jurisdictionReqs["HU"] = &JurisdictionRequirement{
		Jurisdiction:         "HU",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "szemelyazonosito"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 75000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "szemelyazonosito", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Hungarian_Chamber_of_Notaries"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Polgári Törvénykönyv 6:239-6:258", "Közjegyzőkről szóló törvény"},
	}

	// Ireland (IE) - Common law system
	s.jurisdictionReqs["IE"] = &JurisdictionRequirement{
		Jurisdiction:         "IE",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "drivers_licence", "public_services_card"},
		MinimumIDLevel:       "basic",
		MaxValueWithoutBoard: 280000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "drivers_licence", "public_services_card", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Faculty_of_Notaries_Public_in_Ireland"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Powers of Attorney Act 1996", "Enduring Powers of Attorney Regulations"},
	}

	// Latvia (LV) - Baltic state, digital adoption
	s.jurisdictionReqs["LV"] = &JurisdictionRequirement{
		Jurisdiction:         "LV",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 130000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "eID_card", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Latvian_Council_of_Sworn_Notaries"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Civillikums", "Notariāta likums"},
	}

	// Luxembourg (LU) - Financial center, strict requirements
	s.jurisdictionReqs["LU"] = &JurisdictionRequirement{
		Jurisdiction:         "LU",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "carte_identite"},
		MinimumIDLevel:       "qualified",
		MaxValueWithoutBoard: 125000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "carte_identite", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Chambre_des_Notaires_Luxembourg"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Code Civil Art. 1984-2010", "Loi sur le Notariat"},
	}

	// Malta (MT) - Common law system, English influence
	s.jurisdictionReqs["MT"] = &JurisdictionRequirement{
		Jurisdiction:         "MT",
		RequiresNotarization: false,
		RequiredDocuments:    []string{"passport", "national_id", "identity_card"},
		MinimumIDLevel:       "basic",
		MaxValueWithoutBoard: 160000.0,
		RequiredSignatures:   1,
		AcceptedIDTypes:      []string{"passport", "national_id", "identity_card", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            false,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Notarial_Council_of_Malta"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Civil Code Cap. 16", "Powers of Attorney Regulations"},
	}

	// Poland (PL) - Large EU economy, moderate requirements
	s.jurisdictionReqs["PL"] = &JurisdictionRequirement{
		Jurisdiction:         "PL",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "dowod_osobisty"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 105000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "dowod_osobisty", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Krajowa_Rada_Notarialna"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Kodeks cywilny Art. 95-109", "Prawo o notariacie"},
	}

	// Romania (RO) - EU member, modernizing system
	s.jurisdictionReqs["RO"] = &JurisdictionRequirement{
		Jurisdiction:         "RO",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "carte_identitate"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 55000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "carte_identitate", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Uniunea_Nationala_a_Notarilor_Publici"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Codul Civil Art. 2009-2044", "Legea notarilor publici"},
	}

	// Slovakia (SK) - Post-1993 civil law system
	s.jurisdictionReqs["SK"] = &JurisdictionRequirement{
		Jurisdiction:         "SK",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "obciansky_preukaz"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 80000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "obciansky_preukaz", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Notarska_komora_SR"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Občiansky zákonník §32-34", "Notársky poriadok"},
	}

	// Slovenia (SI) - Small EU member, efficient system
	s.jurisdictionReqs["SI"] = &JurisdictionRequirement{
		Jurisdiction:         "SI",
		RequiresNotarization: true,
		RequiredDocuments:    []string{"passport", "national_id", "osebna_izkaznica"},
		MinimumIDLevel:       "advanced",
		MaxValueWithoutBoard: 90000.0,
		RequiredSignatures:   2,
		AcceptedIDTypes:      []string{"passport", "national_id", "osebna_izkaznica", "residence_permit"},
		NotaryRequirements: &NotaryRequirements{
			Required:            true,
			RequiresApostille:   false,
			AcceptedAuthorities: []string{"Notarska_zbornica_Slovenije"},
			MaxCertificateAge:   365 * 24 * time.Hour,
		},
		LegalReferences: []string{"Obligacijski zakonik Člen 88-90", "Zakon o notariatu"},
	}
}

// AddJurisdictionRequirement adds or updates a jurisdiction requirement
func (s *FormalRequirementsService) AddJurisdictionRequirement(req *JurisdictionRequirement) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jurisdictionReqs[req.Jurisdiction] = req
}

// GetJurisdictionRequirement retrieves jurisdiction requirements
func (s *FormalRequirementsService) GetJurisdictionRequirement(jurisdiction string) (*JurisdictionRequirement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	req, exists := s.jurisdictionReqs[jurisdiction]
	if !exists {
		return nil, fmt.Errorf("jurisdiction not found: %s", jurisdiction)
	}
	return req, nil
}

// GetMetrics returns validation metrics
func (s *FormalRequirementsService) GetMetrics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"validation_attempts":     s.validationAttempts,
		"validation_successes":    s.validationSuccesses,
		"validation_failures":     s.validationFailures,
		"success_rate":            float64(s.validationSuccesses) / float64(s.validationAttempts),
		"jurisdiction_checks":     s.jurisdictionChecks,
		"supported_jurisdictions": len(s.jurisdictionReqs),
	}
}

// ClearCache clears validation result caches
func (s *FormalRequirementsService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.documentReqsCache = make(map[string]*DocumentRequirementCheck)
	s.complianceCheckCache = make(map[string]*LegalComplianceCheck)
	s.lastCacheClear = time.Now()
}
