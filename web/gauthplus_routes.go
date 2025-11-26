package web

import (
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauthplus"
	gauthplushandlers "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/handlers/gauthplus"
)

// RegisterGAuthPlusEndpoints registers all GAuth+ management endpoints for the five advanced features:
// - Successor Management (AI takeover scenarios)
// - Delegation Service (AI-to-AI delegations)
// - Dual Control (multi-approver workflows)
// - Capability Assessment (AI capability evaluation)
// - Fiduciary Duty (violation tracking)
//
// These endpoints enable operational management of GAuth+ features integrated into the RFC-0111 authorization chain.
func (s *BetaServer) RegisterGAuthPlusEndpoints(
	successorService gauthplus.SuccessorManagementService,
	delegationService gauthplus.DelegationService,
	dualControlService gauthplus.DualControlService,
	capabilityService gauthplus.CapabilityAssessmentService,
	fiduciaryService gauthplus.FiduciaryDutyService,
) {
	// Create handlers
	successorHandlers := gauthplushandlers.NewSuccessorHandlers(successorService)
	delegationHandlers := gauthplushandlers.NewDelegationHandlers(delegationService)
	dualControlHandlers := gauthplushandlers.NewDualControlHandlers(dualControlService)
	capabilityHandlers := gauthplushandlers.NewCapabilityHandlers(capabilityService)
	fiduciaryHandlers := gauthplushandlers.NewFiduciaryHandlers(fiduciaryService)

	// Successor Management endpoints
	s.router.POST("/api/v1/gauthplus/successors/activate", successorHandlers.ActivateSuccessor)
	s.router.POST("/api/v1/gauthplus/successors/deactivate", successorHandlers.DeactivateSuccessor)
	s.router.GET("/api/v1/gauthplus/successors/active/:poaID", successorHandlers.GetActiveSuccessor)
	s.router.GET("/api/v1/gauthplus/successors/history/:poaID", successorHandlers.ListSuccessorHistory)

	// Delegation Service endpoints
	s.router.POST("/api/v1/gauthplus/delegations", delegationHandlers.CreateDelegation)
	s.router.POST("/api/v1/gauthplus/delegations/:id/revoke", delegationHandlers.RevokeDelegation)
	s.router.POST("/api/v1/gauthplus/delegations/validate", delegationHandlers.ValidateDelegation)
	s.router.GET("/api/v1/gauthplus/delegations/chain/:agentID", delegationHandlers.GetDelegationChain)
	s.router.POST("/api/v1/gauthplus/delegations/check-depth", delegationHandlers.CheckMaxDepth)

	// Dual Control endpoints
	s.router.POST("/api/v1/gauthplus/dual-control/approvals", dualControlHandlers.RequestApproval)
	s.router.POST("/api/v1/gauthplus/dual-control/approvals/:id/approve", dualControlHandlers.ApproveAction)
	s.router.POST("/api/v1/gauthplus/dual-control/approvals/:id/reject", dualControlHandlers.RejectAction)
	s.router.GET("/api/v1/gauthplus/dual-control/approvals/:id/status", dualControlHandlers.GetApprovalStatus)
	s.router.GET("/api/v1/gauthplus/dual-control/approvals/pending", dualControlHandlers.GetPendingApprovals)
	s.router.GET("/api/v1/gauthplus/dual-control/approvals/query", dualControlHandlers.FindApprovalsByPoAAndAction)

	// Capability Assessment endpoints
	s.router.POST("/api/v1/gauthplus/capabilities/assess", capabilityHandlers.CreateAssessment)
	s.router.POST("/api/v1/gauthplus/capabilities/certify", capabilityHandlers.GrantCertification)
	s.router.POST("/api/v1/gauthplus/capabilities/certifications/:id/revoke", capabilityHandlers.RevokeCertification)
	s.router.GET("/api/v1/gauthplus/capabilities/assessments/:agentID", capabilityHandlers.GetLatestAssessment)
	s.router.GET("/api/v1/gauthplus/capabilities/certifications/:agentID", capabilityHandlers.ListCertifications)

	// Fiduciary Duty endpoints
	s.router.POST("/api/v1/gauthplus/fiduciary/violations", fiduciaryHandlers.RecordViolation)
	s.router.POST("/api/v1/gauthplus/fiduciary/violations/:id/resolve", fiduciaryHandlers.ResolveViolation)
	s.router.GET("/api/v1/gauthplus/fiduciary/violations", fiduciaryHandlers.GetViolations)
	s.router.GET("/api/v1/gauthplus/fiduciary/violations/by-severity", fiduciaryHandlers.GetViolationsBySeverity)
}
