package mcp

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/mcp"
)

// MCPHandler handles Model Context Protocol server management endpoints
type MCPHandler struct {
	connManager *mcp.ConnectionManager
}

// NewMCPHandler creates a new MCP handler instance
func NewMCPHandler() *MCPHandler {
	return &MCPHandler{
		connManager: mcp.NewConnectionManager(),
	}
}

// RegisterRoutes registers all MCP endpoints to the given router group
func (h *MCPHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/mcp/servers", h.ListServers)
	r.POST("/mcp/servers", h.RegisterServer)
	r.DELETE("/mcp/servers/:id", h.DisconnectServer)
	r.GET("/mcp/servers/:id/resources", h.ListResources)
	r.POST("/mcp/servers/:id/resources/read", h.ReadResource)
	r.GET("/mcp/servers/:id/tools", h.ListTools)
	r.POST("/mcp/servers/:id/tools/call", h.CallTool)
}

// ListServers returns all registered MCP servers
func (h *MCPHandler) ListServers(c *gin.Context) {
	serverIDs := h.connManager.ListServers()
	status := h.connManager.GetConnectionStatus()
	
	servers := make([]map[string]interface{}, 0, len(serverIDs))
	for _, id := range serverIDs {
		config, err := h.connManager.GetServerConfig(id)
		if err != nil {
			continue
		}
		
		servers = append(servers, map[string]interface{}{
			"id":             config.ID,
			"name":           config.Name,
			"description":    config.Description,
			"transport_type": config.TransportType,
			"command":        config.Command,
			"args":           config.Args,
			"url":            config.URL,
			"status":         status[id],
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"servers": servers,
	})
}

// RegisterServer registers a new MCP server
func (h *MCPHandler) RegisterServer(c *gin.Context) {
	var req struct {
		ID            string   `json:"id" binding:"required"`
		Name          string   `json:"name" binding:"required"`
		Description   string   `json:"description"`
		TransportType string   `json:"transport_type" binding:"required"`
		Command       string   `json:"command"`
		Args          []string `json:"args"`
		URL           string   `json:"url"`
		RequireAuth   bool     `json:"require_auth"`
		AllowedScopes []string `json:"allowed_scopes"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}
	
	config := &mcp.ServerConfig{
		ID:            req.ID,
		Name:          req.Name,
		Description:   req.Description,
		TransportType: req.TransportType,
		Command:       req.Command,
		Args:          req.Args,
		URL:           req.URL,
		RequireAuth:   req.RequireAuth,
		AllowedScopes: req.AllowedScopes,
	}
	
	if err := h.connManager.RegisterServer(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "registration_failed",
			"message": err.Error(),
		})
		return
	}
	
	// Try to connect immediately
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	_, err := h.connManager.GetClient(ctx, req.ID)
	status := "connected"
	if err != nil {
		status = "disconnected"
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"server": map[string]interface{}{
			"id":     req.ID,
			"name":   req.Name,
			"status": status,
		},
	})
}

// DisconnectServer disconnects and removes an MCP server
func (h *MCPHandler) DisconnectServer(c *gin.Context) {
	serverID := c.Param("id")
	
	if err := h.connManager.UnregisterServer(serverID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "disconnect_failed",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Server disconnected successfully",
	})
}

// ListResources lists all resources available on an MCP server
func (h *MCPHandler) ListResources(c *gin.Context) {
	serverID := c.Param("id")
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	
	client, err := h.connManager.GetClient(ctx, serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "server_not_found",
			"message": err.Error(),
		})
		return
	}
	
	resources, err := client.ListResources(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "list_failed",
			"message":   err.Error(),
			"resources": []interface{}{},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"resources": resources,
	})
}

// ReadResource reads the content of a specific resource
func (h *MCPHandler) ReadResource(c *gin.Context) {
	serverID := c.Param("id")
	
	var req struct {
		URI string `json:"uri" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	
	client, err := h.connManager.GetClient(ctx, serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "server_not_found",
			"message": err.Error(),
		})
		return
	}
	
	content, err := client.ReadResource(ctx, req.URI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "read_failed",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"contents": []interface{}{content},
	})
}

// ListTools lists all tools available on an MCP server
func (h *MCPHandler) ListTools(c *gin.Context) {
	serverID := c.Param("id")
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	
	client, err := h.connManager.GetClient(ctx, serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "server_not_found",
			"message": err.Error(),
		})
		return
	}
	
	tools, err := client.ListTools(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "list_failed",
			"message": err.Error(),
			"tools":   []interface{}{},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tools":   tools,
	})
}

// CallTool executes a tool on an MCP server
func (h *MCPHandler) CallTool(c *gin.Context) {
	serverID := c.Param("id")
	
	var req struct {
		Name      string                 `json:"name" binding:"required"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	client, err := h.connManager.GetClient(ctx, serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "server_not_found",
			"message": err.Error(),
		})
		return
	}
	
	result, err := client.CallTool(ctx, req.Name, req.Arguments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "tool_call_failed",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"content": result,
	})
}
