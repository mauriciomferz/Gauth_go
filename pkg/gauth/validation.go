package gauth

import "github.com/mauriciomferz/Gauth_go/pkg/errors"

// validateAuthRequest validates the authorization request.
func (g *GAuth) validateAuthRequest(req AuthorizationRequest) error {
	if req.ClientID == "" {
		return errors.New(errors.ErrInvalidClient, "client ID is required")
	}
	if req.ClientID != g.config.ClientID {
		return errors.New(errors.ErrUnauthorizedClient, "invalid client ID")
	}
	if len(req.Scopes) == 0 {
		return errors.New(errors.ErrInvalidScope, "at least one scope is required")
	}
	return nil
}
