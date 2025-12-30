package agentauth

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// AdvancedClaims extends the standard JWT claims with additional metadata and semantic validation
type AdvancedClaims struct {
	// Standard JWT claims
	Subject   string   `json:"sub"`
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	NotBefore int64    `json:"nbf"`
	JWTID     string   `json:"jti"`

	// Extended claims
	Scope []string `json:"scope,omitempty"`

	// Advanced claims metadata
	TokenType string `json:"typ,omitempty"`       // Token type semantic enforcement
	ClientID  string `json:"client_id,omitempty"` // Client identification

	// Claims set metadata
	ClaimsMetadata *ClaimsMetadata `json:"claims_meta,omitempty"`

	// Custom claims for extensibility
	Custom map[string]interface{} `json:"-"`
}

// ClaimsMetadata provides structured metadata about the claims set
type ClaimsMetadata struct {
	Version      string              `json:"version"`      // Claims schema version
	Capabilities []string            `json:"capabilities"` // Supported capabilities
	Restrictions *ClaimsRestrictions `json:"restrictions,omitempty"`
	Source       string              `json:"source"`     // Claims source (internal, external, etc.)
	Confidence   float64             `json:"confidence"` // Confidence level (0.0-1.0)
}

// ClaimsRestrictions defines usage restrictions
type ClaimsRestrictions struct {
	IPWhitelist    []string    `json:"ip_whitelist,omitempty"`
	TimeWindow     *TimeWindow `json:"time_window,omitempty"`
	UsageLimit     int         `json:"usage_limit,omitempty"`
	GeofenceRegion string      `json:"geofence_region,omitempty"`
}

// TimeWindow defines valid time constraints
type TimeWindow struct {
	StartHour int   `json:"start_hour"` // 0-23
	EndHour   int   `json:"end_hour"`   // 0-23
	Weekdays  []int `json:"weekdays"`   // 0=Sunday, 1=Monday, etc.
}

// PASETOFooter represents structured PASETO footer for advanced use cases
type PASETOFooter struct {
	KeyID     string                 `json:"kid"`
	Algorithm string                 `json:"alg"`
	Issuer    string                 `json:"iss"`
	Metadata  map[string]interface{} `json:"meta,omitempty"`
	Signature string                 `json:"sig,omitempty"` // Optional footer signature
}

// ValidateSemantics performs semantic validation on the claims
func (ac *AdvancedClaims) ValidateSemantics() error {
	now := time.Now().Unix()

	// Temporal validation
	if ac.ExpiresAt > 0 && now > ac.ExpiresAt {
		return fmt.Errorf("token expired at %d, current time %d", ac.ExpiresAt, now)
	}

	if ac.NotBefore > 0 && now < ac.NotBefore {
		return fmt.Errorf("token not valid before %d, current time %d", ac.NotBefore, now)
	}

	if ac.IssuedAt > now+300 { // Allow 5-minute clock skew
		return fmt.Errorf("token issued in future: %d > %d", ac.IssuedAt, now)
	}

	// Token type validation
	if ac.TokenType != "" && !isValidTokenType(ac.TokenType) {
		return fmt.Errorf("invalid token type: %s", ac.TokenType)
	}

	// Audience validation
	if len(ac.Audience) == 0 {
		return fmt.Errorf("audience (aud) claim is required")
	}

	// Subject validation
	if ac.Subject == "" {
		return fmt.Errorf("subject (sub) claim is required")
	}

	// Metadata validation
	if ac.ClaimsMetadata != nil {
		if err := ac.ClaimsMetadata.Validate(); err != nil {
			return fmt.Errorf("claims metadata validation failed: %w", err)
		}
	}

	return nil
}

// Validate performs comprehensive validation on claims metadata
func (cm *ClaimsMetadata) Validate() error {
	if cm.Version == "" {
		return fmt.Errorf("claims metadata version is required")
	}

	if cm.Confidence < 0.0 || cm.Confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got %f", cm.Confidence)
	}

	// Validate restrictions if present
	if cm.Restrictions != nil {
		if err := cm.Restrictions.Validate(); err != nil {
			return fmt.Errorf("restrictions validation failed: %w", err)
		}
	}

	return nil
}

// Validate validates claims restrictions
func (cr *ClaimsRestrictions) Validate() error {
	// Validate time window
	if cr.TimeWindow != nil {
		if cr.TimeWindow.StartHour < 0 || cr.TimeWindow.StartHour > 23 {
			return fmt.Errorf("invalid start_hour: %d", cr.TimeWindow.StartHour)
		}
		if cr.TimeWindow.EndHour < 0 || cr.TimeWindow.EndHour > 23 {
			return fmt.Errorf("invalid end_hour: %d", cr.TimeWindow.EndHour)
		}
		for _, day := range cr.TimeWindow.Weekdays {
			if day < 0 || day > 6 {
				return fmt.Errorf("invalid weekday: %d", day)
			}
		}
	}

	// Validate usage limit
	if cr.UsageLimit < 0 {
		return fmt.Errorf("usage_limit cannot be negative: %d", cr.UsageLimit)
	}

	return nil
}

// IsInTimeWindow checks if current time falls within the allowed time window
func (cr *ClaimsRestrictions) IsInTimeWindow() bool {
	if cr.TimeWindow == nil {
		return true // No restriction
	}

	now := time.Now()
	hour := now.Hour()
	weekday := int(now.Weekday())

	// Check hour range
	if hour < cr.TimeWindow.StartHour || hour > cr.TimeWindow.EndHour {
		return false
	}

	// Check weekdays if specified
	if len(cr.TimeWindow.Weekdays) > 0 {
		allowed := false
		for _, day := range cr.TimeWindow.Weekdays {
			if day == weekday {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

// ToMap converts AdvancedClaims to a map for JSON encoding
func (ac *AdvancedClaims) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Use reflection to populate standard fields
	v := reflect.ValueOf(ac).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if field.Name == "Custom" {
			continue // Handle custom fields separately
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract field name from json tag
		fieldName := jsonTag
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			fieldName = jsonTag[:idx]
		}

		// Skip empty optional fields
		if strings.Contains(jsonTag, "omitempty") && value.IsZero() {
			continue
		}

		result[fieldName] = value.Interface()
	}

	// Add custom fields
	for k, v := range ac.Custom {
		result[k] = v
	}

	return result
}

// FromMap populates AdvancedClaims from a map
func (ac *AdvancedClaims) FromMap(data map[string]interface{}) error {
	// Handle scope conversion from string to []string before unmarshaling
	if scopeVal, exists := data["scope"]; exists {
		if scopeStr, ok := scopeVal.(string); ok && scopeStr != "" {
			data["scope"] = strings.Split(scopeStr, " ")
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := json.Unmarshal(jsonData, ac); err != nil {
		return fmt.Errorf("failed to unmarshal into AdvancedClaims: %w", err)
	}

	// Handle custom fields
	ac.Custom = make(map[string]interface{})
	standardFields := map[string]bool{
		"sub": true, "iss": true, "aud": true, "exp": true, "iat": true,
		"nbf": true, "jti": true, "scope": true, "typ": true, "client_id": true,
		"claims_meta": true,
	}

	for k, v := range data {
		if !standardFields[k] {
			ac.Custom[k] = v
		}
	}

	return nil
}

// CreatePASETOFooter creates a structured PASETO footer
func CreatePASETOFooter(keyID, algorithm, issuer string, metadata map[string]interface{}) (*PASETOFooter, error) {
	footer := &PASETOFooter{
		KeyID:     keyID,
		Algorithm: algorithm,
		Issuer:    issuer,
		Metadata:  metadata,
	}

	return footer, nil
}

// ToJSON converts PASETOFooter to JSON string
func (pf *PASETOFooter) ToJSON() (string, error) {
	data, err := json.Marshal(pf)
	if err != nil {
		return "", fmt.Errorf("failed to marshal PASETO footer: %w", err)
	}
	return string(data), nil
}

// isValidTokenType checks if the token type is valid according to RFC standards
// and AgentAuth-specific types (P2.10 sec1.item2).
func isValidTokenType(tokenType string) bool {
	validTypes := map[string]bool{
		// Standard JWT/PASETO types (RFC 7519)
		"JWT":           true,
		"PASETO":        true,
		"access_token":  true,
		"refresh_token": true,
		"id_token":      true,
		"at+jwt":        true,
		"rt+jwt":        true,
		// AgentAuth-specific types (P2.10 sec1.item2)
		"agentauth.delegation": true,
		"agentauth.token":      true,
		"agentauth.capability": true,
	}
	return validTypes[tokenType]
}

// Example usage and demonstration
func ExampleAdvancedClaims() *AdvancedClaims {
	return &AdvancedClaims{
		Subject:   "user123",
		Issuer:    "https://auth.example.com",
		Audience:  []string{"api.example.com", "web.example.com"},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
		NotBefore: time.Now().Unix(),
		JWTID:     "unique-jwt-id-123",
		Scope:     []string{"read", "write", "admin"},
		TokenType: "JWT",
		ClientID:  "client-app-123",
		ClaimsMetadata: &ClaimsMetadata{
			Version:      "1.0",
			Capabilities: []string{"delegation", "revocation"},
			Source:       "internal",
			Confidence:   0.95,
			Restrictions: &ClaimsRestrictions{
				TimeWindow: &TimeWindow{
					StartHour: 9,                    // 9 AM
					EndHour:   17,                   // 5 PM
					Weekdays:  []int{1, 2, 3, 4, 5}, // Monday-Friday
				},
				UsageLimit: 100,
			},
		},
		Custom: map[string]interface{}{
			"tenant_id":   "tenant-123",
			"session_id":  "session-456",
			"device_info": "mobile-app/1.0",
		},
	}
}
