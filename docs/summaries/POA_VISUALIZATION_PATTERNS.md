---
title: Poa Visualization Patterns
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# PoA Visualization Pattern Enhancement - Complete

## Summary
Successfully extended the PoA visualization to include all major AgentAuth authorization patterns. The visualization now supports 9 different pattern types demonstrating various aspects of the Proof of Authorization system.

## Changes Made

### 1. Updated Files
- **web/static/js/modules/poa-viz.js**: Added ~350 lines of pattern generation functions
- **web/templates/poa-visualization.html**: Updated imports and load logic

### 2. New Pattern Types

#### Cascade Revocation Pattern (`cascade`)
- **Demonstrates**: Hierarchical PoA with parent-child relationships
- **Features**: Shows how revoking a root PoA automatically suspends all descendants
- **Nodes**: 7 (1 root revoked, 6 children suspended)
- **Use Case**: Organizational hierarchy where parent authority revocation cascades down

#### Delegation Chain Pattern (`delegation-chain`)
- **Demonstrates**: Multi-level delegation depth
- **Features**: Shows 5-level delegation chain from issuer to end user
- **Nodes**: 6 (issuer → 4 delegates → consumer)
- **Use Case**: Supply chain or multi-tier service authorization

#### Multi-Signature Pattern (`multisig`)
- **Demonstrates**: Collective authorization with threshold requirements
- **Features**: 3-of-5 threshold signature visualization with signed/pending states
- **Nodes**: 7 (5 signers + 1 multisig PoA + 1 resource)
- **Use Case**: Financial transactions or critical operations requiring multiple approvals

#### Jurisdiction Pattern (`jurisdiction`)
- **Demonstrates**: Geographic/legal boundary constraints
- **Features**: Shows GDPR/HIPAA compliance boundaries, cross-border restrictions
- **Nodes**: 9 (global issuer → 3 regional delegates → 4 services + 1 blocked)
- **Use Case**: International data governance, regulatory compliance

#### Revocation States Pattern (`revocation`)
- **Demonstrates**: Different PoA lifecycle states and transparency logging
- **Features**: Shows active/suspended/revoked/expired/pending states with Merkle tree
- **Nodes**: 7 (5 PoAs in different states + 1 revocation root + 1 transparency log)
- **Use Case**: Audit trails, compliance monitoring, state transitions

#### Full Pattern (`full-pattern`)
- **Demonstrates**: Combined view of all patterns
- **Features**: Cascade + delegation + multisig + jurisdiction in one view
- **Nodes**: 13 (demonstrates complexity of real-world scenarios)
- **Use Case**: Enterprise-grade authorization system visualization

#### Existing Patterns
- **simple**: Basic 3-node graph (Alice → Bob → Charlie)
- **multi**: Multi-party organization (7 nodes, shared authorizations)
- **demo**: Complex graph fetched from backend API (10 nodes, various states)

## Technical Implementation

### Pattern Generator Functions
Each pattern generator returns a standardized data structure:
```javascript
{
    nodes: [
        { 
            id: 'unique-id',
            label: 'Display Name',
            type: 'root|delegation|consumer|signer',
            status: 'active|pending|suspended|revoked',
            position: { x, y, z },
            metadata: { /* pattern-specific data */ }
        }
    ],
    edges: [
        {
            source: 'node-id',
            target: 'node-id',
            type: 'delegation|authorization|signature|...',
            label: 'Edge label'
        }
    ],
    stats: {
        total_nodes: N,
        total_edges: M,
        active_nodes: A,
        pending_nodes: P,
        revoked_nodes: R
    }
}
```

### Export Function
`generateDemoGraph(type)` - Main entry point that routes pattern type to appropriate generator.

### Color Coding
- **Root nodes**: Yellow (#ffeb3b)
- **Delegation nodes**: Blue (#2196f3)
- **Consumer nodes**: Green (#4caf50)
- **Signer nodes**: Orange (#ff9800)
- **Status states**: Active (green), Pending (yellow), Suspended (orange), Revoked (red)

## How to Use

1. **Start the server** (if not running):
   ```bash
   AGENTAUTH_DEV_INDEX=1 go run ./cmd/web-server
   ```

2. **Open the visualization**:
   Navigate to http://localhost:8080/poa-visualization.html

3. **Select pattern**:
   - Choose from dropdown: demo, simple, multi, cascade, delegation-chain, multisig, jurisdiction, revocation, full-pattern
   - Click "Load" button
   - Use mouse to interact:
     - Left-drag: Rotate view
     - Right-drag: Pan view
     - Scroll: Zoom in/out
   - Click "Rotate" to enable auto-rotation
   - Click "Reset View" to return to default camera position

## Pattern Details

### Cascade Revocation
- Root PoA at (0, 2, 0) - REVOKED
- 2 child PoAs at depth 1 - SUSPENDED
- 4 grandchild PoAs at depth 2 - SUSPENDED
- All edges are "parent-child" type
- Metadata includes depth tracking and cascade_trigger flag

### Delegation Chain
- Vertical chain from y=3 to y=-2
- Z-axis progression (-2 to -10) for depth visualization
- Each level clearly labeled with depth metadata
- Single linear path from issuer to consumer

### Multi-Signature
- Central multisig PoA with 5 signers arranged in arc
- 3 signers show ✓ mark (signed)
- 2 signers show no mark (pending)
- Resource node awaiting threshold completion
- Edges differentiate signed vs pending states

### Jurisdiction
- 3 regional branches (EU, US, Asia)
- Each region has 1-2 services in proper jurisdiction
- Blocked cross-border transfer shown as violation
- Metadata includes location, compliance flags (GDPR, HIPAA)

### Revocation States
- Shows all 5 lifecycle states simultaneously
- Transparency log node connected to revoked/expired PoAs
- Merkle tree inclusion proofs shown as edges
- STH (Signed Tree Head) verification metadata

### Full Pattern
- 13 nodes combining multiple patterns
- EU branch uses multisig
- US branch uses delegation chain
- Asia branch demonstrates cascade revocation
- Shows real-world complexity

## Development Notes

### File Locations
- Pattern generators: `/web/static/js/modules/poa-viz.js` (lines 755-1125 approx)
- HTML template: `/web/templates/poa-visualization.html`
- Import statement: `import { PoAGraphVisualizer, ProtocolStepVisualizer, generateDemoGraph }`

### Dependencies
- Three.js v0.160.0 (CDN: jsdelivr)
- OrbitControls for camera interaction
- WebGL for rendering

### Browser Requirements
- Modern browser with ES6 module support
- WebGL-enabled GPU
- Minimum 1024x768 resolution recommended

## Future Enhancements
- [ ] Add time-based animation showing cascade propagation
- [ ] Interactive node clicking for detailed metadata display
- [ ] Graph export to JSON/SVG
- [ ] Filter by status (show only active/pending/revoked)
- [ ] Search/highlight specific nodes
- [ ] Performance mode for graphs >100 nodes
- [ ] Mobile-optimized controls

## Related Documentation
- `WEB_UI_USAGE_GUIDE.md` - General web UI documentation
- `internal/cascade/processor.go` - Cascade revocation implementation
- `web/protocol_flow.go` - Protocol flow API handlers
- `WEBAPP_VISUAL_GUIDE.md` - Overall webapp architecture

## Testing Checklist
- [x] Server starts with AGENTAUTH_DEV_INDEX=1
- [x] Page loads at /poa-visualization.html
- [x] Three.js imports resolve correctly
- [x] Dropdown shows all 9 pattern options
- [x] generateDemoGraph function exported and accessible
- [ ] All patterns render correctly (manual browser test needed)
- [ ] Camera controls work (orbit, zoom, pan)
- [ ] Statistics panel updates correctly
- [ ] Auto-rotation functions properly
- [ ] Reset view button works
- [ ] Mobile responsive (if applicable)

## Completion Status
**Phase 1: Code Implementation** - ✅ COMPLETE
- Pattern generator functions added
- HTML template updated with new patterns
- Import statements corrected
- Load function refactored to use generators

**Phase 2: Browser Testing** - ⏳ PENDING USER VERIFICATION
- User should test all 9 patterns in browser
- Verify visual correctness
- Check for JavaScript console errors
- Validate 3D positioning and layout

**Phase 3: Documentation** - ✅ COMPLETE
- This document serves as comprehensive guide
- Pattern details documented
- Usage instructions provided
- Future enhancements outlined

---

**Created**: 2025-11-07
**Last Updated**: 2025-11-07
**Status**: Ready for user testing
**Server**: Running at localhost:8080 (PID: 5986)
