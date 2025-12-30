package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
	"github.com/mauriciomferz/AgentAuth/pkg/pdp"
	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// MockPDPEngine for testing
type MockPDPEngine struct {
	allowDecision  bool
	decisionReason string
	obligations    []pdp.Obligation
}

func (m *MockPDPEngine) Evaluate(ctx context.Context, req pdp.Request) (pdp.Decision, error) {
	return pdp.Decision{
		Allow:       m.allowDecision,
		Reason:      m.decisionReason,
		Obligations: m.obligations,
	}, nil
}

func (m *MockPDPEngine) Metrics() pdp.MetricsSnapshot {
	return pdp.MetricsSnapshot{}
}

// createTestToken creates a valid extended token for testing
func createTestToken() *agentauth.ExtendedToken {
	now := time.Now()

	return &agentauth.ExtendedToken{
		AccessToken: "test-token-123",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Scope:       []string{"mcp:resource:read", "mcp:tool:call", "mcp:prompt:get"},
		IssuedAt:    now,

		AuthorizationChain: &agentauth.AuthorizationChain{
			OwnersAuthorizer: &agentauth.AuthorizationLink{
				EntityID:         "authorizer-1",
				EntityType:       "natural_person",
				EntityName:       "John Doe",
				Role:             "authorizer",
				Status:           "active",
				ValidFrom:        now.Add(-24 * time.Hour),
				ValidUntil:       now.Add(365 * 24 * time.Hour),
				IdentityVerified: true,
			},
			ClientOwner: &agentauth.AuthorizationLink{
				EntityID:         "owner-1",
				EntityType:       "organization",
				EntityName:       "AI Corp",
				Role:             "owner",
				AuthorizedBy:     "authorizer-1",
				Status:           "active",
				ValidFrom:        now.Add(-24 * time.Hour),
				ValidUntil:       now.Add(365 * 24 * time.Hour),
				IdentityVerified: true,
			},
			Client: &agentauth.AuthorizationLink{
				EntityID:         "client-1",
				EntityType:       "ai_system",
				EntityName:       "AI Agent",
				Role:             "client",
				AuthorizedBy:     "owner-1",
				Status:           "active",
				ValidFrom:        now.Add(-24 * time.Hour),
				ValidUntil:       now.Add(365 * 24 * time.Hour),
				IdentityVerified: true,
			},
			ChainValidated: true,
			ValidationTime: now,
			ChainDepth:     3,
		},

		ClientOwner: &agentauth.ClientOwnerInfo{
			OwnerID:          "owner-1",
			OwnerName:        "AI Corp",
			OwnerType:        "organization",
			IdentityVerified: true,
			VerificationDate: now,
		},

		OwnersAuthorizer: &agentauth.OwnersAuthorizerInfo{
			AuthorizerID:     "authorizer-1",
			AuthorizerName:   "John Doe",
			AuthorizerType:   "managing_director",
			IdentityVerified: true,
			VerificationDate: now,
		},

		LegalFramework: &agentauth.LegalFrameworkInfo{
			ApplicableLaws: []string{"GDPR", "Company Law"},
			Jurisdiction:   "DE",
		},

		IssuedBy: &agentauth.AuthorizationServerInfo{
			ServerID:  "gauth-server-1",
			ServerURL: "https://auth.example.com",
			IssueTime: now,
		},

		VerificationProof: &agentauth.IdentityVerificationChain{
			ChainID:             "verify-chain-1",
			OverallVerification: "verified",
			VerificationTime:    now,
		},

		JurisdictionContext: &agentauth.JurisdictionContext{
			PrimaryJurisdiction: "DE",
			ApplicableLaws:      []string{"GDPR"},
		},

		ComplianceLevel: "high",

		PowerOfAttorney: &poa.PoADefinition{
			Parties: poa.Parties{
				Principal: poa.Principal{
					Type:     "organization",
					Identity: "AI Corp",
				},
				AuthorizedClient: poa.AuthorizedClient{
					Identity: "client-1",
					TypeEnum: poa.ClientTypeLLM,
				},
			},
			Authorization: poa.AuthorizationScope{},
			Requirements:  poa.Requirements{},
		},

		Restrictions: []agentauth.PowerRestriction{
			{
				RestrictionType:  "value_limit",
				Description:      "Max transaction value 10000 EUR",
				Value:            10000.0,
				EnforcementLevel: "mandatory",
			},
		},
	}
}

func TestAuthorizationBridge_AuthorizeResourceRead_Success(t *testing.T) {
	mockPDP := &MockPDPEngine{allowDecision: true, decisionReason: "policy allows"}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	ctx := context.Background()

	allowed, err := bridge.AuthorizeResourceRead(ctx, token, "file:///data/customers.db")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatal("expected authorization to be allowed")
	}
}

func TestAuthorizationBridge_AuthorizeResourceRead_MissingScope(t *testing.T) {
	mockPDP := &MockPDPEngine{allowDecision: true, decisionReason: "policy allows"}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	token.Scope = []string{"other:scope"} // No MCP scope
	ctx := context.Background()

	allowed, err := bridge.AuthorizeResourceRead(ctx, token, "file:///data/customers.db")

	if err == nil {
		t.Fatal("expected error for missing scope")
	}
	if allowed {
		t.Fatal("expected authorization to be denied")
	}
}

func TestAuthorizationBridge_AuthorizeResourceRead_PDPDenies(t *testing.T) {
	mockPDP := &MockPDPEngine{allowDecision: false, decisionReason: "policy denies"}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	ctx := context.Background()

	allowed, err := bridge.AuthorizeResourceRead(ctx, token, "file:///data/sensitive.db")

	if err == nil {
		t.Fatal("expected error when PDP denies")
	}
	if allowed {
		t.Fatal("expected authorization to be denied")
	}
}

func TestAuthorizationBridge_AuthorizeResourceRead_ExpiredToken(t *testing.T) {
	mockPDP := &MockPDPEngine{allowDecision: true, decisionReason: "policy allows"}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	token.IssuedAt = time.Now().Add(-2 * time.Hour)
	token.ExpiresIn = 3600 // 1 hour - expired
	ctx := context.Background()

	allowed, err := bridge.AuthorizeResourceRead(ctx, token, "file:///data/customers.db")

	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if allowed {
		t.Fatal("expected authorization to be denied for expired token")
	}
}

func TestAuthorizationBridge_AuthorizeToolCall_Success(t *testing.T) {
	mockPDP := &MockPDPEngine{allowDecision: true, decisionReason: "policy allows"}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	ctx := context.Background()

	allowed, err := bridge.AuthorizeToolCall(ctx, token, "calculator", map[string]interface{}{
		"expression": "2+2",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatal("expected authorization to be allowed")
	}
}

func TestAuthorizationBridge_AuthorizeToolCall_ValueRestriction(t *testing.T) {
	mockPDP := &MockPDPEngine{allowDecision: true, decisionReason: "policy allows"}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	ctx := context.Background()

	// Tool with value exceeding restriction
	allowed, err := bridge.AuthorizeToolCall(ctx, token, "payment_tool", map[string]interface{}{
		"amount": 15000.0, // Exceeds 10000 limit
	})

	if err == nil {
		t.Fatal("expected error for value restriction violation")
	}
	if allowed {
		t.Fatal("expected authorization to be denied")
	}
}

func TestAuthorizationBridge_AuthorizeToolCall_WithinValueRestriction(t *testing.T) {
	mockPDP := &MockPDPEngine{allowDecision: true, decisionReason: "policy allows"}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	ctx := context.Background()

	// Tool with value within restriction
	allowed, err := bridge.AuthorizeToolCall(ctx, token, "payment_tool", map[string]interface{}{
		"amount": 5000.0, // Within 10000 limit
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatal("expected authorization to be allowed")
	}
}

func TestAuthorizationBridge_AuthorizePromptGet_Success(t *testing.T) {
	mockPDP := &MockPDPEngine{allowDecision: true, decisionReason: "policy allows"}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	ctx := context.Background()

	allowed, err := bridge.AuthorizePromptGet(ctx, token, "customer_service_template")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatal("expected authorization to be allowed")
	}
}

func TestAuthorizationBridge_ExtractMCPScopes(t *testing.T) {
	mockPDP := &MockPDPEngine{}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	token.Scope = []string{
		"mcp:resource:read",
		"mcp:tool:call",
		"openid",
		"profile",
		"mcp:prompt:get",
	}

	mcpScopes := bridge.ExtractMCPScopes(token)

	if len(mcpScopes) != 3 {
		t.Fatalf("expected 3 MCP scopes, got %d", len(mcpScopes))
	}

	expectedScopes := map[string]bool{
		"mcp:resource:read": true,
		"mcp:tool:call":     true,
		"mcp:prompt:get":    true,
	}

	for _, scope := range mcpScopes {
		if !expectedScopes[scope] {
			t.Fatalf("unexpected MCP scope: %s", scope)
		}
	}
}

func TestAuthorizationBridge_ValidateMCPScopes_Valid(t *testing.T) {
	mockPDP := &MockPDPEngine{}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()

	err := bridge.ValidateMCPScopes(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAuthorizationBridge_ValidateMCPScopes_NoScopes(t *testing.T) {
	mockPDP := &MockPDPEngine{}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	token.Scope = []string{"openid", "profile"} // No MCP scopes

	err := bridge.ValidateMCPScopes(token)
	if err == nil {
		t.Fatal("expected error for missing MCP scopes")
	}
}

func TestAuthorizationBridge_ValidateMCPScopes_InvalidFormat(t *testing.T) {
	mockPDP := &MockPDPEngine{}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	token.Scope = []string{"mcp:invalid:scope:format"}

	err := bridge.ValidateMCPScopes(token)
	if err == nil {
		t.Fatal("expected error for invalid MCP scope format")
	}
}

func TestAuthorizationBridge_WildcardScopes(t *testing.T) {
	mockPDP := &MockPDPEngine{allowDecision: true, decisionReason: "policy allows"}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	token.Scope = []string{"mcp:*"} // Wildcard allows all MCP operations
	ctx := context.Background()

	// Test resource read
	allowed, err := bridge.AuthorizeResourceRead(ctx, token, "file:///data/test.txt")
	if err != nil {
		t.Fatalf("expected no error for resource read, got %v", err)
	}
	if !allowed {
		t.Fatal("expected resource read to be allowed with wildcard scope")
	}

	// Test tool call
	allowed, err = bridge.AuthorizeToolCall(ctx, token, "calculator", map[string]interface{}{})
	if err != nil {
		t.Fatalf("expected no error for tool call, got %v", err)
	}
	if !allowed {
		t.Fatal("expected tool call to be allowed with wildcard scope")
	}

	// Test prompt get
	allowed, err = bridge.AuthorizePromptGet(ctx, token, "test_prompt")
	if err != nil {
		t.Fatalf("expected no error for prompt get, got %v", err)
	}
	if !allowed {
		t.Fatal("expected prompt get to be allowed with wildcard scope")
	}
}

func TestAuthorizationBridge_AuthorizeWithDetails(t *testing.T) {
	mockPDP := &MockPDPEngine{
		allowDecision:  true,
		decisionReason: "policy allows access",
		obligations: []pdp.Obligation{
			{ID: "log_access", Mandatory: true},
			{ID: "notify_admin", Mandatory: false},
		},
	}
	bridge := NewAuthorizationBridge(mockPDP)

	token := createTestToken()
	ctx := context.Background()

	result, err := bridge.AuthorizeWithDetails(
		ctx,
		token,
		"mcp:read_resource",
		"file:///data/test.txt",
		nil,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.Allowed {
		t.Fatal("expected authorization to be allowed")
	}

	if result.Reason != "policy allows access" {
		t.Fatalf("expected reason 'policy allows access', got '%s'", result.Reason)
	}

	if result.Decision != "permit" {
		t.Fatalf("expected decision 'permit', got '%s'", result.Decision)
	}

	if len(result.Obligations) != 2 {
		t.Fatalf("expected 2 obligations, got %d", len(result.Obligations))
	}

	if result.TokenID != token.AccessToken {
		t.Fatalf("expected token ID '%s', got '%s'", token.AccessToken, result.TokenID)
	}

	if result.ClientID != token.AuthorizationChain.Client.EntityID {
		t.Fatalf("expected client ID '%s', got '%s'", token.AuthorizationChain.Client.EntityID, result.ClientID)
	}
}

func TestAuthorizationBridge_ExtractResourceType(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"file:///data/test.txt", "file"},
		{"db://localhost/mydb", "database"},
		{"postgres://localhost:5432/mydb", "database"},
		{"http://api.example.com/resource", "http"},
		{"https://api.example.com/resource", "http"},
		{"mcp://server/resource", "mcp"},
		{"unknown://resource", "unknown"},
	}

	for _, tt := range tests {
		result := extractResourceType(tt.uri)
		if result != tt.expected {
			t.Errorf("extractResourceType(%s) = %s, expected %s", tt.uri, result, tt.expected)
		}
	}
}

func TestAuthorizationBridge_IsMonetaryTool(t *testing.T) {
	tests := []struct {
		toolName string
		expected bool
	}{
		{"payment_processor", true},
		{"transfer_funds", true},
		{"create_invoice", true},
		{"purchase_item", true},
		{"transaction_handler", true},
		{"calculator", false},
		{"search_engine", false},
		{"data_analyzer", false},
	}

	for _, tt := range tests {
		result := isMonetaryTool(tt.toolName)
		if result != tt.expected {
			t.Errorf("isMonetaryTool(%s) = %v, expected %v", tt.toolName, result, tt.expected)
		}
	}
}
