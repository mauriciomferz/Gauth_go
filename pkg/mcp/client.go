package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/mauriciomferz/Gauth_go/internal/security"
)

// MCPClient represents an MCP protocol client
type MCPClient struct {
	serverID   string
	serverName string
	transport  Transport
	requestID  int64
}

// Transport defines MCP communication transport interface
type Transport interface {
	Send(ctx context.Context, message []byte) error
	Receive(ctx context.Context) ([]byte, error)
	Close() error
}

// NewMCPClient creates a new MCP client
func NewMCPClient(serverID, serverName string, transport Transport) *MCPClient {
	return &MCPClient{
		serverID:   serverID,
		serverName: serverName,
		transport:  transport,
		requestID:  1,
	}
}

// ListResources queries available resources from MCP server
func (c *MCPClient) ListResources(ctx context.Context) (*ResourcesListResponse, error) {
	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "resources/list",
	}

	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	var result ResourcesListResponse
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resources list: %w", err)
	}

	return &result, nil
}

// ReadResource reads content from specified resource URI
func (c *MCPClient) ReadResource(ctx context.Context, uri string) (*ResourceReadResponse, error) {
	// SECURITY: Validate URI to prevent SSRF attacks
	// For MCP, we allow mcp:// and https:// schemes
	validator := security.NewURIValidatorWithSchemes([]string{"mcp", "https"})
	if err := validator.ValidateURI(uri); err != nil {
		return nil, fmt.Errorf("URI validation failed (SSRF protection): %w", err)
	}

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "resources/read",
		Params: map[string]interface{}{
			"uri": uri,
		},
	}

	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	var result ResourceReadResponse
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resource content: %w", err)
	}

	return &result, nil
}

// ListTools queries available tools from MCP server
func (c *MCPClient) ListTools(ctx context.Context) (*ToolsListResponse, error) {
	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/list",
	}

	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	var result ToolsListResponse
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools list: %w", err)
	}

	return &result, nil
}

// CallTool invokes a tool with given arguments
func (c *MCPClient) CallTool(ctx context.Context, toolName string,
	arguments map[string]interface{}) (*ToolCallResponse, error) {

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}

	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	var result ToolCallResponse
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool response: %w", err)
	}

	return &result, nil
}

// ListPrompts queries available prompts from MCP server
func (c *MCPClient) ListPrompts(ctx context.Context) (*PromptsListResponse, error) {
	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "prompts/list",
	}

	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	var result PromptsListResponse
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse prompts list: %w", err)
	}

	return &result, nil
}

// GetPrompt retrieves a prompt with given arguments
func (c *MCPClient) GetPrompt(ctx context.Context, promptName string,
	arguments map[string]string) (*PromptGetResponse, error) {

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "prompts/get",
		Params: map[string]interface{}{
			"name":      promptName,
			"arguments": arguments,
		},
	}

	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	var result PromptGetResponse
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse prompt response: %w", err)
	}

	return &result, nil
}

// Close closes the MCP client connection
func (c *MCPClient) Close() error {
	return c.transport.Close()
}

// ServerID returns the server identifier
func (c *MCPClient) ServerID() string {
	return c.serverID
}

// ServerName returns the server name
func (c *MCPClient) ServerName() string {
	return c.serverName
}

// sendRequest sends JSON-RPC request and waits for response
func (c *MCPClient) sendRequest(ctx context.Context,
	request *JSONRPCRequest) (*JSONRPCResponse, error) {

	// Marshal request
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
	if err := c.transport.Send(ctx, requestBytes); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	responseBytes, err := c.transport.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	// Parse response
	var response JSONRPCResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for error
	if response.Error != nil {
		return nil, fmt.Errorf("MCP error: %s (code %d)",
			response.Error.Message, response.Error.Code)
	}

	// Verify request ID matches
	if response.ID != request.ID {
		return nil, fmt.Errorf("response ID mismatch: expected %d, got %d",
			request.ID, response.ID)
	}

	return &response, nil
}

// nextRequestID generates next request ID (thread-safe)
func (c *MCPClient) nextRequestID() int64 {
	return atomic.AddInt64(&c.requestID, 1) - 1
}
