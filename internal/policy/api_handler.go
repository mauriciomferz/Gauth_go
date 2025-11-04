package policy

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// APIHandler provides REST API endpoints for policy version management.
type APIHandler struct {
	versionManager *PolicyVersionManager
}

// NewAPIHandler creates a new API handler for policy versioning.
func NewAPIHandler(versionManager *PolicyVersionManager) *APIHandler {
	return &APIHandler{
		versionManager: versionManager,
	}
}

// RegisterRoutes registers all policy versioning API routes.
func (h *APIHandler) RegisterRoutes(router *gin.Engine) {
	policyAPI := router.Group("/api/v1/beta/policy")
	{
		// Version management endpoints
		policyAPI.GET("/versions", h.listVersions)
		policyAPI.GET("/versions/:version", h.getVersionDetails)
		policyAPI.GET("/versions/active", h.getActiveVersion)
		policyAPI.POST("/versions/:version/activate", h.activateVersion)
		policyAPI.POST("/versions/:version/deprecate", h.deprecateVersion)
		policyAPI.POST("/versions/:version/approve", h.approveVersion)

		// Rollback endpoint
		policyAPI.POST("/rollback", h.rollbackVersion)

		// Comparison and diff endpoints
		policyAPI.GET("/compare", h.compareVersions)
		policyAPI.GET("/diff", h.diffVersions)

		// Export and health endpoints
		policyAPI.GET("/metadata/export", h.exportMetadata)
		policyAPI.GET("/health", h.healthCheck)
	}
}

// listVersions returns all policy versions with metadata.
func (h *APIHandler) listVersions(c *gin.Context) {
	versions := h.versionManager.ListVersions()

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"total_versions": len(versions),
		"versions":       versions,
		"active_version": h.versionManager.GetActiveVersion(),
	})
}

// getVersionDetails returns detailed metadata for a specific version.
func (h *APIHandler) getVersionDetails(c *gin.Context) {
	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid version number",
		})
		return
	}

	metadata, err := h.versionManager.GetVersionMetadata(version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"metadata": metadata,
	})
}

// getActiveVersion returns the currently active policy version.
func (h *APIHandler) getActiveVersion(c *gin.Context) {
	activeVersion := h.versionManager.GetActiveVersion()
	metadata, err := h.versionManager.GetVersionMetadata(activeVersion)

	response := gin.H{
		"success":        true,
		"active_version": activeVersion,
	}

	if err == nil {
		response["metadata"] = metadata
	}

	c.JSON(http.StatusOK, response)
}

// activateVersion activates a specific policy version.
func (h *APIHandler) activateVersion(c *gin.Context) {
	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid version number",
		})
		return
	}

	var req struct {
		Actor string `json:"actor"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Actor = "system"
	}

	ctx := context.Background()
	if err := h.versionManager.ActivateVersion(ctx, version, req.Actor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	metadata, _ := h.versionManager.GetVersionMetadata(version)
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"active_version": version,
		"metadata":       metadata,
		"message":        "Version activated successfully",
	})
}

// rollbackVersion rolls back to a previous policy version.
func (h *APIHandler) rollbackVersion(c *gin.Context) {
	versionStr := c.Query("version")
	if versionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "version parameter required",
		})
		return
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid version number",
		})
		return
	}

	var req struct {
		Actor  string `json:"actor"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Actor = "system"
		req.Reason = "Rollback requested"
	}

	ctx := context.Background()
	if err := h.versionManager.RollbackVersion(ctx, version, req.Actor, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	metadata, _ := h.versionManager.GetVersionMetadata(version)
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"active_version": version,
		"metadata":       metadata,
		"message":        "Rollback successful",
	})
}

// deprecateVersion marks a version as deprecated.
func (h *APIHandler) deprecateVersion(c *gin.Context) {
	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid version number",
		})
		return
	}

	var req struct {
		Actor      string     `json:"actor"`
		Reason     string     `json:"reason"`
		SunsetDate *time.Time `json:"sunset_date,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request body",
		})
		return
	}

	ctx := context.Background()
	if err := h.versionManager.DeprecateVersion(ctx, version, req.Reason, req.SunsetDate, req.Actor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	metadata, _ := h.versionManager.GetVersionMetadata(version)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"version":  version,
		"metadata": metadata,
		"message":  "Version deprecated successfully",
	})
}

// approveVersion records an approval for a version.
func (h *APIHandler) approveVersion(c *gin.Context) {
	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid version number",
		})
		return
	}

	var req struct {
		Approver string `json:"approver"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Approver == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "approver required in request body",
		})
		return
	}

	if err := h.versionManager.ApproveVersion(version, req.Approver); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	metadata, _ := h.versionManager.GetVersionMetadata(version)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"version":  version,
		"approver": req.Approver,
		"metadata": metadata,
		"message":  "Approval recorded successfully",
	})
}

// compareVersions compares two policy versions.
func (h *APIHandler) compareVersions(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "from and to version parameters required",
		})
		return
	}

	fromVersion, err1 := strconv.Atoi(fromStr)
	toVersion, err2 := strconv.Atoi(toStr)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid version numbers",
		})
		return
	}

	diff, err := h.versionManager.CompareVersions(fromVersion, toVersion)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"diff":    diff,
	})
}

// diffVersions provides detailed diff between two versions.
func (h *APIHandler) diffVersions(c *gin.Context) {
	// Alias for compareVersions with additional formatting
	h.compareVersions(c)
}

// exportMetadata exports all version metadata.
func (h *APIHandler) exportMetadata(c *gin.Context) {
	export, err := h.versionManager.ExportMetadata()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to export metadata",
		})
		return
	}

	c.Data(http.StatusOK, "application/json", export)
}

// healthCheck returns health status of the policy version manager.
func (h *APIHandler) healthCheck(c *gin.Context) {
	activeVersion := h.versionManager.GetActiveVersion()
	versions := h.versionManager.ListVersions()

	c.JSON(http.StatusOK, gin.H{
		"status":         "healthy",
		"active_version": activeVersion,
		"total_versions": len(versions),
		"timestamp":      time.Now(),
	})
}
