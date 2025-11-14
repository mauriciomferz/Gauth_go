package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// MockTransport is a mock implementation of Transport for testing
type MockTransport struct {
	sendFunc    func(ctx context.Context, message []byte) error
	receiveFunc func(ctx context.Context) ([]byte, error)
	closeFunc   func() error
	sent        [][]byte
	received    [][]byte
	closed      bool
}

func NewMockTransport() *MockTransport {
	return &MockTransport{
		sent:     make([][]byte, 0),
		received: make([][]byte, 0),
	}
}

func (m *MockTransport) Send(ctx context.Context, message []byte) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, message)
	}
	m.sent = append(m.sent, message)
	return nil
}

func (m *MockTransport) Receive(ctx context.Context) ([]byte, error) {
	if m.receiveFunc != nil {
		return m.receiveFunc(ctx)
	}
	if len(m.received) == 0 {
		return nil, fmt.Errorf("no messages to receive")
	}
	msg := m.received[0]
	m.received = m.received[1:]
	return msg, nil
}

func (m *MockTransport) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	m.closed = true
	return nil
}

func (m *MockTransport) QueueResponse(response *JSONRPCResponse) error {
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return err
	}
	m.received = append(m.received, responseBytes)
	return nil
}

func TestMCPClient_ListResources(t *testing.T) {
	transport := NewMockTransport()
	client := NewMCPClient("test-server", "Test Server", transport)

	// Queue mock response
	mockResponse := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: json.RawMessage(`{
			"resources": [
				{
					"uri": "file:///data/test.txt",
					"name": "Test File",
					"mimeType": "text/plain"
				}
			]
		}`),
	}
	if err := transport.QueueResponse(mockResponse); err != nil {
		t.Fatalf("Failed to queue response: %v", err)
	}

	// Call ListResources
	ctx := context.Background()
	result, err := client.ListResources(ctx)
	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}

	// Verify result
	if len(result.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(result.Resources))
	}

	if result.Resources[0].URI != "file:///data/test.txt" {
		t.Errorf("Expected URI 'file:///data/test.txt', got '%s'", result.Resources[0].URI)
	}

	// Verify request was sent
	if len(transport.sent) != 1 {
		t.Errorf("Expected 1 sent message, got %d", len(transport.sent))
	}

	var sentRequest JSONRPCRequest
	if err := json.Unmarshal(transport.sent[0], &sentRequest); err != nil {
		t.Fatalf("Failed to parse sent request: %v", err)
	}

	if sentRequest.Method != "resources/list" {
		t.Errorf("Expected method 'resources/list', got '%s'", sentRequest.Method)
	}
}

func TestMCPClient_ReadResource(t *testing.T) {
	transport := NewMockTransport()
	client := NewMCPClient("test-server", "Test Server", transport)

	// Queue mock response
	mockResponse := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: json.RawMessage(`{
			"contents": [
				{
					"uri": "file:///data/test.txt",
					"mimeType": "text/plain",
					"text": "Hello, World!"
				}
			]
		}`),
	}
	if err := transport.QueueResponse(mockResponse); err != nil {
		t.Fatalf("Failed to queue response: %v", err)
	}

	// Call ReadResource
	ctx := context.Background()
	result, err := client.ReadResource(ctx, "file:///data/test.txt")
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	// Verify result
	if len(result.Contents) != 1 {
		t.Errorf("Expected 1 content, got %d", len(result.Contents))
	}

	if result.Contents[0].Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", result.Contents[0].Text)
	}
}

func TestMCPClient_ListTools(t *testing.T) {
	transport := NewMockTransport()
	client := NewMCPClient("test-server", "Test Server", transport)

	// Queue mock response
	mockResponse := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: json.RawMessage(`{
			"tools": [
				{
					"name": "calculator",
					"description": "Perform arithmetic calculations",
					"inputSchema": {
						"type": "object",
						"properties": {
							"expression": {"type": "string"}
						}
					}
				}
			]
		}`),
	}
	if err := transport.QueueResponse(mockResponse); err != nil {
		t.Fatalf("Failed to queue response: %v", err)
	}

	// Call ListTools
	ctx := context.Background()
	result, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	// Verify result
	if len(result.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(result.Tools))
	}

	if result.Tools[0].Name != "calculator" {
		t.Errorf("Expected tool name 'calculator', got '%s'", result.Tools[0].Name)
	}
}

func TestMCPClient_CallTool(t *testing.T) {
	transport := NewMockTransport()
	client := NewMCPClient("test-server", "Test Server", transport)

	// Queue mock response
	mockResponse := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: json.RawMessage(`{
			"content": [
				{
					"type": "text",
					"text": "4"
				}
			]
		}`),
	}
	if err := transport.QueueResponse(mockResponse); err != nil {
		t.Fatalf("Failed to queue response: %v", err)
	}

	// Call CallTool
	ctx := context.Background()
	result, err := client.CallTool(ctx, "calculator", map[string]interface{}{
		"expression": "2+2",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	// Verify result
	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Text != "4" {
		t.Errorf("Expected text '4', got '%s'", result.Content[0].Text)
	}
}

func TestMCPClient_ErrorHandling(t *testing.T) {
	transport := NewMockTransport()
	client := NewMCPClient("test-server", "Test Server", transport)

	// Queue error response
	errorResponse := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: &JSONRPCError{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}
	if err := transport.QueueResponse(errorResponse); err != nil {
		t.Fatalf("Failed to queue response: %v", err)
	}

	// Call should return error
	ctx := context.Background()
	_, err := client.ListResources(ctx)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify error message contains MCP error details
	expectedMsg := "MCP error: Invalid Request (code -32600)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestMCPClient_RequestIDIncrement(t *testing.T) {
	transport := NewMockTransport()
	client := NewMCPClient("test-server", "Test Server", transport)

	// Mock responses for multiple requests
	for i := 1; i <= 3; i++ {
		mockResponse := &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      int64(i),
			Result:  json.RawMessage(`{"resources":[]}`),
		}
		if err := transport.QueueResponse(mockResponse); err != nil {
			t.Fatalf("Failed to queue response: %v", err)
		}
	}

	ctx := context.Background()

	// Make multiple requests
	for i := 1; i <= 3; i++ {
		_, err := client.ListResources(ctx)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	// Verify request IDs are sequential
	for i := 0; i < 3; i++ {
		var request JSONRPCRequest
		if err := json.Unmarshal(transport.sent[i], &request); err != nil {
			t.Fatalf("Failed to parse request %d: %v", i, err)
		}

		expectedID := int64(i + 1)
		if request.ID != expectedID {
			t.Errorf("Request %d: expected ID %d, got %d", i, expectedID, request.ID)
		}
	}
}

func TestMCPClient_Close(t *testing.T) {
	transport := NewMockTransport()
	client := NewMCPClient("test-server", "Test Server", transport)

	err := client.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if !transport.closed {
		t.Error("Transport was not closed")
	}
}
