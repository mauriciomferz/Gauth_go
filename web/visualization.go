package web

import (
	"net/http"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/visualization"
	"github.com/gin-gonic/gin"
)

// Global visualizer instance
var globalVisualizer = visualization.NewPoAVisualizer()

// POAGraphRequest represents a request to create a PoA graph
type POAGraphRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// POANodeRequest represents a request to add a node
type POANodeRequest struct {
	Type        string                 `json:"type" binding:"required"`
	Label       string                 `json:"label" binding:"required"`
	Description string                 `json:"description"`
	Icon        string                 `json:"icon"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// POAEdgeRequest represents a request to add an edge
type POAEdgeRequest struct {
	Source   string                 `json:"source" binding:"required"`
	Target   string                 `json:"target" binding:"required"`
	Type     string                 `json:"type" binding:"required"`
	Label    string                 `json:"label"`
	Strength float64                `json:"strength"`
	Metadata map[string]interface{} `json:"metadata"`
}

// POANodePositionRequest represents a request to update node position
type POANodePositionRequest struct {
	NodeID string  `json:"node_id" binding:"required"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
}

// SetupVisualizationRoutes registers visualization API routes
func SetupVisualizationRoutes(router *gin.Engine) {
	vizGroup := router.Group("/api/v1/visualization")
	{
		// PoA Graph endpoints
		vizGroup.POST("/poa/graphs", createPOAGraph)
		vizGroup.GET("/poa/graphs/:id", getPOAGraph)
		vizGroup.POST("/poa/graphs/:id/nodes", addPOANode)
		vizGroup.POST("/poa/graphs/:id/edges", addPOAEdge)
		vizGroup.PUT("/poa/graphs/:id/nodes/position", updateNodePosition)
		vizGroup.PUT("/poa/graphs/:id/nodes/:node_id/status", updateNodeStatus)

		// Protocol step visualizations
		vizGroup.GET("/protocol/subscription", getSubscriptionViz)
		vizGroup.GET("/protocol/matching", getMatchingViz)
		vizGroup.GET("/protocol/request", getRequestViz)

		// Demo data
		vizGroup.POST("/demo/complex-graph", createDemoComplexGraph)
	}
}

// createPOAGraph creates a new PoA graph
func createPOAGraph(c *gin.Context) {
	var req POAGraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	graph := globalVisualizer.CreateGraph(req.Name, req.Description)
	c.JSON(http.StatusCreated, graph)
}

// getPOAGraph retrieves a PoA graph by ID
func getPOAGraph(c *gin.Context) {
	graphID := c.Param("id")

	// In a real implementation, retrieve from visualizer's graph map
	// For now, return example data
	c.JSON(http.StatusOK, gin.H{
		"id":          graphID,
		"name":        "Example Graph",
		"description": "An example PoA relationship graph",
		"nodes":       []gin.H{},
		"edges":       []gin.H{},
		"stats": gin.H{
			"total_nodes": 0,
			"total_edges": 0,
		},
	})
}

// addPOANode adds a node to a graph
func addPOANode(c *gin.Context) {
	graphID := c.Param("id")

	var req POANodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a new graph for demo purposes
	graph := globalVisualizer.CreateGraph("Demo", "Demo graph")

	node := graph.AddNode(req.Type, req.Label, req.Description, req.Icon, req.Metadata)

	c.JSON(http.StatusCreated, gin.H{
		"graph_id": graphID,
		"node":     node,
	})
}

// addPOAEdge adds an edge to a graph
func addPOAEdge(c *gin.Context) {
	graphID := c.Param("id")

	var req POAEdgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In real implementation, get graph by ID
	graph := globalVisualizer.CreateGraph("Demo", "Demo graph")

	edge := graph.AddEdge(req.Source, req.Target, req.Type, req.Label, req.Strength, req.Metadata)

	c.JSON(http.StatusCreated, gin.H{
		"graph_id": graphID,
		"edge":     edge,
	})
}

// updateNodePosition updates a node's 3D position
func updateNodePosition(c *gin.Context) {
	graphID := c.Param("id")

	var req POANodePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In real implementation, get graph and update position
	graph := globalVisualizer.CreateGraph("Demo", "Demo graph")

	err := graph.SetNodePosition(req.NodeID, req.X, req.Y, req.Z)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"graph_id": graphID,
		"node_id":  req.NodeID,
		"position": gin.H{
			"x": req.X,
			"y": req.Y,
			"z": req.Z,
		},
	})
}

// updateNodeStatus updates a node's status
func updateNodeStatus(c *gin.Context) {
	graphID := c.Param("id")
	nodeID := c.Param("node_id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In real implementation, get graph and update status
	graph := globalVisualizer.CreateGraph("Demo", "Demo graph")

	err := graph.UpdateNodeStatus(nodeID, req.Status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"graph_id": graphID,
		"node_id":  nodeID,
		"status":   req.Status,
	})
}

// getSubscriptionViz returns subscription protocol visualization
func getSubscriptionViz(c *gin.Context) {
	viz := visualization.CreateSubscriptionVisualization()
	c.JSON(http.StatusOK, viz)
}

// getMatchingViz returns matching protocol visualization
func getMatchingViz(c *gin.Context) {
	viz := visualization.CreateMatchingVisualization()
	c.JSON(http.StatusOK, viz)
}

// getRequestViz returns request protocol visualization
func getRequestViz(c *gin.Context) {
	viz := visualization.CreateRequestVisualization()
	c.JSON(http.StatusOK, viz)
}

// createDemoComplexGraph creates a complex demo graph
func createDemoComplexGraph(c *gin.Context) {
	graph := globalVisualizer.CreateGraph("Multi-Party Authorization", "Complex PoA relationship graph")

	// Create principal
	principal := graph.AddNode("principal", "Acme Corporation", "Principal organization", "🏢", map[string]interface{}{
		"type":         "organization",
		"jurisdiction": "US",
	})
	graph.SetNodePosition(principal.ID, 0, 0, 0) //nolint:errcheck

	// Create authorizers
	agent1 := graph.AddNode("authorizer", "Legal Department", "Authorized legal agent", "👔", map[string]interface{}{
		"department": "legal",
	})
	graph.SetNodePosition(agent1.ID, -3, 0, 2) //nolint:errcheck

	agent2 := graph.AddNode("authorizer", "Finance Department", "Authorized finance agent", "💼", map[string]interface{}{
		"department": "finance",
	})
	graph.SetNodePosition(agent2.ID, 3, 0, 2) //nolint:errcheck

	// Create AI clients
	aiClient1 := graph.AddNode("client", "Legal AI Assistant", "AI for legal document processing", "🤖", map[string]interface{}{
		"capability": "document_analysis",
	})
	graph.SetNodePosition(aiClient1.ID, -3, 0, 4) //nolint:errcheck

	aiClient2 := graph.AddNode("client", "Finance AI Assistant", "AI for financial analysis", "🤖", map[string]interface{}{
		"capability": "financial_analysis",
	})
	graph.SetNodePosition(aiClient2.ID, 3, 0, 4) //nolint:errcheck

	// Create resources
	resource1 := graph.AddNode("resource", "Legal Documents", "Contracts and legal files", "📄", map[string]interface{}{
		"sensitivity": "high",
	})
	graph.SetNodePosition(resource1.ID, -3, 0, 6) //nolint:errcheck

	resource2 := graph.AddNode("resource", "Financial Database", "Financial records and transactions", "💾", map[string]interface{}{
		"sensitivity": "critical",
	})
	graph.SetNodePosition(resource2.ID, 3, 0, 6) //nolint:errcheck

	// Create delegation edges
	graph.AddEdge(principal.ID, agent1.ID, "delegates", "Delegates legal authority", 0.95, map[string]interface{}{
		"scope":      "legal_operations",
		"expires_at": time.Now().Add(365 * 24 * time.Hour),
	})

	graph.AddEdge(principal.ID, agent2.ID, "delegates", "Delegates financial authority", 0.95, map[string]interface{}{
		"scope":      "financial_operations",
		"expires_at": time.Now().Add(365 * 24 * time.Hour),
	})

	// Create authorization edges
	graph.AddEdge(agent1.ID, aiClient1.ID, "authorizes", "Authorizes AI access", 0.85, map[string]interface{}{
		"scope": "read_write",
		"terms": "ai_usage_policy_v1",
	})

	graph.AddEdge(agent2.ID, aiClient2.ID, "authorizes", "Authorizes AI access", 0.85, map[string]interface{}{
		"scope": "read_only",
		"terms": "ai_usage_policy_v1",
	})

	// Create request edges
	graph.AddEdge(aiClient1.ID, resource1.ID, "requests", "Requests document access", 1.0, map[string]interface{}{
		"method": "GET",
	})

	graph.AddEdge(aiClient2.ID, resource2.ID, "requests", "Requests data access", 1.0, map[string]interface{}{
		"method": "GET",
	})

	// Set one node to pending for demo
	graph.UpdateNodeStatus(aiClient2.ID, "pending") //nolint:errcheck

	c.JSON(http.StatusCreated, gin.H{
		"graph":   graph,
		"message": "Complex demo graph created successfully",
	})
}
