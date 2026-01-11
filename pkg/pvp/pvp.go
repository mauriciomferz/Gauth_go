package pvp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
)

// PowerVerificationPoint (PVP) handles identity verification chain validation
// per AAP001 Step VII requirements
type PowerVerificationPoint interface {
	// VerifyIdentityChain verifies the complete identity chain from Resource Owner → Client Owner → Client
	VerifyIdentityChain(ctx context.Context, req *IdentityChainVerificationRequest) (*IdentityChainVerificationResult, error)

	// VerifyIdentityProof verifies a single identity proof credential
	VerifyIdentityProof(ctx context.Context, proof *agentauth.IdentityVerificationChain) (*IdentityProofResult, error)

	// VerifyTrustServiceProvider verifies a trust service provider's credentials
	VerifyTrustServiceProvider(ctx context.Context, tspID string) (*TSPVerificationResult, error)

	// TraceAuthorizationChain traces and validates the authorization chain
	TraceAuthorizationChain(ctx context.Context, chain *agentauth.AuthorizationChain) (*ChainTraceResult, error)

	// BindIdentityToCryptographicKey binds an identity to a cryptographic key
	BindIdentityToCryptographicKey(ctx context.Context, req *IdentityKeyBindingRequest) (*IdentityKeyBindingResult, error)
}

// IdentityChainVerificationRequest contains details for chain verification
type IdentityChainVerificationRequest struct {
	ResourceOwner      *IdentityCredential `json:"resource_owner"`
	ClientOwner        *IdentityCredential `json:"client_owner"`
	OwnersAuthorizer   *IdentityCredential `json:"owners_authorizer,omitempty"`
	Client             *ClientIdentity     `json:"client"`
	PowerOfAttorney    string              `json:"power_of_attorney,omitempty"`
	RequiredTrustLevel string              `json:"required_trust_level"` // "substantial", "high", "eidas_qualified"
}

// IdentityCredential represents an identity credential
type IdentityCredential struct {
	ID                   string                              `json:"id"`
	Type                 string                              `json:"type"` // "natural_person", "legal_person"
	Name                 string                              `json:"name"`
	Identifier           string                              `json:"identifier"` // Tax ID, registration number, etc.
	IdentifierType       string                              `json:"identifier_type"`
	Jurisdiction         string                              `json:"jurisdiction"`
	VerificationMethod   string                              `json:"verification_method"`
	VerificationLevel    agentauth.VerificationLevel         `json:"verification_level"`
	TrustServiceProvider *agentauth.TrustServiceProviderInfo `json:"trust_service_provider,omitempty"`
	Proof                *IdentityProof                      `json:"proof,omitempty"`
	IssuedAt             time.Time                           `json:"issued_at"`
	ExpiresAt            time.Time                           `json:"expires_at,omitempty"`
}

// ClientIdentity represents client identity information
type ClientIdentity struct {
	ClientID          string    `json:"client_id"`
	ClientName        string    `json:"client_name"`
	PublicKey         string    `json:"public_key"`
	ClientCertificate string    `json:"client_certificate,omitempty"`
	RegistrationDate  time.Time `json:"registration_date"`
}

// IdentityProof represents cryptographic proof of identity
type IdentityProof struct {
	Algorithm        string    `json:"algorithm"` // "RS256", "ES256", "EdDSA"
	Signature        string    `json:"signature"`
	PublicKey        string    `json:"public_key"`
	Certificate      string    `json:"certificate,omitempty"`
	CertificateChain []string  `json:"certificate_chain,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
	Nonce            string    `json:"nonce,omitempty"`
}

// IdentityChainVerificationResult contains verification results
type IdentityChainVerificationResult struct {
	Valid                    bool                 `json:"valid"`
	TrustLevel               string               `json:"trust_level"`
	ResourceOwnerVerified    bool                 `json:"resource_owner_verified"`
	ClientOwnerVerified      bool                 `json:"client_owner_verified"`
	OwnersAuthorizerVerified bool                 `json:"owners_authorizer_verified"`
	ClientVerified           bool                 `json:"client_verified"`
	ChainIntegrity           bool                 `json:"chain_integrity"`
	AuthorizationProof       string               `json:"authorization_proof,omitempty"`
	VerificationTimestamp    time.Time            `json:"verification_timestamp"`
	VerificationDetails      []VerificationDetail `json:"verification_details"`
	Warnings                 []string             `json:"warnings,omitempty"`
}

// VerificationDetail contains details for a single verification step
type VerificationDetail struct {
	Step       string    `json:"step"`
	Entity     string    `json:"entity"`
	Verified   bool      `json:"verified"`
	Method     string    `json:"method"`
	TrustLevel string    `json:"trust_level"`
	Timestamp  time.Time `json:"timestamp"`
	Details    string    `json:"details,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// IdentityProofResult contains identity proof verification results
type IdentityProofResult struct {
	Valid              bool                                `json:"valid"`
	VerificationLevel  agentauth.VerificationLevel         `json:"verification_level"`
	TrustLevel         string                              `json:"trust_level"`
	TSPVerified        bool                                `json:"tsp_verified"`
	TSPDetails         *agentauth.TrustServiceProviderInfo `json:"tsp_details,omitempty"`
	CryptographicProof bool                                `json:"cryptographic_proof"`
	Timestamp          time.Time                           `json:"timestamp"`
	Details            string                              `json:"details,omitempty"`
}

// TSPVerificationResult contains trust service provider verification results
type TSPVerificationResult struct {
	Valid            bool      `json:"valid"`
	TSPID            string    `json:"tsp_id"`
	TSPName          string    `json:"tsp_name"`
	TrustListStatus  string    `json:"trust_list_status"` // "qualified", "non-qualified"
	Jurisdiction     string    `json:"jurisdiction"`
	Accreditation    string    `json:"accreditation,omitempty"`
	ValidFrom        time.Time `json:"valid_from"`
	ValidUntil       time.Time `json:"valid_until,omitempty"`
	VerificationDate time.Time `json:"verification_date"`
}

// ChainTraceResult contains authorization chain trace results
type ChainTraceResult struct {
	Valid            bool             `json:"valid"`
	ChainLength      int              `json:"chain_length"`
	ChainLinks       []ChainLinkTrace `json:"chain_links"`
	IntegrityHash    string           `json:"integrity_hash"`
	VerificationDate time.Time        `json:"verification_date"`
	Warnings         []string         `json:"warnings,omitempty"`
}

// ChainLinkTrace contains trace information for a chain link
type ChainLinkTrace struct {
	LinkIndex          int       `json:"link_index"`
	FromEntity         string    `json:"from_entity"`
	ToEntity           string    `json:"to_entity"`
	RelationshipType   string    `json:"relationship_type"`
	LegalBasis         string    `json:"legal_basis"`
	Verified           bool      `json:"verified"`
	VerificationMethod string    `json:"verification_method"`
	Timestamp          time.Time `json:"timestamp"`
}

// IdentityKeyBindingRequest contains details for identity-key binding
type IdentityKeyBindingRequest struct {
	IdentityID         string              `json:"identity_id"`
	IdentityCredential *IdentityCredential `json:"identity_credential"`
	PublicKey          string              `json:"public_key"`
	KeyAlgorithm       string              `json:"key_algorithm"`
	BindingProof       *IdentityProof      `json:"binding_proof"`
}

// IdentityKeyBindingResult contains identity-key binding results
type IdentityKeyBindingResult struct {
	Bound            bool      `json:"bound"`
	BindingID        string    `json:"binding_id"`
	BindingHash      string    `json:"binding_hash"`
	BindingTimestamp time.Time `json:"binding_timestamp"`
	ValidUntil       time.Time `json:"valid_until,omitempty"`
	Details          string    `json:"details,omitempty"`
}

// DefaultPVP is the default PVP implementation
type DefaultPVP struct {
	trustListURL      string
	trustProviders    map[string]*agentauth.TrustServiceProviderInfo
	verificationCache map[string]*IdentityProofResult
	cacheExpiry       time.Duration
}

// NewDefaultPVP creates a new default PVP
func NewDefaultPVP(trustListURL string) *DefaultPVP {
	pvp := &DefaultPVP{
		trustListURL:      trustListURL,
		trustProviders:    make(map[string]*agentauth.TrustServiceProviderInfo),
		verificationCache: make(map[string]*IdentityProofResult),
		cacheExpiry:       15 * time.Minute,
	}

	// Seed with common trust service providers
	pvp.seedTrustProviders()

	return pvp
}

// seedTrustProviders populates with known TSPs
func (p *DefaultPVP) seedTrustProviders() {
	// eIDAS qualified trust service providers
	p.trustProviders["TSP-DE-001"] = &agentauth.TrustServiceProviderInfo{
		ProviderID:       "TSP-DE-001",
		ProviderName:     "Bundesdruckerei GmbH",
		ProviderType:     "qualified",
		Jurisdiction:     "DE",
		AccreditationRef: "BNetzA - https://www.bundesnetzagentur.de/tsl-de.xml",
		ServiceTypes:     []string{"electronic_signature", "identity_verification"},
	}

	p.trustProviders["TSP-GB-001"] = &agentauth.TrustServiceProviderInfo{
		ProviderID:       "TSP-GB-001",
		ProviderName:     "GOV.UK Verify",
		ProviderType:     "qualified",
		Jurisdiction:     "GB",
		AccreditationRef: "UK Cabinet Office - https://www.gov.uk/government/publications/govuk-verify",
		ServiceTypes:     []string{"identity_verification"},
	}
}

// VerifyIdentityChain verifies the complete identity chain
func (p *DefaultPVP) VerifyIdentityChain(
	ctx context.Context, req *IdentityChainVerificationRequest,
) (*IdentityChainVerificationResult, error) {
	result := &IdentityChainVerificationResult{
		VerificationTimestamp: time.Now(),
		VerificationDetails:   make([]VerificationDetail, 0),
	}

	// Step 1: Verify Resource Owner identity
	if req.ResourceOwner != nil {
		roDetail := p.verifyIdentity(ctx, req.ResourceOwner, "Resource Owner")
		result.VerificationDetails = append(result.VerificationDetails, roDetail)
		result.ResourceOwnerVerified = roDetail.Verified
	} else {
		result.ResourceOwnerVerified = false
		result.VerificationDetails = append(result.VerificationDetails, VerificationDetail{
			Step:      "Verify Resource Owner Identity",
			Entity:    "N/A",
			Verified:  false,
			Error:     "resource owner credential missing",
			Timestamp: time.Now(),
		})
	}

	// Step 2: Verify Client Owner identity
	if req.ClientOwner != nil {
		coDetail := p.verifyIdentity(ctx, req.ClientOwner, "Client Owner")
		result.VerificationDetails = append(result.VerificationDetails, coDetail)
		result.ClientOwnerVerified = coDetail.Verified
	} else {
		result.ClientOwnerVerified = false
		result.VerificationDetails = append(result.VerificationDetails, VerificationDetail{
			Step:      "Verify Client Owner Identity",
			Entity:    "N/A",
			Verified:  false,
			Error:     "client owner credential missing",
			Timestamp: time.Now(),
		})
	}

	// Step 3: Verify Owner's Authorizer (if present)
	if req.OwnersAuthorizer != nil {
		oaDetail := p.verifyIdentity(ctx, req.OwnersAuthorizer, "Owner's Authorizer")
		result.VerificationDetails = append(result.VerificationDetails, oaDetail)
		result.OwnersAuthorizerVerified = oaDetail.Verified
	} else {
		result.OwnersAuthorizerVerified = true // Not required in all cases
	}

	// Step 4: Verify Client identity
	if req.Client != nil {
		clientDetail := p.verifyClient(ctx, req.Client)
		result.VerificationDetails = append(result.VerificationDetails, clientDetail)
		result.ClientVerified = clientDetail.Verified
	} else {
		result.ClientVerified = false
		result.VerificationDetails = append(result.VerificationDetails, VerificationDetail{
			Step:      "Verify Client Identity",
			Entity:    "N/A",
			Verified:  false,
			Error:     "client identity missing",
			Timestamp: time.Now(),
		})
	}

	// Step 5: Verify chain integrity
	chainIntegrity := p.verifyChainIntegrity(ctx, req)
	result.ChainIntegrity = chainIntegrity

	// Determine overall validity
	result.Valid = result.ResourceOwnerVerified &&
		result.ClientOwnerVerified &&
		result.OwnersAuthorizerVerified &&
		result.ClientVerified &&
		result.ChainIntegrity

	// Determine trust level
	result.TrustLevel = p.determineTrustLevel(result)

	// Generate authorization proof
	if result.Valid {
		result.AuthorizationProof = p.generateAuthorizationProof(result)
	}

	return result, nil
}

// verifyIdentity verifies a single identity credential
func (p *DefaultPVP) verifyIdentity(ctx context.Context, cred *IdentityCredential, role string) VerificationDetail {
	detail := VerificationDetail{
		Step:      fmt.Sprintf("Verify %s Identity", role),
		Entity:    cred.Name,
		Timestamp: time.Now(),
	}

	// Check expiration
	if !cred.ExpiresAt.IsZero() && time.Now().After(cred.ExpiresAt) {
		detail.Verified = false
		detail.Error = "credential expired"
		return detail
	}

	// Verify TSP if present
	if cred.TrustServiceProvider != nil {
		tspResult, err := p.VerifyTrustServiceProvider(ctx, cred.TrustServiceProvider.ProviderID)
		if err != nil || !tspResult.Valid {
			detail.Verified = false
			detail.Error = "trust service provider verification failed"
			return detail
		}
		detail.TrustLevel = cred.TrustServiceProvider.ProviderType
	}

	// Verify cryptographic proof
	if cred.Proof != nil {
		proofValid := p.verifyCryptographicProof(cred.Proof)
		if !proofValid {
			detail.Verified = false
			detail.Error = "cryptographic proof verification failed"
			return detail
		}
	}

	detail.Verified = true
	detail.Method = cred.VerificationMethod
	detail.TrustLevel = cred.VerificationLevel.AssuranceLevel
	detail.Details = fmt.Sprintf("Verified via %s", cred.VerificationMethod)

	return detail
}

// verifyClient verifies client identity
func (p *DefaultPVP) verifyClient(ctx context.Context, client *ClientIdentity) VerificationDetail {
	detail := VerificationDetail{
		Step:      "Verify Client Identity",
		Entity:    client.ClientName,
		Timestamp: time.Now(),
		Method:    "client_certificate",
	}

	// Verify client certificate if present
	if client.ClientCertificate != "" {
		certValid := p.verifyClientCertificate(client.ClientCertificate)
		if !certValid {
			detail.Verified = false
			detail.Error = "client certificate verification failed"
			return detail
		}
	}

	detail.Verified = true
	detail.Details = fmt.Sprintf("Client registered on %s", client.RegistrationDate.Format("2006-01-02"))

	return detail
}

// verifyChainIntegrity verifies the authorization chain integrity
func (p *DefaultPVP) verifyChainIntegrity(ctx context.Context, req *IdentityChainVerificationRequest) bool {
	// Verify linkage between entities

	// 1. Resource Owner → Proof of Authorization → Client Owner
	if req.PowerOfAttorney == "" {
		return false
	}

	// 2. Client Owner → Client (ownership/control relationship)
	// In real implementation, verify commercial register linkage

	// 3. Owner's Authorizer → Client Owner (if present)
	// Verify statutory authority (placeholder)

	return true
}

// determineTrustLevel determines overall trust level
func (p *DefaultPVP) determineTrustLevel(result *IdentityChainVerificationResult) string {
	minLevel := "low"

	for _, detail := range result.VerificationDetails {
		if detail.TrustLevel != "" {
			level := detail.TrustLevel
			if level == "eidas_qualified" {
				return "eidas_qualified"
			}
			if level == "high" && minLevel != "eidas_qualified" {
				minLevel = "high"
			}
			if level == "substantial" && minLevel == "low" {
				minLevel = "substantial"
			}
		}
	}

	return minLevel
}

// generateAuthorizationProof generates cryptographic authorization proof
func (p *DefaultPVP) generateAuthorizationProof(result *IdentityChainVerificationResult) string {
	data := fmt.Sprintf("%v|%v|%v",
		result.ResourceOwnerVerified,
		result.ClientOwnerVerified,
		result.VerificationTimestamp.Unix())

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// VerifyIdentityProof verifies a single identity proof credential
func (p *DefaultPVP) VerifyIdentityProof(
	ctx context.Context, proof *agentauth.IdentityVerificationChain,
) (*IdentityProofResult, error) {
	result := &IdentityProofResult{
		Timestamp: time.Now(),
	}

	// Determine overall verification level from chain
	if len(proof.VerificationLevels) > 0 {
		result.VerificationLevel = proof.VerificationLevels[0]
	}

	// Verify trust service provider
	if proof.TrustServiceProvider != nil {
		tspResult, err := p.VerifyTrustServiceProvider(ctx, proof.TrustServiceProvider.ProviderID)
		if err != nil || !tspResult.Valid {
			result.Valid = false
			result.Details = "trust service provider verification failed"
			return result, nil
		}
		result.TSPVerified = true
		result.TSPDetails = proof.TrustServiceProvider
		result.TrustLevel = proof.TrustServiceProvider.ProviderType
	}

	// Verify cryptographic proof
	result.CryptographicProof = proof.CryptographicProof != ""
	result.Valid = proof.OverallVerification == "verified"

	return result, nil
}

// VerifyTrustServiceProvider verifies a trust service provider
func (p *DefaultPVP) VerifyTrustServiceProvider(ctx context.Context, tspID string) (*TSPVerificationResult, error) {
	tsp, exists := p.trustProviders[tspID]
	if !exists {
		return &TSPVerificationResult{
			Valid:            false,
			TSPID:            tspID,
			VerificationDate: time.Now(),
		}, nil
	}

	return &TSPVerificationResult{
		Valid:            true,
		TSPID:            tsp.ProviderID,
		TSPName:          tsp.ProviderName,
		TrustListStatus:  tsp.ProviderType,
		Jurisdiction:     tsp.Jurisdiction,
		Accreditation:    tsp.AccreditationRef,
		ValidFrom:        time.Now().Add(-365 * 24 * time.Hour), // Mock
		VerificationDate: time.Now(),
	}, nil
}

// TraceAuthorizationChain traces and validates authorization chain
func (p *DefaultPVP) TraceAuthorizationChain(
	ctx context.Context, chain *agentauth.AuthorizationChain,
) (*ChainTraceResult, error) {
	result := &ChainTraceResult{
		ChainLinks:       make([]ChainLinkTrace, 0),
		VerificationDate: time.Now(),
	}

	linkCount := 0

	// Trace Owner's Authorizer → Client Owner
	if chain.OwnersAuthorizer != nil {
		linkCount++
		legalBasis := "statutory_authority"
		if chain.OwnersAuthorizer.LegalBasis != nil {
			legalBasis = chain.OwnersAuthorizer.LegalBasis.BasisType
		}
		link := ChainLinkTrace{
			LinkIndex:          linkCount,
			FromEntity:         chain.OwnersAuthorizer.EntityName,
			ToEntity:           chain.ClientOwner.EntityName,
			RelationshipType:   "statutory_authority",
			LegalBasis:         legalBasis,
			Verified:           chain.OwnersAuthorizer.IdentityVerified,
			VerificationMethod: chain.OwnersAuthorizer.VerificationMethod,
			Timestamp:          time.Now(),
		}
		result.ChainLinks = append(result.ChainLinks, link)
	}

	// Trace Client Owner → Client
	linkCount++
	clientOwnerLegalBasis := "ownership"
	if chain.ClientOwner.LegalBasis != nil {
		clientOwnerLegalBasis = chain.ClientOwner.LegalBasis.BasisType
	}
	link := ChainLinkTrace{
		LinkIndex:          linkCount,
		FromEntity:         chain.ClientOwner.EntityName,
		ToEntity:           chain.Client.EntityName,
		RelationshipType:   "ownership",
		LegalBasis:         clientOwnerLegalBasis,
		Verified:           chain.ClientOwner.IdentityVerified && chain.Client.IdentityVerified,
		VerificationMethod: chain.ClientOwner.VerificationMethod,
		Timestamp:          time.Now(),
	}
	result.ChainLinks = append(result.ChainLinks, link)

	result.ChainLength = linkCount
	result.Valid = true
	result.IntegrityHash = p.calculateChainHash(result.ChainLinks)

	return result, nil
}

// calculateChainHash calculates integrity hash for chain
func (p *DefaultPVP) calculateChainHash(links []ChainLinkTrace) string {
	data := ""
	for _, link := range links {
		data += fmt.Sprintf("%s→%s|", link.FromEntity, link.ToEntity)
	}
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// BindIdentityToCryptographicKey binds identity to key
func (p *DefaultPVP) BindIdentityToCryptographicKey(
	ctx context.Context, req *IdentityKeyBindingRequest,
) (*IdentityKeyBindingResult, error) {
	// Verify identity credential
	if req.IdentityCredential.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("identity credential expired")
	}

	// Verify binding proof
	if req.BindingProof == nil {
		return nil, fmt.Errorf("binding proof required")
	}

	if !p.verifyCryptographicProof(req.BindingProof) {
		return nil, fmt.Errorf("binding proof verification failed")
	}

	// Create binding
	bindingData := fmt.Sprintf("%s|%s|%d",
		req.IdentityID,
		req.PublicKey,
		time.Now().Unix())

	hash := sha256.Sum256([]byte(bindingData))
	bindingHash := hex.EncodeToString(hash[:])

	return &IdentityKeyBindingResult{
		Bound:            true,
		BindingID:        fmt.Sprintf("BIND-%s", bindingHash[:16]),
		BindingHash:      bindingHash,
		BindingTimestamp: time.Now(),
		ValidUntil:       req.IdentityCredential.ExpiresAt,
		Details:          "Identity successfully bound to cryptographic key",
	}, nil
}

// verifyCryptographicProof verifies cryptographic proof (simplified)
func (p *DefaultPVP) verifyCryptographicProof(proof *IdentityProof) bool {
	// In real implementation:
	// 1. Parse public key
	// 2. Verify signature algorithm
	// 3. Verify signature over claimed data
	// 4. Check certificate chain if present
	// 5. Verify timestamp freshness

	// Simplified mock verification
	return proof != nil && proof.Signature != "" && proof.PublicKey != ""
}

// verifyClientCertificate verifies client certificate (simplified)
func (p *DefaultPVP) verifyClientCertificate(cert string) bool {
	// In real implementation:
	// 1. Parse X.509 certificate
	// 2. Verify certificate chain
	// 3. Check certificate expiration
	// 4. Verify certificate revocation status
	// 5. Validate certificate usage constraints

	return cert != ""
}

// Ensure DefaultPVP implements PowerVerificationPoint
var _ PowerVerificationPoint = (*DefaultPVP)(nil)
