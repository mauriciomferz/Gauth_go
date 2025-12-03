// Package gagent - Tests for MCP Agent Integration
package gagent

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/mcp"
	"github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// Mock MCP Client for testing
type mockMCPClient struct {
	listResourcesFunc func(ctx context.Context) (*mcp.ResourcesListResponse, error)
	readResourceFunc  func(ctx context.Context, uri string) (*mcp.ResourceReadResponse, error)
	listToolsFunc     func(ctx context.Context) (*mcp.ToolsListResponse, error)
	callToolFunc      func(ctx context.Context, name string, args map[string]interface{}) (*mcp.ToolCallResponse, error)
	listPromptsFunc   func(ctx context.Context) (*mcp.PromptsListResponse, error)
	getPromptFunc     func(ctx context.Context, name string, args map[string]string) (*mcp.PromptGetResponse, error)
}

func (m *mockMCPClient) ListResources(ctx context.Context) (*mcp.ResourcesListResponse, error) {
	if m.listResourcesFunc != nil {
		return m.listResourcesFunc(ctx)
	}
	return &mcp.ResourcesListResponse{
		Resources: []mcp.Resource{
			{URI: "file:///test.txt", Name: "Test File"},
		},
	}, nil
}

func (m *mockMCPClient) ReadResource(ctx context.Context, uri string) (*mcp.ResourceReadResponse, error) {
	if m.readResourceFunc != nil {
		return m.readResourceFunc(ctx, uri)
	}
	return &mcp.ResourceReadResponse{
		Contents: []mcp.ResourceContent{
			{URI: uri, Text: "test content"},
		},
	}, nil
}

func (m *mockMCPClient) ListTools(ctx context.Context) (*mcp.ToolsListResponse, error) {
	if m.listToolsFunc != nil {
		return m.listToolsFunc(ctx)
	}
	return &mcp.ToolsListResponse{
		Tools: []mcp.Tool{
			{Name: "calculator", Description: "Math calculator"},
		},
	}, nil
}

func (m *mockMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*mcp.ToolCallResponse, error) {
	if m.callToolFunc != nil {
		return m.callToolFunc(ctx, name, args)
	}
	return &mcp.ToolCallResponse{
		Content: []mcp.ToolContent{
			{Type: "text", Text: "42"},
		},
	}, nil
}

func (m *mockMCPClient) ListPrompts(ctx context.Context) (*mcp.PromptsListResponse, error) {
	if m.listPromptsFunc != nil {
		return m.listPromptsFunc(ctx)
	}
	return &mcp.PromptsListResponse{
		Prompts: []mcp.Prompt{
			{Name: "greeting", Description: "Greeting prompt"},
		},
	}, nil
}

func (m *mockMCPClient) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.PromptGetResponse, error) {
	if m.getPromptFunc != nil {
		return m.getPromptFunc(ctx, name, args)
	}
	return &mcp.PromptGetResponse{
		Messages: []mcp.PromptMessage{
			{Role: "user", Content: "Hello!"},
		},
	}, nil
}

func (m *mockMCPClient) Close() error {
	return nil
}

// Mock Authorization Bridge for testing
type mockAuthBridge struct {
	authorizeResourceReadFunc func(ctx context.Context, token *gauth.ExtendedToken, uri string) (bool, error)
	authorizeToolCallFunc     func(ctx context.Context, token *gauth.ExtendedToken, name string, args map[string]interface{}) (bool, error)
	authorizePromptGetFunc    func(ctx context.Context, token *gauth.ExtendedToken, name string) (bool, error)
}

func (m *mockAuthBridge) AuthorizeResourceRead(ctx context.Context, token *gauth.ExtendedToken, uri string) (bool, error) {
	if m.authorizeResourceReadFunc != nil {
		return m.authorizeResourceReadFunc(ctx, token, uri)
	}
	return true, nil
}

func (m *mockAuthBridge) AuthorizeToolCall(ctx context.Context, token *gauth.ExtendedToken, name string, args map[string]interface{}) (bool, error) {
	if m.authorizeToolCallFunc != nil {
		return m.authorizeToolCallFunc(ctx, token, name, args)
	}
	return true, nil
}

func (m *mockAuthBridge) AuthorizePromptGet(ctx context.Context, token *gauth.ExtendedToken, name string) (bool, error) {
	if m.authorizePromptGetFunc != nil {
		return m.authorizePromptGetFunc(ctx, token, name)
	}
	return true, nil
}

// createTestToken creates a valid test token
func createTestToken() *gauth.ExtendedToken {
	now := time.Now()
	return &gauth.ExtendedToken{
		AccessToken: "test_access_token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Scope: []string{
			"mcp:resource:read",
			"mcp:tool:call",
			"mcp:prompt:get",
		},
		IssuedAt:        now,
		RequestID:       "req-123",
		GrantID:         "grant-456",
		PowerOfAttorney: &poa.PoADefinition{
			// Minimal PoA for testing
		},
		ClientOwner: &gauth.ClientOwnerInfo{
			OwnerID:          "client-owner-1",
			OwnerName:        "Test Owner Org",
			OwnerType:        "organization",
			IdentityVerified: true,
			VerificationDate: now,
		},
		OwnersAuthorizer: &gauth.OwnersAuthorizerInfo{
			AuthorizerID:        "authorizer-1",
			AuthorizerName:      "Test Board",
			AuthorizerType:      "board_member",
			StatutoryAuthority:  "Managing Director",
			IdentityVerified:    true,
			VerificationDate:    now,
			RelationshipToOwner: "board",
			AuthorizationBasis:  "statutory",
		},
		ResourceOwner: &gauth.ResourceOwnerInfo{
			OwnerID:          "resource-owner-1",
			OwnerName:        "Test Resource Owner",
			OwnerType:        "organization",
			IdentityVerified: true,
			VerificationDate: now,
		},
		AuthorizationChain: &gauth.AuthorizationChain{
			OwnersAuthorizer: &gauth.AuthorizationLink{
				EntityID:          "authorizer-1",
				EntityType:        "organization",
				EntityName:        "Test Board",
				Role:              "authorizer",
				AuthorizedBy:      "", // Root of chain
				AuthorizationDate: now,
				AuthorizationType: "statutory",
				IdentityVerified:  true,
				ScopeOfAuthority:  []string{"authorize_agents"},
				ValidFrom:         now.Add(-24 * time.Hour), // Started yesterday
				ValidUntil:        now.Add(365 * 24 * time.Hour),
				Status:            "active",
			},
			ClientOwner: &gauth.AuthorizationLink{
				EntityID:          "client-owner-1",
				EntityType:        "organization",
				EntityName:        "Test Owner Org",
				Role:              "owner",
				AuthorizedBy:      "authorizer-1", // Authorized by owner's authorizer
				AuthorizationDate: now,
				AuthorizationType: "contractual",
				IdentityVerified:  true,
				ScopeOfAuthority:  []string{"own_ai_systems"},
				ValidFrom:         now.Add(-24 * time.Hour), // Started yesterday
				ValidUntil:        now.Add(365 * 24 * time.Hour),
				Status:            "active",
			},
			Client: &gauth.AuthorizationLink{
				EntityID:          "client-1",
				EntityType:        "ai_agent",
				EntityName:        "Test AI Agent",
				Role:              "client",
				AuthorizedBy:      "client-owner-1", // Authorized by client owner
				AuthorizationDate: now,
				AuthorizationType: "delegated",
				IdentityVerified:  true,
				ScopeOfAuthority:  []string{"mcp_operations"},
				ValidFrom:         now.Add(-24 * time.Hour), // Started yesterday
				ValidUntil:        now.Add(365 * 24 * time.Hour),
				Status:            "active",
			},
			ChainValidated: true,
			ValidationTime: now,
			ValidatorID:    "validator-1",
		},
		LegalFramework: &gauth.LegalFrameworkInfo{
			Jurisdiction: "US",
		},
		IssuedBy: &gauth.AuthorizationServerInfo{
			ServerID:  "gauth-server-1",
			ServerURL: "https://auth.example.com",
			Issuer:    "GAuth Authorization Server",
			IssueTime: now,
		},
		VerificationProof: &gauth.IdentityVerificationChain{
			ChainID:             "verification-chain-1",
			OverallVerification: "verified",
			VerificationTime:    now,
			VerifierEntity:      "test-verifier",
		},
	}
}

func TestNewMCPAgent(t *testing.T) {
	token := createTestToken()
	client := &mockMCPClient{}
	bridge := &mockAuthBridge{}

	tests := []struct {
		name    string
		config  *MCPAgentConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &MCPAgentConfig{
				AgentID:    "agent-1",
				Token:      token,
				MCPClient:  client,
				AuthBridge: bridge,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty agent ID",
			config: &MCPAgentConfig{
				AgentID:    "",
				Token:      token,
				MCPClient:  client,
				AuthBridge: bridge,
			},
			wantErr: true,
		},
		{
			name: "nil token",
			config: &MCPAgentConfig{
				AgentID:    "agent-1",
				Token:      nil,
				MCPClient:  client,
				AuthBridge: bridge,
			},
			wantErr: true,
		},
		{
			name: "nil MCP client",
			config: &MCPAgentConfig{
				AgentID:    "agent-1",
				Token:      token,
				MCPClient:  nil,
				AuthBridge: bridge,
			},
			wantErr: true,
		},
		{
			name: "nil auth bridge",
			config: &MCPAgentConfig{
				AgentID:    "agent-1",
				Token:      token,
				MCPClient:  client,
				AuthBridge: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewMCPAgent(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMCPAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && agent == nil {
				t.Error("NewMCPAgent() returned nil agent")
			}
		})
	}
}

func TestMCPAgent_ReadResource(t *testing.T) {
	token := createTestToken()
	client := &mockMCPClient{}
	bridge := &mockAuthBridge{}
	auditLogger := mcp.NewInMemoryAuditLogger(100)

	config := &MCPAgentConfig{
		AgentID:     "agent-1",
		Token:       token,
		MCPClient:   client,
		AuthBridge:  bridge,
		AuditLogger: auditLogger,
	}

	agent, err := NewMCPAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()

	t.Run("successful read", func(t *testing.T) {
		response, err := agent.ReadResource(ctx, "file:///test.txt")
		if err != nil {
			t.Errorf("ReadResource() error = %v", err)
			return
		}
		if len(response.Contents) == 0 {
			t.Error("ReadResource() returned empty contents")
		}

		// Check audit log
		entries := auditLogger.GetAllEntries()
		if len(entries) == 0 {
			t.Error("No audit log entries created")
		}
	})

	t.Run("authorization denied", func(t *testing.T) {
		bridge.authorizeResourceReadFunc = func(ctx context.Context, token *gauth.ExtendedToken, uri string) (bool, error) {
			return false, nil
		}

		_, err := agent.ReadResource(ctx, "file:///denied.txt")
		if err == nil {
			t.Error("ReadResource() should have failed with authorization error")
		}
	})
}

func TestMCPAgent_CallTool(t *testing.T) {
	token := createTestToken()
	client := &mockMCPClient{}
	bridge := &mockAuthBridge{}
	auditLogger := mcp.NewInMemoryAuditLogger(100)

	config := &MCPAgentConfig{
		AgentID:     "agent-1",
		Token:       token,
		MCPClient:   client,
		AuthBridge:  bridge,
		AuditLogger: auditLogger,
	}

	agent, err := NewMCPAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()

	t.Run("successful call", func(t *testing.T) {
		args := map[string]interface{}{
			"expression": "2 + 2",
		}

		response, err := agent.CallTool(ctx, "calculator", args)
		if err != nil {
			t.Errorf("CallTool() error = %v", err)
			return
		}
		if len(response.Content) == 0 {
			t.Error("CallTool() returned empty content")
		}

		// Check audit log
		entries := auditLogger.GetAllEntries()
		if len(entries) == 0 {
			t.Error("No audit log entries created")
		}
	})

	t.Run("authorization denied", func(t *testing.T) {
		bridge.authorizeToolCallFunc = func(ctx context.Context, token *gauth.ExtendedToken, name string, args map[string]interface{}) (bool, error) {
			return false, nil
		}

		args := map[string]interface{}{"test": "value"}
		_, err := agent.CallTool(ctx, "restricted-tool", args)
		if err == nil {
			t.Error("CallTool() should have failed with authorization error")
		}
	})
}

func TestMCPAgent_GetPrompt(t *testing.T) {
	token := createTestToken()
	client := &mockMCPClient{}
	bridge := &mockAuthBridge{}
	auditLogger := mcp.NewInMemoryAuditLogger(100)

	config := &MCPAgentConfig{
		AgentID:     "agent-1",
		Token:       token,
		MCPClient:   client,
		AuthBridge:  bridge,
		AuditLogger: auditLogger,
	}

	agent, err := NewMCPAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		args := map[string]string{
			"name": "Alice",
		}

		response, err := agent.GetPrompt(ctx, "greeting", args)
		if err != nil {
			t.Errorf("GetPrompt() error = %v", err)
			return
		}
		if len(response.Messages) == 0 {
			t.Error("GetPrompt() returned empty messages")
		}

		// Check audit log
		entries := auditLogger.GetAllEntries()
		if len(entries) == 0 {
			t.Error("No audit log entries created")
		}
	})

	t.Run("authorization denied", func(t *testing.T) {
		bridge.authorizePromptGetFunc = func(ctx context.Context, token *gauth.ExtendedToken, name string) (bool, error) {
			return false, nil
		}

		args := map[string]string{}
		_, err := agent.GetPrompt(ctx, "restricted-prompt", args)
		if err == nil {
			t.Error("GetPrompt() should have failed with authorization error")
		}
	})
}

func TestMCPAgent_ListOperations(t *testing.T) {
	token := createTestToken()
	client := &mockMCPClient{}
	bridge := &mockAuthBridge{}

	config := &MCPAgentConfig{
		AgentID:    "agent-1",
		Token:      token,
		MCPClient:  client,
		AuthBridge: bridge,
	}

	agent, err := NewMCPAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()

	t.Run("list resources", func(t *testing.T) {
		response, err := agent.ListResources(ctx)
		if err != nil {
			t.Errorf("ListResources() error = %v", err)
			return
		}
		if len(response.Resources) == 0 {
			t.Error("ListResources() returned empty list")
		}
	})

	t.Run("list tools", func(t *testing.T) {
		response, err := agent.ListTools(ctx)
		if err != nil {
			t.Errorf("ListTools() error = %v", err)
			return
		}
		if len(response.Tools) == 0 {
			t.Error("ListTools() returned empty list")
		}
	})

	t.Run("list prompts", func(t *testing.T) {
		response, err := agent.ListPrompts(ctx)
		if err != nil {
			t.Errorf("ListPrompts() error = %v", err)
			return
		}
		if len(response.Prompts) == 0 {
			t.Error("ListPrompts() returned empty list")
		}
	})
}

func TestMCPAgent_AuditLogging(t *testing.T) {
	token := createTestToken()
	client := &mockMCPClient{}
	bridge := &mockAuthBridge{}
	auditLogger := mcp.NewInMemoryAuditLogger(100)

	config := &MCPAgentConfig{
		AgentID:     "agent-1",
		Token:       token,
		MCPClient:   client,
		AuthBridge:  bridge,
		AuditLogger: auditLogger,
	}

	agent, err := NewMCPAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx := context.Background()

	// Perform several operations
	_, _ = agent.ReadResource(ctx, "file:///test1.txt")
	_, _ = agent.CallTool(ctx, "calculator", map[string]interface{}{"x": 1})
	_, _ = agent.GetPrompt(ctx, "greeting", map[string]string{})

	// Check audit logs
	entries := auditLogger.GetAllEntries()
	if len(entries) != 3 {
		t.Errorf("Expected 3 audit entries, got %d", len(entries))
	}

	// Verify audit entry fields
	for _, entry := range entries {
		if entry.AgentID != "agent-1" {
			t.Errorf("Audit entry has wrong agent ID: %s", entry.AgentID)
		}
		if entry.RequestID != token.RequestID {
			t.Errorf("Audit entry has wrong request ID: %s", entry.RequestID)
		}
		if !entry.Authorized {
			t.Error("Audit entry should be authorized")
		}
		if entry.Duration == 0 {
			t.Error("Audit entry should have duration")
		}
	}
}

func TestMCPAgent_GettersAndClosures(t *testing.T) {
	token := createTestToken()
	client := &mockMCPClient{}
	bridge := &mockAuthBridge{}

	config := &MCPAgentConfig{
		AgentID:    "agent-test-123",
		Token:      token,
		MCPClient:  client,
		AuthBridge: bridge,
	}

	agent, err := NewMCPAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	t.Run("get token", func(t *testing.T) {
		gotToken := agent.GetToken()
		if gotToken != token {
			t.Error("GetToken() returned different token")
		}
	})

	t.Run("get agent ID", func(t *testing.T) {
		gotID := agent.GetAgentID()
		if gotID != "agent-test-123" {
			t.Errorf("GetAgentID() = %s, want %s", gotID, "agent-test-123")
		}
	})

	t.Run("close", func(t *testing.T) {
		err := agent.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}
