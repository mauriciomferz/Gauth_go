package resources

// Messages contains all user-facing messages
var Messages = struct {
	Errors struct {
		InvalidToken          string
		RateLimitExceeded     string
		InvalidAuthentication string
		InsufficientScope     string
		CircuitBreakerOpen    string
		InvalidGrantType      string
		MissingClientID       string
		InvalidClientSecret   string
		ServiceUnavailable    string
	}
	Info struct {
		TokenIssued          string
		TokenRevoked         string
		CircuitBreakerClosed string
		RateLimitReset       string
	}
}{
	Errors: struct {
		InvalidToken          string
		RateLimitExceeded     string
		InvalidAuthentication string
		InsufficientScope     string
		CircuitBreakerOpen    string
		InvalidGrantType      string
		MissingClientID       string
		InvalidClientSecret   string
		ServiceUnavailable    string
	}{
		InvalidToken:          "Token is invalid or expired",
		RateLimitExceeded:     "Rate limit exceeded. Please try again later",
		InvalidAuthentication: "Invalid authentication credentials",
		InsufficientScope:     "Insufficient scope for this operation",
		CircuitBreakerOpen:    "Service temporarily unavailable due to high error rate",
		InvalidGrantType:      "Invalid or unsupported grant type",
		MissingClientID:       "Client ID is required",
		InvalidClientSecret:   "Invalid client secret",
		ServiceUnavailable:    "Service is temporarily unavailable",
	},
	Info: struct {
		TokenIssued          string
		TokenRevoked         string
		CircuitBreakerClosed string
		RateLimitReset       string
	}{
		TokenIssued:          "Token successfully issued",
		TokenRevoked:         "Token successfully revoked",
		CircuitBreakerClosed: "Service has recovered and is accepting requests",
		RateLimitReset:       "Rate limit window has reset",
	},
}

// ErrorCodes defines error codes for API responses
var ErrorCodes = struct {
	InvalidToken          string
	RateLimitExceeded     string
	InvalidAuthentication string
	InsufficientScope     string
	CircuitBreakerOpen    string
	InvalidGrantType      string
	MissingClientID       string
	InvalidClientSecret   string
	ServiceUnavailable    string
}{
	InvalidToken:          "invalid_token",
	RateLimitExceeded:     "rate_limit_exceeded",
	InvalidAuthentication: "invalid_authentication",
	InsufficientScope:     "insufficient_scope",
	CircuitBreakerOpen:    "circuit_breaker_open",
	InvalidGrantType:      "invalid_grant_type",
	MissingClientID:       "missing_client_id",
	InvalidClientSecret:   "invalid_client_secret",
	ServiceUnavailable:    "service_unavailable",
}
