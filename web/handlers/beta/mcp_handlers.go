package beta

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/mcp"
)

// MCPHandlers provides HTTP endpoints for Model Context Protocol operations
type MCPHandlers struct {
	connectionManager *mcp.ConnectionManager
}

// NewMCPHandlers creates a new MCP handlers instance
func NewMCPHandlers(manager *mcp.ConnectionManager) *MCPHandlers {
	return &MCPHandlers{
		connectionManager: manager,
	}
}

// RegisterServer handles POST /api/v1/beta/mcp/servers
// Registers a new MCP server configuration
func (h *MCPHandlers) RegisterServer(c *gin.Context) {
	var req mcp.ServerConfig

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if err := h.connectionManager.RegisterServer(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "registration_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"server_id": req.ID,
		"message":   "MCP server registered successfully",
	})
}

// ListServers handles GET /api/v1/beta/mcp/servers
// Lists all registered MCP servers with their connection status
func (h *MCPHandlers) ListServers(c *gin.Context) {
	serverIDs := h.connectionManager.ListServers()
	statusMap := h.connectionManager.GetConnectionStatus()

	response := make([]gin.H, 0, len(serverIDs))
	for _, serverID := range serverIDs {
		config, err := h.connectionManager.GetServerConfig(serverID)
		if err != nil {
			continue
		}

		response = append(response, gin.H{
			"id":             config.ID,
			"name":           config.Name,
			"description":    config.Description,
			"transport_type": config.TransportType,
			"command":        config.Command,
			"args":           config.Args,
			"url":            config.URL,
			"status":         statusMap[serverID],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"servers": response,
		"count":   len(response),
	})
}

// ListResources handles GET /api/v1/beta/mcp/servers/:id/resources
// Lists available resources from an MCP server
func (h *MCPHandlers) ListResources(c *gin.Context) {
	serverID := c.Param("id")

	client, err := h.connectionManager.GetClient(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "server_not_found",
			"message": "MCP server not found or not connected",
		})
		return
	}

	resourcesResp, err := client.ListResources(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "list_failed",
			"message": err.Error(),
		})
		return
	}

	response := make([]gin.H, 0, len(resourcesResp.Resources))
	for _, resource := range resourcesResp.Resources {
		response = append(response, gin.H{
			"uri":         resource.URI,
			"name":        resource.Name,
			"description": resource.Description,
			"mime_type":   resource.MimeType,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id": serverID,
		"resources": response,
		"count":     len(response),
	})
}

// ReadResource handles POST /api/v1/beta/mcp/servers/:id/resources/read
// Reads content from an MCP resource
func (h *MCPHandlers) ReadResource(c *gin.Context) {
	serverID := c.Param("id")

	var req struct {
		URI string `json:"uri" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	client, err := h.connectionManager.GetClient(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "server_not_found",
			"message": "MCP server not found or not connected",
		})
		return
	}

	resourceResp, err := client.ReadResource(c.Request.Context(), req.URI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "read_failed",
			"message": err.Error(),
		})
		return
	}

	response := make([]gin.H, 0, len(resourceResp.Contents))
	for _, content := range resourceResp.Contents {
		response = append(response, gin.H{
			"uri":       content.URI,
			"mime_type": content.MimeType,
			"text":      content.Text,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id": serverID,
		"uri":       req.URI,
		"contents":  response,
	})
}

// CallTool handles POST /api/v1/beta/mcp/servers/:id/tools/call
// Invokes an MCP tool with specified arguments
func (h *MCPHandlers) CallTool(c *gin.Context) {
	serverID := c.Param("id")

	var req struct {
		Name      string                 `json:"name" binding:"required"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	client, err := h.connectionManager.GetClient(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "server_not_found",
			"message": "MCP server not found or not connected",
		})
		return
	}

	toolResp, err := client.CallTool(c.Request.Context(), req.Name, req.Arguments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "tool_call_failed",
			"message": err.Error(),
		})
		return
	}

	response := make([]gin.H, 0, len(toolResp.Content))
	for _, content := range toolResp.Content {
		response = append(response, gin.H{
			"type": content.Type,
			"text": content.Text,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id": serverID,
		"tool":      req.Name,
		"is_error":  toolResp.IsError,
		"content":   response,
	})
}

// ListTools handles GET /api/v1/beta/mcp/servers/:id/tools
// Lists available tools from an MCP server
func (h *MCPHandlers) ListTools(c *gin.Context) {
	serverID := c.Param("id")

	client, err := h.connectionManager.GetClient(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "server_not_found",
			"message": "MCP server not found or not connected",
		})
		return
	}

	toolsResp, err := client.ListTools(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "list_failed",
			"message": err.Error(),
		})
		return
	}

	response := make([]gin.H, 0, len(toolsResp.Tools))
	for _, tool := range toolsResp.Tools {
		response = append(response, gin.H{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": tool.InputSchema,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"server_id": serverID,
		"tools":     response,
		"count":     len(response),
	})
}

// DisconnectServer handles DELETE /api/v1/beta/mcp/servers/:id
// Disconnects and unregisters an MCP server
func (h *MCPHandlers) DisconnectServer(c *gin.Context) {
	serverID := c.Param("id")

	if err := h.connectionManager.UnregisterServer(serverID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "disconnect_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"server_id": serverID,
		"message":   "MCP server disconnected successfully",
	})
}
