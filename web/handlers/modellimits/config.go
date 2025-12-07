package modellimits

import "encoding/json"

// ModelLimitsRaw mirrors the on-disk JSON schema for model limits including per-user overrides.
// Exported for visibility in handler.
type ModelLimitsRaw struct {
	ModelLimits map[string]struct {
		MaxInputTokens       int      `json:"max_input_tokens"`
		MaxOutputTokens      int      `json:"max_output_tokens"`
		MaxRequestsPerMinute int      `json:"max_requests_per_minute"` // legacy per-minute limit (backward compat)
		RateLimitsExtended   []string `json:"rate_limits_extended"`    // multi-period limits ("5000/hour", "100K/day")
	} `json:"model_limits"`
	UserLimits map[string]map[string]struct {
		MaxInputTokens       int      `json:"max_input_tokens"`
		MaxOutputTokens      int      `json:"max_output_tokens"`
		MaxRequestsPerMinute int      `json:"max_requests_per_minute"` // legacy per-minute limit
		RateLimitsExtended   []string `json:"rate_limits_extended"`    // multi-period limits
	} `json:"user_limits"`
}

// ParseModelLimitsJSON parses raw JSON bytes into ModelLimitsRaw. It is resilient: on JSON unmarshal
// error it returns a zero-value struct and the error for caller decision. Intended for use by fuzz tests.
func ParseModelLimitsJSON(b []byte) (ModelLimitsRaw, error) {
	var raw ModelLimitsRaw
	if err := json.Unmarshal(b, &raw); err != nil {
		return ModelLimitsRaw{}, err
	}
	return raw, nil
}
