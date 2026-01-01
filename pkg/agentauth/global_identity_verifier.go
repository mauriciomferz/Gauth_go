package agentauth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth/external"
)

// GlobalIdentityVerifier implements IdentityDocumentVerifier by routing to country-specific connectors
type GlobalIdentityVerifier struct {
	usVerifier  *external.USIdentityVerifier
	frConnector *external.FranceIdentityConnector
	itConnector *external.ItalyIdentityConnector
	esConnector *external.SpainIdentityConnector
	strictMode  bool
}

// NewGlobalIdentityVerifier creates a new global identity verifier
func NewGlobalIdentityVerifier(
	usVerifier *external.USIdentityVerifier,
	frConnector *external.FranceIdentityConnector,
	itConnector *external.ItalyIdentityConnector,
	esConnector *external.SpainIdentityConnector,
	strictMode bool,
) *GlobalIdentityVerifier {
	return &GlobalIdentityVerifier{
		usVerifier:  usVerifier,
		frConnector: frConnector,
		itConnector: itConnector,
		esConnector: esConnector,
		strictMode:  strictMode,
	}
}

// VerifyIdentityDocument verifies an identity document
func (v *GlobalIdentityVerifier) VerifyIdentityDocument(
	ctx context.Context,
	doc *IdentityDocument,
) (*IDVerificationResult, error) {
	// 1. Determine jurisdiction/connector
	country := strings.ToUpper(doc.IssuingCountry)
	if country == "" {
		// Try to infer from document type or extra data if necessary
		// For now, fail if unknown
		return &IDVerificationResult{
			Valid:            false,
			VerificationDate: time.Now(),
			Issues:           []string{"Issuing country not specified"},
		}, nil
	}

	// 2. Route to specific connector
	switch country {
	case "US", "USA":
		return v.verifyUSDocument(ctx, doc)
	case "FR", "FRA":
		return v.verifyFranceDocument(ctx, doc)
	case "IT", "ITA":
		return v.verifyItalyDocument(ctx, doc)
	case "ES", "ESP":
		return v.verifySpainDocument(ctx, doc)
	default:
		// Unsupported country
		return &IDVerificationResult{
			Valid:            false,
			VerificationDate: time.Now(),
			Issues:           []string{fmt.Sprintf("Unsupported issuing country: %s", country)},
			Warnings:         []string{"Connector not available for this jurisdiction"},
		}, nil
	}
}

// VerifyGovernmentID verifies a government ID (part of interface)
func (v *GlobalIdentityVerifier) VerifyGovernmentID(
	ctx context.Context,
	idNumber, idType, issuingCountry string,
) (*GovernmentIDInfo, error) {
	// Construct a dummy document to reuse routing logic or implement direct calls
	// For simplicity, we'll map this to a document verification
	doc := &IdentityDocument{
		DocumentNumber: idNumber,
		DocumentType:   idType,
		IssuingCountry: issuingCountry,
	}

	res, err := v.VerifyIdentityDocument(ctx, doc)
	if err != nil {
		return nil, err
	}

	if !res.Valid {
		return &GovernmentIDInfo{
			IDNumber:       idNumber,
			IDType:         idType,
			IssuingCountry: issuingCountry,
			Status:         "invalid",
			Metadata:       map[string]interface{}{"issues": res.Issues},
		}, nil
	}

	return &GovernmentIDInfo{
		IDNumber:       idNumber,
		IDType:         idType,
		IssuingCountry: issuingCountry,
		Status:         "valid",
		IssueDate:      time.Now(),                  // Approximation, normally would extract from result details
		ExpiryDate:     time.Now().AddDate(1, 0, 0), // Approximation
	}, nil
}

// CheckIDExpiration checks ID expiration (part of interface)
func (v *GlobalIdentityVerifier) CheckIDExpiration(
	ctx context.Context,
	idNumber, idType string,
) (bool, time.Time, error) {
	// This would typically require fetching the document first
	// For now, return not implemented or basiccheck
	return false, time.Time{}, fmt.Errorf("CheckIDExpiration not directly supported by global adapter")
}

// VerifyBiometricMatch verifies biometric match (part of interface)
func (v *GlobalIdentityVerifier) VerifyBiometricMatch(
	ctx context.Context,
	biometric []byte,
	idNumber string,
) (bool, float64, error) {
	// This requires country context which isn't passed here
	// Assuming this might be called after document verification
	return false, 0.0, fmt.Errorf("VerifyBiometricMatch requires country context")
}

// -----------------------------------------------------------------------------
// Country-Specific Adapters
// -----------------------------------------------------------------------------

func (v *GlobalIdentityVerifier) verifyUSDocument(ctx context.Context, doc *IdentityDocument) (*IDVerificationResult, error) {
	if v.usVerifier == nil {
		return v.connectorNotConfigured("US"), nil
	}

	// Map to US request
	// Assuming Passport for simplicity if not specified
	if strings.EqualFold(doc.DocumentType, "passport") {
		req := &external.PassportVerificationRequest{
			PassportNumber: doc.DocumentNumber,
			Nationality:    "US",
			// Missing fields would need to be populated from SubjectData or similar
			FirstName:      doc.SubjectData["first_name"],
			LastName:       doc.SubjectData["last_name"],
			DateOfBirth:    parseDate(doc.SubjectData["dob"]),
			IssueDate:      doc.IssueDate, // mapped from doc struct
			ExpirationDate: doc.ExpirationDate,
		}

		res, err := v.usVerifier.VerifyPassport(ctx, req)
		if err != nil {
			return nil, err
		}
		return v.mapUSResult(res), nil
	}

	// Default fallthrough
	return &IDVerificationResult{
		Valid:            false,
		VerificationDate: time.Now(),
		Issues:           []string{"Unsupported US document type or missing data"},
	}, nil
}

func (v *GlobalIdentityVerifier) verifyFranceDocument(ctx context.Context, doc *IdentityDocument) (*IDVerificationResult, error) {
	if v.frConnector == nil {
		return v.connectorNotConfigured("FR"), nil
	}

	if strings.EqualFold(doc.DocumentType, "passport") {
		req := &external.FrenchPassportRequest{
			PassportNumber: doc.DocumentNumber,
			FirstName:      doc.SubjectData["first_name"],
			LastName:       doc.SubjectData["last_name"],
			DateOfBirth:    doc.SubjectData["dob"], // string YYYY-MM-DD
			Nationality:    "FRA",
		}
		res, err := v.frConnector.VerifyFrenchPassport(ctx, req)
		if err != nil {
			return nil, err
		}

		return &IDVerificationResult{
			Valid:                 res.Valid,
			VerificationDate:      time.Now(),
			DocumentAuthentic:     res.Valid,
			NotExpired:            res.Status == "valid",
			BiometricMatch:        res.BiometricVerified,
			SecurityFeatureOK:     res.MRZVerified,
			IssuingAuthorityValid: true,
			Issues:                v.errorToIssues(res.Error),
		}, nil
	}
	// Add CNI support similarly

	return &IDVerificationResult{
		Valid:            false,
		VerificationDate: time.Now(),
		Issues:           []string{"Unsupported FR document type"},
	}, nil
}

func (v *GlobalIdentityVerifier) verifyItalyDocument(ctx context.Context, doc *IdentityDocument) (*IDVerificationResult, error) {
	if v.itConnector == nil {
		return v.connectorNotConfigured("IT"), nil
	}

	if strings.EqualFold(doc.DocumentType, "tax_id") || strings.EqualFold(doc.DocumentType, "codice_fiscale") {
		req := &external.CodiceFiscaleRequest{
			CodiceFiscale: doc.DocumentNumber,
			FirstName:     doc.SubjectData["first_name"],
			LastName:      doc.SubjectData["last_name"],
			DateOfBirth:   doc.SubjectData["dob"],
			Gender:        doc.SubjectData["gender"],
		}
		res, err := v.itConnector.ValidateCodiceFiscale(ctx, req)
		if err != nil {
			return nil, err
		}

		return &IDVerificationResult{
			Valid:             res.Valid,
			VerificationDate:  time.Now(),
			DocumentAuthentic: res.Valid,
			NotExpired:        true, // Tax IDs don't expire usually
			Issues:            v.errorToIssues(res.Error),
		}, nil
	}

	return &IDVerificationResult{
		Valid:            false,
		VerificationDate: time.Now(),
		Issues:           []string{"Unsupported IT document type"},
	}, nil
}

func (v *GlobalIdentityVerifier) verifySpainDocument(ctx context.Context, doc *IdentityDocument) (*IDVerificationResult, error) {
	if v.esConnector == nil {
		return v.connectorNotConfigured("ES"), nil
	}

	if strings.EqualFold(doc.DocumentType, "national_id") || strings.EqualFold(doc.DocumentType, "dni") {
		req := &external.DNIValidationRequest{
			DocumentNumber: doc.DocumentNumber,
			DocumentType:   "DNI",
			FirstName:      doc.SubjectData["first_name"],
			FirstSurname:   doc.SubjectData["last_name"],
		}
		res, err := v.esConnector.ValidateDNI(ctx, req)
		if err != nil {
			return nil, err
		}

		return &IDVerificationResult{
			Valid:             res.Valid,
			VerificationDate:  time.Now(),
			DocumentAuthentic: res.Valid,
			NotExpired:        res.Status == "valid",
			Issues:            v.errorToIssues(res.Error),
		}, nil
	}

	return &IDVerificationResult{
		Valid:            false,
		VerificationDate: time.Now(),
		Issues:           []string{"Unsupported ES document type"},
	}, nil
}

// Helpers

func (v *GlobalIdentityVerifier) connectorNotConfigured(country string) *IDVerificationResult {
	return &IDVerificationResult{
		Valid:            false,
		VerificationDate: time.Now(),
		Issues:           []string{fmt.Sprintf("Connector for %s is not configured (nil)", country)},
		Warnings:         []string{"Service unavailable"},
	}
}

func (v *GlobalIdentityVerifier) errorToIssues(errStr string) []string {
	if errStr == "" {
		return []string{}
	}
	return []string{errStr}
}

func (v *GlobalIdentityVerifier) mapUSResult(res *external.IdentityVerificationResult) *IDVerificationResult {
	return &IDVerificationResult{
		Valid:             res.Verified,
		VerificationDate:  res.VerificationTimestamp,
		DocumentAuthentic: res.Checks.DocumentAuthenticity.Status == "passed",
		NotExpired:        res.Checks.DocumentExpiration.Status == "passed",
		BiometricMatch:    res.Checks.FaceMatch != nil && res.Checks.FaceMatch.Status == "passed",
		BiometricScore:    res.ConfidenceScore,
		SecurityFeatureOK: true, // simplified
		Issues:            convertErrors(res.Errors),
		Warnings:          res.Warnings,
	}
}

func convertErrors(errs []external.VerificationError) []string {
	var issues []string
	for _, e := range errs {
		issues = append(issues, e.Message)
	}
	return issues
}

func parseDate(d string) time.Time {
	t, _ := time.Parse("2006-01-02", d)
	return t
}

// VerifyIdentityProof implements the PowerVerificationPoint interface
func (v *GlobalIdentityVerifier) VerifyIdentityProof(ctx context.Context, request *IdentityProofRequest) (*IdentityProofResult, error) {
	// Extract document from proof data
	docData, ok := request.ProofData["document"].(map[string]interface{})
	if !ok {
		// Try to construct from request fields if proof data is structurally different
		// For now, assume proof data contains document fields
		return &IdentityProofResult{
			Valid:         false,
			SubjectID:     request.SubjectID,
			VerifiedAt:    time.Now(),
			FailureReason: "Missing document data in proof request",
		}, nil
	}

	docNumber, _ := docData["number"].(string)
	docType, _ := docData["type"].(string)
	country, _ := docData["country"].(string)

	if docNumber == "" || docType == "" || country == "" {
		// Try to see if top-level request has hints (logic depends on caller)
		return &IdentityProofResult{
			Valid:         false,
			SubjectID:     request.SubjectID,
			VerifiedAt:    time.Now(),
			FailureReason: "Incomplete document data (number, type, country required)",
		}, nil
	}

	doc := &IdentityDocument{
		DocumentNumber: docNumber,
		DocumentType:   docType,
		IssuingCountry: country,
		SubjectData:    make(map[string]string),
	}

	// Copy other fields like name, dob
	if fn, ok := docData["first_name"].(string); ok {
		doc.SubjectData["first_name"] = fn
	}
	if ln, ok := docData["last_name"].(string); ok {
		doc.SubjectData["last_name"] = ln
	}
	if dob, ok := docData["dob"].(string); ok {
		doc.SubjectData["dob"] = dob
	}

	// Verify
	res, err := v.VerifyIdentityDocument(ctx, doc)
	if err != nil {
		return nil, err
	}

	result := &IdentityProofResult{
		Valid:      res.Valid,
		SubjectID:  request.SubjectID,
		VerifiedAt: res.VerificationDate,
		TrustLevel: "substantial", // Default, could be derived
	}

	if !res.Valid {
		result.FailureReason = strings.Join(res.Issues, "; ")
	} else {
		result.Identity = fmt.Sprintf("%s:%s", country, docNumber)
	}

	return result, nil
}
