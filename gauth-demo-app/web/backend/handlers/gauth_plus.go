package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

// GAuthPlusHandler provides comprehensive GAuth+ protocol handlers
type GAuthPlusHandler struct {
	service *services.GAuthPlusService
	logger  *logrus.Logger
}

// NewGAuthPlusHandler creates a new GAuth+ handler
func NewGAuthPlusHandler(service *services.GAuthPlusService, logger *logrus.Logger) *GAuthPlusHandler {
	return &GAuthPlusHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterAIAuthorization registers comprehensive AI authorization on blockchain
// @Summary Register AI Authorization
// @Description Register comprehensive AI authorization with power-of-attorney on blockchain commercial register
// @Tags GAuth+
// @Accept json
// @Produce json
// @Param authorization body services.AIAuthorizationRecord true "AI Authorization Record"
// @Success 200 {object} services.AIAuthorizationRecord
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/gauth-plus/authorize [post]
func (h *GAuthPlusHandler) RegisterAIAuthorization(c *gin.Context) {
	var req services.AIAuthorizationRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid authorization request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid authorization record format",
			"details": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"ai_system_id":      req.AISystemID,
		"authorizing_party": req.AuthorizingParty.ID,
	}).Info("Processing comprehensive AI authorization registration")

	// Register authorization on blockchain
	record, err := h.service.RegisterAIAuthorization(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to register AI authorization")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "registration_failed",
			"message": "Failed to register AI authorization",
			"details": err.Error(),
		})
		return
	}

	// Successful registration response
	c.JSON(http.StatusOK, gin.H{
		"success":              true,
		"message":              "AI authorization successfully registered on blockchain commercial register",
		"authorization_record": record,
		"blockchain_hash":      record.BlockchainHash,
		"commercial_register":  gin.H{
			"registered":        true,
			"registry_type":     "blockchain_commercial_register",
			"verification":      "cryptographically_verified",
		},
		"compliance_status": gin.H{
			"gauth_plus_compliant": true,
			"power_of_attorney":    "verified",
			"dual_control":         record.DualControlPrinciple.Enabled,
			"authorization_cascade": "validated",
			"ultimate_human":       record.AuthorizationCascade.UltimateHuman.Name,
		},
		"authority_summary": h.generateAuthoritySummary(record),
		"next_steps": []string{
			"AI system can now act within registered powers",
			"All actions will be validated against blockchain registry",
			"Dual control approval required for sensitive operations",
			"Authority can be verified by any relying party",
		},
	})
}

// ValidateAIAuthority validates an AI's authority to perform specific actions
// @Summary Validate AI Authority
// @Description Validate an AI system's authority to perform specific actions against blockchain registry
// @Tags GAuth+
// @Accept json
// @Produce json
// @Param ai_system_id query string true "AI System ID"
// @Param action query string true "Action to validate"
// @Param context body map[string]interface{} false "Action context"
// @Success 200 {object} services.AuthorityValidationResult
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/gauth-plus/validate [post]
func (h *GAuthPlusHandler) ValidateAIAuthority(c *gin.Context) {
	aiSystemID := c.Query("ai_system_id")
	action := c.Query("action")

	if aiSystemID == "" || action == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_parameters",
			"message": "ai_system_id and action parameters are required",
		})
		return
	}

	// Parse optional context
	var context map[string]interface{}
	if err := c.ShouldBindJSON(&context); err != nil {
		context = make(map[string]interface{}) // Use empty context if none provided
	}

	h.logger.WithFields(logrus.Fields{
		"ai_system_id": aiSystemID,
		"action":       action,
	}).Info("Validating AI authority")

	// Validate authority against blockchain registry
	result, err := h.service.ValidateAIAuthority(c.Request.Context(), aiSystemID, action, context)
	if err != nil {
		h.logger.WithError(err).Error("Authority validation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "validation_failed",
			"message": "Failed to validate AI authority",
			"details": err.Error(),
		})
		return
	}

	// Return validation result
	response := gin.H{
		"validation_result": result,
		"blockchain_verified": true,
		"commercial_register": gin.H{
			"verified":     result.Valid,
			"registry_access": "public_blockchain_verification",
		},
	}

	if result.Valid {
		response["authority_confirmed"] = gin.H{
			"authorized":         true,
			"action":            action,
			"authorization_source": "blockchain_commercial_register",
			"verification_hash":  result.Record.BlockchainHash,
			"ultimate_human":     result.Record.AuthorizationCascade.UltimateHuman.Name,
		}
		c.JSON(http.StatusOK, response)
	} else {
		response["authority_denied"] = gin.H{
			"authorized": false,
			"reason":     result.Reason,
			"action":     action,
		}
		c.JSON(http.StatusForbidden, response)
	}
}

// GetCommercialRegisterEntry retrieves AI system entry from commercial register
// @Summary Get Commercial Register Entry
// @Description Retrieve AI system entry from blockchain commercial register
// @Tags GAuth+
// @Produce json
// @Param ai_system_id path string true "AI System ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/gauth-plus/commercial-register/{ai_system_id} [get]
func (h *GAuthPlusHandler) GetCommercialRegisterEntry(c *gin.Context) {
	aiSystemID := c.Param("ai_system_id")
	
	h.logger.WithFields(logrus.Fields{
		"ai_system_id": aiSystemID,
	}).Info("Retrieving commercial register entry")

	// Retrieve from blockchain registry
	record, err := h.service.ValidateAIAuthority(c.Request.Context(), aiSystemID, "view_registration", nil)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "AI system not found in commercial register",
		})
		return
	}

	// Format commercial register entry
	entry := gin.H{
		"ai_system_id": aiSystemID,
		"commercial_register": gin.H{
			"registration_type":   "blockchain_commercial_register",
			"registration_date":   record.Record.CreatedAt,
			"status":             record.Record.Status,
			"blockchain_hash":     record.Record.BlockchainHash,
			"publicly_verifiable": true,
		},
		"authorizing_party": gin.H{
			"name":               record.Record.AuthorizingParty.Name,
			"type":               record.Record.AuthorizingParty.Type,
			"registered_office":  record.Record.AuthorizingParty.RegisteredOffice,
			"verification_status": record.Record.AuthorizingParty.VerificationStatus,
		},
		"powers_registered": h.formatRegisteredPowers(record.Record.PowersGranted),
		"decision_authority": gin.H{
			"autonomous_decisions": record.Record.DecisionAuthority.AutonomousDecisions,
			"approval_required":    record.Record.DecisionAuthority.ApprovalRequired,
		},
		"transaction_rights": gin.H{
			"allowed_types": record.Record.TransactionRights.AllowedTransactionTypes,
			"limits":        record.Record.TransactionRights.TransactionLimits,
		},
		"dual_control": gin.H{
			"enabled":           record.Record.DualControlPrinciple.Enabled,
			"requires_approval": record.Record.DualControlPrinciple.RequiredForActions,
		},
		"authorization_cascade": gin.H{
			"ultimate_human":     record.Record.AuthorizationCascade.UltimateHuman,
			"accountability_chain": record.Record.AuthorizationCascade.AccountabilityChain,
		},
		"verification": gin.H{
			"blockchain_verifiable": true,
			"global_access":        true,
			"relying_party_verification": "enabled",
		},
	}

	c.JSON(http.StatusOK, entry)
}

// CreateAuthorizingParty creates and verifies an authorizing party
// @Summary Create Authorizing Party
// @Description Create and verify an authorizing party for AI authorization
// @Tags GAuth+
// @Accept json
// @Produce json
// @Param party body services.AuthorizingParty true "Authorizing Party"
// @Success 200 {object} services.AuthorizingParty
// @Router /api/v1/gauth-plus/authorizing-party [post]
func (h *GAuthPlusHandler) CreateAuthorizingParty(c *gin.Context) {
	var party services.AuthorizingParty
	if err := c.ShouldBindJSON(&party); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid authorizing party format",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"party_id":   party.ID,
		"party_name": party.Name,
		"party_type": party.Type,
	}).Info("Creating authorizing party")

	// Generate ID and set timestamps
	party.ID = generateID("auth_party")
	party.CreatedAt = time.Now()
	party.VerificationStatus = "pending_verification"

	// Perform identity verification (simplified for demo)
	if err := h.verifyAuthorizingPartyIdentity(&party); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "verification_failed",
			"message": "Failed to verify authorizing party identity",
			"details": err.Error(),
		})
		return
	}

	party.VerificationStatus = "verified"

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"authorizing_party": party,
		"verification": gin.H{
			"identity_verified": true,
			"legal_capacity":   party.LegalCapacity.Verified,
			"authority_level":  party.AuthorityLevel,
		},
		"next_steps": []string{
			"Authorizing party can now create AI authorizations",
			"Powers will be recorded in blockchain commercial register",
			"All delegations will be publicly verifiable",
		},
	})
}

// GetAuthorizationCascade retrieves the authorization cascade for an AI system
// @Summary Get Authorization Cascade  
// @Description Retrieve the complete authorization cascade showing human accountability chain
// @Tags GAuth+
// @Produce json
// @Param ai_system_id path string true "AI System ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/gauth-plus/cascade/{ai_system_id} [get]
func (h *GAuthPlusHandler) GetAuthorizationCascade(c *gin.Context) {
	aiSystemID := c.Param("ai_system_id")

	h.logger.WithFields(logrus.Fields{
		"ai_system_id": aiSystemID,
	}).Info("Retrieving authorization cascade")

	// Get authorization record
	result, err := h.service.ValidateAIAuthority(c.Request.Context(), aiSystemID, "view_cascade", nil)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Authorization cascade not found",
		})
		return
	}

	cascade := result.Record.AuthorizationCascade

	c.JSON(http.StatusOK, gin.H{
		"ai_system_id": aiSystemID,
		"authorization_cascade": gin.H{
			"ultimate_human": cascade.UltimateHuman,
			"human_authority": cascade.HumanAuthority,
			"cascade_chain": cascade.CascadeChain,
			"accountability_chain": cascade.AccountabilityChain,
		},
		"verification": gin.H{
			"human_at_top":       cascade.HumanAuthority.IsUltimate,
			"cascade_validated":  true,
			"blockchain_verified": true,
		},
		"compliance": gin.H{
			"dual_control_principle": "enforced",
			"human_accountability": "guaranteed",
			"organizational_fault_reduction": "active",
			"trust_preservation": "maintained",
		},
	})
}

// QueryCommercialRegister queries the commercial register with various filters
// @Summary Query Commercial Register
// @Description Query blockchain commercial register for AI systems with filters
// @Tags GAuth+
// @Produce json
// @Param authorizing_party query string false "Filter by authorizing party"
// @Param power_type query string false "Filter by power type"
// @Param status query string false "Filter by status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/gauth-plus/commercial-register [get]
func (h *GAuthPlusHandler) QueryCommercialRegister(c *gin.Context) {
	authorizingParty := c.Query("authorizing_party")
	powerType := c.Query("power_type")
	status := c.Query("status")
	
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	h.logger.WithFields(logrus.Fields{
		"authorizing_party": authorizingParty,
		"power_type":       powerType,
		"status":           status,
		"page":             page,
		"limit":            limit,
	}).Info("Querying commercial register")

	// Mock query results for demo
	entries := h.generateMockRegisterEntries(authorizingParty, powerType, status, page, limit)

	c.JSON(http.StatusOK, gin.H{
		"commercial_register_query": gin.H{
			"total_entries": len(entries),
			"page":         page,
			"limit":        limit,
			"filters": gin.H{
				"authorizing_party": authorizingParty,
				"power_type":       powerType,
				"status":           status,
			},
		},
		"entries": entries,
		"registry_info": gin.H{
			"type":               "blockchain_commercial_register",
			"global_accessibility": true,
			"verification_method": "cryptographic_proof",
			"relying_party_access": "unrestricted",
		},
	})
}

// generateAuthoritySummary creates a summary of AI authority for responses
func (h *GAuthPlusHandler) generateAuthoritySummary(record *services.AIAuthorizationRecord) map[string]interface{} {
	summary := map[string]interface{}{
		"basic_powers_count":   len(record.PowersGranted.BasicPowers),
		"derived_powers_count": len(record.PowersGranted.DerivedPowers),
		"decision_categories":  []string{},
		"transaction_types":    []string{},
		"dual_control_enabled": record.DualControlPrinciple.Enabled,
	}

	// Categorize decisions
	if record.DecisionAuthority != nil {
		summary["autonomous_decisions_count"] = len(record.DecisionAuthority.AutonomousDecisions)
		summary["approval_required_count"] = len(record.DecisionAuthority.ApprovalRequired)
	}

	// Categorize transactions  
	if record.TransactionRights != nil {
		summary["allowed_transaction_types"] = record.TransactionRights.AllowedTransactionTypes
		summary["prohibited_transactions"] = record.TransactionRights.ProhibitedTransactions
	}

	return summary
}

// formatRegisteredPowers formats powers for commercial register display
func (h *GAuthPlusHandler) formatRegisteredPowers(powers *services.PowersGranted) map[string]interface{} {
	if powers == nil {
		return nil
	}

	formatted := map[string]interface{}{
		"basic_powers":      powers.BasicPowers,
		"derived_powers":    powers.DerivedPowers,
		"power_derivation":  powers.PowerDerivation,
	}

	if powers.StandardPowers != nil {
		formatted["standard_powers"] = gin.H{
			"financial_powers":      powers.StandardPowers.FinancialPowers != nil,
			"contractual_powers":    powers.StandardPowers.ContractualPowers != nil,
			"operational_powers":    powers.StandardPowers.OperationalPowers != nil,
			"representation_powers": powers.StandardPowers.RepresentationPowers != nil,
			"compliance_powers":     powers.StandardPowers.CompliancePowers != nil,
		}
	}

	return formatted
}

// verifyAuthorizingPartyIdentity performs identity verification (simplified)
func (h *GAuthPlusHandler) verifyAuthorizingPartyIdentity(party *services.AuthorizingParty) error {
	// In a real implementation, this would perform comprehensive identity verification
	// For demo purposes, we'll simulate verification

	if party.Name == "" {
		return fmt.Errorf("name is required")
	}

	if party.Type == "" {
		return fmt.Errorf("party type is required")
	}

	// Set up legal capacity
	party.LegalCapacity = &services.LegalCapacity{
		Verified:         true,
		VerificationDate: time.Now(),
		VerifiedBy:       "gauth_plus_verification_service",
		Jurisdiction:     "US", // Default for demo
		LegalFramework:   "corporate_power_of_attorney_act_2024",
	}

	// Set authority level
	if party.AuthorityLevel == "" {
		party.AuthorityLevel = "primary"
	}

	return nil
}

// generateMockRegisterEntries generates mock commercial register entries for demo
func (h *GAuthPlusHandler) generateMockRegisterEntries(authorizingParty, powerType, status string, page, limit int) []map[string]interface{} {
	entries := []map[string]interface{}{
		{
			"ai_system_id": "ai_financial_assistant_001",
			"authorizing_party": gin.H{
				"name": "Enterprise Financial Corp",
				"type": "corporation",
			},
			"powers": []string{"financial_operations", "contract_signing"},
			"status": "active",
			"registration_date": time.Now().Add(-30 * 24 * time.Hour),
			"blockchain_hash": "0x1234567890abcdef",
		},
		{
			"ai_system_id": "ai_legal_assistant_002", 
			"authorizing_party": gin.H{
				"name": "Law Firm Partners LLC",
				"type": "corporation",
			},
			"powers": []string{"legal_research", "document_drafting"},
			"status": "active",
			"registration_date": time.Now().Add(-15 * 24 * time.Hour),
			"blockchain_hash": "0xabcdef1234567890",
		},
		{
			"ai_system_id": "ai_trading_bot_003",
			"authorizing_party": gin.H{
				"name": "Investment Management Inc",
				"type": "corporation", 
			},
			"powers": []string{"trading_operations", "portfolio_management"},
			"status": "active",
			"registration_date": time.Now().Add(-7 * 24 * time.Hour),
			"blockchain_hash": "0x567890abcdef1234",
		},
	}

	// Apply filters (simplified)
	filtered := []map[string]interface{}{}
	for _, entry := range entries {
		include := true
		
		if authorizingParty != "" {
			if partyInfo, ok := entry["authorizing_party"].(gin.H); ok {
				if name, ok := partyInfo["name"].(string); ok {
					if !strings.Contains(strings.ToLower(name), strings.ToLower(authorizingParty)) {
						include = false
					}
				}
			}
		}
		
		if powerType != "" {
			if powers, ok := entry["powers"].([]string); ok {
				found := false
				for _, power := range powers {
					if strings.Contains(strings.ToLower(power), strings.ToLower(powerType)) {
						found = true
						break
					}
				}
				if !found {
					include = false
				}
			}
		}
		
		if status != "" && entry["status"] != status {
			include = false
		}
		
		if include {
			filtered = append(filtered, entry)
		}
	}

	// Apply pagination
	startIdx := (page - 1) * limit
	endIdx := startIdx + limit
	
	if startIdx >= len(filtered) {
		return []map[string]interface{}{}
	}
	
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}
	
	return filtered[startIdx:endIdx]
}

// generateID generates a unique identifier with prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}