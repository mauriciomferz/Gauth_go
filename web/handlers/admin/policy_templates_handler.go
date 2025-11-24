package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PolicyTemplatesHandler handles policy template management
type PolicyTemplatesHandler struct {
	db *pgxpool.Pool
}

// NewPolicyTemplatesHandler creates a new policy templates handler
func NewPolicyTemplatesHandler(db *pgxpool.Pool) *PolicyTemplatesHandler {
	return &PolicyTemplatesHandler{db: db}
}

// PolicyTemplate represents a policy template
type PolicyTemplate struct {
	ID                   string                 `json:"id"`
	TenantID             string                 `json:"tenant_id"`
	Name                 string                 `json:"name"`
	Description          *string                `json:"description,omitempty"`
	Category             string                 `json:"category"`
	Industry             *string                `json:"industry,omitempty"`
	TemplateType         string                 `json:"template_type"`
	PolicyRules          map[string]interface{} `json:"policy_rules"`
	Variables            []interface{}          `json:"variables,omitempty"`
	Version              int                    `json:"version"`
	IsLatest             bool                   `json:"is_latest"`
	ParentTemplateID     *string                `json:"parent_template_id,omitempty"`
	Status               string                 `json:"status"`
	Visibility           string                 `json:"visibility"`
	IsMarketplaceItem    bool                   `json:"is_marketplace_item"`
	MarketplaceRating    *float64               `json:"marketplace_rating,omitempty"`
	MarketplaceDownloads *int                   `json:"marketplace_downloads,omitempty"`
	MarketplacePrice     *float64               `json:"marketplace_price,omitempty"`
	AuthorID             *string                `json:"author_id,omitempty"`
	License              *string                `json:"license,omitempty"`
	Tags                 []string               `json:"tags,omitempty"`
	ComplianceFrameworks []string               `json:"compliance_frameworks,omitempty"`
	CreatedBy            string                 `json:"created_by"`
	UpdatedBy            *string                `json:"updated_by,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
	PublishedAt          *time.Time             `json:"published_at,omitempty"`
	DeprecatedAt         *time.Time             `json:"deprecated_at,omitempty"`
}

// CreatePolicyTemplateRequest represents a request to create a template
type CreatePolicyTemplateRequest struct {
	Name                 string                 `json:"name" binding:"required"`
	Description          string                 `json:"description"`
	Category             string                 `json:"category" binding:"required"`
	Industry             string                 `json:"industry"`
	TemplateType         string                 `json:"template_type" binding:"required"`
	PolicyRules          map[string]interface{} `json:"policy_rules" binding:"required"`
	Variables            []interface{}          `json:"variables"`
	Visibility           string                 `json:"visibility"`
	Tags                 []string               `json:"tags"`
	ComplianceFrameworks []string               `json:"compliance_frameworks"`
}

// UpdatePolicyTemplateRequest represents a request to update a template
type UpdatePolicyTemplateRequest struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	Category             string                 `json:"category"`
	Industry             string                 `json:"industry"`
	TemplateType         string                 `json:"template_type"`
	PolicyRules          map[string]interface{} `json:"policy_rules"`
	Variables            []interface{}          `json:"variables"`
	Status               string                 `json:"status"`
	Visibility           string                 `json:"visibility"`
	Tags                 []string               `json:"tags"`
	ComplianceFrameworks []string               `json:"compliance_frameworks"`
	Changelog            string                 `json:"changelog"`
}

// CloneTemplateRequest represents a request to clone a template
type CloneTemplateRequest struct {
	NewName         string                 `json:"new_name" binding:"required"`
	NewTenantID     string                 `json:"new_tenant_id"`
	ForkType        string                 `json:"fork_type"` // 'clone', 'inherit', 'customize'
	Customizations  map[string]interface{} `json:"customizations"`
	IncludeVersions bool                   `json:"include_versions"`
}

// ListPolicyTemplates returns all policy templates for a tenant
func (h *PolicyTemplatesHandler) ListPolicyTemplates(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	// Query parameters
	category := c.Query("category")
	industry := c.Query("industry")
	templateType := c.Query("template_type")
	status := c.Query("status")
	visibility := c.Query("visibility")
	isMarketplace := c.Query("is_marketplace") == "true"
	onlyLatest := c.DefaultQuery("only_latest", "true") == "true"
	search := c.Query("search")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	offset := (page - 1) * pageSize

	ctx := context.Background()

	// Build query
	query := `
		SELECT id, tenant_id, name, description, category, industry, template_type,
			   policy_rules, variables, version, is_latest, parent_template_id,
			   status, visibility, is_marketplace_item, marketplace_rating,
			   marketplace_downloads, marketplace_price, author_id, license,
			   tags, compliance_frameworks, created_by, updated_by,
			   created_at, updated_at, published_at, deprecated_at
		FROM policy_templates
		WHERE (tenant_id = $1 OR visibility IN ('public', 'marketplace'))
	`
	args := []interface{}{tenantID}
	argCount := 1

	if category != "" {
		argCount++
		query += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, category)
	}
	if industry != "" {
		argCount++
		query += fmt.Sprintf(" AND industry = $%d", argCount)
		args = append(args, industry)
	}
	if templateType != "" {
		argCount++
		query += fmt.Sprintf(" AND template_type = $%d", argCount)
		args = append(args, templateType)
	}
	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}
	if visibility != "" {
		argCount++
		query += fmt.Sprintf(" AND visibility = $%d", argCount)
		args = append(args, visibility)
	}
	if isMarketplace {
		query += " AND is_marketplace_item = true"
	}
	if onlyLatest {
		query += " AND is_latest = true"
	}
	if search != "" {
		argCount++
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)
	args = append(args, pageSize, offset)

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query templates", "details": err.Error()})
		return
	}
	defer rows.Close()

	templates := []PolicyTemplate{}
	for rows.Next() {
		var t PolicyTemplate
		var policyRulesJSON []byte
		var variablesJSON []byte

		err := rows.Scan(
			&t.ID, &t.TenantID, &t.Name, &t.Description, &t.Category, &t.Industry,
			&t.TemplateType, &policyRulesJSON, &variablesJSON, &t.Version, &t.IsLatest,
			&t.ParentTemplateID, &t.Status, &t.Visibility, &t.IsMarketplaceItem,
			&t.MarketplaceRating, &t.MarketplaceDownloads, &t.MarketplacePrice,
			&t.AuthorID, &t.License, &t.Tags, &t.ComplianceFrameworks,
			&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt,
			&t.PublishedAt, &t.DeprecatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan template", "details": err.Error()})
			return
		}

		json.Unmarshal(policyRulesJSON, &t.PolicyRules)
		json.Unmarshal(variablesJSON, &t.Variables)

		templates = append(templates, t)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM policy_templates WHERE (tenant_id = $1 OR visibility IN ('public', 'marketplace'))`
	var totalCount int
	err = h.db.QueryRow(ctx, countQuery, tenantID).Scan(&totalCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count templates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates":   templates,
		"total":       totalCount,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (totalCount + pageSize - 1) / pageSize,
	})
}

// GetPolicyTemplate returns a single policy template
func (h *PolicyTemplatesHandler) GetPolicyTemplate(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	templateID := c.Param("id")
	ctx := context.Background()

	query := `
		SELECT id, tenant_id, name, description, category, industry, template_type,
			   policy_rules, variables, version, is_latest, parent_template_id,
			   status, visibility, is_marketplace_item, marketplace_rating,
			   marketplace_downloads, marketplace_price, author_id, license,
			   tags, compliance_frameworks, created_by, updated_by,
			   created_at, updated_at, published_at, deprecated_at
		FROM policy_templates
		WHERE id = $1 AND (tenant_id = $2 OR visibility IN ('public', 'marketplace'))
	`

	var t PolicyTemplate
	var policyRulesJSON []byte
	var variablesJSON []byte

	err := h.db.QueryRow(ctx, query, templateID, tenantID).Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Description, &t.Category, &t.Industry,
		&t.TemplateType, &policyRulesJSON, &variablesJSON, &t.Version, &t.IsLatest,
		&t.ParentTemplateID, &t.Status, &t.Visibility, &t.IsMarketplaceItem,
		&t.MarketplaceRating, &t.MarketplaceDownloads, &t.MarketplacePrice,
		&t.AuthorID, &t.License, &t.Tags, &t.ComplianceFrameworks,
		&t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt,
		&t.PublishedAt, &t.DeprecatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get template", "details": err.Error()})
		return
	}

	json.Unmarshal(policyRulesJSON, &t.PolicyRules)
	json.Unmarshal(variablesJSON, &t.Variables)

	c.JSON(http.StatusOK, t)
}

// CreatePolicyTemplate creates a new policy template
func (h *PolicyTemplatesHandler) CreatePolicyTemplate(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	var req CreatePolicyTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// TODO: Get authenticated user from context
	createdBy := "admin" // Placeholder

	ctx := context.Background()
	templateID := uuid.New().String()

	policyRulesJSON, _ := json.Marshal(req.PolicyRules)
	variablesJSON, _ := json.Marshal(req.Variables)

	visibility := req.Visibility
	if visibility == "" {
		visibility = "private"
	}

	query := `
		INSERT INTO policy_templates (
			id, tenant_id, name, description, category, industry, template_type,
			policy_rules, variables, visibility, tags, compliance_frameworks,
			created_by, version, is_latest, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 1, true, 'draft')
		RETURNING id, created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := h.db.QueryRow(ctx, query,
		templateID, tenantID, req.Name, req.Description, req.Category, req.Industry,
		req.TemplateType, policyRulesJSON, variablesJSON, visibility,
		req.Tags, req.ComplianceFrameworks, createdBy,
	).Scan(&templateID, &createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         templateID,
		"created_at": createdAt,
		"updated_at": updatedAt,
		"message":    "Template created successfully",
	})
}

// UpdatePolicyTemplate updates an existing policy template (creates new version)
func (h *PolicyTemplatesHandler) UpdatePolicyTemplate(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	templateID := c.Param("id")

	var req UpdatePolicyTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// TODO: Get authenticated user from context
	updatedBy := "admin" // Placeholder

	ctx := context.Background()

	// Get current version
	var currentVersion int
	err := h.db.QueryRow(ctx, "SELECT version FROM policy_templates WHERE id = $1 AND tenant_id = $2 AND is_latest = true", templateID, tenantID).Scan(&currentVersion)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Mark current version as not latest
	_, err = h.db.Exec(ctx, "UPDATE policy_templates SET is_latest = false WHERE id = $1 AND version = $2", templateID, currentVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update version", "details": err.Error()})
		return
	}

	// Create new version
	newVersion := currentVersion + 1
	policyRulesJSON, _ := json.Marshal(req.PolicyRules)
	variablesJSON, _ := json.Marshal(req.Variables)

	query := `
		INSERT INTO policy_templates (
			id, tenant_id, name, description, category, industry, template_type,
			policy_rules, variables, version, is_latest, status, visibility,
			tags, compliance_frameworks, updated_by, created_by
		)
		SELECT id, tenant_id, COALESCE($2, name), COALESCE($3, description), COALESCE($4, category),
			   COALESCE($5, industry), COALESCE($6, template_type), COALESCE($7, policy_rules),
			   COALESCE($8, variables), $9, true, COALESCE($10, status), COALESCE($11, visibility),
			   COALESCE($12, tags), COALESCE($13, compliance_frameworks), $14, created_by
		FROM policy_templates
		WHERE id = $1 AND version = $15
		RETURNING updated_at
	`

	var updatedAt time.Time
	err = h.db.QueryRow(ctx, query,
		templateID, req.Name, req.Description, req.Category, req.Industry,
		req.TemplateType, policyRulesJSON, variablesJSON, newVersion,
		req.Status, req.Visibility, req.Tags, req.ComplianceFrameworks,
		updatedBy, currentVersion,
	).Scan(&updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new version", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          templateID,
		"version":     newVersion,
		"updated_at":  updatedAt,
		"message":     "Template updated successfully (new version created)",
	})
}

// ClonePolicyTemplate clones a template
func (h *PolicyTemplatesHandler) ClonePolicyTemplate(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	originalTemplateID := c.Param("id")

	var req CloneTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// TODO: Get authenticated user from context
	createdBy := "admin" // Placeholder

	ctx := context.Background()
	newTemplateID := uuid.New().String()

	targetTenantID := req.NewTenantID
	if targetTenantID == "" {
		targetTenantID = tenantID
	}

	forkType := req.ForkType
	if forkType == "" {
		forkType = "clone"
	}

	// Clone the template
	query := `
		INSERT INTO policy_templates (
			id, tenant_id, name, description, category, industry, template_type,
			policy_rules, variables, version, is_latest, parent_template_id,
			status, visibility, tags, compliance_frameworks, created_by
		)
		SELECT $1, $2, $3, description, category, industry, template_type,
			   policy_rules, variables, 1, true, id,
			   'draft', 'private', tags, compliance_frameworks, $4
		FROM policy_templates
		WHERE id = $5 AND (tenant_id = $6 OR visibility IN ('public', 'marketplace'))
		AND is_latest = true
		RETURNING created_at
	`

	var createdAt time.Time
	err := h.db.QueryRow(ctx, query,
		newTemplateID, targetTenantID, req.NewName, createdBy, originalTemplateID, tenantID,
	).Scan(&createdAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clone template", "details": err.Error()})
		return
	}

	// Record the fork
	customizationsJSON, _ := json.Marshal(req.Customizations)
	_, err = h.db.Exec(ctx, `
		INSERT INTO policy_template_forks (original_template_id, forked_template_id, tenant_id, fork_type, customizations, forked_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, originalTemplateID, newTemplateID, targetTenantID, forkType, customizationsJSON, createdBy)

	if err != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to record fork: %v\n", err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          newTemplateID,
		"name":        req.NewName,
		"created_at":  createdAt,
		"parent_id":   originalTemplateID,
		"message":     "Template cloned successfully",
	})
}

// DeletePolicyTemplate deletes (archives) a policy template
func (h *PolicyTemplatesHandler) DeletePolicyTemplate(c *gin.Context) {
	tenantID := c.GetHeader("X-Tenant-ID")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header is required"})
		return
	}

	templateID := c.Param("id")
	ctx := context.Background()

	// Soft delete by archiving
	query := `UPDATE policy_templates SET status = 'archived', updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND tenant_id = $2`
	result, err := h.db.Exec(ctx, query, templateID, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template", "details": err.Error()})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template archived successfully"})
}
