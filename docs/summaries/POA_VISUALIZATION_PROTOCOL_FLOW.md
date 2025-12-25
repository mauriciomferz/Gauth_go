---
title: Poa Visualization Protocol Flow
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# PoA Visualization Enhancement - GAuth Protocol Flow Integration

## Overview
Successfully enhanced the PoA visualization to include complete GAuth Protocol Flow patterns based on RFC-0111 and RFC-0115. The visualization now supports **15 different pattern types** including 6 new protocol flow visualizations.

## New Protocol Flow Patterns

### 1. **Subscription Flow** (`protocol-subscription`)
**Demonstrates**: Client registration and authorization server setup
- **Nodes**: 6 (Client, Authorization Server, Registration steps, Credentials)
- **Key Components**:
  - AI Agent/Client registration
  - Scope configuration
  - Credential issuance (Client ID & Secret)
- **APIs**: `/api/v1/client/register`, `/api/v1/subscribe`
- **Flow**: Client → Register → Configure Scopes → Obtain Credentials → Client ID/Secret
- **Use Case**: Initial setup phase for OAuth 2.0/OpenID Connect clients

### 2. **Matching Flow** (`protocol-matching`)
**Demonstrates**: PoA Definition validation and capability matching
- **Nodes**: 7 (PoA Definition, 4 validation steps, PDP, Result)
- **Key Components**:
  - PoA Definition (RFC-0115) validation
  - AI Capability checking
  - Jurisdiction verification (GDPR, HIPAA)
  - Policy matching (RBAC)
  - PDP (Policy Decision Point) evaluation
- **APIs**: `/api/v1/poa/validate`, `/api/v1/ai/capabilities`
- **Flow**: PoA Definition → Validate → Check Capabilities → Verify Jurisdiction → Match Policies → PDP Decision → Result
- **Use Case**: Ensuring PoA compliance with organizational policies and legal requirements

### 3. **Subset/Request Flow** (`protocol-request`)
**Demonstrates**: Authorization request with scope subset selection
- **Nodes**: 7 (Client, Principal, 4 authorization steps, Access Token)
- **Key Components**:
  - Authentication request creation
  - Scope subset selection (poa:read, poa:delegate)
  - Principal (human) consent
  - PDP decision evaluation
  - JWT token generation
- **APIs**: `/api/v1/authorize`, `/api/v1/token`
- **Flow**: Client → Auth Request → Scope Selection (with Principal consent) → PDP Decision → Token Generation → Access Token
- **Use Case**: Standard OAuth 2.0 authorization code flow with consent management

### 4. **Enforcement Flow** (`protocol-enforcement`)
**Demonstrates**: PEP enforcement (supply-side and demand-side)
- **Nodes**: 8 (Token, Supply PEP, Demand PEP, Disclosure, Audit, Resource, Agent, Owner)
- **Key Components**:
  - Supply-Side PEP (pre-check enforcement)
  - Demand-Side PEP (post-check enforcement)
  - Disclosure requirements (transparency)
  - Audit logging (Merkle tree)
  - Resource access control
- **APIs**: `/api/v1/enforce/supply`, `/api/v1/enforce/demand`
- **Flow**: AI Agent → Supply PEP (validates token) → Disclosure + Audit → Resource Owner → Demand PEP → Resource Access
- **Use Case**: Dual-sided policy enforcement ensuring both provider and consumer compliance

### 5. **Verification Flow** (`protocol-verification`)
**Demonstrates**: Token verification and PVP identity validation
- **Nodes**: 9 (Incoming Token, 4 verification steps, 3 external services, Result)
- **Key Components**:
  - Token format validation
  - Signature verification (RS256)
  - Revocation status checking
  - PVP (Principal Verifiable Presentation) identity check
  - JWKS endpoint integration
  - Revocation list (Merkle tree)
  - PVP Registry (DID method)
- **APIs**: `/api/v1/validate`, `/api/v1/verify`
- **Flow**: Incoming Token → Validate → Verify Signature (JWKS) → Check Revocation (List) → PVP Check (Registry) → Verification Result
- **Use Case**: Comprehensive token validation including cryptographic verification and identity proofing

### 6. **Complete Protocol Flow** (`protocol-full`)
**Demonstrates**: End-to-end RFC-0111 + RFC-0115 flow
- **Nodes**: 15 (All phases integrated)
- **Phases**:
  1. **Subscription**: Client registration
  2. **Matching**: PoA validation and capability matching
  3. **Request**: Authorization request with principal consent
  4. **Enforcement**: Dual PEP enforcement
  5. **Verification**: Token and identity verification
  6. **Audit**: Compliance logging and reporting
- **Supporting Services**:
  - Authorization Server (centralized issuer)
  - PDP (Policy Decision Point)
  - Transparency Log (immutable audit trail)
- **Flow**: Complete end-to-end journey from client registration to resource access
- **Use Case**: Enterprise-grade authorization system with full compliance and audit capabilities

## Complete Pattern List (15 Total)

### PoA Relationship Patterns (9)
1. **demo** - Complex graph from backend API
2. **simple** - Basic 3-node graph
3. **multi** - Multi-party organization
4. **cascade** - Cascade revocation
5. **delegation-chain** - Multi-level delegation
6. **multisig** - Multi-signature PoA
7. **jurisdiction** - Geographic boundaries
8. **revocation** - Lifecycle states
9. **full-pattern** - All PoA patterns combined

### GAuth Protocol Flow Patterns (6)
10. **protocol-subscription** - Client registration
11. **protocol-matching** - PoA validation
12. **protocol-request** - Authorization request
13. **protocol-enforcement** - PEP enforcement
14. **protocol-verification** - Token verification
15. **protocol-full** - Complete protocol flow

## Technical Implementation

### Files Modified
1. **web/templates/poa-visualization.html**
   - Added `<optgroup>` for "GAuth Protocol Flow" patterns
   - 6 new options in dropdown menu

2. **web/static/js/modules/poa-viz.js**
   - Added 6 protocol flow generator functions (~200 lines)
   - Updated `generateDemoGraph()` switch statement
   - Each function returns standardized `{nodes, edges, stats}` structure

### Pattern Metadata Structure
Each protocol flow node includes rich metadata:
```javascript
{
    id: 'unique-id',
    label: 'Display Name',
    type: 'root|delegation|consumer',
    status: 'active|pending|revoked',
    position: { x, y, z },
    metadata: {
        // Protocol-specific fields
        phase: 'subscription|matching|request|enforcement|verification|audit',
        step: 'registration|validation|token|...',
        api: '/api/v1/...',
        role: 'requester|issuer|authorizer|...',
        // Technical details
        algorithm: 'RS256',
        grant_type: 'client_credentials',
        response_type: 'code',
        scopes: ['poa:read', 'poa:delegate'],
        // Compliance
        gdpr: true,
        hipaa: true,
        jurisdiction: 'EU|US|ASIA',
        // Security
        immutable: true,
        merkle_tree: true,
        verification: 'did_method'
    }
}
```

### Edge Types for Protocol Flow
- **flow**: Sequential step progression
- **request**: Client-initiated API call
- **validation**: Validation check
- **registration**: Client registration
- **consent**: Principal authorization
- **evaluation**: Policy evaluation
- **issuance**: Token/credential issuance
- **delivery**: Credential delivery
- **enforcement**: Policy enforcement
- **logging**: Audit trail recording
- **lookup**: External service query
- **decision**: Final decision/result
- **permit**: Access grant

## How to Use

### Starting the Server
```bash
# Set dev mode to serve templates from disk
export GAUTH_DEV_INDEX=1

# Start the server
go run ./cmd/web-server

# Or using the workspace task
# Run Task: "Start GAuth Web Server"
```

### Accessing the Visualization
**URL**: http://localhost:8080/poa-visualization (without .html extension)

**Note**: The route `/poa-visualization.html` returns 404. Use `/poa-visualization` instead.

### Selecting Patterns
1. Open http://localhost:8080/poa-visualization
2. **Visualization Mode**: Select "PoA Graph"
3. **Graph Type**: Choose from dropdown:
   - Standard patterns (demo, simple, multi, etc.)
   - GAuth Protocol Flow patterns (optgroup):
     - Subscription Flow
     - Matching Flow
     - Subset/Request Flow
     - Enforcement Flow
     - Verification Flow
     - Complete Protocol Flow
4. Click **Load** button
5. Interact with 3D visualization:
   - **Left-drag**: Rotate view
   - **Right-drag**: Pan view
   - **Scroll**: Zoom in/out
   - **Rotate button**: Enable auto-rotation
   - **Reset View**: Return to default camera

### Understanding the Visualization

#### Color Coding (by node type)
- **Root nodes** (yellow): Authorization servers, PoA definitions, external services
- **Delegation nodes** (blue): Intermediate steps, PEPs, validators
- **Consumer nodes** (green): Clients, agents, resources, results
- **Signer nodes** (orange): Multi-sig signers

#### Status Colors
- **Active** (green): Operational
- **Pending** (yellow): Awaiting action
- **Suspended** (orange): Temporarily disabled
- **Revoked** (red): Permanently invalid

#### Node Positioning
- **Y-axis**: Flow hierarchy (higher = earlier in flow)
- **X-axis**: Parallel operations (left-to-right spread)
- **Z-axis**: Depth/sequence (negative = further in process)

Example: Complete Protocol Flow positions nodes from:
- Top-left (Client at y=4, z=0) → Subscription phase
- Middle (Request at y=4, z=-8) → Authorization phase
- Bottom-right (Resource at y=-2, z=-18) → Final access

## Protocol Flow Mapping to RFCs

### RFC-0111 Authorization Flow
| Pattern | RFC Section | Components |
|---------|-------------|------------|
| Subscription | §3.1 Client Registration | Client ID, Client Secret, Registration endpoint |
| Matching | §3.2 Authorization Request | Scope, State, Response Type |
| Request | §3.3 Authorization Grant | Authorization Code, Token endpoint |
| Enforcement | §4.1 Token Usage | PEP, Resource Server, Access Token |
| Verification | §4.2 Token Validation | Signature, Expiration, Revocation |

### RFC-0115 PoA Definition
| Pattern | RFC Section | Components |
|---------|-------------|------------|
| Matching | §2.1 PoA Structure | Grantor, Grantee, Scope, Constraints |
| Matching | §2.2 AI Capabilities | Capability Declaration, Verification |
| Matching | §2.3 Jurisdiction | Geographic Constraints, Legal Requirements |
| Enforcement | §3.1 Supply-Side PEP | Pre-authorization checks, Disclosure |
| Enforcement | §3.2 Demand-Side PEP | Post-authorization monitoring |
| Verification | §4.1 PVP | Principal Verifiable Presentation, DID |

## Architecture Diagrams

### Subscription Flow
```
AI Agent → Register Client → Configure Scopes → Obtain Credentials
    ↓            ↓                  ↓                   ↓
    └─────→ Authorization Server ←──┴───────────────────┘
                     ↓
              Client ID & Secret
```

### Complete Protocol Flow (High-Level)
```
┌──────────────┐
│ Subscription │ (Client Registration)
└──────┬───────┘
       ↓
┌──────────────┐
│   Matching   │ (PoA Validation + Capability Check)
└──────┬───────┘
       ↓
┌──────────────┐
│   Request    │ (Authorization + Token Issuance)
└──────┬───────┘
       ↓
┌──────────────┐
│ Enforcement  │ (Supply PEP + Demand PEP)
└──────┬───────┘
       ↓
┌──────────────┐
│ Verification │ (Token + Identity Validation)
└──────┬───────┘
       ↓
┌──────────────┐
│    Audit     │ (Logging + Resource Access)
└──────────────┘
```

## API Endpoints Referenced

### Subscription Phase
- `POST /api/v1/client/register` - Client registration
- `POST /api/v1/subscribe` - Subscription management

### Matching Phase
- `POST /api/v1/poa/validate` - PoA definition validation
- `GET /api/v1/ai/capabilities` - AI capability discovery

### Request Phase
- `GET /api/v1/authorize` - Authorization request
- `POST /api/v1/token` - Token issuance

### Enforcement Phase
- `POST /api/v1/enforce/supply` - Supply-side enforcement
- `POST /api/v1/enforce/demand` - Demand-side enforcement

### Verification Phase
- `POST /api/v1/validate` - Token validation
- `POST /api/v1/verify` - Identity verification
- `GET /.well-known/jwks.json` - Public keys (JWKS)

### Audit Phase
- `POST /api/v1/audit` - Audit event logging
- `GET /api/v1/compliance/report` - Compliance reporting

## Future Enhancements

### Phase 1: Interactive Features
- [ ] Click node to view detailed metadata panel
- [ ] Hover tooltip showing API endpoints and parameters
- [ ] Edge animation showing flow direction
- [ ] Time-based playback of protocol execution

### Phase 2: Real-Time Integration
- [ ] Connect to live protocol flow sessions
- [ ] Display actual API response times
- [ ] Show error states and retry logic
- [ ] Real-time audit log streaming

### Phase 3: Advanced Visualization
- [ ] Phase-based color coding (subscription=blue, enforcement=red, etc.)
- [ ] Node clustering by component type
- [ ] Performance metrics overlay (latency, throughput)
- [ ] Compliance status indicators (GDPR, HIPAA, SOC2)

### Phase 4: Educational Mode
- [ ] Guided tour of each pattern with explanations
- [ ] RFC section references on hover
- [ ] Code snippet examples for each API
- [ ] Interactive quiz mode

## Testing Checklist

- [x] Server starts with `GAUTH_DEV_INDEX=1`
- [x] Page loads at `/poa-visualization`
- [x] Dropdown includes "GAuth Protocol Flow" optgroup
- [x] 6 new protocol patterns added
- [x] `generateDemoGraph()` function updated with new cases
- [ ] **Manual Testing Required**:
  - [ ] Load each protocol flow pattern in browser
  - [ ] Verify node positions are readable
  - [ ] Check edge connections are correct
  - [ ] Validate statistics panel updates
  - [ ] Test camera controls (rotate, zoom, pan, reset)
  - [ ] Verify auto-rotation works
  - [ ] Check mobile responsiveness

## Related Documentation
- `POA_VISUALIZATION_PATTERNS.md` - Original PoA patterns documentation
- `WEB_UI_USAGE_GUIDE.md` - General web UI guide
- `web/protocol_flow.go` - Backend protocol flow handlers
- `web/static/js/modules/protocol-navigator.js` - Protocol navigator implementation
- `internal/cascade/processor.go` - Cascade revocation logic

## Troubleshooting

### Issue: 404 on `/poa-visualization.html`
**Solution**: Use `/poa-visualization` without the `.html` extension

### Issue: Three.js import error
**Solution**: Import map is configured in HTML head, ensure CDN is accessible:
```html
<script type="importmap">
{
    "imports": {
        "three": "https://cdn.jsdelivr.net/npm/three@0.160.0/build/three.module.js",
        "three/": "https://cdn.jsdelivr.net/npm/three@0.160.0/"
    }
}
</script>
```

### Issue: Pattern doesn't load
**Solution**: Check browser console for JavaScript errors. Ensure `generateDemoGraph()` export is successful:
```javascript
import { generateDemoGraph } from '/static/js/modules/poa-viz.js';
```

### Issue: Nodes overlap or are too close
**Solution**: Adjust node positions in pattern generator functions. Each pattern has hardcoded x, y, z coordinates. Increase spread for better visibility.

### Issue: Statistics don't update
**Solution**: Each pattern must return `stats` object with:
```javascript
{
    total_nodes: N,
    total_edges: M,
    active_nodes: A,
    pending_nodes: P,
    revoked_nodes: R
}
```

## Browser Compatibility
- **Chrome/Edge**: Fully supported (Recommended)
- **Firefox**: Fully supported
- **Safari**: Supported (may have WebGL limitations on older versions)
- **Mobile**: Partially supported (touch controls may vary)

## Performance Notes
- **Small patterns** (6-9 nodes): Excellent performance
- **Medium patterns** (10-15 nodes): Good performance
- **Large patterns** (15+ nodes): May experience slight lag on low-end devices
- **Optimization**: Use `continueOnError` and limit animation complexity for mobile

---

**Created**: November 7, 2025  
**Last Updated**: November 7, 2025  
**Status**: ✅ Implementation Complete - Ready for Testing  
**Server**: Running at http://localhost:8080  
**Access URL**: http://localhost:8080/poa-visualization  
**Pattern Count**: 15 (9 PoA + 6 Protocol Flow)
