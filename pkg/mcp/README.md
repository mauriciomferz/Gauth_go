# MCP (Model Context Protocol) Integration

This package implements the Model Context Protocol (MCP) integration for GAuth 1.0, satisfying RFC-0111 building block requirements.

## Overview

MCP is an open standard developed by Anthropic for connecting AI applications to external systems. This implementation enables GAuth-authorized AI agents to access external resources (databases, files, APIs) and invoke tools (calculators, search engines, etc.) through standardized MCP servers.

## Architecture

```
┌──────────────┐         MCP Protocol        ┌──────────────┐
│              │ <─────────────────────────> │              │
│  MCP Client  │  (Bidirectional Conn.)      │  MCP Server  │
│  (AI Agent)  │                             │  (Resources) │
│              │                             │              │
└──────────────┘                             └──────────────┘
     ↓                                              ↓
GAuth Authorization                          • Databases
Extended Tokens                              • File Systems
PDP Policies                                 • APIs/Tools
```

## Current Status: Phase 1 Complete ✅

**Implemented Components**:
- ✅ **MCP Client SDK** (`client.go`) - JSON-RPC 2.0 protocol client
- ✅ **Protocol Types** (`types.go`) - MCP message structures
- ✅ **Stdio Transport** (`transport_stdio.go`) - Process-based communication
- ✅ **Connection Manager** (`connection_manager.go`) - Multi-server management
- ✅ **Unit Tests** - 45.2% test coverage, all tests passing

**Functionality**:
- ✅ List/read resources from MCP servers
- ✅ List/call tools on MCP servers
- ✅ List/get prompts from MCP servers
- ✅ Manage multiple MCP server connections
- ✅ Stdio transport (for subprocess-based MCP servers)

## Usage

### Registering an MCP Server

```go
manager := mcp.NewConnectionManager()

config := &mcp.ServerConfig{
    ID:            "docs-server",
    Name:          "Documentation Server",
    TransportType: "stdio",
    Command:       "/path/to/mcp-server",
    Args:          []string{"--config", "config.json"},
    RequireAuth:   true,
    AllowedScopes: []string{
        "mcp:resource:read:docs/*",
        "mcp:tool:call:search",
    },
}

if err := manager.RegisterServer(config); err != nil {
    log.Fatalf("Failed to register server: %v", err)
}
```

### Connecting to an MCP Server

```go
ctx := context.Background()

// Get client (creates connection if needed)
client, err := manager.GetClient(ctx, "docs-server")
if err != nil {
    log.Fatalf("Failed to get client: %v", err)
}

// List available resources
resources, err := client.ListResources(ctx)
if err != nil {
    log.Fatalf("Failed to list resources: %v", err)
}

for _, resource := range resources.Resources {
    fmt.Printf("Resource: %s (%s)\n", resource.Name, resource.URI)
}
```

### Reading a Resource

```go
content, err := client.ReadResource(ctx, "file:///docs/manual.pdf")
if err != nil {
    log.Fatalf("Failed to read resource: %v", err)
}

for _, item := range content.Contents {
    fmt.Printf("Content: %s\n", item.Text)
}
```

### Calling a Tool

```go
result, err := client.CallTool(ctx, "calculator", map[string]interface{}{
    "expression": "2 + 2",
})
if err != nil {
    log.Fatalf("Failed to call tool: %v", err)
}

for _, item := range result.Content {
    fmt.Printf("Result: %s\n", item.Text)
}
```

## MCP Protocol Specification

- **Official Specification**: https://modelcontextprotocol.io/
- **GitHub Repository**: https://github.com/modelcontextprotocol/specification
- **Protocol**: JSON-RPC 2.0
- **Transports**: stdio (implemented), WebSocket (Phase 4), HTTP-SSE (Phase 4)

## MCP Primitives

1. **Resources** - Data sources the AI can read (files, database records, API responses)
2. **Tools** - Functions the AI can invoke (calculators, search, API calls)
3. **Prompts** - Templated messages or instructions for AI models
4. **Sampling** - AI model completion requests routed through MCP (future)

## Implementation Phases

### Phase 1: Core MCP Client ✅ COMPLETE
- ✅ MCP Client SDK
- ✅ Protocol Types
- ✅ Stdio Transport
- ✅ Connection Manager
- ✅ Unit Tests (45.2% coverage)

**Status**: Complete (November 12, 2025)

### Phase 2: Authorization Bridge (In Progress)
- ⏳ Authorization Bridge (map GAuth tokens → MCP permissions)
- ⏳ Extended Token with MCP Scopes
- ⏳ PDP Integration for MCP policies
- ⏳ Integration Tests

**Estimated**: 4-5 days

### Phase 3: Agent Integration & Audit (Next)
- 📅 MCP Agent wrapper (`pkg/gagent/mcp_integration.go`)
- 📅 Audit Logger for MCP operations
- 📅 REST API endpoints for MCP
- 📅 E2E Tests

**Estimated**: 5-6 days

### Phase 4: Production Hardening (Future)
- 📅 WebSocket transport
- 📅 HTTP-SSE transport
- 📅 Connection pooling & retry logic
- 📅 Rate limiting
- 📅 Monitoring & metrics

**Estimated**: 4-5 days

## Testing

Run all tests:
```bash
go test -v ./pkg/mcp/...
```

Run with coverage:
```bash
go test -v ./pkg/mcp/... -cover
```

Run specific test:
```bash
go test -v ./pkg/mcp/... -run TestMCPClient_ListResources
```

## RFC-0111 Compliance

**Before MCP Implementation**: 0% (0/100 points)  
**After Phase 1**: 30% (30/100 points) - Core client functional  
**After Phase 2**: 60% (60/100 points) - Authorization integrated  
**After Phase 3**: 85% (85/100 points) - Agent integration complete  
**After Phase 4**: 95% (95/100 points) - Production-ready

**Overall GAuth Compliance Impact**:
- Before MCP: 68% RFC-0111 compliant (with OIDC Phases 1-2)
- After MCP Phase 3: **75% RFC-0111 compliant** (+7% increase)

## Security Considerations

1. **Server Allowlist** - Only registered MCP servers permitted
2. **Scope Enforcement** - PDP validates MCP operations against token scopes
3. **Audit Logging** - All MCP interactions logged for compliance
4. **Transport Security** - Future: mTLS for WebSocket/HTTP transports
5. **Data Classification** - Tag resources with sensitivity levels

## Design Document

For detailed architecture, design decisions, and implementation strategy, see:
`/MCP_INTEGRATION_DESIGN.md` (1,700+ lines)

## Contributing

MCP integration follows the standard GAuth development workflow:
1. Design documented in `MCP_INTEGRATION_DESIGN.md`
2. Implementation in phases (currently Phase 1 complete)
3. Unit tests required (45%+ coverage)
4. Integration tests in Phase 2
5. E2E tests in Phase 3

## References

- RFC-0111 (GAuth 1.0) - Section 1: Scope (Building Blocks)
- Model Context Protocol Specification: https://modelcontextprotocol.io/
- JSON-RPC 2.0 Specification: https://www.jsonrpc.org/specification
- MCP TypeScript SDK: https://github.com/modelcontextprotocol/typescript-sdk

## License

Part of the GAuth 1.0 implementation - same license as parent project.

---

**Document Status**: Phase 1 Complete  
**Last Updated**: November 12, 2025  
**Next Review**: After Phase 2 completion (Authorization Bridge)
