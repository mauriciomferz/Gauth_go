// This file previously contained a duplicate GAuth implementation. All GAuth logic should use the canonical implementation in pkg/gauth/gauth.go.

// RequestToken issues a new token based on an authorization grant
func (g *GAuth) RequestToken(req TokenRequest) (*TokenResponse, error) {
	if !g.rateLimiter.Allow(req.GrantID) {
		return nil, errors.New(errors.ErrRateLimitExceeded, "rate limit exceeded")
	}

	token, err := generateToken()
	if err != nil {
		return nil, errors.New(errors.ErrInternalError, "failed to generate token")
	}
	tokenData := tokenstore.TokenData{
		Valid:      true,
		ValidUntil: time.Now().Add(g.config.AccessTokenExpiry),
		ClientID:   req.GrantID,
		Scope:      req.Scope,
	}

	if err := g.tokenStore.Store(token, tokenData); err != nil {
		return nil, err
	}

	return &TokenResponse{
		Token:        token,
		ValidUntil:   tokenData.ValidUntil,
		Scope:        tokenData.Scope,
		Restrictions: req.Restrictions,
	}, nil
}

// ValidateToken checks if a token is valid and returns its associated data
// ValidateToken validates the given token and returns its data
func (g *GAuth) ValidateToken(token string) (*tokenstore.TokenData, error) {
	data, exists := g.tokenStore.Get(token)
	if !exists {
		return nil, errors.New(errors.ErrInvalidToken, "token not found")
	}

	if !data.Valid || time.Now().After(data.ValidUntil) {
		return nil, errors.New(errors.ErrTokenExpired, "token has expired")
	}

	return &data, nil
}

func validateConfig(config Config) error {
	if config.AuthServerURL == "" {
		return errors.New(errors.ErrMissingConfig, "auth server URL is required")
	}
	if config.ClientID == "" {
		return errors.New(errors.ErrMissingConfig, "client ID is required")
	}
	if config.ClientSecret == "" {
		return errors.New(errors.ErrMissingConfig, "client secret is required")
	}
	if config.AccessTokenExpiry == 0 {
		config.AccessTokenExpiry = time.Hour // Default to 1 hour
	}
	return nil
}

func (g *GAuth) validateAuthRequest(req AuthorizationRequest) error {
	if req.ClientID == "" {
		return errors.New(errors.ErrInvalidConfig, "client ID is required")
	}
	if req.ClientID != g.config.ClientID {
		return errors.New(errors.ErrUnauthorized, "invalid client ID")
	}
	if len(req.Scopes) == 0 {
		return errors.New(errors.ErrInvalidConfig, "at least one scope is required")
	}
	return nil
}
