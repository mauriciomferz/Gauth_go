package mcp

import (
	"context"
	"fmt"
	"sync"
)

// ServerConfig represents MCP server configuration
type ServerConfig struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	TransportType string            `json:"transport_type"`    // "stdio", "websocket", "http-sse"
	Command       string            `json:"command,omitempty"` // For stdio
	Args          []string          `json:"args,omitempty"`    // For stdio
	URL           string            `json:"url,omitempty"`     // For websocket/http
	RequireAuth   bool              `json:"require_auth"`
	AllowedScopes []string          `json:"allowed_scopes,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// ConnectionManager manages connections to multiple MCP servers
type ConnectionManager struct {
	servers map[string]*ServerConfig
	clients map[string]*MCPClient
	mu      sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		servers: make(map[string]*ServerConfig),
		clients: make(map[string]*MCPClient),
	}
}

// RegisterServer registers an MCP server configuration
func (m *ConnectionManager) RegisterServer(config *ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.ID == "" {
		return fmt.Errorf("server ID is required")
	}

	if config.Name == "" {
		return fmt.Errorf("server name is required")
	}

	if config.TransportType == "" {
		return fmt.Errorf("transport type is required")
	}

	// Validate transport-specific fields
	switch config.TransportType {
	case "stdio":
		if config.Command == "" {
			return fmt.Errorf("command is required for stdio transport")
		}
	case "websocket", "http-sse":
		if config.URL == "" {
			return fmt.Errorf("URL is required for %s transport", config.TransportType)
		}
	default:
		return fmt.Errorf("unsupported transport type: %s", config.TransportType)
	}

	m.servers[config.ID] = config
	return nil
}

// UnregisterServer unregisters an MCP server and closes its connection
func (m *ConnectionManager) UnregisterServer(serverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close existing client connection
	if client, exists := m.clients[serverID]; exists {
		if err := client.Close(); err != nil {
			return fmt.Errorf("failed to close client connection: %w", err)
		}
		delete(m.clients, serverID)
	}

	// Remove server config
	delete(m.servers, serverID)
	return nil
}

// GetClient returns an MCP client for the specified server, creating connection if needed
func (m *ConnectionManager) GetClient(ctx context.Context, serverID string) (*MCPClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if client already exists
	if client, exists := m.clients[serverID]; exists {
		return client, nil
	}

	// Get server config
	config, exists := m.servers[serverID]
	if !exists {
		return nil, fmt.Errorf("server not registered: %s", serverID)
	}

	// Create transport based on type
	var transport Transport
	var err error

	switch config.TransportType {
	case "stdio":
		transport, err = NewStdioTransport(ctx, config.Command, config.Args...)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdio transport: %w", err)
		}
	case "websocket":
		wsTransport := NewWebSocketTransport(config.URL, nil)
		if err := wsTransport.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect websocket transport: %w", err)
		}
		transport = wsTransport
	case "http-sse":
		sseTransport := NewSSETransport(config.URL, nil)
		if err := sseTransport.Connect(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect SSE transport: %w", err)
		}
		transport = sseTransport
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", config.TransportType)
	}

	// Create client
	client := NewMCPClient(config.ID, config.Name, transport)
	m.clients[serverID] = client

	return client, nil
}

// ListServers returns list of registered server IDs
func (m *ConnectionManager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	serverIDs := make([]string, 0, len(m.servers))
	for id := range m.servers {
		serverIDs = append(serverIDs, id)
	}
	return serverIDs
}

// GetServerConfig returns server configuration
func (m *ConnectionManager) GetServerConfig(serverID string) (*ServerConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.servers[serverID]
	if !exists {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	// Return copy to prevent modification
	configCopy := *config
	return &configCopy, nil
}

// CloseAll closes all MCP client connections
func (m *ConnectionManager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []error
	for serverID, client := range m.clients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close %s: %w", serverID, err))
		}
	}

	// Clear clients map
	m.clients = make(map[string]*MCPClient)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}

// GetConnectionStatus returns status of all connections
func (m *ConnectionManager) GetConnectionStatus() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]string)
	for serverID := range m.servers {
		if _, connected := m.clients[serverID]; connected {
			status[serverID] = "connected"
		} else {
			status[serverID] = "disconnected"
		}
	}
	return status
}
