package visualization

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// PoANode represents a node in the Proof of Authorization relationship graph
type PoANode struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // principal, authorizer, client, resource
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Icon        string                 `json:"icon"`
	Status      string                 `json:"status"` // active, pending, revoked, expired
	Metadata    map[string]interface{} `json:"metadata"`
	Position    *Position              `json:"position,omitempty"` // For 3D coordinates
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PoAEdge represents a relationship/connection between PoA nodes
type PoAEdge struct {
	ID            string                 `json:"id"`
	Source        string                 `json:"source"` // Source node ID
	Target        string                 `json:"target"` // Target node ID
	Type          string                 `json:"type"`   // authorizes, delegates, requests, validates
	Label         string                 `json:"label"`
	Strength      float64                `json:"strength"` // 0.0-1.0 connection strength
	Bidirectional bool                   `json:"bidirectional"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
}

// Position represents 3D coordinates for visualization
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// PoAGraph represents the complete Proof of Authorization relationship graph
type PoAGraph struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Nodes       []PoANode              `json:"nodes"`
	Edges       []PoAEdge              `json:"edges"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Stats       PoAGraphStats          `json:"stats"`
}

// PoAGraphStats contains statistics about the graph
type PoAGraphStats struct {
	TotalNodes         int     `json:"total_nodes"`
	TotalEdges         int     `json:"total_edges"`
	ActiveNodes        int     `json:"active_nodes"`
	PendingNodes       int     `json:"pending_nodes"`
	RevokedNodes       int     `json:"revoked_nodes"`
	AverageConnections float64 `json:"average_connections"`
	MaxDepth           int     `json:"max_depth"`
}

// ProtocolStepVisualization represents a 3D visualization of a protocol step
type ProtocolStepVisualization struct {
	StepID      string                 `json:"step_id"`
	StepName    string                 `json:"step_name"`
	Description string                 `json:"description"`
	Icon        string                 `json:"icon"`
	Layers      []VisualizationLayer   `json:"layers"`
	Connections []LayerConnection      `json:"connections"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

// VisualizationLayer represents a layer in the 3D protocol visualization
type VisualizationLayer struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Level       int                    `json:"level"` // Z-axis level (0=bottom, higher=top)
	Components  []LayerComponent       `json:"components"`
	Color       string                 `json:"color"`
	Opacity     float64                `json:"opacity"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// LayerComponent represents a component within a visualization layer
type LayerComponent struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // actor, process, data, validation
	Label    string                 `json:"label"`
	Icon     string                 `json:"icon"`
	Position Position               `json:"position"`
	Size     float64                `json:"size"`
	Color    string                 `json:"color"`
	Status   string                 `json:"status"` // active, processing, completed, error
	Metadata map[string]interface{} `json:"metadata"`
}

// LayerConnection represents a connection between layer components
type LayerConnection struct {
	ID          string                 `json:"id"`
	SourceLayer string                 `json:"source_layer"`
	SourceComp  string                 `json:"source_component"`
	TargetLayer string                 `json:"target_layer"`
	TargetComp  string                 `json:"target_component"`
	Type        string                 `json:"type"` // data-flow, control-flow, dependency
	Animated    bool                   `json:"animated"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PoAVisualizer creates and manages PoA visualizations
type PoAVisualizer struct {
	graphs map[string]*PoAGraph
}

// NewPoAVisualizer creates a new visualizer
func NewPoAVisualizer() *PoAVisualizer {
	return &PoAVisualizer{
		graphs: make(map[string]*PoAGraph),
	}
}

// CreateGraph creates a new PoA graph
func (v *PoAVisualizer) CreateGraph(name, description string) *PoAGraph {
	now := time.Now()
	id := generateGraphID(name, now)

	graph := &PoAGraph{
		ID:          id,
		Name:        name,
		Description: description,
		Nodes:       []PoANode{},
		Edges:       []PoAEdge{},
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
		Stats:       PoAGraphStats{},
	}

	v.graphs[id] = graph
	return graph
}

// AddNode adds a node to the graph
func (g *PoAGraph) AddNode(nodeType, label, description, icon string, metadata map[string]interface{}) *PoANode {
	now := time.Now()
	id := generateNodeID(nodeType, label, now)

	node := PoANode{
		ID:          id,
		Type:        nodeType,
		Label:       label,
		Description: description,
		Icon:        icon,
		Status:      "active",
		Metadata:    metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	g.Nodes = append(g.Nodes, node)
	g.UpdatedAt = now
	g.updateStats()

	return &node
}

// AddEdge adds an edge between two nodes
func (g *PoAGraph) AddEdge(sourceID, targetID, edgeType, label string, strength float64, metadata map[string]interface{}) *PoAEdge {
	now := time.Now()
	id := generateEdgeID(sourceID, targetID, edgeType)

	edge := PoAEdge{
		ID:            id,
		Source:        sourceID,
		Target:        targetID,
		Type:          edgeType,
		Label:         label,
		Strength:      strength,
		Bidirectional: false,
		Metadata:      metadata,
		CreatedAt:     now,
	}

	g.Edges = append(g.Edges, edge)
	g.UpdatedAt = now
	g.updateStats()

	return &edge
}

// SetNodePosition sets 3D position for a node
func (g *PoAGraph) SetNodePosition(nodeID string, x, y, z float64) error {
	for i := range g.Nodes {
		if g.Nodes[i].ID == nodeID {
			g.Nodes[i].Position = &Position{X: x, Y: y, Z: z}
			g.Nodes[i].UpdatedAt = time.Now()
			g.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("node not found: %s", nodeID)
}

// UpdateNodeStatus updates the status of a node
func (g *PoAGraph) UpdateNodeStatus(nodeID, status string) error {
	for i := range g.Nodes {
		if g.Nodes[i].ID == nodeID {
			g.Nodes[i].Status = status
			g.Nodes[i].UpdatedAt = time.Now()
			g.UpdatedAt = time.Now()
			g.updateStats()
			return nil
		}
	}
	return fmt.Errorf("node not found: %s", nodeID)
}

// GetNode retrieves a node by ID
func (g *PoAGraph) GetNode(nodeID string) *PoANode {
	for i := range g.Nodes {
		if g.Nodes[i].ID == nodeID {
			return &g.Nodes[i]
		}
	}
	return nil
}

// GetConnectedNodes returns all nodes connected to the given node
func (g *PoAGraph) GetConnectedNodes(nodeID string) []PoANode {
	connected := []PoANode{}

	for _, edge := range g.Edges {
		var targetID string
		if edge.Source == nodeID {
			targetID = edge.Target
		} else if edge.Target == nodeID && edge.Bidirectional {
			targetID = edge.Source
		}

		if targetID != "" {
			if node := g.GetNode(targetID); node != nil {
				connected = append(connected, *node)
			}
		}
	}

	return connected
}

// updateStats recalculates graph statistics
func (g *PoAGraph) updateStats() {
	g.Stats.TotalNodes = len(g.Nodes)
	g.Stats.TotalEdges = len(g.Edges)

	activeCount := 0
	pendingCount := 0
	revokedCount := 0

	for _, node := range g.Nodes {
		switch node.Status {
		case "active":
			activeCount++
		case "pending":
			pendingCount++
		case "revoked":
			revokedCount++
		}
	}

	g.Stats.ActiveNodes = activeCount
	g.Stats.PendingNodes = pendingCount
	g.Stats.RevokedNodes = revokedCount

	if len(g.Nodes) > 0 {
		g.Stats.AverageConnections = float64(len(g.Edges)*2) / float64(len(g.Nodes))
	}

	// Calculate max depth (simplified - would need proper graph traversal)
	g.Stats.MaxDepth = calculateMaxDepth(g)
}

// CreateSubscriptionVisualization creates visualization for subscription step
func CreateSubscriptionVisualization() *ProtocolStepVisualization {
	viz := &ProtocolStepVisualization{
		StepID:      "subscription",
		StepName:    "Subscription",
		Description: "Client registration and authorization server setup",
		Icon:        "📝",
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Layer 0: Client Registration
	layer0 := VisualizationLayer{
		ID:          "layer-client",
		Name:        "Client Layer",
		Description: "AI Client registration and configuration",
		Level:       0,
		Color:       "#667eea",
		Opacity:     0.9,
		Components: []LayerComponent{
			{
				ID:       "client-ai",
				Type:     "actor",
				Label:    "AI Client",
				Icon:     "🤖",
				Position: Position{X: -2, Y: 0, Z: 0},
				Size:     1.0,
				Color:    "#667eea",
				Status:   "active",
				Metadata: map[string]interface{}{"role": "requestor"},
			},
		},
	}

	// Layer 1: Authorization Server
	layer1 := VisualizationLayer{
		ID:          "layer-authz",
		Name:        "Authorization Layer",
		Description: "Authorization server and token endpoint",
		Level:       1,
		Color:       "#f59e0b",
		Opacity:     0.9,
		Components: []LayerComponent{
			{
				ID:       "authz-server",
				Type:     "process",
				Label:    "Authorization Server",
				Icon:     "🏛️",
				Position: Position{X: 0, Y: 0, Z: 1},
				Size:     1.2,
				Color:    "#f59e0b",
				Status:   "processing",
				Metadata: map[string]interface{}{"endpoints": []string{"/register", "/token"}},
			},
		},
	}

	// Layer 2: Credential Storage
	layer2 := VisualizationLayer{
		ID:          "layer-storage",
		Name:        "Storage Layer",
		Description: "Client credentials and configuration storage",
		Level:       2,
		Color:       "#10b981",
		Opacity:     0.8,
		Components: []LayerComponent{
			{
				ID:       "credential-store",
				Type:     "data",
				Label:    "Credential Store",
				Icon:     "🔐",
				Position: Position{X: 2, Y: 0, Z: 2},
				Size:     0.8,
				Color:    "#10b981",
				Status:   "active",
				Metadata: map[string]interface{}{"type": "secure_storage"},
			},
		},
	}

	viz.Layers = []VisualizationLayer{layer0, layer1, layer2}

	// Connections
	viz.Connections = []LayerConnection{
		{
			ID:          "conn-register",
			SourceLayer: "layer-client",
			SourceComp:  "client-ai",
			TargetLayer: "layer-authz",
			TargetComp:  "authz-server",
			Type:        "control-flow",
			Animated:    true,
			Metadata:    map[string]interface{}{"action": "register"},
		},
		{
			ID:          "conn-store",
			SourceLayer: "layer-authz",
			SourceComp:  "authz-server",
			TargetLayer: "layer-storage",
			TargetComp:  "credential-store",
			Type:        "data-flow",
			Animated:    true,
			Metadata:    map[string]interface{}{"action": "persist"},
		},
	}

	return viz
}

// CreateMatchingVisualization creates visualization for matching step
func CreateMatchingVisualization() *ProtocolStepVisualization {
	viz := &ProtocolStepVisualization{
		StepID:      "matching",
		StepName:    "Matching",
		Description: "PoA Definition validation and capability matching",
		Icon:        "🔍",
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Layer 0: PoA Definition
	layer0 := VisualizationLayer{
		ID:          "layer-poa",
		Name:        "PoA Definition Layer",
		Description: "Proof of Authorization definition document",
		Level:       0,
		Color:       "#8b5cf6",
		Opacity:     0.9,
		Components: []LayerComponent{
			{
				ID:       "poa-definition",
				Type:     "data",
				Label:    "PoA Definition",
				Icon:     "📄",
				Position: Position{X: -2, Y: 0, Z: 0},
				Size:     1.0,
				Color:    "#8b5cf6",
				Status:   "active",
				Metadata: map[string]interface{}{"sections": []string{"parties", "scope", "requirements"}},
			},
		},
	}

	// Layer 1: Validation Engines
	layer1 := VisualizationLayer{
		ID:          "layer-validation",
		Name:        "Validation Layer",
		Description: "Multiple validation engines processing PoA",
		Level:       1,
		Color:       "#3b82f6",
		Opacity:     0.9,
		Components: []LayerComponent{
			{
				ID:       "poa-validator",
				Type:     "validation",
				Label:    "PoA Validator",
				Icon:     "✓",
				Position: Position{X: -1, Y: 1, Z: 1},
				Size:     0.8,
				Color:    "#3b82f6",
				Status:   "processing",
				Metadata: map[string]interface{}{"type": "structure"},
			},
			{
				ID:       "capability-matcher",
				Type:     "validation",
				Label:    "Capability Matcher",
				Icon:     "🎯",
				Position: Position{X: 0, Y: -1, Z: 1},
				Size:     0.8,
				Color:    "#06b6d4",
				Status:   "processing",
				Metadata: map[string]interface{}{"type": "capability"},
			},
			{
				ID:       "jurisdiction-validator",
				Type:     "validation",
				Label:    "Jurisdiction Validator",
				Icon:     "⚖️",
				Position: Position{X: 1, Y: 1, Z: 1},
				Size:     0.8,
				Color:    "#0ea5e9",
				Status:   "processing",
				Metadata: map[string]interface{}{"type": "legal"},
			},
		},
	}

	// Layer 2: Decision
	layer2 := VisualizationLayer{
		ID:          "layer-decision",
		Name:        "Decision Layer",
		Description: "Matching decision outcome",
		Level:       2,
		Color:       "#10b981",
		Opacity:     0.9,
		Components: []LayerComponent{
			{
				ID:       "match-result",
				Type:     "process",
				Label:    "Match Result",
				Icon:     "✓",
				Position: Position{X: 0, Y: 0, Z: 2},
				Size:     1.2,
				Color:    "#10b981",
				Status:   "completed",
				Metadata: map[string]interface{}{"outcome": "approved"},
			},
		},
	}

	viz.Layers = []VisualizationLayer{layer0, layer1, layer2}

	// Connections
	viz.Connections = []LayerConnection{
		{
			ID:          "conn-validate-poa",
			SourceLayer: "layer-poa",
			SourceComp:  "poa-definition",
			TargetLayer: "layer-validation",
			TargetComp:  "poa-validator",
			Type:        "data-flow",
			Animated:    true,
		},
		{
			ID:          "conn-validate-capability",
			SourceLayer: "layer-poa",
			SourceComp:  "poa-definition",
			TargetLayer: "layer-validation",
			TargetComp:  "capability-matcher",
			Type:        "data-flow",
			Animated:    true,
		},
		{
			ID:          "conn-validate-jurisdiction",
			SourceLayer: "layer-poa",
			SourceComp:  "poa-definition",
			TargetLayer: "layer-validation",
			TargetComp:  "jurisdiction-validator",
			Type:        "data-flow",
			Animated:    true,
		},
		{
			ID:          "conn-decision",
			SourceLayer: "layer-validation",
			SourceComp:  "poa-validator",
			TargetLayer: "layer-decision",
			TargetComp:  "match-result",
			Type:        "control-flow",
			Animated:    true,
		},
	}

	return viz
}

// CreateRequestVisualization creates visualization for subset/request step
func CreateRequestVisualization() *ProtocolStepVisualization {
	viz := &ProtocolStepVisualization{
		StepID:      "subset_request",
		StepName:    "Subset/Request",
		Description: "Authorization request with scope subset selection",
		Icon:        "🎯",
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Layer 0: Authorization Request
	layer0 := VisualizationLayer{
		ID:          "layer-request",
		Name:        "Request Layer",
		Description: "Client authorization request",
		Level:       0,
		Color:       "#ec4899",
		Opacity:     0.9,
		Components: []LayerComponent{
			{
				ID:       "authz-request",
				Type:     "data",
				Label:    "Authorization Request",
				Icon:     "📋",
				Position: Position{X: -2, Y: 0, Z: 0},
				Size:     1.0,
				Color:    "#ec4899",
				Status:   "active",
				Metadata: map[string]interface{}{"scopes": []string{"read", "write"}},
			},
		},
	}

	// Layer 1: PDP Processing
	layer1 := VisualizationLayer{
		ID:          "layer-pdp",
		Name:        "PDP Layer",
		Description: "Policy Decision Point evaluation",
		Level:       1,
		Color:       "#f59e0b",
		Opacity:     0.9,
		Components: []LayerComponent{
			{
				ID:       "pdp-engine",
				Type:     "process",
				Label:    "PDP Engine",
				Icon:     "⚙️",
				Position: Position{X: 0, Y: 0, Z: 1},
				Size:     1.2,
				Color:    "#f59e0b",
				Status:   "processing",
				Metadata: map[string]interface{}{"algorithm": "permit-overrides"},
			},
		},
	}

	// Layer 2: Token Generation
	layer2 := VisualizationLayer{
		ID:          "layer-token",
		Name:        "Token Layer",
		Description: "Access token generation",
		Level:       2,
		Color:       "#10b981",
		Opacity:     0.9,
		Components: []LayerComponent{
			{
				ID:       "token-generator",
				Type:     "process",
				Label:    "Token Generator",
				Icon:     "🎫",
				Position: Position{X: 2, Y: 0, Z: 2},
				Size:     1.0,
				Color:    "#10b981",
				Status:   "completed",
				Metadata: map[string]interface{}{"format": "JWT"},
			},
		},
	}

	viz.Layers = []VisualizationLayer{layer0, layer1, layer2}

	// Connections
	viz.Connections = []LayerConnection{
		{
			ID:          "conn-evaluate",
			SourceLayer: "layer-request",
			SourceComp:  "authz-request",
			TargetLayer: "layer-pdp",
			TargetComp:  "pdp-engine",
			Type:        "control-flow",
			Animated:    true,
		},
		{
			ID:          "conn-generate",
			SourceLayer: "layer-pdp",
			SourceComp:  "pdp-engine",
			TargetLayer: "layer-token",
			TargetComp:  "token-generator",
			Type:        "data-flow",
			Animated:    true,
		},
	}

	return viz
}

// Utility functions

func generateGraphID(name string, t time.Time) string {
	data := fmt.Sprintf("%s-%d", name, t.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "graph-" + hex.EncodeToString(hash[:])[:16]
}

func generateNodeID(nodeType, label string, t time.Time) string {
	data := fmt.Sprintf("%s-%s-%d", nodeType, label, t.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "node-" + hex.EncodeToString(hash[:])[:12]
}

func generateEdgeID(source, target, edgeType string) string {
	data := fmt.Sprintf("%s-%s-%s", source, target, edgeType)
	hash := sha256.Sum256([]byte(data))
	return "edge-" + hex.EncodeToString(hash[:])[:12]
}

func calculateMaxDepth(g *PoAGraph) int {
	// Simplified depth calculation - would need proper graph traversal for accuracy
	if len(g.Nodes) == 0 {
		return 0
	}

	maxZ := 0.0
	for _, node := range g.Nodes {
		if node.Position != nil && node.Position.Z > maxZ {
			maxZ = node.Position.Z
		}
	}

	return int(maxZ) + 1
}
