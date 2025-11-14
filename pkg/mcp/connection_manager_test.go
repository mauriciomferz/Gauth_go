package mcp

import (
	"context"
	"testing"
)

func TestConnectionManager_RegisterServer(t *testing.T) {
	manager := NewConnectionManager()

	config := &ServerConfig{
		ID:            "test-server",
		Name:          "Test Server",
		TransportType: "stdio",
		Command:       "/usr/bin/test",
		Args:          []string{"arg1", "arg2"},
	}

	err := manager.RegisterServer(config)
	if err != nil {
		t.Fatalf("RegisterServer failed: %v", err)
	}

	// Verify server was registered
	servers := manager.ListServers()
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}

	if servers[0] != "test-server" {
		t.Errorf("Expected server ID 'test-server', got '%s'", servers[0])
	}
}

func TestConnectionManager_RegisterServer_Validation(t *testing.T) {
	manager := NewConnectionManager()

	tests := []struct {
		name      string
		config    *ServerConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "missing ID",
			config: &ServerConfig{
				Name:          "Test",
				TransportType: "stdio",
				Command:       "/usr/bin/test",
			},
			expectErr: true,
			errMsg:    "server ID is required",
		},
		{
			name: "missing name",
			config: &ServerConfig{
				ID:            "test",
				TransportType: "stdio",
				Command:       "/usr/bin/test",
			},
			expectErr: true,
			errMsg:    "server name is required",
		},
		{
			name: "missing transport type",
			config: &ServerConfig{
				ID:      "test",
				Name:    "Test",
				Command: "/usr/bin/test",
			},
			expectErr: true,
			errMsg:    "transport type is required",
		},
		{
			name: "stdio missing command",
			config: &ServerConfig{
				ID:            "test",
				Name:          "Test",
				TransportType: "stdio",
			},
			expectErr: true,
			errMsg:    "command is required for stdio transport",
		},
		{
			name: "websocket missing URL",
			config: &ServerConfig{
				ID:            "test",
				Name:          "Test",
				TransportType: "websocket",
			},
			expectErr: true,
			errMsg:    "URL is required for websocket transport",
		},
		{
			name: "unsupported transport type",
			config: &ServerConfig{
				ID:            "test",
				Name:          "Test",
				TransportType: "invalid",
			},
			expectErr: true,
			errMsg:    "unsupported transport type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RegisterServer(tt.config)
			if tt.expectErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConnectionManager_UnregisterServer(t *testing.T) {
	manager := NewConnectionManager()

	config := &ServerConfig{
		ID:            "test-server",
		Name:          "Test Server",
		TransportType: "stdio",
		Command:       "/usr/bin/test",
	}

	if err := manager.RegisterServer(config); err != nil {
		t.Fatalf("RegisterServer failed: %v", err)
	}

	// Unregister server
	if err := manager.UnregisterServer("test-server"); err != nil {
		t.Fatalf("UnregisterServer failed: %v", err)
	}

	// Verify server was removed
	servers := manager.ListServers()
	if len(servers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(servers))
	}
}

func TestConnectionManager_GetServerConfig(t *testing.T) {
	manager := NewConnectionManager()

	config := &ServerConfig{
		ID:            "test-server",
		Name:          "Test Server",
		Description:   "A test server",
		TransportType: "stdio",
		Command:       "/usr/bin/test",
		Args:          []string{"arg1"},
		RequireAuth:   true,
		AllowedScopes: []string{"scope1", "scope2"},
	}

	if err := manager.RegisterServer(config); err != nil {
		t.Fatalf("RegisterServer failed: %v", err)
	}

	// Get server config
	retrieved, err := manager.GetServerConfig("test-server")
	if err != nil {
		t.Fatalf("GetServerConfig failed: %v", err)
	}

	// Verify config fields
	if retrieved.ID != config.ID {
		t.Errorf("Expected ID '%s', got '%s'", config.ID, retrieved.ID)
	}
	if retrieved.Name != config.Name {
		t.Errorf("Expected Name '%s', got '%s'", config.Name, retrieved.Name)
	}
	if retrieved.Description != config.Description {
		t.Errorf("Expected Description '%s', got '%s'", config.Description, retrieved.Description)
	}
	if retrieved.TransportType != config.TransportType {
		t.Errorf("Expected TransportType '%s', got '%s'", config.TransportType, retrieved.TransportType)
	}
	if retrieved.RequireAuth != config.RequireAuth {
		t.Errorf("Expected RequireAuth %v, got %v", config.RequireAuth, retrieved.RequireAuth)
	}
}

func TestConnectionManager_GetServerConfig_NotFound(t *testing.T) {
	manager := NewConnectionManager()

	_, err := manager.GetServerConfig("nonexistent")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedMsg := "server not found: nonexistent"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestConnectionManager_GetConnectionStatus(t *testing.T) {
	manager := NewConnectionManager()

	// Register two servers
	config1 := &ServerConfig{
		ID:            "server1",
		Name:          "Server 1",
		TransportType: "stdio",
		Command:       "/usr/bin/test1",
	}
	config2 := &ServerConfig{
		ID:            "server2",
		Name:          "Server 2",
		TransportType: "stdio",
		Command:       "/usr/bin/test2",
	}

	if err := manager.RegisterServer(config1); err != nil {
		t.Fatalf("RegisterServer failed: %v", err)
	}
	if err := manager.RegisterServer(config2); err != nil {
		t.Fatalf("RegisterServer failed: %v", err)
	}

	// Get status (both should be disconnected)
	status := manager.GetConnectionStatus()
	if len(status) != 2 {
		t.Errorf("Expected 2 status entries, got %d", len(status))
	}

	if status["server1"] != "disconnected" {
		t.Errorf("Expected server1 status 'disconnected', got '%s'", status["server1"])
	}
	if status["server2"] != "disconnected" {
		t.Errorf("Expected server2 status 'disconnected', got '%s'", status["server2"])
	}
}

func TestConnectionManager_CloseAll(t *testing.T) {
	manager := NewConnectionManager()

	// Register server
	config := &ServerConfig{
		ID:            "test-server",
		Name:          "Test Server",
		TransportType: "stdio",
		Command:       "/usr/bin/test",
	}

	if err := manager.RegisterServer(config); err != nil {
		t.Fatalf("RegisterServer failed: %v", err)
	}

	// CloseAll should not error even with no connections
	if err := manager.CloseAll(); err != nil {
		t.Fatalf("CloseAll failed: %v", err)
	}
}

func TestConnectionManager_ListServers_Empty(t *testing.T) {
	manager := NewConnectionManager()

	servers := manager.ListServers()
	if len(servers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(servers))
	}
}

func TestConnectionManager_GetClient_NotRegistered(t *testing.T) {
	manager := NewConnectionManager()

	ctx := context.Background()
	_, err := manager.GetClient(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedMsg := "server not registered: nonexistent"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}
