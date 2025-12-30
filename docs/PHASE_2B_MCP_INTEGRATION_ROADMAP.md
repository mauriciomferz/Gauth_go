# Phase 2B: Model Context Protocol (MCP) Integration Roadmap

**Date**: November 16, 2025  
**Target Timeline**: Q1 2026 (6 weeks)  
**Status**: Planning Phase  
**Priority**: P1 - Strategic Initiative

---

## Executive Summary

**Phase 2B implements Model Context Protocol (MCP) as the AI-to-system connectivity layer**, enabling AgentAuth-authorized AI agents to access external data sources and tools through standardized bidirectional connections. This brings AgentAuth from 68% to 75% AAP-001 compliance by satisfying the MCP building block requirement.

### Why MCP Integration?

**AAP-001 Requirement**:
> "AgentAuth builds on the following standards as building blocks: MCP or its alternatives, including but not limited to: MCP Implementation on Github"

**Current Gap**: AgentAuth has NO MCP implementation (0% coverage)  
**Impact**: Cannot authorize AI agents to access external resources  
**Strategic Value**: Positions AgentAuth as the authorization layer for AI-to-AI and AI-to-system interactions

### What Is MCP?

**Model Context Protocol** is an open-source standard developed by Anthropic for connecting AI applications to external systems. Think of MCP as **"USB-C for AI"** - a standardized way to plug AI into external resources.

**MCP Enables**:
- 🔌 AI agents connecting to databases, APIs, file systems
- 🛠️ AI tools invoking functions, running commands, processing data
- 🔄 Bidirectional communication between AI and systems
- 🌐 Standardized resource discovery and tool execution

**AgentAuth's Role**: Authorization layer ensuring AI agents only access resources they're authorized for via Extended Tokens with MCP scopes.

---

## Design Foundation

### Comprehensive Design Document ✅

**MCP_INTEGRATION_DESIGN.md** (1,727 lines) provides:
- Complete architecture specification
- Component designs with code examples
- Security considerations and threat models
- Testing strategy (unit, integration, E2E)
- Migration path (3 phases)
- AAP-001 compliance mapping

**Key Design Decisions**:
1. **Hybrid Model**: AgentAuth as MCP client host (not MCP server)
2. **Authorization Bridge**: Extended Tokens authorize MCP resource access
3. **Policy Enforcement**: PDP validates all MCP operations
4. **Audit Trail**: Complete logging of AI-to-resource interactions

---

## Implementation Phases

### Phase 2B.1: MCP Client Foundation (Week 1-2)

**Duration**: 2 weeks  
**Objective**: Basic MCP protocol implementation

#### Deliverables

**1. MCP Client Core** (`pkg/mcp/client.go` - 400 lines)
```go
type MCPClient struct {
    transport Transport          // stdio, SSE, WebSocket
    pendingRequests map[string]chan *Response
    handlers MessageHandlers
}

// Core methods
func (c *MCPClient) Initialize() error
func (c *MCPClient) ListResources() ([]Resource, error)
func (c *MCPClient) ReadResource(uri string) (*ResourceContent, error)
func (c *MCPClient) ListTools() ([]Tool, error)
func (c *MCPClient) CallTool(name string, args map[string]any) (*ToolResult, error)
```

**2. Transport Layer** (`pkg/mcp/transport/` - 300 lines)
- stdio transport (pipes to MCP server process)
- SSE transport (server-sent events)
- WebSocket transport (future)

**3. JSON-RPC 2.0 Implementation** (`pkg/mcp/jsonrpc.go` - 200 lines)
- Request/response serialization
- Error handling (parse error, invalid request, etc.)
- Request ID tracking

**4. Unit Tests** (`pkg/mcp/*_test.go` - 500 lines)
- 85%+ test coverage target
- Mock MCP server for testing
- Error case coverage

#### Success Criteria
- [ ] MCP client can connect to MCP server via stdio
- [ ] Can list resources from MCP server
- [ ] Can read resource content
- [ ] Can list available tools
- [ ] Can call tools with arguments
- [ ] 85%+ unit test coverage
- [ ] Zero memory leaks in connection handling

---

### Phase 2B.2: Connection Management (Week 2)

**Duration**: 1 week  
**Objective**: Multi-server connection lifecycle management

#### Deliverables

**1. Connection Manager** (`pkg/mcp/connection_manager.go` - 350 lines)
```go
type ConnectionManager struct {
    servers map[string]*MCPServerConfig
    clients map[string]*MCPClient
    mu sync.RWMutex
}

func (m *ConnectionManager) RegisterServer(config *MCPServerConfig) error
func (m *ConnectionManager) ConnectToServer(serverID string) (*MCPClient, error)
func (m *ConnectionManager) GetClient(serverID string) (*MCPClient, error)
func (m *ConnectionManager) DisconnectServer(serverID string) error
func (m *ConnectionManager) ListServers() []*MCPServerConfig
```

**2. Server Configuration** (`pkg/mcp/config.go` - 150 lines)
```yaml
mcp:
  enabled: true
  servers:
    - id: "filesystem-mcp"
      name: "File System Access"
      command: "npx"
      args: ["@modelcontextprotocol/server-filesystem", "/data"]
      env:
        MCP_LOG_LEVEL: "info"
      
    - id: "database-mcp"
      name: "PostgreSQL Database"
      command: "mcp-server-postgres"
      args: ["postgresql://localhost:5432/mydb"]
```

**3. Health Checking** (`pkg/mcp/health.go` - 100 lines)
- Periodic health checks
- Automatic reconnection on failure
- Connection status monitoring

#### Success Criteria
- [ ] Can register multiple MCP servers
- [ ] Can connect to multiple servers simultaneously
- [ ] Connection pooling/reuse works
- [ ] Automatic reconnection on disconnect
- [ ] Health check detects server failures
- [ ] Configuration loads from YAML

---

### Phase 2B.3: Authorization Bridge (Week 3)

**Duration**: 1 week  
**Objective**: Integrate AgentAuth authorization with MCP operations

#### Deliverables

**1. Authorization Bridge** (`pkg/mcp/authorization_bridge.go` - 400 lines)
```go
type AuthorizationBridge struct {
    pdp *agentauth.PDP
    scopeParser *MCPScopeParser
}

func (a *AuthorizationBridge) ValidateMCPScopes(token *ExtendedToken) error
func (a *AuthorizationBridge) AuthorizeResourceRead(token, serverID, resourceURI) error
func (a *AuthorizationBridge) AuthorizeToolCall(token, serverID, toolName) error
func (a *AuthorizationBridge) CheckResourcePolicy(resource Resource) bool
```

**2. MCP Scope Parser** (`pkg/mcp/scope_parser.go` - 200 lines)
```go
// Scope format: mcp:resource:read:server-id/resource-uri
// Example: mcp:resource:read:db-server/customers/public

type MCPScope struct {
    Operation   string   // "resource:read", "tool:call"
    ServerID    string   // "db-server"
    ResourceURI string   // "customers/public"
    ToolName    string   // "search_customers"
}

func ParseMCPScope(scope string) (*MCPScope, error)
func (s *MCPScope) MatchesResource(serverID, uri string) bool
func (s *MCPScope) MatchesTool(serverID, toolName string) bool
```

**3. Policy Rules** (`policies/mcp_policies.rego` - 150 lines)
```rego
# Allow resource read if token has matching scope
allow_resource_read {
    input.token.scopes[_] == concat(":", [
        "mcp:resource:read",
        input.server_id,
        input.resource_uri
    ])
}

# Deny dangerous tool calls without explicit approval
deny_tool_call {
    tool_is_dangerous(input.tool_name)
    not input.approval_granted
}
```

**4. Integration Tests** (`pkg/mcp/integration_test.go` - 400 lines)
- Agent reads resource with valid token
- Agent denied resource without scope
- Tool call authorized
- Dangerous tool requires approval

#### Success Criteria
- [ ] Extended Tokens support MCP scopes
- [ ] PDP validates MCP resource reads
- [ ] PDP validates MCP tool calls
- [ ] Policy rules enforce scope matching
- [ ] Unauthorized operations logged
- [ ] Integration tests pass

---

### Phase 2B.4: Audit Trail & Compliance (Week 4)

**Duration**: 1 week  
**Objective**: Complete audit logging for compliance

#### Deliverables

**1. MCP Audit Logger** (`pkg/mcp/audit_logger.go` - 300 lines)
```go
type MCPAuditEvent struct {
    Timestamp     time.Time
    EventType     string  // "resource_read", "tool_call"
    TokenID       string
    AgentID       string
    ServerID      string
    ResourceURI   string
    ToolName      string
    ToolArgs      map[string]any
    Result        string  // "allowed", "denied"
    DenialReason  string
}

func (l *MCPAuditLogger) LogResourceRead(event *MCPAuditEvent)
func (l *MCPAuditLogger) LogToolCall(event *MCPAuditEvent)
func (l *MCPAuditLogger) LogAuthorizationFailure(event *MCPAuditEvent)
func (l *MCPAuditLogger) GenerateComplianceReport(start, end time.Time) (*Report, error)
```

**2. Compliance Report Generator** (`pkg/mcp/compliance_report.go` - 250 lines)
- MCP operations by agent
- Resources accessed by AI
- Tool invocations summary
- Authorization denials
- Dangerous operations log

**3. Prometheus Metrics** (`pkg/mcp/metrics.go` - 150 lines)
```go
var (
    mcpResourceReadsTotal = prometheus.NewCounterVec(...)
    mcpToolCallsTotal = prometheus.NewCounterVec(...)
    mcpAuthzDenialsTotal = prometheus.NewCounterVec(...)
    mcpServerHealthGauge = prometheus.NewGaugeVec(...)
)
```

#### Success Criteria
- [ ] All MCP operations logged
- [ ] Compliance reports generated
- [ ] Prometheus metrics exported
- [ ] Audit log searchable
- [ ] Retention policy configured

---

### Phase 2B.5: API & UI Integration (Week 5)

**Duration**: 1 week  
**Objective**: Expose MCP functionality via API and UI

#### Deliverables

**1. MCP HTTP Endpoints** (`web/handlers/beta/mcp_handlers.go` - 400 lines)
```go
// POST /api/v1/beta/mcp/servers
// Register MCP server configuration
func RegisterMCPServerHandler(c *gin.Context)

// GET /api/v1/beta/mcp/servers
// List registered MCP servers
func ListMCPServersHandler(c *gin.Context)

// GET /api/v1/beta/mcp/servers/:id/resources
// List resources from MCP server
func ListMCPResourcesHandler(c *gin.Context)

// POST /api/v1/beta/mcp/servers/:id/resources/:uri/read
// Read resource content (requires valid Extended Token)
func ReadMCPResourceHandler(c *gin.Context)

// POST /api/v1/beta/mcp/servers/:id/tools/:name/call
// Call MCP tool (requires valid Extended Token)
func CallMCPToolHandler(c *gin.Context)
```

**2. React UI - MCP Page** (`web/ui-react/src/pages/MCP.tsx` - 500 lines)
- MCP server registry
- Resource browser
- Tool invocation interface
- Operation history
- Real-time connection status

**3. API Client Updates** (`web/ui-react/src/lib/api.ts` - 150 lines)
```typescript
async registerMCPServer(config: MCPServerConfig): Promise<void>
async listMCPServers(): Promise<MCPServerInfo[]>
async listMCPResources(serverID: string): Promise<Resource[]>
async readMCPResource(serverID: string, uri: string): Promise<ResourceContent>
async callMCPTool(serverID, toolName, args): Promise<ToolResult>
```

**4. OpenAPI Spec Updates** (`docs/openapi/agentauth-api.yaml` - +200 lines)
- MCP endpoint documentation
- Request/response schemas
- Authentication requirements

#### Success Criteria
- [ ] 5 MCP endpoints implemented
- [ ] MCP UI page complete
- [ ] Can register server via UI
- [ ] Can browse resources via UI
- [ ] Can call tools via UI
- [ ] OpenAPI spec updated

---

### Phase 2B.6: Testing & Documentation (Week 6)

**Duration**: 1 week  
**Objective**: Comprehensive testing and documentation

#### Deliverables

**1. E2E Tests** (`e2e/mcp_test.go` - 600 lines)
- Complete agent-to-MCP workflows
- Multi-server scenarios
- Authorization denial scenarios
- Performance tests (latency, throughput)

**2. Example MCP Servers** (`examples/mcp/` - 400 lines)
```bash
examples/mcp/
├── filesystem/         # File system MCP server
├── database/          # PostgreSQL MCP server
├── calculator/        # Simple calculator tool
└── README.md          # Setup instructions
```

**3. Integration Guide** (`docs/MCP_INTEGRATION_GUIDE.md` - 800 lines)
- MCP overview and concepts
- AgentAuth authorization model
- Registering MCP servers
- Creating Extended Tokens with MCP scopes
- Resource access examples
- Tool invocation examples
- Security best practices
- Troubleshooting guide

**4. API Documentation Updates**
- `docs/API_EXAMPLES.md` - Add MCP workflow examples
- `docs/QUICKSTART.md` - Add MCP quick start section
- Update all SDK examples

**5. Migration Guide** (`docs/MCP_MIGRATION_GUIDE.md` - 400 lines)
- Upgrading from no-MCP to MCP-enabled
- Configuration migration
- Policy migration
- Testing checklist

#### Success Criteria
- [ ] 90%+ code coverage on MCP components
- [ ] E2E tests pass
- [ ] 3 example MCP servers working
- [ ] Integration guide complete
- [ ] API docs updated
- [ ] Migration guide complete

---

## Technical Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        AgentAuth System                          │
│                                                              │
│  ┌──────────────┐        ┌──────────────┐                  │
│  │  AI Agent    │──────▶ │   Extended   │                  │
│  │  (Claude,    │        │    Token     │                  │
│  │   GPT-4)     │        │ (MCP Scopes) │                  │
│  └──────────────┘        └──────┬───────┘                  │
│                                  │                           │
│                                  ▼                           │
│                    ┌─────────────────────────┐              │
│                    │  Authorization Bridge   │              │
│                    │  - Scope validation     │              │
│                    │  - PDP integration      │              │
│                    └─────────┬───────────────┘              │
│                              │                               │
│                              ▼                               │
│                    ┌─────────────────────────┐              │
│                    │  Connection Manager     │              │
│                    │  - Server registry      │              │
│                    │  - Client pooling       │              │
│                    └─────────┬───────────────┘              │
│                              │                               │
│               ┌──────────────┼──────────────┐              │
│               ▼              ▼               ▼              │
│         ┌─────────┐    ┌─────────┐    ┌─────────┐         │
│         │ MCP     │    │ MCP     │    │ MCP     │         │
│         │ Client  │    │ Client  │    │ Client  │         │
│         │    1    │    │    2    │    │    3    │         │
│         └────┬────┘    └────┬────┘    └────┬────┘         │
└──────────────┼──────────────┼──────────────┼──────────────┘
               │              │               │
               │              │               │
    ┌──────────┼──────────────┼───────────────┼──────────┐
    │  External MCP Servers (stdio/SSE/WebSocket)        │
    │                                                      │
    │  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
    │  │  Filesystem  │  │  Database    │  │  Custom  │ │
    │  │  MCP Server  │  │  MCP Server  │  │  Tools   │ │
    │  └──────────────┘  └──────────────┘  └──────────┘ │
    └──────────────────────────────────────────────────────┘
```

### Data Flow

**1. Agent Requests Resource Access**
```
AI Agent → AgentAuth → Validate Extended Token → Check MCP Scopes → 
Authorize Operation → Connect to MCP Server → Read Resource → 
Audit Log → Return Content → AI Agent
```

**2. Agent Calls Tool**
```
AI Agent → AgentAuth → Validate Token → Check Tool Permission → 
PDP Evaluates Policy → Connect to MCP Server → Execute Tool → 
Audit Log → Return Result → AI Agent
```

---

## MCP Scope Format

### Scope Syntax

```
mcp:<operation>:<permission>:<server-id>/<resource-path>

Components:
- mcp: Protocol identifier
- operation: "resource" or "tool"
- permission: "read", "write", "call"
- server-id: Registered MCP server ID
- resource-path: Resource URI or tool name
```

### Examples

```
# Read access to specific database table
mcp:resource:read:db-server/customers/public

# Write access to file system
mcp:resource:write:fs-server/data/reports

# Tool execution permission
mcp:tool:call:calc-server/add

# Wildcard for all resources on server
mcp:resource:read:db-server/*

# Wildcard for all tools
mcp:tool:call:*/*
```

---

## Security Considerations

### Threat Model

**T1: Unauthorized Resource Access**
- **Mitigation**: Scope-based authorization, PDP validation
- **Detection**: Audit logging, anomaly detection

**T2: Dangerous Tool Execution**
- **Mitigation**: Tool classification, approval workflow
- **Detection**: Audit alerts, rate limiting

**T3: MCP Server Compromise**
- **Mitigation**: Server health checks, TLS/mTLS
- **Detection**: Connection monitoring, anomaly detection

**T4: Token Theft/Replay**
- **Mitigation**: Short-lived tokens, token binding
- **Detection**: Unusual usage patterns, geolocation checks

**T5: Scope Escalation**
- **Mitigation**: Immutable scopes, scope validation
- **Detection**: Audit log analysis

### Security Best Practices

1. **Minimal Scopes**: Grant only required MCP scopes
2. **Short-Lived Tokens**: MCP tokens expire quickly (15 minutes)
3. **Tool Classification**: Mark dangerous tools, require approval
4. **Audit Everything**: Log all MCP operations
5. **Rate Limiting**: Prevent abuse via rate limits
6. **Encryption**: TLS for all MCP connections (future)

---

## Performance Targets

### Latency
- MCP resource read: <100ms (p95)
- MCP tool call: <500ms (p95)
- Authorization check: <5ms (p99)
- Connection establishment: <2s (p95)

### Throughput
- Concurrent MCP operations: 100+
- MCP servers connected: 10+
- Operations per second: 1,000+ (with proper sizing)

### Resource Usage
- Memory per MCP client: <10MB
- CPU overhead: <5%
- Connection pool size: Configurable (default: 10)

---

## Dependencies

### Go Packages
```go
import (
    "github.com/gorilla/websocket"        // WebSocket transport
    "github.com/creack/pty"               // PTY for stdio transport
    "encoding/json"                       // JSON-RPC serialization
    "github.com/prometheus/client_golang" // Metrics
)
```

### External Tools
- MCP server implementations (npm packages, binaries)
- Test MCP servers for development

### Configuration
```yaml
# Minimum Go version: 1.21+
# New config section: mcp
# New database tables: mcp_servers, mcp_audit_log (optional)
```

---

## Testing Strategy

### Test Coverage Targets
- Unit tests: 85%+
- Integration tests: 70%+
- E2E tests: Critical paths covered

### Test Matrix

| Component | Unit | Integration | E2E |
|-----------|------|-------------|-----|
| MCP Client | ✅ | ✅ | ✅ |
| Connection Manager | ✅ | ✅ | ✅ |
| Authorization Bridge | ✅ | ✅ | ✅ |
| Audit Logger | ✅ | ✅ | ⚠️ |
| API Handlers | ✅ | ✅ | ✅ |
| UI Components | ✅ | ⚠️ | ✅ |

### Performance Tests
- Load test: 1,000 concurrent MCP operations
- Stress test: Connection pool exhaustion
- Endurance test: 24-hour continuous operation

---

## Migration Path

### Phase 1: MCP Optional (Week 1-2)
**Configuration**: MCP disabled by default
```yaml
mcp:
  enabled: false  # Opt-in
```
- MCP code deployed but inactive
- No impact on existing functionality
- Feature flag controlled

### Phase 2: MCP Available (Week 3-5)
**Configuration**: MCP enabled, no servers registered
```yaml
mcp:
  enabled: true
  servers: []  # Empty by default
```
- Agents can request MCP scopes
- System ready for MCP server registration
- Documentation available

### Phase 3: MCP Production (Week 6+)
**Configuration**: MCP enabled with example servers
```yaml
mcp:
  enabled: true
  servers:
    - id: "example-fs"
      command: "mcp-server-filesystem"
      args: ["/data"]
```
- Production MCP servers registered
- AI agents using MCP in production
- Monitoring and alerts configured

---

## Risk Assessment

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| MCP protocol changes | Low | High | Pin to stable MCP version |
| Performance issues | Medium | Medium | Load testing, optimization |
| Security vulnerabilities | Low | High | Security review, audit logging |
| Integration complexity | Medium | Medium | Phased rollout, feature flags |

### Business Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Low adoption | Medium | Low | Documentation, examples |
| MCP standard not adopted | Low | High | AAP-001 still requires it |
| Competition | Low | Low | First-mover advantage |

---

## Success Criteria

### Functional Requirements
- [ ] Connect to MCP servers via stdio, SSE, WebSocket
- [ ] List resources from MCP servers
- [ ] Read resource content with authorization
- [ ] List tools from MCP servers
- [ ] Call tools with authorization
- [ ] Register/manage multiple MCP servers
- [ ] Audit all MCP operations
- [ ] UI for MCP management

### Non-Functional Requirements
- [ ] 85%+ test coverage
- [ ] <100ms resource read latency (p95)
- [ ] <500ms tool call latency (p95)
- [ ] 100+ concurrent operations
- [ ] Zero memory leaks
- [ ] Comprehensive documentation

### Compliance Requirements
- [ ] AAP-001 MCP requirement satisfied
- [ ] All operations auditable
- [ ] Compliance reports generated
- [ ] Security best practices followed

---

## Resource Requirements

### Development Team
- **Backend Engineer**: 1 FTE (6 weeks)
- **Frontend Engineer**: 0.5 FTE (2 weeks)
- **QA Engineer**: 0.5 FTE (2 weeks)
- **Tech Writer**: 0.25 FTE (1 week)

### Infrastructure
- Test MCP servers (development)
- CI/CD pipeline updates
- Staging environment for E2E tests

### External Dependencies
- MCP specification stable (✅ Available)
- Example MCP servers (✅ Available)
- Go libraries (✅ Available)

---

## Timeline & Milestones

### Week 1-2: Foundation
- ✅ MCP client implementation
- ✅ Transport layer (stdio, SSE)
- ✅ JSON-RPC 2.0
- ✅ Unit tests (85%+ coverage)
- **Milestone**: Basic MCP connectivity working

### Week 2: Connection Management
- ✅ Connection manager
- ✅ Multi-server support
- ✅ Health checking
- ✅ Configuration loading
- **Milestone**: Can connect to multiple MCP servers

### Week 3: Authorization
- ✅ Authorization bridge
- ✅ MCP scope parser
- ✅ PDP integration
- ✅ Policy rules
- **Milestone**: Authorization enforced on MCP operations

### Week 4: Compliance
- ✅ Audit logger
- ✅ Compliance reports
- ✅ Prometheus metrics
- **Milestone**: Full audit trail for compliance

### Week 5: Integration
- ✅ HTTP API endpoints
- ✅ React UI page
- ✅ API client updates
- ✅ OpenAPI spec updates
- **Milestone**: MCP accessible via API and UI

### Week 6: Testing & Docs
- ✅ E2E tests
- ✅ Example MCP servers
- ✅ Integration guide
- ✅ Migration guide
- **Milestone**: Production-ready with docs

---

## Post-Launch Enhancements

### Q2 2026: Advanced Features
- WebSocket transport for real-time MCP
- mTLS for MCP server authentication
- Tool approval workflow UI
- Advanced policy rules (context-aware)

### Q3 2026: Performance & Scale
- Connection pooling optimization
- Caching layer for MCP resources
- Distributed MCP client (multi-instance)
- Performance monitoring dashboard

### Q4 2026: Enterprise Features
- MCP server marketplace
- Certified MCP servers registry
- Enterprise support packages
- Compliance certification

---

## Conclusion

**Phase 2B brings AgentAuth from 68% to 75% AAP-001 compliance** by implementing the MCP building block requirement. This strategic initiative positions AgentAuth as the authorization layer for the emerging AI agent ecosystem, enabling secure, auditable, policy-driven access to external resources and tools.

**Key Value Propositions**:
1. ✅ AAP-001 compliance achieved (MCP requirement)
2. 🤖 AI-ready authorization system
3. 🔒 Secure AI-to-system interactions
4. 📊 Complete audit trail for compliance
5. 🚀 First-mover advantage in AI authorization

**Recommended Action**: Proceed with Phase 2B in Q1 2026 as planned.

---

**Document Status**: Planning Complete  
**Last Updated**: November 16, 2025  
**Next Review**: Before Phase 2B kickoff (Q1 2026)
