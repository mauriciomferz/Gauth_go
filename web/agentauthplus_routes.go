package web

import (
	"github.com/mauriciomferz/AgentAuth/pkg/agentauthplus"
	agentauthplushandlers "github.com/mauriciomferz/AgentAuth/web/handlers/agentauthplus"
)

// RegisterAgentAuthPlusEndpoints registers all AgentAuth+ management endpoints for the five advanced features:
// - Successor Management (AI takeover scenarios)
// - Delegation Service (AI-to-AI delegations)
// - Dual Control (multi-approver workflows)
// - Capability Assessment (AI capability evaluation)
// - Fiduciary Duty (violation tracking)
//
// These endpoints enable operational management of AgentAuth+ features integrated into the AAP001 authorization chain.
func (s *BetaServer) RegisterAgentAuthPlusEndpoints(
	successorService agentauthplus.SuccessorManagementService,
	delegationService agentauthplus.DelegationService,
	dualControlService agentauthplus.DualControlService,
	capabilityService agentauthplus.CapabilityAssessmentService,
	fiduciaryService agentauthplus.FiduciaryDutyService,
) {
	// Create handlers
	successorHandlers := agentauthplushandlers.NewSuccessorHandlers(successorService)
	delegationHandlers := agentauthplushandlers.NewDelegationHandlers(delegationService)
	dualControlHandlers := agentauthplushandlers.NewDualControlHandlers(dualControlService)
	capabilityHandlers := agentauthplushandlers.NewCapabilityHandlers(capabilityService)
	fiduciaryHandlers := agentauthplushandlers.NewFiduciaryHandlers(fiduciaryService)

	// Successor Management endpoints
	s.router.POST("/api/v1/agentauthplus/successors/activate", successorHandlers.ActivateSuccessor)
	s.router.POST("/api/v1/agentauthplus/successors/deactivate", successorHandlers.DeactivateSuccessor)
	s.router.GET("/api/v1/agentauthplus/successors/active/:poaID", successorHandlers.GetActiveSuccessor)
	s.router.GET("/api/v1/agentauthplus/successors/history/:poaID", successorHandlers.ListSuccessorHistory)

	// Delegation Service endpoints
	s.router.POST("/api/v1/agentauthplus/delegations", delegationHandlers.CreateDelegation)
	s.router.POST("/api/v1/agentauthplus/delegations/:id/revoke", delegationHandlers.RevokeDelegation)
	s.router.POST("/api/v1/agentauthplus/delegations/validate", delegationHandlers.ValidateDelegation)
	s.router.GET("/api/v1/agentauthplus/delegations/chain/:agentID", delegationHandlers.GetDelegationChain)
	s.router.POST("/api/v1/agentauthplus/delegations/check-depth", delegationHandlers.CheckMaxDepth)

	// Dual Control endpoints
	s.router.POST("/api/v1/agentauthplus/dual-control/approvals", dualControlHandlers.RequestApproval)
	s.router.POST("/api/v1/agentauthplus/dual-control/approvals/:id/approve", dualControlHandlers.ApproveAction)
	s.router.POST("/api/v1/agentauthplus/dual-control/approvals/:id/reject", dualControlHandlers.RejectAction)
	s.router.GET("/api/v1/agentauthplus/dual-control/approvals/:id/status", dualControlHandlers.GetApprovalStatus)
	s.router.GET("/api/v1/agentauthplus/dual-control/approvals/pending", dualControlHandlers.GetPendingApprovals)
	s.router.GET("/api/v1/agentauthplus/dual-control/approvals/query", dualControlHandlers.FindApprovalsByPoAAndAction)

	// Capability Assessment endpoints
	s.router.POST("/api/v1/agentauthplus/capabilities/assess", capabilityHandlers.CreateAssessment)
	s.router.POST("/api/v1/agentauthplus/capabilities/certify", capabilityHandlers.GrantCertification)
	s.router.POST("/api/v1/agentauthplus/capabilities/certifications/:id/revoke", capabilityHandlers.RevokeCertification)
	s.router.GET("/api/v1/agentauthplus/capabilities/assessments/:agentID", capabilityHandlers.GetLatestAssessment)
	s.router.GET("/api/v1/agentauthplus/capabilities/certifications/:agentID", capabilityHandlers.ListCertifications)

	// Fiduciary Duty endpoints
	s.router.POST("/api/v1/agentauthplus/fiduciary/violations", fiduciaryHandlers.RecordViolation)
	s.router.POST("/api/v1/agentauthplus/fiduciary/violations/:id/resolve", fiduciaryHandlers.ResolveViolation)
	s.router.GET("/api/v1/agentauthplus/fiduciary/violations", fiduciaryHandlers.GetViolations)
	s.router.GET("/api/v1/agentauthplus/fiduciary/violations/by-severity", fiduciaryHandlers.GetViolationsBySeverity)
}
