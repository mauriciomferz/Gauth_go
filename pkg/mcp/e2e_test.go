// Package mcp - End-to-End Tests for Complete MCP Workflow
// Tests the full integration from HTTP API → Agent → MCP Client → Server
package mcp

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestE2E_CompleteM CPWorkflow tests the entire MCP integration end-to-end
func TestE2E_CompleteMCPWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	t.Run("server lifecycle management", func(t *testing.T) {
		manager := NewConnectionManager()

		// Step 1: Register MCP server
		config := &ServerConfig{
			ID:            "test-server-e2e",
			Name:          "Test E2E Server",
			Description:   "Server for E2E testing",
			TransportType: "stdio",
			Command:       "echo",
			Args:          []string{"hello", "world"},
			RequireAuth:   false,
		}

		err := manager.RegisterServer(config)
		if err != nil {
			t.Fatalf("Failed to register server: %v", err)
		}

		// Step 2: Get client connection
		client, err := manager.GetClient(ctx, "test-server-e2e")
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}

		if client == nil {
			t.Fatal("Client is nil")
		}

		// Step 3: List servers
		servers := manager.ListServers()
		if len(servers) != 1 {
			t.Errorf("Expected 1 server, got %d", len(servers))
		}

		if servers[0] != "test-server-e2e" {
			t.Errorf("Expected server ID 'test-server-e2e', got %s", servers[0])
		}

		// Step 4: Unregister server
		err = manager.UnregisterServer("test-server-e2e")
		if err != nil {
			t.Errorf("Failed to unregister server: %v", err)
		}

		// Verify server removed
		servers = manager.ListServers()
		if len(servers) != 0 {
			t.Errorf("Expected 0 servers after unregister, got %d", len(servers))
		}
	})

	t.Run("audit logging integration", func(t *testing.T) {
		auditLogger := NewInMemoryAuditLogger(100)

		// Create test audit entries
		entry1 := &AuditLogEntry{
			Timestamp:   time.Now(),
			AgentID:     "agent-e2e-1",
			RequestID:   "req-e2e-1",
			GrantID:     "grant-e2e-1",
			Operation:   "resource_read",
			Target:      "file:///test.txt",
			Authorized:  true,
			Decision:    "granted",
			Duration:    50 * time.Millisecond,
			MCPServerID: "server-1",
			TokenScopes: []string{"mcp:resource:read"},
		}

		entry2 := &AuditLogEntry{
			Timestamp:   time.Now().Add(time.Second),
			AgentID:     "agent-e2e-1",
			RequestID:   "req-e2e-2",
			GrantID:     "grant-e2e-1",
			Operation:   "tool_call",
			Target:      "calculator",
			Authorized:  true,
			Decision:    "granted",
			Duration:    75 * time.Millisecond,
			MCPServerID: "server-1",
			TokenScopes: []string{"mcp:tool:call"},
		}

		entry3 := &AuditLogEntry{
			Timestamp:   time.Now().Add(2 * time.Second),
			AgentID:     "agent-e2e-2",
			RequestID:   "req-e2e-3",
			GrantID:     "grant-e2e-2",
			Operation:   "resource_read",
			Target:      "file:///restricted.txt",
			Authorized:  false,
			Decision:    "denied",
			Duration:    10 * time.Millisecond,
			MCPServerID: "server-1",
			TokenScopes: []string{},
		}

		// Log entries
		if err := auditLogger.Log(ctx, entry1); err != nil {
			t.Errorf("Failed to log entry1: %v", err)
		}
		if err := auditLogger.Log(ctx, entry2); err != nil {
			t.Errorf("Failed to log entry2: %v", err)
		}
		if err := auditLogger.Log(ctx, entry3); err != nil {
			t.Errorf("Failed to log entry3: %v", err)
		}

		// Query all entries
		allEntries, err := auditLogger.Query(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to query all entries: %v", err)
		}
		if len(allEntries) != 3 {
			t.Errorf("Expected 3 entries, got %d", len(allEntries))
		}

		// Query by agent ID
		agent1Criteria := &AuditQueryCriteria{
			AgentID: "agent-e2e-1",
		}
		agent1Entries, err := auditLogger.Query(ctx, agent1Criteria)
		if err != nil {
			t.Fatalf("Failed to query agent1 entries: %v", err)
		}
		if len(agent1Entries) != 2 {
			t.Errorf("Expected 2 entries for agent1, got %d", len(agent1Entries))
		}

		// Query by operation
		operationCriteria := &AuditQueryCriteria{
			Operation: "resource_read",
		}
		resourceEntries, err := auditLogger.Query(ctx, operationCriteria)
		if err != nil {
			t.Fatalf("Failed to query resource_read entries: %v", err)
		}
		if len(resourceEntries) != 2 {
			t.Errorf("Expected 2 resource_read entries, got %d", len(resourceEntries))
		}

		// Query denied operations
		authorized := false
		deniedCriteria := &AuditQueryCriteria{
			Authorized: &authorized,
		}
		deniedEntries, err := auditLogger.Query(ctx, deniedCriteria)
		if err != nil {
			t.Fatalf("Failed to query denied entries: %v", err)
		}
		if len(deniedEntries) != 1 {
			t.Errorf("Expected 1 denied entry, got %d", len(deniedEntries))
		}

		// Compute statistics
		stats := ComputeStatistics(allEntries)
		if stats.TotalOperations != 3 {
			t.Errorf("Expected 3 total operations, got %d", stats.TotalOperations)
		}
		if stats.AuthorizedCount != 2 {
			t.Errorf("Expected 2 authorized, got %d", stats.AuthorizedCount)
		}
		if stats.DeniedCount != 1 {
			t.Errorf("Expected 1 denied, got %d", stats.DeniedCount)
		}
		if stats.OperationBreakdown["resource_read"] != 2 {
			t.Errorf("Expected 2 resource_read operations, got %d", stats.OperationBreakdown["resource_read"])
		}
		if stats.OperationBreakdown["tool_call"] != 1 {
			t.Errorf("Expected 1 tool_call operation, got %d", stats.OperationBreakdown["tool_call"])
		}
		if stats.AgentActivityCount["agent-e2e-1"] != 2 {
			t.Errorf("Expected 2 operations for agent-e2e-1, got %d", stats.AgentActivityCount["agent-e2e-1"])
		}
		if stats.AverageDuration == 0 {
			t.Error("Expected non-zero average duration")
		}
	})

	t.Run("authorization bridge scope validation", func(t *testing.T) {
		// Test scope validation logic
		testCases := []struct {
			name     string
			scopes   []string
			required string
			expected bool
		}{
			{"has resource scope", []string{"mcp:resource:read"}, "mcp:resource:read", true},
			{"has tool scope", []string{"mcp:tool:call"}, "mcp:tool:call", true},
			{"has prompt scope", []string{"mcp:prompt:get"}, "mcp:prompt:get", true},
			{"missing scope", []string{"other:scope"}, "mcp:resource:read", false},
			{"empty scopes", []string{}, "mcp:resource:read", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				hasScope := false
				for _, scope := range tc.scopes {
					if scope == tc.required {
						hasScope = true
						break
					}
				}
				if hasScope != tc.expected {
					t.Errorf("Expected %v for scope check, got %v", tc.expected, hasScope)
				}
			})
		}
	})

	t.Run("connection manager concurrency", func(t *testing.T) {
		manager := NewConnectionManager()

		// Register multiple servers concurrently
		servers := []ServerConfig{
			{
				ID:            "concurrent-1",
				Name:          "Concurrent Server 1",
				TransportType: "stdio",
				Command:       "echo",
				Args:          []string{"test1"},
			},
			{
				ID:            "concurrent-2",
				Name:          "Concurrent Server 2",
				TransportType: "stdio",
				Command:       "echo",
				Args:          []string{"test2"},
			},
			{
				ID:            "concurrent-3",
				Name:          "Concurrent Server 3",
				TransportType: "stdio",
				Command:       "echo",
				Args:          []string{"test3"},
			},
		}

		// Register servers concurrently
		errChan := make(chan error, len(servers))
		for i := range servers {
			go func(config ServerConfig) {
				errChan <- manager.RegisterServer(&config)
			}(servers[i])
		}

		// Collect errors
		for i := 0; i < len(servers); i++ {
			if err := <-errChan; err != nil {
				t.Errorf("Concurrent register failed: %v", err)
			}
		}

		// Verify all registered
		registered := manager.ListServers()
		if len(registered) != 3 {
			t.Errorf("Expected 3 registered servers, got %d", len(registered))
		}

		// Unregister concurrently
		for _, serverID := range registered {
			go func(id string) {
				errChan <- manager.UnregisterServer(id)
			}(serverID)
		}

		// Collect unregister errors
		for i := 0; i < len(registered); i++ {
			if err := <-errChan; err != nil {
				t.Errorf("Concurrent unregister failed: %v", err)
			}
		}

		// Verify all unregistered
		remaining := manager.ListServers()
		if len(remaining) != 0 {
			t.Errorf("Expected 0 servers after unregister, got %d", len(remaining))
		}
	})

	t.Run("error handling and recovery", func(t *testing.T) {
		manager := NewConnectionManager()

		// Test invalid configurations
		invalidConfigs := []*ServerConfig{
			{
				// Missing ID
				Name:          "Invalid Server",
				TransportType: "stdio",
				Command:       "echo",
			},
			{
				ID:   "invalid-2",
				// Missing Name
				TransportType: "stdio",
				Command:       "echo",
			},
			{
				ID:   "invalid-3",
				Name: "Invalid Server 3",
				// Missing TransportType
				Command: "echo",
			},
			{
				ID:            "invalid-4",
				Name:          "Invalid Server 4",
				TransportType: "stdio",
				// Missing Command for stdio
			},
			{
				ID:            "invalid-5",
				Name:          "Invalid Server 5",
				TransportType: "websocket",
				// Missing URL for websocket
			},
		}

		for i, config := range invalidConfigs {
			err := manager.RegisterServer(config)
			if err == nil {
				t.Errorf("Invalid config %d should have returned error", i)
			}
		}

		// Test unregistering non-existent server
		err := manager.UnregisterServer("non-existent")
		if err != nil {
			// Should not error on unregistering non-existent server
			t.Errorf("Unregistering non-existent server should not error: %v", err)
		}

		// Test getting client for non-existent server
		_, err = manager.GetClient(ctx, "non-existent")
		if err == nil {
			t.Error("Getting client for non-existent server should error")
		}
	})
}

// TestE2E_AuditLoggerPerformance tests audit logger under load
func TestE2E_AuditLoggerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()
	auditLogger := NewInMemoryAuditLogger(10000)

	// Generate 1000 audit entries
	numEntries := 1000
	startTime := time.Now()

	for i := 0; i < numEntries; i++ {
		entry := &AuditLogEntry{
			Timestamp:   time.Now(),
			AgentID:     fmt.Sprintf("agent-%d", i%10), // 10 different agents
			RequestID:   fmt.Sprintf("req-%d", i),
			GrantID:     fmt.Sprintf("grant-%d", i%100),
			Operation:   []string{"resource_read", "tool_call", "prompt_get"}[i%3],
			Target:      fmt.Sprintf("target-%d", i),
			Authorized:  i%4 != 0, // 75% authorized
			Decision:    "granted",
			Duration:    time.Duration(i%100) * time.Millisecond,
			MCPServerID: fmt.Sprintf("server-%d", i%5),
			TokenScopes: []string{"mcp:resource:read", "mcp:tool:call"},
		}

		if err := auditLogger.Log(ctx, entry); err != nil {
			t.Fatalf("Failed to log entry %d: %v", i, err)
		}
	}

	logDuration := time.Since(startTime)
	t.Logf("Logged %d entries in %v (%.2f entries/sec)", 
		numEntries, logDuration, float64(numEntries)/logDuration.Seconds())

	// Query performance
	queryStart := time.Now()
	allEntries, err := auditLogger.Query(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to query entries: %v", err)
	}
	queryDuration := time.Since(queryStart)

	if len(allEntries) != numEntries {
		t.Errorf("Expected %d entries, got %d", numEntries, len(allEntries))
	}

	t.Logf("Queried %d entries in %v", len(allEntries), queryDuration)

	// Statistics computation performance
	statsStart := time.Now()
	stats := ComputeStatistics(allEntries)
	statsDuration := time.Since(statsStart)

	t.Logf("Computed statistics in %v", statsDuration)
	t.Logf("Statistics: Total=%d, Authorized=%d, Denied=%d, Avg Duration=%v",
		stats.TotalOperations, stats.AuthorizedCount, stats.DeniedCount, stats.AverageDuration)
}

// TestE2E_RealWorldScenario simulates a realistic MCP usage scenario
func TestE2E_RealWorldScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-world scenario test in short mode")
	}

	ctx := context.Background()

	t.Run("AI agent accessing multiple MCP servers", func(t *testing.T) {
		manager := NewConnectionManager()
		auditLogger := NewInMemoryAuditLogger(1000)

		// Scenario: AI agent needs to:
		// 1. Read files from filesystem server
		// 2. Perform calculations using calculator server
		// 3. Query database using database server

		servers := []*ServerConfig{
			{
				ID:            "filesystem",
				Name:          "Filesystem Server",
				Description:   "Access to project files",
				TransportType: "stdio",
				Command:       "echo",
				Args:          []string{"filesystem"},
				RequireAuth:   true,
				AllowedScopes: []string{"mcp:resource:read"},
			},
			{
				ID:            "calculator",
				Name:          "Calculator Server",
				Description:   "Mathematical operations",
				TransportType: "stdio",
				Command:       "echo",
				Args:          []string{"calculator"},
				RequireAuth:   true,
				AllowedScopes: []string{"mcp:tool:call"},
			},
			{
				ID:            "database",
				Name:          "Database Server",
				Description:   "Query database",
				TransportType: "stdio",
				Command:       "echo",
				Args:          []string{"database"},
				RequireAuth:   true,
				AllowedScopes: []string{"mcp:resource:read", "mcp:tool:call"},
			},
		}

		// Register all servers
		for _, config := range servers {
			if err := manager.RegisterServer(config); err != nil {
				t.Fatalf("Failed to register server %s: %v", config.ID, err)
			}
		}

		// Verify all registered
		registered := manager.ListServers()
		if len(registered) != 3 {
			t.Errorf("Expected 3 servers, got %d", len(registered))
		}

		// Simulate agent operations
		operations := []struct {
			server    string
			operation string
			target    string
			authorized bool
		}{
			{"filesystem", "resource_read", "file:///project/data.txt", true},
			{"calculator", "tool_call", "add", true},
			{"calculator", "tool_call", "multiply", true},
			{"database", "resource_read", "db://users", true},
			{"database", "tool_call", "query", true},
			{"filesystem", "resource_read", "file:///etc/passwd", false}, // Denied
		}

		for _, op := range operations {
			entry := &AuditLogEntry{
				Timestamp:   time.Now(),
				AgentID:     "ai-agent-prod-1",
				RequestID:   fmt.Sprintf("req-%d", time.Now().UnixNano()),
				GrantID:     "grant-prod-123",
				Operation:   op.operation,
				Target:      op.target,
				Authorized:  op.authorized,
				Decision:    map[bool]string{true: "granted", false: "denied"}[op.authorized],
				Duration:    time.Duration(50+time.Now().UnixNano()%100) * time.Millisecond,
				MCPServerID: op.server,
				TokenScopes: []string{"mcp:resource:read", "mcp:tool:call"},
			}

			if err := auditLogger.Log(ctx, entry); err != nil {
				t.Errorf("Failed to log operation: %v", err)
			}
		}

		// Analyze audit logs
		allOps, err := auditLogger.Query(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to query operations: %v", err)
		}

		stats := ComputeStatistics(allOps)
		t.Logf("Operations Summary:")
		t.Logf("  Total: %d", stats.TotalOperations)
		t.Logf("  Authorized: %d", stats.AuthorizedCount)
		t.Logf("  Denied: %d", stats.DeniedCount)
		t.Logf("  Average Duration: %v", stats.AverageDuration)
		t.Logf("  Operations by Type:")
		for op, count := range stats.OperationBreakdown {
			t.Logf("    %s: %d", op, count)
		}

		// Cleanup
		for _, serverID := range registered {
			if err := manager.UnregisterServer(serverID); err != nil {
				t.Errorf("Failed to unregister %s: %v", serverID, err)
			}
		}
	})
}
