package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mauriciomferz/AgentAuth/pkg/config"
)

// ConfigHandler handles configuration management endpoints
type ConfigHandler struct {
	repo *config.Repository
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(db *pgxpool.Pool) *ConfigHandler {
	return &ConfigHandler{
		repo: config.NewRepository(db),
	}
}

// ConfigVariable represents an environment variable
type ConfigVariable struct {
	Key          string    `json:"key"`
	Value        string    `json:"value"`
	Type         string    `json:"type"` // string, number, boolean, json
	Sensitive    bool      `json:"sensitive"`
	Description  string    `json:"description"`
	LastModified time.Time `json:"lastModified"`
	ModifiedBy   string    `json:"modifiedBy"`
}

// ConfigContent represents YAML/JSON configuration
type ConfigContent struct {
	Content string `json:"content"`
}

// ServiceStatus represents service reload status
type ServiceStatus struct {
	Name          string    `json:"name"`
	Status        string    `json:"status"` // running, stopped, error
	LastReload    time.Time `json:"lastReload"`
	ConfigVersion string    `json:"configVersion"`
	Uptime        string    `json:"uptime"`
}

// ConfigVersion represents a configuration version
type ConfigVersion struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Author      string    `json:"author"`
	Message     string    `json:"message"`
	ChangeCount int       `json:"changeCount"`
	Type        string    `json:"type"` // manual, auto, rollback
}

// TenantOverride represents tenant-specific configuration overrides
type TenantOverride struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenantId"`
	TenantName string            `json:"tenantName"`
	Overrides  map[string]string `json:"overrides"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
	Active     bool              `json:"active"`
}

// FeatureFlag represents a feature flag
type FeatureFlag struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Enabled       bool      `json:"enabled"`
	Type          string    `json:"type"` // boolean, percentage, targeting
	Percentage    int       `json:"percentage,omitempty"`
	TargetTenants []string  `json:"targetTenants,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// VariableRequest represents a request to create/update a variable
type VariableRequest struct {
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value" binding:"required"`
	Type        string `json:"type"`
	Sensitive   bool   `json:"sensitive"`
	Description string `json:"description"`
}

// ReloadRequest represents a service reload request
type ReloadRequest struct {
	Service string `json:"service" binding:"required"`
}

// OverrideRequest represents a tenant override request
type OverrideRequest struct {
	TenantID   string            `json:"tenantId" binding:"required"`
	TenantName string            `json:"tenantName" binding:"required"`
	Overrides  map[string]string `json:"overrides" binding:"required"`
}

// FlagRequest represents a feature flag request
type FlagRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	Enabled       bool     `json:"enabled"`
	Type          string   `json:"type" binding:"required"`
	Percentage    int      `json:"percentage,omitempty"`
	TargetTenants []string `json:"targetTenants,omitempty"`
}

// ToggleRequest represents a toggle request
type ToggleRequest struct {
	Active  *bool `json:"active,omitempty"`
	Enabled *bool `json:"enabled,omitempty"`
}

// ListVariables lists all environment variables
func (h *ConfigHandler) ListVariables(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	vars, err := h.repo.ListVariables(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list variables"})
		return
	}

	variables := make([]ConfigVariable, 0, len(vars))
	for _, v := range vars {
		value := v.VariableValue
		if v.IsSensitive {
			value = "••••••••"
		}

		var description string
		if v.Description != nil {
			description = *v.Description
		}

		var modifiedBy string
		if v.UpdatedBy != nil {
			modifiedBy = *v.UpdatedBy
		}

		variables = append(variables, ConfigVariable{
			Key:          v.VariableKey,
			Value:        value,
			Type:         v.VariableType,
			Sensitive:    v.IsSensitive,
			Description:  description,
			LastModified: v.UpdatedAt,
			ModifiedBy:   modifiedBy,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"variables": variables,
	})
}

// CreateVariable creates a new environment variable
func (h *ConfigHandler) CreateVariable(c *gin.Context) {
	var req VariableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	varType := req.Type
	if varType == "" {
		varType = "string"
	}

	variable := config.ConfigVariableRecord{
		TenantID:      &tenantID,
		VariableKey:   req.Key,
		VariableValue: req.Value,
		VariableType:  varType,
		Scope:         "tenant",
		IsSensitive:   req.Sensitive,
		UpdatedBy:     &userID,
	}

	if req.Description != "" {
		variable.Description = &req.Description
	}

	if err := h.repo.CreateVariable(c.Request.Context(), variable); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create variable"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Variable created successfully",
		"key":     req.Key,
	})
}

// UpdateVariable updates an existing environment variable
func (h *ConfigHandler) UpdateVariable(c *gin.Context) {
	key := c.Param("key")
	var req VariableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	varType := req.Type
	if varType == "" {
		varType = "string"
	}

	variable := config.ConfigVariableRecord{
		VariableValue: req.Value,
		VariableType:  varType,
		IsSensitive:   req.Sensitive,
		UpdatedBy:     &userID,
	}

	if req.Description != "" {
		variable.Description = &req.Description
	}

	if err := h.repo.UpdateVariable(c.Request.Context(), tenantID, key, variable); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update variable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Variable updated successfully",
		"key":     key,
	})
}

// DeleteVariable deletes an environment variable
func (h *ConfigHandler) DeleteVariable(c *gin.Context) {
	key := c.Param("key")
	tenantID := c.GetString("tenant_id")

	if err := h.repo.DeleteVariable(c.Request.Context(), tenantID, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete variable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Variable deleted successfully",
		"key":     key,
	})
}

// GetYAMLConfig retrieves YAML configuration
func (h *ConfigHandler) GetYAMLConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	file, err := h.repo.GetConfigFile(c.Request.Context(), tenantID, "gauth-config", "yaml")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": file.FileContent,
	})
}

// UpdateYAMLConfig updates YAML configuration
func (h *ConfigHandler) UpdateYAMLConfig(c *gin.Context) {
	var req ConfigContent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	file := config.ConfigFileRecord{
		TenantID:    &tenantID,
		FileName:    "gauth-config",
		FileFormat:  "yaml",
		FileContent: req.Content,
		SizeBytes:   func(s string) *int { l := len(s); return &l }(req.Content),
		UpdatedBy:   &userID,
	}

	if err := h.repo.CreateConfigFile(c.Request.Context(), file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "YAML configuration updated successfully",
	})
}

// GetJSONConfig retrieves JSON configuration
func (h *ConfigHandler) GetJSONConfig(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	file, err := h.repo.GetConfigFile(c.Request.Context(), tenantID, "gauth-config", "json")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": file.FileContent,
	})
}

// UpdateJSONConfig updates JSON configuration
func (h *ConfigHandler) UpdateJSONConfig(c *gin.Context) {
	var req ConfigContent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	file := config.ConfigFileRecord{
		TenantID:    &tenantID,
		FileName:    "gauth-config",
		FileFormat:  "json",
		FileContent: req.Content,
		SizeBytes:   func(s string) *int { l := len(s); return &l }(req.Content),
		UpdatedBy:   &userID,
	}

	if err := h.repo.CreateConfigFile(c.Request.Context(), file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "JSON configuration updated successfully",
	})
}

// ListServices lists all services and their reload status
func (h *ConfigHandler) ListServices(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	configs, err := h.repo.ListServiceConfigs(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list services"})
		return
	}

	services := make([]ServiceStatus, 0, len(configs))
	for _, cfg := range configs {
		uptime := "0m"
		if cfg.DeployedAt != nil {
			duration := time.Since(*cfg.DeployedAt)
			hours := int(duration.Hours())
			minutes := int(duration.Minutes()) % 60
			uptime = time.Duration(hours*int(time.Hour) + minutes*int(time.Minute)).String()
		}

		lastReload := time.Now()
		if cfg.LastReloadAt != nil {
			lastReload = *cfg.LastReloadAt
		}

		services = append(services, ServiceStatus{
			Name:          cfg.ServiceName,
			Status:        cfg.Status,
			LastReload:    lastReload,
			ConfigVersion: cfg.ConfigVersion,
			Uptime:        uptime,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"services": services,
	})
}

// ReloadService triggers a hot reload for a specific service or all services
func (h *ConfigHandler) ReloadService(c *gin.Context) {
	var req ReloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")

	if err := h.repo.UpdateServiceReload(c.Request.Context(), tenantID, req.Service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Service reload initiated",
		"service": req.Service,
	})
}

// ListVersions lists configuration version history
func (h *ConfigHandler) ListVersions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	files, err := h.repo.ListConfigVersions(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list versions"})
		return
	}

	versions := make([]ConfigVersion, 0, len(files))
	for _, file := range files {
		author := "system"
		if file.UpdatedBy != nil {
			author = *file.UpdatedBy
		}

		message := ""
		if file.Description != nil {
			message = *file.Description
		}

		versions = append(versions, ConfigVersion{
			ID:          file.ID,
			Timestamp:   file.CreatedAt,
			Author:      author,
			Message:     message,
			ChangeCount: 0,
			Type:        "manual",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
	})
}

// GetVersionDiff retrieves the diff for a specific version
func (h *ConfigHandler) GetVersionDiff(c *gin.Context) {
	versionID := c.Param("id")
	tenantID := c.GetString("tenant_id")

	file, err := h.repo.GetConfigVersion(c.Request.Context(), tenantID, versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	diffContent := "Configuration content for version " + versionID

	c.JSON(http.StatusOK, gin.H{
		"versionId": versionID,
		"diff":      diffContent,
		"content":   file.FileContent,
	})
}

// RollbackVersion rolls back to a specific configuration version
func (h *ConfigHandler) RollbackVersion(c *gin.Context) {
	versionID := c.Param("id")
	tenantID := c.GetString("tenant_id")

	file, err := h.repo.GetConfigVersion(c.Request.Context(), tenantID, versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Version not found"})
		return
	}

	userID := c.GetString("user_id")
	rollbackFile := config.ConfigFileRecord{
		TenantID:    file.TenantID,
		FileName:    file.FileName,
		FileFormat:  file.FileFormat,
		FileContent: file.FileContent,
		SizeBytes:   file.SizeBytes,
		UpdatedBy:   &userID,
	}

	if err := h.repo.CreateConfigFile(c.Request.Context(), rollbackFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rollback version"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Rollback completed successfully",
		"versionId": versionID,
	})
}

// ListTenantOverrides lists all tenant-specific configuration overrides
func (h *ConfigHandler) ListTenantOverrides(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	overridesDB, err := h.repo.ListTenantOverrides(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list overrides"})
		return
	}

	overridesMap := make(map[string]*TenantOverride)
	for _, o := range overridesDB {
		if existing, ok := overridesMap[o.TenantID]; ok {
			existing.Overrides[o.ConfigKey] = o.OverrideValue
		} else {
			overridesMap[o.TenantID] = &TenantOverride{
				ID:         o.ID,
				TenantID:   o.TenantID,
				TenantName: o.TenantID,
				Overrides: map[string]string{
					o.ConfigKey: o.OverrideValue,
				},
				CreatedAt: o.CreatedAt,
				UpdatedAt: o.UpdatedAt,
				Active:    o.Enabled,
			}
		}
	}

	overrides := make([]TenantOverride, 0, len(overridesMap))
	for _, override := range overridesMap {
		overrides = append(overrides, *override)
	}

	c.JSON(http.StatusOK, gin.H{
		"overrides": overrides,
	})
}

// CreateTenantOverride creates a new tenant-specific override
func (h *ConfigHandler) CreateTenantOverride(c *gin.Context) {
	var req OverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	for key, value := range req.Overrides {
		override := config.TenantOverrideRecord{
			TenantID:      req.TenantID,
			ConfigKey:     key,
			OverrideValue: value,
			OverrideType:  "string",
			CreatedBy:     &userID,
		}

		if err := h.repo.CreateTenantOverride(c.Request.Context(), override); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create override"})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Tenant override created successfully",
		"tenantId": req.TenantID,
	})
}

// ToggleTenantOverride enables or disables a tenant override
func (h *ConfigHandler) ToggleTenantOverride(c *gin.Context) {
	overrideID := c.Param("id")
	var req ToggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	enabled := false
	if req.Active != nil {
		enabled = *req.Active
	}

	if err := h.repo.ToggleTenantOverride(c.Request.Context(), tenantID, overrideID, enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle override"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Override status updated",
		"overrideId": overrideID,
		StatusActive: enabled,
	})
}

// DeleteTenantOverride deletes a tenant override
func (h *ConfigHandler) DeleteTenantOverride(c *gin.Context) {
	overrideID := c.Param("id")
	tenantID := c.GetString("tenant_id")

	if err := h.repo.DeleteTenantOverride(c.Request.Context(), tenantID, overrideID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete override"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Override deleted successfully",
		"overrideId": overrideID,
	})
}

// ListFeatureFlags lists all feature flags
func (h *ConfigHandler) ListFeatureFlags(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	flagsDB, err := h.repo.ListFeatureFlags(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list flags"})
		return
	}

	flags := make([]FeatureFlag, 0, len(flagsDB))
	for _, f := range flagsDB {
		description := ""
		if f.Description != nil {
			description = *f.Description
		}

		flagType := "boolean"
		if f.RolloutPercentage > 0 {
			flagType = "percentage"
		} else if f.TargetingRules != nil {
			flagType = "targeting"
		}

		var targetTenants []string
		if f.TargetingRules != nil {
			targetTenants = []string{}
		}

		flags = append(flags, FeatureFlag{
			ID:            f.ID,
			Name:          f.FlagKey,
			Description:   description,
			Enabled:       f.Enabled,
			Type:          flagType,
			Percentage:    f.RolloutPercentage,
			TargetTenants: targetTenants,
			CreatedAt:     f.CreatedAt,
			UpdatedAt:     f.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"flags": flags,
	})
}

// CreateFeatureFlag creates a new feature flag
func (h *ConfigHandler) CreateFeatureFlag(c *gin.Context) {
	var req FlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Type != "boolean" && req.Type != "percentage" && req.Type != "targeting" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid flag type"})
		return
	}

	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	flag := config.FeatureFlagRecord{
		TenantID:          &tenantID,
		FlagKey:           req.Name,
		FlagName:          req.Name,
		Enabled:           req.Enabled,
		RolloutPercentage: req.Percentage,
		UpdatedBy:         &userID,
	}

	if req.Description != "" {
		flag.Description = &req.Description
	}

	if err := h.repo.CreateFeatureFlag(c.Request.Context(), flag); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create flag"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Feature flag created successfully",
		"name":    req.Name,
	})
}

// ToggleFeatureFlag enables or disables a feature flag
func (h *ConfigHandler) ToggleFeatureFlag(c *gin.Context) {
	flagID := c.Param("id")
	var req ToggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	enabled := false
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if err := h.repo.ToggleFeatureFlag(c.Request.Context(), tenantID, flagID, enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle flag"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Flag status updated",
		"flagId":  flagID,
		"enabled": enabled,
	})
}

// DeleteFeatureFlag deletes a feature flag
func (h *ConfigHandler) DeleteFeatureFlag(c *gin.Context) {
	flagID := c.Param("id")
	tenantID := c.GetString("tenant_id")

	if err := h.repo.DeleteFeatureFlag(c.Request.Context(), tenantID, flagID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete flag"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Flag deleted successfully",
		"flagId":  flagID,
	})
}

// RegisterRoutes registers all configuration management routes
func (h *ConfigHandler) RegisterRoutes(router *gin.RouterGroup) {
	config := router.Group("/config")
	{
		// Environment variables
		config.GET("/variables", h.ListVariables)
		config.POST("/variables", h.CreateVariable)
		config.PUT("/variables/:key", h.UpdateVariable)
		config.DELETE("/variables/:key", h.DeleteVariable)

		// YAML/JSON configuration
		config.GET("/yaml", h.GetYAMLConfig)
		config.PUT("/yaml", h.UpdateYAMLConfig)
		config.GET("/json", h.GetJSONConfig)
		config.PUT("/json", h.UpdateJSONConfig)

		// Service management
		config.GET("/services", h.ListServices)
		config.POST("/reload", h.ReloadService)

		// Version history
		config.GET("/versions", h.ListVersions)
		config.GET("/versions/:id/diff", h.GetVersionDiff)
		config.POST("/versions/:id/rollback", h.RollbackVersion)

		// Tenant overrides
		config.GET("/tenant-overrides", h.ListTenantOverrides)
		config.POST("/tenant-overrides", h.CreateTenantOverride)
		config.POST("/tenant-overrides/:id/toggle", h.ToggleTenantOverride)
		config.DELETE("/tenant-overrides/:id", h.DeleteTenantOverride)

		// Feature flags
		config.GET("/feature-flags", h.ListFeatureFlags)
		config.POST("/feature-flags", h.CreateFeatureFlag)
		config.POST("/feature-flags/:id/toggle", h.ToggleFeatureFlag)
		config.DELETE("/feature-flags/:id", h.DeleteFeatureFlag)
	}
}
