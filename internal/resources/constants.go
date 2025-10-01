package resources

// Rate Limit Events
const (
	EventRateLimitExceeded = "rate_limit.exceeded"
	EventRateLimitReset    = "rate_limit.reset"
)

// Authentication Events
const (
	EventAuthSuccess  = "auth.success"
	EventAuthFailure  = "auth.failure"
	EventTokenIssued  = "token.issued"
	EventTokenRevoked = "token.revoked"
	EventTokenExpired = "token.expired"
)

// Authorization Events
const (
	EventAccessGranted  = "access.granted"
	EventAccessDenied   = "access.denied"
	EventScopeViolation = "scope.violation"
)

// Service Events
const (
	EventServiceStarted = "service.started"
	EventServiceStopped = "service.stopped"
	EventCircuitOpen    = "circuit.open"
	EventCircuitClosed  = "circuit.closed"
)

// Error Messages
const (
	ErrMsgInvalidToken        = "Token is invalid or expired"
	ErrMsgRateLimitExceeded   = "Rate limit exceeded. Please try again in %s"
	ErrMsgUnauthorized        = "Unauthorized access"
	ErrMsgInsufficientScope   = "Insufficient scope for this operation"
	ErrMsgInvalidClientID     = "Invalid client ID"
	ErrMsgInvalidClientSecret = "Invalid client secret"
	ErrMsgServiceUnavailable  = "Service temporarily unavailable"
	ErrMsgInvalidGrantType    = "Invalid or unsupported grant type"
	ErrMsgInvalidRequest      = "Invalid request parameters"
)

// Success Messages
const (
	MsgTokenIssued    = "Token successfully issued"
	MsgTokenRevoked   = "Token successfully revoked"
	MsgTokenValidated = "Token successfully validated"
	MsgAccessGranted  = "Access granted"
	MsgCircuitClosed  = "Circuit breaker closed"
	MsgRateLimitReset = "Rate limit reset"
)
