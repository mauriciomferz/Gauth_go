package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/audit"
	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// OAuth2 grant types
const (
	GrantTypeAuthCode     = "authorization_code"
	GrantTypeClientCreds  = "client_credentials"
	GrantTypePassword     = "password"
	GrantTypeRefreshToken = "refresh_token"
)

// OAuth2Config extends the base Config with OAuth2-specific settings
type OAuth2Config struct {
	Config
	TokenURL         string
	AuthorizeURL     string
	RedirectURL      string
	Endpoints        map[string]string
	TokenStore       token.EnhancedStore
	ValidateUserFunc func(context.Context, string, string) error
}

// OAuth2TokenResponse represents the OAuth2 token endpoint response
type OAuth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// oauth2Authenticator implements the Authenticator interface for OAuth2
type oauth2Authenticator struct {
	config       OAuth2Config
	httpClient   *http.Client
}

func newOAuth2Authenticator(config Config) (Authenticator, error) {
	oauthConfig, ok := config.ExtraConfig.(OAuth2Config)
	if !ok {
		return nil, errors.New("invalid OAuth2 config")
	}

	if oauthConfig.TokenURL == "" {
		return nil, errors.New("token URL is required")
	}

	return &oauth2Authenticator{
		config:     oauthConfig,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (a *oauth2Authenticator) Initialize(ctx context.Context) error {
	if a.config.TokenStore != nil {
		return a.config.TokenStore.Initialize(ctx)
	}
	return nil
}

func (a *oauth2Authenticator) Close() error {
	if a.config.TokenStore != nil {
		return a.config.TokenStore.Close()
	}
	return nil
}

func (a *oauth2Authenticator) ValidateCredentials(ctx context.Context, creds interface{}) error {
	switch c := creds.(type) {
	case basicCredentials:
		if a.config.ValidateUserFunc != nil {
			return a.config.ValidateUserFunc(ctx, c.Username, c.Password)
		}
		return errors.New("no user validation function configured")
	default:
		return fmt.Errorf("unsupported credentials type: %T", creds)
	}
}

func (a *oauth2Authenticator) GenerateToken(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	// Prepare request data based on grant type
	data, err := a.prepareTokenRequestData(req)
	if err != nil {
		return nil, err
	}

	// Make HTTP request and get response
	oauthResp, err := a.makeTokenRequest(data)
	if err != nil {
		return nil, err
	}

	// Create response and handle storage/audit
	return a.createAndStoreTokenResponse(ctx, req, oauthResp)
}

func (a *oauth2Authenticator) ValidateToken(ctx context.Context, tokenStr string) (*TokenData, error) {
	if a.config.TokenStore != nil {
		// Check if token is in the store
		data, err := a.config.TokenStore.Get(ctx, tokenStr)
		if err != nil {
			return nil, fmt.Errorf("token not found in store: %w", err)
		}
		// Convert *token.Token to *auth.TokenData for compatibility
		return ConvertTokenToTokenData(data), nil
	}

	// If no token store, validate against introspection endpoint
	if endpoint, ok := a.config.Endpoints["introspection"]; ok {
		data := url.Values{}
		data.Set("token", tokenStr)
		data.Set("client_id", a.config.ClientID)
		data.Set("client_secret", a.config.ClientSecret)

		resp, err := a.httpClient.PostForm(endpoint, data)
		if err != nil {
			return nil, fmt.Errorf("introspection request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, ErrInvalidToken
		}

		var introspection struct {
			Active   bool   `json:"active"`
			Scope    string `json:"scope"`
			ClientID string `json:"client_id"`
			Username string `json:"username"`
			Exp      int64  `json:"exp"`
			Iat      int64  `json:"iat"`
			Sub      string `json:"sub"`
			Aud      string `json:"aud"`
			Iss      string `json:"iss"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&introspection); err != nil {
			return nil, fmt.Errorf("failed to decode introspection response: %w", err)
		}

		if !introspection.Active {
			return nil, ErrInvalidToken
		}

		return &TokenData{
			Valid:     true,
			Subject:   introspection.Sub,
			Issuer:    introspection.Iss,
			Audience:  introspection.Aud,
			IssuedAt:  time.Unix(introspection.Iat, 0),
			ExpiresAt: time.Unix(introspection.Exp, 0),
			Scope:     strings.Split(introspection.Scope, " "),
		}, nil
	}

	return nil, errors.New("no token validation method configured")
}

func (a *oauth2Authenticator) RevokeToken(ctx context.Context, tokenStr string) error {
	if endpoint, ok := a.config.Endpoints["revocation"]; ok {
		data := url.Values{}
		data.Set("token", tokenStr)
		data.Set("client_id", a.config.ClientID)
		data.Set("client_secret", a.config.ClientSecret)

		resp, err := a.httpClient.PostForm(endpoint, data)
		if err != nil {
			return fmt.Errorf("revocation request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("revocation failed with status %d", resp.StatusCode)
		}
	}

	if a.config.TokenStore != nil {
		if err := a.config.TokenStore.Remove(ctx, tokenStr); err != nil {
			return fmt.Errorf("failed to remove token from store: %w", err)
		}
	}

	if a.config.AuditLogger != nil {
		a.config.AuditLogger.Log(ctx, &audit.Entry{
			Type:   audit.TypeToken,
			Action: audit.ActionTokenRevoke,
			Result: audit.ResultSuccess,
		})
	}

	return nil
}

// prepareTokenRequestData prepares URL values based on grant type
func (a *oauth2Authenticator) prepareTokenRequestData(req TokenRequest) (url.Values, error) {
	data := url.Values{}
	data.Set("client_id", a.config.ClientID)
	data.Set("client_secret", a.config.ClientSecret)

	switch req.GrantType {
	case GrantTypeClientCreds:
		data.Set("grant_type", "client_credentials")
		if len(req.Scopes) > 0 {
			data.Set("scope", strings.Join(req.Scopes, " "))
		}

	case GrantTypePassword:
		creds, ok := req.Metadata["credentials"].(basicCredentials)
		if !ok {
			return nil, errors.New("invalid credentials for password grant")
		}
		data.Set("grant_type", "password")
		data.Set("username", creds.Username)
		data.Set("password", creds.Password)
		if len(req.Scopes) > 0 {
			data.Set("scope", strings.Join(req.Scopes, " "))
		}

	case GrantTypeAuthCode:
		code, ok := req.Metadata["code"].(string)
		if !ok {
			return nil, errors.New("authorization code required")
		}
		data.Set("grant_type", "authorization_code")
		data.Set("code", code)
		data.Set("redirect_uri", a.config.RedirectURL)

	case GrantTypeRefreshToken:
		refreshToken, ok := req.Metadata["refresh_token"].(string)
		if !ok {
			return nil, errors.New("refresh token required")
		}
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", refreshToken)

	default:
		return nil, fmt.Errorf("unsupported grant type: %s", req.GrantType)
	}

	return data, nil
}

// makeTokenRequest makes HTTP request and handles response
func (a *oauth2Authenticator) makeTokenRequest(data url.Values) (*OAuth2TokenResponse, error) {
	resp, err := a.httpClient.PostForm(a.config.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, body)
	}

	var oauthResp OAuth2TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &oauthResp, nil
}

// createAndStoreTokenResponse creates response and handles storage/audit
func (a *oauth2Authenticator) createAndStoreTokenResponse(ctx context.Context, req TokenRequest, oauthResp *OAuth2TokenResponse) (*TokenResponse, error) {
	tokenResp := &TokenResponse{
		Token:     oauthResp.AccessToken,
		TokenType: oauthResp.TokenType,
		ExpiresIn: oauthResp.ExpiresIn,
		Scope:     strings.Split(oauthResp.Scope, " "),
		Claims: map[string]interface{}{
			"refresh_token": oauthResp.RefreshToken,
		},
	}

	if a.config.TokenStore != nil {
		if err := a.config.TokenStore.Store(ctx, tokenResp); err != nil {
			return nil, fmt.Errorf("failed to store token: %w", err)
		}
	}

	if a.config.AuditLogger != nil {
		a.config.AuditLogger.Log(ctx, &audit.Entry{
			Type:    audit.TypeToken,
			Action:  audit.ActionTokenGenerate,
			ActorID: req.Subject,
			Result:  audit.ResultSuccess,
			Metadata: map[string]string{
				"grant_type": req.GrantType,
				"scope":      oauthResp.Scope,
			},
		})
	}

	return tokenResp, nil
}

// ConvertTokenToTokenData maps a *token.Token to a *auth.TokenData for compatibility
func ConvertTokenToTokenData(t *token.Token) *TokenData {
	if t == nil {
		return nil
	}
	return &TokenData{
		Subject:   t.Subject,
		Issuer:    t.Issuer,
		Audience:  "", // Not directly available
		IssuedAt:  t.IssuedAt,
		ExpiresAt: t.ExpiresAt,
		Scope:     t.Scopes,
		Claims:    Claims{}, // Not available on Token
	}
}
