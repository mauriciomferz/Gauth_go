package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test JSON-RPC message parsing
func TestJSONRPCParsing(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMethod string
		wantID     int64
		wantError  bool
	}{
		{
			name:       "Valid initialize request",
			input:      `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
			wantMethod: "initialize",
			wantID:     1,
			wantError:  false,
		},
		{
			name:       "Valid resources/list request",
			input:      `{"jsonrpc":"2.0","id":2,"method":"resources/list","params":{}}`,
			wantMethod: "resources/list",
			wantID:     2,
			wantError:  false,
		},
		{
			name:      "Invalid JSON",
			input:     `{"jsonrpc":"2.0","id":1,`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req JSONRPCRequest
			err := json.Unmarshal([]byte(tt.input), &req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if req.Method != tt.wantMethod {
				t.Errorf("Expected method '%s', got '%s'", tt.wantMethod, req.Method)
			}

			if req.ID != tt.wantID {
				t.Errorf("Expected ID %d, got %d", tt.wantID, req.ID)
			}
		})
	}
}

// Test SSE transport connection
func TestSSETransportConnect(t *testing.T) {
	// Create a test server that sends SSE events
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Send test event
		_, _ = w.Write([]byte("data: {\"jsonrpc\":\"2.0\",\"method\":\"ping\"}\n\n"))
		flusher.Flush()

		// Keep connection open briefly
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	// Create SSE transport
	transport := NewSSETransport(server.URL, nil)

	// Start connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Give it time to receive message
	time.Sleep(200 * time.Millisecond)

	// Close
	err = transport.Close()
	if err != nil {
		t.Errorf("Failed to close: %v", err)
	}
}

// Test resource structure
func TestResourceStructure(t *testing.T) {
	resource := Resource{
		URI:         "file:///test.txt",
		Name:        "test.txt",
		Description: "Test file",
		MimeType:    "text/plain",
		Metadata: map[string]string{
			"size": "100",
		},
	}

	// Serialize to JSON
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}

	// Deserialize back
	var decoded Resource
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal resource: %v", err)
	}

	if decoded.URI != resource.URI {
		t.Errorf("Expected URI '%s', got '%s'", resource.URI, decoded.URI)
	}
	if decoded.Name != resource.Name {
		t.Errorf("Expected Name '%s', got '%s'", resource.Name, decoded.Name)
	}
}

// Test error response structure
func TestJSONRPCError(t *testing.T) {
	errorResp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: &JSONRPCError{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}

	// Serialize
	data, err := json.Marshal(errorResp)
	if err != nil {
		t.Fatalf("Failed to marshal error: %v", err)
	}

	// Deserialize
	var decoded JSONRPCResponse
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal error: %v", err)
	}

	if decoded.Error == nil {
		t.Fatal("Expected error to be present")
	}
	if decoded.Error.Code != -32600 {
		t.Errorf("Expected code -32600, got %d", decoded.Error.Code)
	}
	if decoded.Error.Message != "Invalid Request" {
		t.Errorf("Expected message 'Invalid Request', got '%s'", decoded.Error.Message)
	}
}

// Benchmark JSON-RPC parsing
func BenchmarkJSONRPCParsing(b *testing.B) {
	input := []byte(`{"jsonrpc":"2.0","id":1,"method":"resources/list","params":{}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req JSONRPCRequest
		_ = json.Unmarshal(input, &req)
	}
}

// Benchmark resource marshaling
func BenchmarkResourceMarshal(b *testing.B) {
	resource := Resource{
		URI:         "file:///test.txt",
		Name:        "test.txt",
		Description: "Test file",
		MimeType:    "text/plain",
		Metadata: map[string]string{
			"size": "100",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(resource)
	}
}
