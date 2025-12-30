package token

import "github.com/mauriciomferz/AgentAuth/pkg/agentauth"

// SetAgentAuthService sets the AgentAuth service for RFC compliant operations
func (h *Handler) SetAgentAuthService(service agentauth.AgentAuth) {
	h.AgentAuthService = service
}
