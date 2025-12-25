---
title: Poa Visualization
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# PoA Visualization System

Think Machine-inspired 3D visualization for Power of Attorney relationships and protocol flows.

## Overview

The PoA Visualization system provides interactive 3D visualizations for understanding:
- **Power of Attorney Graphs**: Relationship networks between principals, agents, AI clients, and resources
- **Protocol Steps**: Layered 3D visualization of GAuth protocol steps (Subscription, Matching, Request)

## Architecture

### Backend (`pkg/visualization/`)

- **`poa_viz.go`**: Core visualization data structures and graph management
  - `PoAGraph`: Manages nodes and edges with 3D positions
  - `PoANode`: Represents entities (principal, client, resource, authorizer)
  - `PoAEdge`: Represents relationships (authorizes, delegates, requests)
  - `ProtocolStepVisualization`: 3D layered protocol step representation
  - Pre-built visualizations for Subscription, Matching, and Request steps

- **`poa_viz_test.go`**: Comprehensive test coverage (13/13 tests passing)
  - Graph creation and manipulation
  - Node position management
  - Status tracking and statistics
  - Protocol step generation
  - Complex multi-party scenarios

### Frontend (`web/static/`)

- **`js/modules/poa-viz.js`**: Three.js-based 3D visualization
  - `PoAGraphVisualizer`: Renders PoA relationship graphs with curved edges
  - `ProtocolStepVisualizer`: Renders layered protocol steps with animated connections
  - Real-time animations, lighting effects, camera controls
  - Status-based coloring (active=green, pending=orange, revoked=red, expired=gray)

- **`css/poa-viz.css`**: Think Machine-inspired dark UI
  - Gradient backgrounds and grid effects
  - Control panels with glassmorphism
  - Scan line animations
  - Responsive design with mobile breakpoints
  - Accessibility features (reduced motion, high contrast)

### API Endpoints (`web/visualization.go`)

```
POST   /api/v1/visualization/poa/graphs                  # Create graph
GET    /api/v1/visualization/poa/graphs/:id              # Get graph by ID
POST   /api/v1/visualization/poa/graphs/:id/nodes        # Add node
POST   /api/v1/visualization/poa/graphs/:id/edges        # Add edge
PUT    /api/v1/visualization/poa/graphs/:id/nodes/position  # Update node position
PUT    /api/v1/visualization/poa/graphs/:id/nodes/:node_id/status  # Update node status

GET    /api/v1/visualization/protocol/subscription       # Subscription step viz
GET    /api/v1/visualization/protocol/matching           # Matching step viz
GET    /api/v1/visualization/protocol/request            # Request step viz

POST   /api/v1/visualization/demo/complex-graph          # Generate demo graph
```

### Demo Page (`web/templates/poa-visualization.html`)

Interactive demonstration with:
- Mode selector (PoA Graph / Protocol Steps)
- Graph type selector (Demo, Simple, Multi-Party)
- Protocol step selector (Subscription, Matching, Request)
- Real-time statistics panel
- Legend with color meanings
- Control buttons (Load, Clear, Rotate, Reset View)
- Info panel with contextual help

## Features

### PoA Graph Visualization

**Node Types:**
- 🏢 **Principal**: Octahedron geometry (organizations, individuals)
- 🤖 **Client**: Tetrahedron geometry (AI applications)
- 💾 **Resource**: Box geometry (data, services)
- 👔 **Authorizer**: Sphere geometry (agents, delegates)

**Edge Types:**
- **Authorizes** (Purple #667eea): Authorization relationships
- **Delegates** (Orange #f59e0b): Delegation chains
- **Requests** (Green #10b981): Access requests
- **Validates** (Blue #3b82f6): Validation flows

**Status Indicators:**
- **Active** (Green): Operational and authorized
- **Pending** (Orange): Awaiting approval, animated pulse
- **Revoked** (Red): Authorization revoked
- **Expired** (Gray): Time-limited authorization expired

### Protocol Step Visualization

**Layered 3D Rendering:**
- Each protocol step rendered as stacked layers in 3D space
- Components positioned on cylindrical platforms (Z-axis = layer level)
- Animated connections show data flow and control flow
- Processing indicators (rotating rings) for active components

**Subscription Step:**
- Layer 0: Client registration
- Layer 1: Authorization server
- Layer 2: Credential storage

**Matching Step:**
- Layer 0: PoA Definition document
- Layer 1: Validation engines (PoA, Capability, Jurisdiction)
- Layer 2: Match decision result

**Request Step:**
- Layer 0: Authorization request
- Layer 1: PDP evaluation
- Layer 2: Token generation

### Visual Design

**Think Machine Aesthetics:**
- Dark background with fog effects (#0a0a0a)
- Dramatic point lighting (purple, orange, green)
- Grid helper for spatial reference
- Glassmorphism UI panels with blur
- Emissive materials for node glow
- Curved Bézier edges for visual flow
- Scan line animation effect

**Animations:**
- Node rotation (0.002 rad/frame)
- Pending node pulse (sin wave scale)
- Connection opacity modulation
- Processing indicator rotation (0.05 rad/frame)
- Auto-rotate camera option

**Controls:**
- Orbit controls (mouse drag to rotate)
- Zoom with mouse wheel (min=3, max=50)
- Damping for smooth motion
- Reset view button
- Auto-rotate toggle

## Usage

### Basic PoA Graph

```go
import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/visualization"

// Create visualizer
viz := visualization.NewPoAVisualizer()

// Create graph
graph := viz.CreateGraph("Authorization Network", "Multi-party PoA relationships")

// Add principal
principal := graph.AddNode("principal", "Acme Corp", "Principal organization", "🏢", nil)
graph.SetNodePosition(principal.ID, 0, 0, 0)

// Add AI client
aiClient := graph.AddNode("client", "AI Assistant", "AI application", "🤖", nil)
graph.SetNodePosition(aiClient.ID, 3, 0, 2)

// Add authorization edge
graph.AddEdge(principal.ID, aiClient.ID, "authorizes", "Authorizes AI", 1.0, nil)

// Update status
graph.UpdateNodeStatus(aiClient.ID, "active")

// Access statistics
stats := graph.Stats  // total_nodes, total_edges, active_nodes, etc.
```

### Protocol Step Visualization

```go
// Generate subscription visualization
subViz := visualization.CreateSubscriptionVisualization()
// Returns ProtocolStepVisualization with 3 layers, 2 connections

// Generate matching visualization
matchViz := visualization.CreateMatchingVisualization()
// Returns ProtocolStepVisualization with 3 layers, 3 validators, 4 connections

// Generate request visualization
reqViz := visualization.CreateRequestVisualization()
// Returns ProtocolStepVisualization with 3 layers (request, PDP, token)
```

### API Integration

```javascript
// Load demo complex graph
const response = await fetch('/api/v1/visualization/demo/complex-graph', {
    method: 'POST'
});
const { graph } = await response.json();

// Create visualizer
import { PoAGraphVisualizer } from '/static/js/modules/poa-viz.js';
const viz = new PoAGraphVisualizer(document.getElementById('canvas'));

// Load and render
await viz.loadGraph(graph);
```

## Testing

Run comprehensive test suite:

```bash
go test ./pkg/visualization/... -v
```

**Test Coverage:**
- `TestNewPoAVisualizer`: Visualizer initialization
- `TestCreateGraph`: Graph creation and metadata
- `TestAddNode`: Node addition and validation
- `TestAddEdge`: Edge creation between nodes
- `TestSetNodePosition`: 3D coordinate management
- `TestUpdateNodeStatus`: Status tracking and stats
- `TestGetConnectedNodes`: Relationship traversal
- `TestGraphStats`: Statistics calculation
- `TestCreateSubscriptionVisualization`: Subscription step generation
- `TestCreateMatchingVisualization`: Matching step with validators
- `TestCreateRequestVisualization`: Request step with PDP
- `TestGenerateIDs`: ID generation consistency
- `TestComplexGraph`: Multi-party authorization scenario

**All tests passing: 13/13**

## Demo

Access the interactive demo:

```bash
# Start server with dev mode for hot reload
GAUTH_DEV_INDEX=1 go run ./cmd/web-server

# Open browser
open http://localhost:8080/poa-visualization
```

**Demo Controls:**
1. Select "Visualization Mode": PoA Graph or Protocol Steps
2. Choose graph type or protocol step
3. Click "Load" to render
4. Use mouse to orbit, zoom, and explore
5. Toggle "Rotate" for auto-rotation
6. Click "Clear" to reset

## Integration with Items 1 & 2

**Item 1 (Enforcement):**
- Visualizes enforcement decision trees
- Shows PEP (supply/demand-side) relationships
- Displays policy rule graph structures
- Maps AI integration interfaces

**Item 2 (Protocol Navigator):**
- Complements step-by-step navigation with 3D spatial view
- Protocol navigator tracks progress, visualization shows structure
- Breadcrumb navigation in 2D, layered protocol steps in 3D
- Session state from navigator can drive visualization updates

## RFC Compliance

**GiFo-RFC-0111 (GAuth 1.0):**
- Visualizes Section 3.A parties (Principal, Authorizer, Client, Resource)
- Represents subscription, matching, request protocol steps
- Shows delegation chains and authorization relationships

**GiFo-RFC-0115 (PoA Definition):**
- Visualizes PoA Definition structure and sections
- Renders parties, scope, and requirements graph
- Shows validation flow (structural, capability, jurisdiction)

## Performance Considerations

**Optimization:**
- Three.js GPU-accelerated rendering
- Frustum culling for off-screen objects
- Level-of-detail for large graphs (future)
- Batch geometry updates
- Damped animation frames

**Limits:**
- Recommended max nodes: 100 for smooth interaction
- Recommended max edges: 200
- Mobile devices: reduce particle effects

## Future Enhancements

**Planned Features:**
- Real-time updates via WebSocket
- VR mode for immersive exploration
- Timeline playback (show authorization evolution)
- Heat map overlays (activity, risk scores)
- Export to image/video
- Graph layout algorithms (force-directed, hierarchical)
- Minimap for large graphs
- Search and filter nodes/edges

## Dependencies

**Backend:**
- Go 1.21+
- github.com/gin-gonic/gin

**Frontend:**
- Three.js 0.160.0 (CDN)
- OrbitControls (Three.js examples)
- Modern browser with WebGL support

## License

Apache 2.0 - Gimel Foundation gGmbH i.G.

## Related Documentation

- `pkg/enforcement/`: Item 1 - Enforcement mechanisms
- `web/static/js/modules/protocol-navigator.js`: Item 2 - Protocol flow navigator
- `docs/RFC_ARCHITECTURE.md`: P*P architecture (PEP, PDP, PIP, PAP, PVP)
- `README.md`: Main project documentation
