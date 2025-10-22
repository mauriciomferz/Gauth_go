package web

import "encoding/json"

// modelLimitsRaw mirrors the on-disk JSON schema for model limits including per-user overrides.
type modelLimitsRaw struct {
	ModelLimits map[string]struct {
		MaxInputTokens       int `json:"max_input_tokens"`
		MaxOutputTokens      int `json:"max_output_tokens"`
		MaxRequestsPerMinute int `json:"max_requests_per_minute"`
	} `json:"model_limits"`
	UserLimits map[string]map[string]struct {
		MaxInputTokens       int `json:"max_input_tokens"`
		MaxOutputTokens      int `json:"max_output_tokens"`
		MaxRequestsPerMinute int `json:"max_requests_per_minute"`
	} `json:"user_limits"`
}

// parseModelLimitsJSON parses raw JSON bytes into modelLimitsRaw. It is resilient: on JSON unmarshal
// error it returns a zero-value struct and the error for caller decision. Intended for use by fuzz tests.
func parseModelLimitsJSON(b []byte) (modelLimitsRaw, error) {
	var raw modelLimitsRaw
	if err := json.Unmarshal(b, &raw); err != nil {
		return modelLimitsRaw{}, err
	}
	return raw, nil
}
