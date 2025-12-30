package token

import "github.com/mauriciomferz/AgentAuth/pkg/gauth"

// SetAgentAuthService sets the AgentAuth service for RFC compliant operations
func (h *Handler) SetAgentAuthService(service gauth.AgentAuth) {
	h.AgentAuthService = service
}
