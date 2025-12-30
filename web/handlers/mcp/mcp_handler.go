package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/gagent"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
	"github.com/mauriciomferz/AgentAuth/pkg/mcp"
)

// MCPHandler handles Model Context Protocol server management endpoints
type MCPHandler struct {
	connManager  *mcp.ConnectionManager
	authBridge   gagent.AuthorizationBridge
	auditLogger  mcp.AuditLogger
	tokenService *gauth.ExtendedTokenService
}

// NewMCPHandler creates a new MCP handler instance with security dependencies
func NewMCPHandler(authBridge gagent.AuthorizationBridge, auditLogger mcp.AuditLogger, tokenService *gauth.ExtendedTokenService) *MCPHandler {
	return &MCPHandler{
		connManager:  mcp.NewConnectionManager(),
		authBridge:   authBridge,
		auditLogger:  auditLogger,
		tokenService: tokenService,
	}
}

// RegisterRoutes registers all MCP endpoints to the given router group
func (h *MCPHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/mcp/health", h.HealthCheck)
	r.GET("/mcp/servers", h.ListServers)
	r.POST("/mcp/servers", h.RegisterServer)
	r.DELETE("/mcp/servers/:id", h.DisconnectServer)
	r.GET("/mcp/servers/:id/status", h.GetServerStatus)
	r.GET("/mcp/servers/:id/resources", h.ListResources)
	r.POST("/mcp/servers/:id/resources/read", h.ReadResource)
	r.GET("/mcp/servers/:id/tools", h.ListTools)
	r.POST("/mcp/servers/:id/tools/call", h.CallTool)
	r.GET("/mcp/servers/:id/prompts", h.ListPrompts)
	r.GET("/mcp/servers/:id/prompts/:name", h.GetPrompt)
}

// authorize extracts and validates the token from the request
func (h *MCPHandler) authorize(c *gin.Context) (*gauth.ExtendedToken, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
		return nil, errors.New("missing or invalid authorization header")
	}

	tokenString := authHeader[7:]
	if h.tokenService == nil {
		return nil, errors.New("token service not initialized")
	}

	result, err := h.tokenService.ValidateExtendedToken(c.Request.Context(), tokenString)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !result.Valid || result.ExtendedToken == nil {
		return nil, errors.New("invalid token")
	}

	return result.ExtendedToken, nil
}

// HealthCheck returns the health status of the MCP handler
func (h *MCPHandler) HealthCheck(c *gin.Context) {
	serverIDs := h.connManager.ListServers()
	status := h.connManager.GetConnectionStatus()

	connected := 0
	disconnected := 0
	for _, stat := range status {
		if stat == "connected" {
			connected++
		} else {
			disconnected++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "healthy",
		"servers": gin.H{
			"total":        len(serverIDs),
			"connected":    connected,
			"disconnected": disconnected,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetServerStatus returns the status of a specific MCP server
func (h *MCPHandler) GetServerStatus(c *gin.Context) {
	serverID := c.Param("id")

	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Server ID is required",
		})
		return
	}

	config, err := h.connManager.GetServerConfig(serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "server_not_found",
			"message": fmt.Sprintf("Server '%s' not found", serverID),
		})
		return
	}

	status := h.connManager.GetConnectionStatus()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"server": gin.H{
			"id":             config.ID,
			"name":           config.Name,
			"description":    config.Description,
			"transport_type": config.TransportType,
			"status":         status[serverID],
		},
	})
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

	// Validate transport type
	if req.TransportType != "stdio" && req.TransportType != "http" && req.TransportType != "https" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_transport_type",
			"message": "Transport type must be 'stdio', 'http', or 'https'",
		})
		return
	}

	// Validate stdio transport has command
	if req.TransportType == "stdio" && req.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "missing_command",
			"message": "Command is required for stdio transport",
		})
		return
	}

	// Validate http/https transport has URL
	if (req.TransportType == "http" || req.TransportType == "https") && req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "missing_url",
			"message": "URL is required for http/https transport",
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

	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Server ID is required",
		})
		return
	}

	if err := h.connManager.UnregisterServer(serverID); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"success": false,
				"error":   "timeout",
				"message": "Server disconnection timed out",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "disconnect_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Server disconnected successfully",
		"server_id": serverID,
	})
}

// ListResources lists all resources available on an MCP server
func (h *MCPHandler) ListResources(c *gin.Context) {
	serverID := c.Param("id")

	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid_request",
			"message": "Server ID is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	client, err := h.connManager.GetClient(ctx, serverID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"success": false,
				"error":   "timeout",
				"message": "Connection to server timed out",
			})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "server_not_found",
			"message": fmt.Sprintf("Server '%s' not found or not connected", serverID),
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

	// Authorization
	token, err := h.authorize(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "unauthorized",
			"message": err.Error(),
		})
		return
	}

	// MCP Authorization Bridge Check
	authorized, err := h.authBridge.AuthorizeResourceRead(c.Request.Context(), token, req.URI)

	// Audit Log
	h.auditLogger.Log(c.Request.Context(), &mcp.AuditLogEntry{ // #nosec G104
		Timestamp:  time.Now(),
		Operation:  "resource_read",
		AgentID:    token.AuthorizationChain.Client.EntityID,
		Target:     req.URI,
		Authorized: authorized,
		Decision: func() string {
			if authorized {
				return "granted"
			} else if err != nil {
				return err.Error()
			} else {
				return "denied"
			}
		}(),
		MCPServerID: serverID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "authorization_error",
			"message": err.Error(),
		})
		return
	}

	if !authorized {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "forbidden",
			"message": "Access denied by policy",
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

	// Authorization
	token, err := h.authorize(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "unauthorized",
			"message": err.Error(),
		})
		return
	}

	// MCP Authorization Bridge Check
	authorized, err := h.authBridge.AuthorizeToolCall(c.Request.Context(), token, req.Name, req.Arguments)

	// Audit Log
	h.auditLogger.Log(c.Request.Context(), &mcp.AuditLogEntry{ // #nosec G104
		Timestamp:  time.Now(),
		Operation:  "tool_call",
		AgentID:    token.AuthorizationChain.Client.EntityID,
		Target:     req.Name,
		Authorized: authorized,
		Decision: func() string {
			if authorized {
				return "granted"
			} else if err != nil {
				return err.Error()
			} else {
				return "denied"
			}
		}(),
		MCPServerID: serverID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "authorization_error",
			"message": err.Error(),
		})
		return
	}

	if !authorized {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "forbidden",
			"message": "Access denied by policy",
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

// ListPrompts lists all prompts available on an MCP server
func (h *MCPHandler) ListPrompts(c *gin.Context) {
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

	prompts, err := client.ListPrompts(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "list_failed",
			"message": err.Error(),
			"prompts": []interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"prompts": prompts,
	})
}

// GetPrompt retrieves a specific prompt from an MCP server
func (h *MCPHandler) GetPrompt(c *gin.Context) {
	serverID := c.Param("id")
	promptName := c.Param("name")

	// Authorization
	token, err := h.authorize(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "unauthorized",
			"message": err.Error(),
		})
		return
	}

	// MCP Authorization Bridge Check
	authorized, authErr := h.authBridge.AuthorizePromptGet(c.Request.Context(), token, promptName)

	// Audit Log
	h.auditLogger.Log(c.Request.Context(), &mcp.AuditLogEntry{ // #nosec G104
		Timestamp:  time.Now(),
		Operation:  "prompt_get",
		AgentID:    token.AuthorizationChain.Client.EntityID,
		Target:     promptName,
		Authorized: authorized,
		Decision: func() string {
			if authorized {
				return "granted"
			} else if authErr != nil {
				return authErr.Error()
			} else {
				return "denied"
			}
		}(),
		MCPServerID: serverID,
	})

	if authErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "authorization_error",
			"message": authErr.Error(),
		})
		return
	}

	if !authorized {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "forbidden",
			"message": "Access denied by policy",
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

	arguments := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			arguments[k] = v[0]
		}
	}

	result, err := client.GetPrompt(ctx, promptName, arguments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "prompt_get_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"content": result,
	})
}
