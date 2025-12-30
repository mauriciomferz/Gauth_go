# MCP Integration - Quick Start Guide

## Overview

The MCP (Model Context Protocol) integration in AgentAuth allows you to register and manage MCP servers, browse their resources, and invoke tools through both REST API and web UI interfaces.

**Protocol:** JSON-RPC 2.0  
**Transports:** stdio (implemented), WebSocket (planned), HTTP-SSE (planned)  
**Feature Gate:** `GAUTH_MCP_ENABLED=1`

---

## Prerequisites

### Required Software
- **Go 1.21+** - For backend server
- **Node.js 18+** - For frontend and MCP servers
- **npm or yarn** - Package manager

### Optional (for testing with real MCP servers)
```bash
# Install official MCP servers from Anthropic
npm install -g @modelcontextprotocol/server-filesystem
npm install -g @modelcontextprotocol/server-calculator
```

---

## Installation & Setup

### 1. Start the Backend Server

```bash
cd /path/to/Gauth_go

# Enable MCP and RFC-0111 features
export GAUTH_MCP_ENABLED=1
export GAUTH_AAP-001_ENABLED=1

# Start the server
go run ./cmd/web-server
```

**Expected output:**
```
[MCP] Connection manager initialized (GAUTH_MCP_ENABLED=1)
[MCP] Endpoints registered:
[MCP]   POST   /api/v1/beta/mcp/servers (Register MCP server)
[MCP]   GET    /api/v1/beta/mcp/servers (List MCP servers)
[MCP]   GET    /api/v1/beta/mcp/servers/:id/resources (List resources)
[MCP]   POST   /api/v1/beta/mcp/servers/:id/resources/read (Read resource)
[MCP]   POST   /api/v1/beta/mcp/servers/:id/tools/call (Call tool)
[MCP]   GET    /api/v1/beta/mcp/servers/:id/tools (List tools)
[MCP]   DELETE /api/v1/beta/mcp/servers/:id (Disconnect server)
[startup] BetaServer starting on http://localhost:8080
```

### 2. Start the Frontend Server

```bash
cd /path/to/Gauth_go/web/ui-react

# Start Vite development server
npm run dev
```

**Expected output:**
```
VITE v5.4.21  ready in 165 ms
➜  Local:   http://localhost:3001/
```

### 3. Access the MCP Interface

Open your browser and navigate to:
```
http://localhost:3001/mcp
```

You should see the MCP management dashboard with:
- Server registration form
- Statistics (Total Servers, Connected, Available Tools)
- Server list panel
- Server details view

---

## Quick Tutorial: Your First MCP Server

### Option A: Using the Web UI

#### Step 1: Register a Server

1. Click the **"Register Server"** button
2. Fill in the form:
   - **Server ID:** `filesystem` (unique identifier)
   - **Server Name:** `Filesystem MCP Server`
   - **Description:** `Provides file system access`
   - **Transport Type:** Select `stdio`
   - **Command:** `npx`
   - **Arguments:** `-y @modelcontextprotocol/server-filesystem /tmp`
   - **Require Authentication:** Leave unchecked for testing

3. Click **"Register Server"**

#### Step 2: View Server Status

- The server card should appear in the server list
- Status indicator shows "connected" (green) or "disconnected" (gray)
- Click the server card to view details

#### Step 3: Browse Resources

1. Select your registered server from the list
2. The resources panel automatically loads
3. Click **"Read"** on any resource to view its content
4. Resource content displays in formatted panel

#### Step 4: Execute Tools

1. With a server selected, scroll to the **Tools** section
2. Select a tool from the dropdown
3. Enter JSON arguments in the text area:
   ```json
   {
     "param1": "value1",
     "param2": "value2"
   }
   ```
4. Click **"Execute Tool"**
5. View the tool result below

#### Step 5: Disconnect Server

- Click the red **trash icon** on the server card
- Server is unregistered and removed from the list

---

### Option B: Using the REST API

#### 1. Register a Server

```bash
curl -X POST http://localhost:8080/api/v1/beta/mcp/servers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "filesystem",
    "name": "Filesystem MCP Server",
    "description": "Provides file system access",
    "transport_type": "stdio",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
    "require_auth": false
  }'
```

**Response:**
```json
{
  "success": true,
  "server_id": "filesystem",
  "message": "MCP server registered successfully"
}
```

#### 2. List Servers

```bash
curl http://localhost:8080/api/v1/beta/mcp/servers | jq '.'
```

**Response:**
```json
{
  "count": 1,
  "servers": [
    {
      "id": "filesystem",
      "name": "Filesystem MCP Server",
      "description": "Provides file system access",
      "status": "connected",
      "transport_type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
      "url": ""
    }
  ]
}
```

#### 3. List Resources

```bash
curl http://localhost:8080/api/v1/beta/mcp/servers/filesystem/resources | jq '.'
```

**Response:**
```json
{
  "resources": [
    {
      "uri": "file:///tmp/example.txt",
      "name": "example.txt",
      "description": "Text file",
      "mimeType": "text/plain"
    }
  ]
}
```

#### 4. Read a Resource

```bash
curl -X POST http://localhost:8080/api/v1/beta/mcp/servers/filesystem/resources/read \
  -H "Content-Type: application/json" \
  -d '{"uri": "file:///tmp/example.txt"}' | jq '.'
```

**Response:**
```json
{
  "contents": [
    {
      "uri": "file:///tmp/example.txt",
      "mimeType": "text/plain",
      "text": "Hello from MCP!"
    }
  ]
}
```

#### 5. List Tools

```bash
curl http://localhost:8080/api/v1/beta/mcp/servers/filesystem/tools | jq '.'
```

**Response:**
```json
{
  "tools": [
    {
      "name": "read_file",
      "description": "Read contents of a file",
      "inputSchema": {
        "type": "object",
        "properties": {
          "path": {
            "type": "string",
            "description": "File path to read"
          }
        },
        "required": ["path"]
      }
    }
  ]
}
```

#### 6. Call a Tool

```bash
curl -X POST http://localhost:8080/api/v1/beta/mcp/servers/filesystem/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "read_file",
    "arguments": {
      "path": "/tmp/example.txt"
    }
  }' | jq '.'
```

**Response:**
```json
{
  "content": [
    {
      "type": "text",
      "text": "Hello from MCP!"
    }
  ],
  "isError": false
}
```

#### 7. Disconnect Server

```bash
curl -X DELETE http://localhost:8080/api/v1/beta/mcp/servers/filesystem | jq '.'
```

**Response:**
```json
{
  "success": true,
  "message": "MCP server disconnected successfully"
}
```

---

## Example MCP Servers

### 1. Filesystem Server

**Purpose:** Provides read/write access to specified directories

```bash
# Install
npm install -g @modelcontextprotocol/server-filesystem

# Registration via API
curl -X POST http://localhost:8080/api/v1/beta/mcp/servers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "filesystem",
    "name": "Filesystem Server",
    "transport_type": "stdio",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp", "/Users/yourname/Documents"]
  }'
```

**Available Tools:**
- `read_file` - Read file contents
- `write_file` - Write to file
- `list_directory` - List directory contents
- `create_directory` - Create new directory
- `delete_file` - Remove file

### 2. Calculator Server

**Purpose:** Performs mathematical calculations

```bash
# Install
npm install -g @modelcontextprotocol/server-calculator

# Registration via API
curl -X POST http://localhost:8080/api/v1/beta/mcp/servers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "calculator",
    "name": "Calculator Server",
    "transport_type": "stdio",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-calculator"]
  }'
```

**Available Tools:**
- `add` - Addition
- `subtract` - Subtraction
- `multiply` - Multiplication
- `divide` - Division

### 3. Custom Server

You can create your own MCP server following the protocol specification:

```javascript
// my-mcp-server.js
const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');

const server = new Server({
  name: 'my-custom-server',
  version: '1.0.0'
}, {
  capabilities: {
    resources: {},
    tools: {}
  }
});

// Define tools
server.setRequestHandler('tools/list', async () => ({
  tools: [{
    name: 'greet',
    description: 'Greet someone',
    inputSchema: {
      type: 'object',
      properties: {
        name: { type: 'string' }
      }
    }
  }]
}));

server.setRequestHandler('tools/call', async (request) => {
  if (request.params.name === 'greet') {
    return {
      content: [{
        type: 'text',
        text: `Hello, ${request.params.arguments.name}!`
      }]
    };
  }
});

const transport = new StdioServerTransport();
server.connect(transport);
```

**Register:**
```bash
curl -X POST http://localhost:8080/api/v1/beta/mcp/servers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "custom",
    "name": "My Custom Server",
    "transport_type": "stdio",
    "command": "node",
    "args": ["/path/to/my-mcp-server.js"]
  }'
```

---

## Troubleshooting

### Server Won't Connect

**Symptom:** Server status shows "disconnected"

**Solutions:**
1. Check the command is correct and executable:
   ```bash
   which npx
   npx -y @modelcontextprotocol/server-filesystem --help
   ```

2. Verify the MCP server package is installed:
   ```bash
   npm list -g @modelcontextprotocol/server-filesystem
   ```

3. Check server logs (future enhancement - not yet implemented)

### Resources Not Loading

**Symptom:** Resources panel shows empty or error

**Solutions:**
1. Ensure server is connected (status should be "connected")
2. Verify the MCP server actually provides resources
3. Check the resource URI format is correct

### Tool Execution Fails

**Symptom:** Tool call returns error

**Solutions:**
1. Validate JSON arguments against tool schema:
   ```bash
   # Check tool schema first
   curl http://localhost:8080/api/v1/beta/mcp/servers/YOUR_SERVER_ID/tools | jq '.tools[] | select(.name=="TOOL_NAME") | .inputSchema'
   ```

2. Ensure all required parameters are provided
3. Check parameter types match the schema

### Port Already in Use

**Symptom:** Backend won't start - "address already in use"

**Solutions:**
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or use this one-liner
lsof -ti:8080 | xargs kill -9
```

### Frontend Not Loading

**Symptom:** Browser shows connection refused at http://localhost:3001

**Solutions:**
1. Check if Vite dev server is running:
   ```bash
   lsof -i :3001
   ```

2. Restart the frontend:
   ```bash
   cd web/ui-react
   npm run dev
   ```

3. Clear browser cache and reload

---

## API Reference

### Base URL
```
http://localhost:8080/api/v1/beta/mcp
```

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/servers` | Register MCP server |
| GET | `/servers` | List all servers |
| GET | `/servers/:id/resources` | List server resources |
| POST | `/servers/:id/resources/read` | Read resource content |
| GET | `/servers/:id/tools` | List server tools |
| POST | `/servers/:id/tools/call` | Invoke tool |
| DELETE | `/servers/:id` | Disconnect server |

### Request/Response Examples

See "Option B: Using the REST API" section above for detailed curl examples.

---

## Security Considerations

### Current Implementation
- MCP servers run as subprocess with stdio transport
- No authentication required by default (can be enabled per server)
- Servers run with same permissions as AgentAuth process

### Best Practices
1. **Limit Directory Access:** When using filesystem server, specify only necessary directories
2. **Enable Authentication:** Set `require_auth: true` for production servers
3. **Validate Input:** Always validate tool arguments before execution
4. **Audit Logging:** MCP operations are logged for compliance (audit_logger.go)
5. **Network Isolation:** Use stdio transport for local servers only

### Future Enhancements
- Token-based authentication for MCP operations
- Resource access control lists (ACLs)
- Rate limiting on tool invocations
- Sandboxed server execution
- WebSocket with TLS support

---

## Performance Tips

1. **Connection Pooling:** Reuse MCP server connections (managed automatically)
2. **Resource Caching:** Cache frequently accessed resources (future enhancement)
3. **Batch Operations:** Group multiple tool calls when possible
4. **Timeout Configuration:** Adjust timeouts for long-running tools (future enhancement)

---

## Integration Examples

### Using MCP in Your Application

```javascript
// Example: Integrate MCP filesystem server with your app
async function readUserDocument(userId, filename) {
  // 1. Register filesystem server if not already registered
  const serverConfig = {
    id: `user-${userId}-fs`,
    name: `User ${userId} Filesystem`,
    transport_type: 'stdio',
    command: 'npx',
    args: ['-y', '@modelcontextprotocol/server-filesystem', `/data/users/${userId}`]
  };
  
  await fetch('http://localhost:8080/api/v1/beta/mcp/servers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(serverConfig)
  });
  
  // 2. Read the document
  const response = await fetch(
    `http://localhost:8080/api/v1/beta/mcp/servers/user-${userId}-fs/resources/read`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ uri: `file:///data/users/${userId}/${filename}` })
    }
  );
  
  const data = await response.json();
  return data.contents[0].text;
}
```

---

## Additional Resources

- **MCP Protocol Specification:** https://spec.modelcontextprotocol.io/
- **Official MCP SDK:** https://github.com/modelcontextprotocol/typescript-sdk
- **AgentAuth RFC-0111 Documentation:** `/docs/RFC_0111_IMPLEMENTATION.md`
- **Phase 2B Completion Report:** `/PHASE_2B_MCP_COMPLETION_REPORT.md`

---

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review the completion report for technical details
3. Check server logs for error messages
4. Submit issues to the project repository

---

**Last Updated:** November 16, 2025  
**Version:** 1.0.0  
**Status:** Production Ready (stdio transport)
