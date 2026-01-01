// Package gagent - AgentAuth Agent Integration with MCP
// Provides high-level API for AI agents to use MCP resources with authorization
package gagent

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/mcp"
)

// MCPClient defines the interface for MCP protocol operations
type MCPClient interface {
	ListResources(ctx context.Context) (*mcp.ResourcesListResponse, error)
	ReadResource(ctx context.Context, uri string) (*mcp.ResourceReadResponse, error)
	ListTools(ctx context.Context) (*mcp.ToolsListResponse, error)
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*mcp.ToolCallResponse, error)
	ListPrompts(ctx context.Context) (*mcp.PromptsListResponse, error)
	GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.PromptGetResponse, error)
	Close() error
}

// AuthorizationBridge defines the interface for MCP authorization
type AuthorizationBridge interface {
	AuthorizeResourceRead(ctx context.Context, token *agentauth.ExtendedToken, uri string) (bool, error)
	AuthorizeToolCall(ctx context.Context, token *agentauth.ExtendedToken, name string, args map[string]interface{}) (bool, error)
	AuthorizePromptGet(ctx context.Context, token *agentauth.ExtendedToken, name string) (bool, error)
}

// MCPAgent is a high-level wrapper for AI agents to access MCP resources
// with automatic AgentAuth authorization enforcement
type MCPAgent struct {
	mcpClient   MCPClient
	authBridge  AuthorizationBridge
	auditLogger mcp.AuditLogger
	agentID     string
	token       *agentauth.ExtendedToken
}

// MCPAgentConfig configures an MCP agent instance
type MCPAgentConfig struct {
	AgentID     string                   // Unique agent identifier
	Token       *agentauth.ExtendedToken // AgentAuth authorization token
	MCPClient   MCPClient                // MCP client for server communication
	AuthBridge  AuthorizationBridge      // Authorization bridge
	AuditLogger mcp.AuditLogger          // Audit logger (optional)
}

// NewMCPAgent creates a new MCP agent with authorization
func NewMCPAgent(config *MCPAgentConfig) (*MCPAgent, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.AgentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}
	if config.Token == nil {
		return nil, fmt.Errorf("token cannot be nil")
	}
	if config.MCPClient == nil {
		return nil, fmt.Errorf("MCP client cannot be nil")
	}
	if config.AuthBridge == nil {
		return nil, fmt.Errorf("authorization bridge cannot be nil")
	}

	// Validate token
	if err := config.Token.Validate(); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	return &MCPAgent{
		mcpClient:   config.MCPClient,
		authBridge:  config.AuthBridge,
		auditLogger: config.AuditLogger,
		agentID:     config.AgentID,
		token:       config.Token,
	}, nil
}

// ReadResource reads an MCP resource with authorization check
func (a *MCPAgent) ReadResource(ctx context.Context, resourceURI string) (*mcp.ResourceReadResponse, error) {
	startTime := time.Now()

	// Authorize resource access
	authorized, err := a.authBridge.AuthorizeResourceRead(ctx, a.token, resourceURI)
	if err != nil {
		a.logAudit(ctx, "resource_read", resourceURI, false, err.Error())
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	if !authorized {
		a.logAudit(ctx, "resource_read", resourceURI, false, "access denied")
		return nil, fmt.Errorf("access denied to resource: %s", resourceURI)
	}

	// Read resource via MCP
	response, err := a.mcpClient.ReadResource(ctx, resourceURI)
	if err != nil {
		a.logAudit(ctx, "resource_read", resourceURI, false, fmt.Sprintf("MCP error: %v", err))
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}

	// Log successful access
	duration := time.Since(startTime)
	a.logAuditSuccess(ctx, "resource_read", resourceURI, duration)

	return response, nil
}

// CallTool invokes an MCP tool with authorization check
func (a *MCPAgent) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*mcp.ToolCallResponse, error) {
	startTime := time.Now()

	// Authorize tool invocation
	authorized, err := a.authBridge.AuthorizeToolCall(ctx, a.token, toolName, arguments)
	if err != nil {
		a.logAudit(ctx, "tool_call", toolName, false, err.Error())
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	if !authorized {
		a.logAudit(ctx, "tool_call", toolName, false, "access denied")
		return nil, fmt.Errorf("access denied to tool: %s", toolName)
	}

	// Call tool via MCP
	response, err := a.mcpClient.CallTool(ctx, toolName, arguments)
	if err != nil {
		a.logAudit(ctx, "tool_call", toolName, false, fmt.Sprintf("MCP error: %v", err))
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}

	// Log successful call
	duration := time.Since(startTime)
	a.logAuditSuccess(ctx, "tool_call", toolName, duration)

	return response, nil
}

// GetPrompt retrieves an MCP prompt with authorization check
func (a *MCPAgent) GetPrompt(ctx context.Context, promptName string, arguments map[string]string) (*mcp.PromptGetResponse, error) {
	startTime := time.Now()

	// Authorize prompt access
	authorized, err := a.authBridge.AuthorizePromptGet(ctx, a.token, promptName)
	if err != nil {
		a.logAudit(ctx, "prompt_get", promptName, false, err.Error())
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	if !authorized {
		a.logAudit(ctx, "prompt_get", promptName, false, "access denied")
		return nil, fmt.Errorf("access denied to prompt: %s", promptName)
	}

	// Get prompt via MCP
	response, err := a.mcpClient.GetPrompt(ctx, promptName, arguments)
	if err != nil {
		a.logAudit(ctx, "prompt_get", promptName, false, fmt.Sprintf("MCP error: %v", err))
		return nil, fmt.Errorf("failed to get prompt: %w", err)
	}

	// Log successful access
	duration := time.Since(startTime)
	a.logAuditSuccess(ctx, "prompt_get", promptName, duration)

	return response, nil
}

// ListResources lists available MCP resources
func (a *MCPAgent) ListResources(ctx context.Context) (*mcp.ResourcesListResponse, error) {
	// List operation typically doesn't require authorization for discovery
	// but the actual reading of resources will be authorized
	response, err := a.mcpClient.ListResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	return response, nil
}

// ListTools lists available MCP tools
func (a *MCPAgent) ListTools(ctx context.Context) (*mcp.ToolsListResponse, error) {
	// List operation typically doesn't require authorization for discovery
	response, err := a.mcpClient.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return response, nil
}

// ListPrompts lists available MCP prompts
func (a *MCPAgent) ListPrompts(ctx context.Context) (*mcp.PromptsListResponse, error) {
	// List operation typically doesn't require authorization for discovery
	response, err := a.mcpClient.ListPrompts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	return response, nil
}

// GetToken returns the agent's current authorization token
func (a *MCPAgent) GetToken() *agentauth.ExtendedToken {
	return a.token
}

// GetAgentID returns the agent's unique identifier
func (a *MCPAgent) GetAgentID() string {
	return a.agentID
}

// logAudit logs an operation with failure
func (a *MCPAgent) logAudit(ctx context.Context, operation, target string, authorized bool, reason string) {
	if a.auditLogger != nil {
		entry := &mcp.AuditLogEntry{
			Timestamp:   time.Now(),
			AgentID:     a.agentID,
			RequestID:   a.token.RequestID,
			GrantID:     a.token.GrantID,
			Operation:   operation,
			Target:      target,
			Authorized:  authorized,
			Decision:    reason,
			TokenScopes: a.token.Scope,
		}
		_ = a.auditLogger.Log(ctx, entry) // Best effort logging
	}
}

// logAuditSuccess logs a successful operation
func (a *MCPAgent) logAuditSuccess(ctx context.Context, operation, target string, duration time.Duration) {
	if a.auditLogger != nil {
		entry := &mcp.AuditLogEntry{
			Timestamp:   time.Now(),
			AgentID:     a.agentID,
			RequestID:   a.token.RequestID,
			GrantID:     a.token.GrantID,
			Operation:   operation,
			Target:      target,
			Authorized:  true,
			Decision:    "granted",
			Duration:    duration,
			TokenScopes: a.token.Scope,
		}
		_ = a.auditLogger.Log(ctx, entry) // Best effort logging
	}
}

// Close closes the underlying MCP client connection
func (a *MCPAgent) Close() error {
	if a.mcpClient != nil {
		return a.mcpClient.Close()
	}
	return nil
}
