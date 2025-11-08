package visualization

import (
	"testing"
	"time"
)

func TestNewPoAVisualizer(t *testing.T) {
	viz := NewPoAVisualizer()
	if viz == nil {
		t.Fatal("NewPoAVisualizer returned nil")
	}

	if viz.graphs == nil {
		t.Error("graphs map not initialized")
	}
}

func TestCreateGraph(t *testing.T) {
	viz := NewPoAVisualizer()

	graph := viz.CreateGraph("Test Graph", "A test graph for PoA relationships")

	if graph == nil {
		t.Fatal("CreateGraph returned nil")
	}

	if graph.ID == "" {
		t.Error("Graph ID not set")
	}

	if graph.Name != "Test Graph" {
		t.Errorf("Expected name 'Test Graph', got '%s'", graph.Name)
	}

	if graph.Description != "A test graph for PoA relationships" {
		t.Errorf("Expected description 'A test graph for PoA relationships', got '%s'", graph.Description)
	}

	if len(graph.Nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(graph.Nodes))
	}

	if len(graph.Edges) != 0 {
		t.Errorf("Expected 0 edges, got %d", len(graph.Edges))
	}
}

func TestAddNode(t *testing.T) {
	viz := NewPoAVisualizer()
	graph := viz.CreateGraph("Test", "Test graph")

	metadata := map[string]interface{}{
		"role": "principal",
	}

	node := graph.AddNode("principal", "Alice", "Principal user", "👤", metadata)

	if node == nil {
		t.Fatal("AddNode returned nil")
	}

	if node.ID == "" {
		t.Error("Node ID not set")
	}

	if node.Type != "principal" {
		t.Errorf("Expected type 'principal', got '%s'", node.Type)
	}

	if node.Label != "Alice" {
		t.Errorf("Expected label 'Alice', got '%s'", node.Label)
	}

	if node.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", node.Status)
	}

	if len(graph.Nodes) != 1 {
		t.Errorf("Expected 1 node in graph, got %d", len(graph.Nodes))
	}

	if graph.Stats.TotalNodes != 1 {
		t.Errorf("Expected stats.TotalNodes=1, got %d", graph.Stats.TotalNodes)
	}
}

func TestAddEdge(t *testing.T) {
	viz := NewPoAVisualizer()
	graph := viz.CreateGraph("Test", "Test graph")

	node1 := graph.AddNode("principal", "Alice", "Principal", "👤", nil)
	node2 := graph.AddNode("client", "AI Agent", "AI Client", "🤖", nil)

	metadata := map[string]interface{}{
		"scope": "read",
	}

	edge := graph.AddEdge(node1.ID, node2.ID, "authorizes", "Authorizes AI", 1.0, metadata)

	if edge == nil {
		t.Fatal("AddEdge returned nil")
	}

	if edge.ID == "" {
		t.Error("Edge ID not set")
	}

	if edge.Source != node1.ID {
		t.Errorf("Expected source '%s', got '%s'", node1.ID, edge.Source)
	}

	if edge.Target != node2.ID {
		t.Errorf("Expected target '%s', got '%s'", node2.ID, edge.Target)
	}

	if edge.Type != "authorizes" {
		t.Errorf("Expected type 'authorizes', got '%s'", edge.Type)
	}

	if edge.Strength != 1.0 {
		t.Errorf("Expected strength 1.0, got %f", edge.Strength)
	}

	if len(graph.Edges) != 1 {
		t.Errorf("Expected 1 edge in graph, got %d", len(graph.Edges))
	}

	if graph.Stats.TotalEdges != 1 {
		t.Errorf("Expected stats.TotalEdges=1, got %d", graph.Stats.TotalEdges)
	}
}

func TestSetNodePosition(t *testing.T) {
	viz := NewPoAVisualizer()
	graph := viz.CreateGraph("Test", "Test graph")

	node := graph.AddNode("principal", "Alice", "Principal", "👤", nil)

	err := graph.SetNodePosition(node.ID, 1.5, 2.5, 3.5)
	if err != nil {
		t.Fatalf("SetNodePosition failed: %v", err)
	}

	retrievedNode := graph.GetNode(node.ID)
	if retrievedNode == nil {
		t.Fatal("GetNode returned nil")
	}

	if retrievedNode.Position == nil {
		t.Fatal("Node position not set")
	}

	if retrievedNode.Position.X != 1.5 {
		t.Errorf("Expected X=1.5, got %f", retrievedNode.Position.X)
	}

	if retrievedNode.Position.Y != 2.5 {
		t.Errorf("Expected Y=2.5, got %f", retrievedNode.Position.Y)
	}

	if retrievedNode.Position.Z != 3.5 {
		t.Errorf("Expected Z=3.5, got %f", retrievedNode.Position.Z)
	}
}

func TestUpdateNodeStatus(t *testing.T) {
	viz := NewPoAVisualizer()
	graph := viz.CreateGraph("Test", "Test graph")

	node := graph.AddNode("principal", "Alice", "Principal", "👤", nil)

	if node.Status != "active" {
		t.Errorf("Expected initial status 'active', got '%s'", node.Status)
	}

	err := graph.UpdateNodeStatus(node.ID, "revoked")
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	retrievedNode := graph.GetNode(node.ID)
	if retrievedNode.Status != "revoked" {
		t.Errorf("Expected updated status 'revoked', got '%s'", retrievedNode.Status)
	}

	if graph.Stats.RevokedNodes != 1 {
		t.Errorf("Expected stats.RevokedNodes=1, got %d", graph.Stats.RevokedNodes)
	}

	if graph.Stats.ActiveNodes != 0 {
		t.Errorf("Expected stats.ActiveNodes=0, got %d", graph.Stats.ActiveNodes)
	}
}

func TestGetConnectedNodes(t *testing.T) {
	viz := NewPoAVisualizer()
	graph := viz.CreateGraph("Test", "Test graph")

	node1 := graph.AddNode("principal", "Alice", "Principal", "👤", nil)
	node2 := graph.AddNode("client", "AI Agent 1", "AI Client", "🤖", nil)
	node3 := graph.AddNode("client", "AI Agent 2", "AI Client", "🤖", nil)
	_ = graph.AddNode("resource", "Database", "Data resource", "💾", nil) // node4 intentionally disconnected

	// Connect node1 -> node2, node1 -> node3
	graph.AddEdge(node1.ID, node2.ID, "authorizes", "Auth 1", 1.0, nil)
	graph.AddEdge(node1.ID, node3.ID, "authorizes", "Auth 2", 1.0, nil)

	connected := graph.GetConnectedNodes(node1.ID)

	if len(connected) != 2 {
		t.Errorf("Expected 2 connected nodes, got %d", len(connected))
	}

	// Check that node2 and node3 are in the connected list
	foundNode2 := false
	foundNode3 := false
	for _, n := range connected {
		if n.ID == node2.ID {
			foundNode2 = true
		}
		if n.ID == node3.ID {
			foundNode3 = true
		}
	}

	if !foundNode2 {
		t.Error("Expected node2 in connected nodes")
	}

	if !foundNode3 {
		t.Error("Expected node3 in connected nodes")
	}
}

func TestGraphStats(t *testing.T) {
	viz := NewPoAVisualizer()
	graph := viz.CreateGraph("Test", "Test graph")

	// Add various nodes with different statuses
	node1 := graph.AddNode("principal", "Alice", "Principal", "👤", nil)
	node2 := graph.AddNode("client", "AI Agent 1", "AI", "🤖", nil)
	node3 := graph.AddNode("client", "AI Agent 2", "AI", "🤖", nil)

	_ = graph.UpdateNodeStatus(node2.ID, "pending") //nolint:errcheck
	_ = graph.UpdateNodeStatus(node3.ID, "revoked") //nolint:errcheck

	// Add edges
	graph.AddEdge(node1.ID, node2.ID, "authorizes", "Auth", 1.0, nil)
	graph.AddEdge(node1.ID, node3.ID, "authorizes", "Auth", 1.0, nil)

	stats := graph.Stats

	if stats.TotalNodes != 3 {
		t.Errorf("Expected TotalNodes=3, got %d", stats.TotalNodes)
	}

	if stats.TotalEdges != 2 {
		t.Errorf("Expected TotalEdges=2, got %d", stats.TotalEdges)
	}

	if stats.ActiveNodes != 1 {
		t.Errorf("Expected ActiveNodes=1, got %d", stats.ActiveNodes)
	}

	if stats.PendingNodes != 1 {
		t.Errorf("Expected PendingNodes=1, got %d", stats.PendingNodes)
	}

	if stats.RevokedNodes != 1 {
		t.Errorf("Expected RevokedNodes=1, got %d", stats.RevokedNodes)
	}

	// Average connections: (2 edges * 2 endpoints) / 3 nodes = 1.33...
	expectedAvg := float64(2*2) / float64(3)
	if stats.AverageConnections != expectedAvg {
		t.Errorf("Expected AverageConnections=%.2f, got %.2f", expectedAvg, stats.AverageConnections)
	}
}

func TestCreateSubscriptionVisualization(t *testing.T) {
	viz := CreateSubscriptionVisualization()

	if viz == nil {
		t.Fatal("CreateSubscriptionVisualization returned nil")
	}

	if viz.StepID != "subscription" {
		t.Errorf("Expected StepID 'subscription', got '%s'", viz.StepID)
	}

	if viz.StepName != "Subscription" {
		t.Errorf("Expected StepName 'Subscription', got '%s'", viz.StepName)
	}

	if len(viz.Layers) != 3 {
		t.Errorf("Expected 3 layers, got %d", len(viz.Layers))
	}

	// Check layer IDs
	expectedLayers := []string{"layer-client", "layer-authz", "layer-storage"}
	for i, expected := range expectedLayers {
		if i >= len(viz.Layers) {
			t.Errorf("Missing layer at index %d", i)
			continue
		}
		if viz.Layers[i].ID != expected {
			t.Errorf("Expected layer ID '%s', got '%s'", expected, viz.Layers[i].ID)
		}
	}

	if len(viz.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(viz.Connections))
	}

	// Verify components exist
	if len(viz.Layers[0].Components) != 1 {
		t.Errorf("Expected 1 component in client layer, got %d", len(viz.Layers[0].Components))
	}

	if len(viz.Layers[1].Components) != 1 {
		t.Errorf("Expected 1 component in authz layer, got %d", len(viz.Layers[1].Components))
	}
}

func TestCreateMatchingVisualization(t *testing.T) {
	viz := CreateMatchingVisualization()

	if viz == nil {
		t.Fatal("CreateMatchingVisualization returned nil")
	}

	if viz.StepID != "matching" {
		t.Errorf("Expected StepID 'matching', got '%s'", viz.StepID)
	}

	if len(viz.Layers) != 3 {
		t.Errorf("Expected 3 layers, got %d", len(viz.Layers))
	}

	// Check that validation layer has multiple validators
	validationLayer := viz.Layers[1]
	if validationLayer.ID != "layer-validation" {
		t.Error("Expected second layer to be validation layer")
	}

	if len(validationLayer.Components) != 3 {
		t.Errorf("Expected 3 validation components, got %d", len(validationLayer.Components))
	}

	// Verify validator types
	expectedValidators := []string{"poa-validator", "capability-matcher", "jurisdiction-validator"}
	for _, expected := range expectedValidators {
		found := false
		for _, comp := range validationLayer.Components {
			if comp.ID == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected validator '%s' not found", expected)
		}
	}

	if len(viz.Connections) != 4 {
		t.Errorf("Expected 4 connections, got %d", len(viz.Connections))
	}
}

func TestCreateRequestVisualization(t *testing.T) {
	viz := CreateRequestVisualization()

	if viz == nil {
		t.Fatal("CreateRequestVisualization returned nil")
	}

	if viz.StepID != "subset_request" {
		t.Errorf("Expected StepID 'subset_request', got '%s'", viz.StepID)
	}

	if viz.StepName != "Subset/Request" {
		t.Errorf("Expected StepName 'Subset/Request', got '%s'", viz.StepName)
	}

	if len(viz.Layers) != 3 {
		t.Errorf("Expected 3 layers, got %d", len(viz.Layers))
	}

	// Check PDP layer
	pdpLayer := viz.Layers[1]
	if pdpLayer.ID != "layer-pdp" {
		t.Error("Expected second layer to be PDP layer")
	}

	if len(pdpLayer.Components) != 1 {
		t.Errorf("Expected 1 PDP component, got %d", len(pdpLayer.Components))
	}

	pdpComponent := pdpLayer.Components[0]
	if pdpComponent.ID != "pdp-engine" {
		t.Errorf("Expected component ID 'pdp-engine', got '%s'", pdpComponent.ID)
	}

	if len(viz.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(viz.Connections))
	}
}

func TestGenerateIDs(t *testing.T) {
	// Test graph ID generation
	id1 := generateGraphID("test", mustParseTime("2024-01-01T00:00:00Z"))
	id2 := generateGraphID("test", mustParseTime("2024-01-01T00:00:01Z"))

	if id1 == id2 {
		t.Error("Expected different IDs for different timestamps")
	}

	if id1[:6] != "graph-" {
		t.Errorf("Expected graph ID to start with 'graph-', got '%s'", id1[:6])
	}

	// Test node ID generation
	nodeID1 := generateNodeID("principal", "Alice", mustParseTime("2024-01-01T00:00:00Z"))
	nodeID2 := generateNodeID("principal", "Bob", mustParseTime("2024-01-01T00:00:00Z"))

	if nodeID1 == nodeID2 {
		t.Error("Expected different IDs for different labels")
	}

	if nodeID1[:5] != "node-" {
		t.Errorf("Expected node ID to start with 'node-', got '%s'", nodeID1[:5])
	}

	// Test edge ID generation
	edgeID1 := generateEdgeID("source1", "target1", "authorizes")
	edgeID2 := generateEdgeID("source1", "target2", "authorizes")

	if edgeID1 == edgeID2 {
		t.Error("Expected different IDs for different targets")
	}

	if edgeID1[:5] != "edge-" {
		t.Errorf("Expected edge ID to start with 'edge-', got '%s'", edgeID1[:5])
	}
}

func TestComplexGraph(t *testing.T) {
	viz := NewPoAVisualizer()
	graph := viz.CreateGraph("Complex PoA", "Multi-party authorization")

	// Create a realistic PoA structure
	principal := graph.AddNode("principal", "Alice Corp", "Principal organization", "🏢", nil)
	agent := graph.AddNode("authorizer", "Bob Legal", "Authorized agent", "👔", nil)
	aiClient := graph.AddNode("client", "AI Assistant", "AI client application", "🤖", nil)
	resource := graph.AddNode("resource", "Financial DB", "Financial database", "💾", nil)

	// Set 3D positions
	_ = graph.SetNodePosition(principal.ID, 0, 0, 0) //nolint:errcheck
	_ = graph.SetNodePosition(agent.ID, 2, 0, 1) //nolint:errcheck
	_ = graph.SetNodePosition(aiClient.ID, 0, 2, 2) //nolint:errcheck
	graph.SetNodePosition(resource.ID, 2, 2, 3)

	// Create authorization chain
	graph.AddEdge(principal.ID, agent.ID, "delegates", "Delegates authority", 0.9, map[string]interface{}{
		"scope": "financial_transactions",
	})
	graph.AddEdge(agent.ID, aiClient.ID, "authorizes", "Authorizes AI access", 0.8, map[string]interface{}{
		"scope": "read_only",
	})
	graph.AddEdge(aiClient.ID, resource.ID, "requests", "Requests data", 1.0, nil)

	// Verify graph structure
	if len(graph.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(graph.Nodes))
	}

	if len(graph.Edges) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(graph.Edges))
	}

	// Verify connections
	principalConnections := graph.GetConnectedNodes(principal.ID)
	if len(principalConnections) != 1 {
		t.Errorf("Expected 1 connection from principal, got %d", len(principalConnections))
	}

	// Verify stats
	if graph.Stats.TotalNodes != 4 {
		t.Errorf("Expected stats.TotalNodes=4, got %d", graph.Stats.TotalNodes)
	}

	if graph.Stats.ActiveNodes != 4 {
		t.Errorf("Expected stats.ActiveNodes=4, got %d", graph.Stats.ActiveNodes)
	}

	// Verify max depth from positions
	if graph.Stats.MaxDepth < 3 {
		t.Errorf("Expected max depth >= 3, got %d", graph.Stats.MaxDepth)
	}
}

// Helper function for testing
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
