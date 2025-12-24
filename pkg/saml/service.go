package saml

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service handles SAML logic
type Service struct {
	repo *Repository
}

// NewService creates a new SAML service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// BuildAuthnRequest creates a SAML AuthnRequest URL for the given provider
// In a real implementation, this would use a library like russellhaering/gosaml2 or crewjam/saml
// This is a minimal REFERENCE implementation constructing the XML manually for demonstration.
func (s *Service) BuildAuthnRequest(ctx context.Context, tenantID, providerID string) (string, error) {
	provider, err := s.repo.GetProvider(ctx, tenantID, providerID)
	if err != nil {
		return "", err
	}

	id := "_" + uuid.New().String()
	issueInstant := time.Now().UTC().Format(time.RFC3339)
	issuer := fmt.Sprintf("https://gauth.example.com/api/saml/acs/%s", providerID)

	xmlTemplate := `<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ` +
		`xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="%s" Version="2.0" IssueInstant="%s" ` +
		`ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" AssertionConsumerServiceURL="%s">` +
		`<saml:Issuer>%s</saml:Issuer><samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified" ` +
		`AllowCreate="true"/></samlp:AuthnRequest>`

	authnRequest := fmt.Sprintf(xmlTemplate, id, issueInstant, issuer, issuer)

	// In a real implementation we would DEFLATE + Base64 encode for Redirect binding
	// For simplicity, we just Base64 encode here as a placeholder for the logic
	encodedRequest := base64.StdEncoding.EncodeToString([]byte(authnRequest))

	// Construct redirect URL
	return fmt.Sprintf("%s?SAMLRequest=%s", provider.SSOURL, encodedRequest), nil
}

// ParseResponse validates and parses a SAMLResponse
// Again, this is a reference stub. Real signature validation requires a robust XML crypto library.
func (s *Service) ParseResponse(ctx context.Context, tenantID, providerID, samlResponse string) (*UserIdentity, error) {
	provider, err := s.repo.GetProvider(ctx, tenantID, providerID)
	if err != nil {
		return nil, err
	}

	xmlData, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SAMLResponse: %w", err)
	}

	// Reference check: Validate we can at least parse the XML
	var response struct {
		XMLName xml.Name `xml:"Response"`
		Example string   `xml:",innerxml"`
	}
	if err := xml.Unmarshal(xmlData, &response); err != nil {
		return nil, fmt.Errorf("invalid SAML XML: %w", err)
	}

	// MOCK VALIDATION LOGIC for Reference Implementation
	// In production, MUST verify signature using provider.Certificate
	if provider.Certificate == "" {
		return nil, fmt.Errorf("provider has no certificate configured")
	}

	// Return extracted identity (Mocked)
	return &UserIdentity{
		ExternalID: "demo-user-123",
		Email:      "demo@example.com",
		FirstName:  "Demo",
		LastName:   "User",
		Attributes: map[string]string{
			"role": "user",
		},
	}, nil
}

// UserIdentity represents the extracted user data from SAML
type UserIdentity struct {
	ExternalID string
	Email      string
	FirstName  string
	LastName   string
	Attributes map[string]string
}

// ListProviders returns all providers for a tenant
func (s *Service) ListProviders(ctx context.Context, tenantID string) ([]SAMLProvider, error) {
	return s.repo.ListProviders(ctx, tenantID)
}

// GetProvider returns a provider by ID
func (s *Service) GetProvider(ctx context.Context, tenantID, providerID string) (*SAMLProvider, error) {
	return s.repo.GetProvider(ctx, tenantID, providerID)
}

// CreateProvider creates a new provider
func (s *Service) CreateProvider(ctx context.Context, p *SAMLProvider) error {
	// Set ID if missing
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return s.repo.CreateProvider(ctx, p)
}

// UpdateProvider updates an existing provider
func (s *Service) UpdateProvider(ctx context.Context, p *SAMLProvider) error {
	return s.repo.UpdateProvider(ctx, p)
}

// DeleteProvider deletes a provider
func (s *Service) DeleteProvider(ctx context.Context, tenantID, providerID string) error {
	return s.repo.DeleteProvider(ctx, tenantID, providerID)
}
