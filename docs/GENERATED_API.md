# Generated API Surface
_Auto-generated on 2025-10-12T23:29:27Z – experimental_

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/cmd/demo
```go

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/cmd/gauth-server
```go
Package main provides a demonstration of the GAuth protocol implementation

This demo shows the basic flow of the GAuth protocol, including: - Authorization
request and grant - Token issuance - Transaction processing - Audit logging -
Token expiration
```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/cmd/security-test
```go

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/cmd/web-server
```go

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit
```go
package audit // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"


CONSTANTS

const (
	TypeAuth     = "auth"
	TypeToken    = "token"
	TypeResource = "resource"

	// Actor types
	ActorUser    = "user"
	ActorService = "service"
	ActorSystem  = "system"

	ActionLogin          = "login"
	ActionLogout         = "logout"
	ActionResourceAccess = "resource_access"

	// Result types
	ResultSuccess = "success"
	ResultFailure = "failure"
)

TYPES

type BenchmarkMetrics struct{}
    --- BENCHMARK METRICS STRICT COMPATIBILITY PATCH ---

func NewMetrics(_ string) *BenchmarkMetrics

func (m *BenchmarkMetrics) ObserveEntry(...interface{})

func (m *BenchmarkMetrics) ObserveStorageOperation(...interface{})

type Entry struct {
}
    Entry is a stub for compatibility

type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Subject   string                 `json:"subject,omitempty"`
	Object    string                 `json:"object,omitempty"`
	Action    string                 `json:"action"`
	Result    string                 `json:"result"`
	ClientID  string                 `json:"client_id,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Severity  string                 `json:"severity"`
}
    Event represents an audit event

func NewEvent(eventType EventType, action, result string) *Event

type EventType string
    EventType represents different types of audit events

const (
	EventTypeAuthentication EventType = "authentication"
	EventTypeAuthorization  EventType = "authorization"
	EventTypeTokenIssue     EventType = "token_issue"
	EventTypeTokenRevoke    EventType = "token_revoke"
	EventTypeResourceAccess EventType = "resource_access"
	EventTypeError          EventType = "error"
)
type FileConfig struct {
	Directory string
}

type FileStorage struct{}
    FileStorage is a stub for benchmark compatibility Store and Close methods
    are also stubbed (Add similar for RedisStorage and SQLStorage)

func NewFileStorage(cfg FileConfig) (*FileStorage, error)

func (fs *FileStorage) Close() error

func (fs *FileStorage) Store(ctx context.Context, entry *Entry) error

type Filter struct {
	EventTypes []EventType `json:"event_types,omitempty"`
	Subject    string      `json:"subject,omitempty"`
	StartTime  *time.Time  `json:"start_time,omitempty"`
	EndTime    *time.Time  `json:"end_time,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
}
    Filter represents query filters for audit events

type MemoryLogger struct {
	// Has unexported fields.
}
    MemoryLogger implements audit logging in memory

func NewAuditLogger() *MemoryLogger
    NewAuditLogger creates a new audit logger NewAuditLogger returns a new
    MemoryLogger for compatibility

func NewMemoryLogger(logger common.Logger) *MemoryLogger
    NewMemoryLogger creates a new memory-based audit logger

func (ml *MemoryLogger) Close() error
    Close closes the audit logger

func (ml *MemoryLogger) Log(ctx context.Context, entry interface{}) error
    Log logs an audit event

func (ml *MemoryLogger) Query(ctx context.Context, filter *Filter) ([]*Event, error)
    Query queries audit events based on filter

type RedisConfig struct {
	Addr      string
	Addresses []string
	KeyPrefix string
}
    RedisConfig is a stub for benchmark compatibility

type RedisStorage struct{}
    FileStorage is a stub for benchmark compatibility Store and Close methods
    are also stubbed (Add similar for RedisStorage and SQLStorage)

func NewRedisStorage(cfg RedisConfig) (*RedisStorage, error)

func (rs *RedisStorage) Close() error

func (rs *RedisStorage) Search(ctx context.Context, filter *Filter) ([]*Entry, error)

func (rs *RedisStorage) Store(ctx context.Context, entry *Entry) error

type SQLConfig struct {
	Driver string
	DSN    string
}
    SQLConfig is a stub for benchmark compatibility

type SQLStorage struct{}
    FileStorage is a stub for benchmark compatibility Store and Close methods
    are also stubbed (Add similar for RedisStorage and SQLStorage)

func NewSQLStorage(cfg SQLConfig) (*SQLStorage, error)

func (ss *SQLStorage) Close() error

func (ss *SQLStorage) Search(ctx context.Context, filter *Filter) ([]*Entry, error)

func (ss *SQLStorage) Store(ctx context.Context, entry *Entry) error

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/auth
```go
package auth // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/auth"


CONSTANTS

const (
	PrincipalTypeOrganization = "Organization"
	OrgTypeCommercial         = "Commercial"
	ClientAI                  = "AI"
	ClientTypeLLM             = "LLM"
	ClientTypeAgenticAI       = "AgenticAI"
	AuthorizationType         = "Authorization"

	DepositTransaction   = gauth.DepositTransaction
	TransactionPending   = gauth.TransactionPending
	TransactionCompleted = gauth.TransactionCompleted
	TransactionFailed    = gauth.TransactionFailed
	TransactionCancelled = gauth.TransactionCancelled
)

VARIABLES

var (
	ErrInvalidJurisdiction = fmt.Errorf("invalid jurisdiction")
	ErrDisallowedScope     = fmt.Errorf("disallowed scope capability")
	ErrMissingFields       = fmt.Errorf("missing required power-of-attorney fields")
)
    --- Structured error types for POA validation ---

var (
	ErrInvalidToken  = gauth.ErrInvalidToken
	ErrUnauthorized  = gauth.ErrUnauthorized
	ErrTokenExpired  = gauth.ErrTokenExpired
	ErrInvalidGrant  = gauth.ErrInvalidGrant
	ErrInvalidClient = gauth.ErrInvalidClient
)
    Re-export variables

var (
	New                         = gauth.New
	NewResourceServer           = gauth.NewResourceServer
	NewPowerAdministrationPoint = gauth.NewPowerAdministrationPoint
)
    Re-export functions


FUNCTIONS

func ValidateToken(token string) (interface{}, error)
    ValidateToken validates an authentication token This is a simple
    implementation that always returns success for demo purposes


TYPES

type AttestationRequirement struct {
	Attesters []string
}

type AuthResult struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	Scope        string    `json:"scope,omitempty"`
	Subject      string    `json:"subject"`
	IssuedAt     time.Time `json:"issued_at"`
}
    AuthResult represents the result of an authentication operation

type Authenticator interface {
	Authenticate(ctx context.Context, credentials *Credentials) (*AuthResult, error)
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error)
}
    Authenticator defines the interface for authentication operations

type AuthorizedRepresentative struct {
	Name                string
	RegisteredAuthority string
	RegisterEntry       string
	AuthorityType       string
}

type Authorizer struct {
	ClientOwner string
}

type Claims struct {
	UserID    string         `json:"user_id"`
	SessionID string         `json:"session_id"`
	Scopes    []string       `json:"scopes"`
	ExpiresAt ExpirationTime `json:"exp"`
	IssuedAt  int64          `json:"iat"`
	Issuer    string         `json:"iss"`
	Audience  string         `json:"aud"`
}
    Claims represents JWT token claims

type Credentials struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	GrantType    string `json:"grant_type,omitempty"`
	Scope        string `json:"scope,omitempty"`
}
    Credentials represents user credentials

type DelegationRequest struct {
	PrincipalID            string
	DelegateID             string
	ValidityPeriod         ValidityPeriod
	AttestationRequirement AttestationRequirement
}

type DelegationResponse struct {
	DelegationID     string
	Status           string
	ValidUntil       time.Time
	Attestations     []string
	ComplianceStatus string
}

type ExpirationTime struct {
	Time time.Time
}
    ExpirationTime wraps int64 with Time method for compatibility

func (et ExpirationTime) Unix() int64
    Unix returns the Unix timestamp

type JWTService struct {
	// Has unexported fields.
}
    JWTService represents a JWT service for token operations

func NewProperJWTService(issuer, audience string) (*JWTService, error)
    NewProperJWTService creates a new JWT service

func (j *JWTService) CreateToken(userID string, scopes []string, duration time.Duration) (string, error)
    CreateToken creates a new JWT token

func (j *JWTService) ValidateToken(token string) (*Claims, error)
    ValidateToken validates a JWT token and returns claims

type Organization struct {
	Type                string
	Name                string
	RegisterEntry       string
	ManagingDirector    string
	RegisteredAuthority string
}

type PoADefinition struct {
	Principal Principal
	Client    string
}

type PowerOfAttorneyRequest struct {
	ClientID     string
	ResponseType string
	Scope        string
	RedirectURI  string
	State        string
	PowerType    string
	PrincipalID  string
	AIAgentID    string
	Jurisdiction string
	LegalBasis   string
}

type PowerOfAttorneyResponse struct {
	AuthorizationCode string
	LegalCompliance   bool
	AuditRecordID     string
}
    RFC Functional Test compatibility stubs These types and methods are required
    for the rfc_functional_test example to build.

type PowerRestrictions struct{}

type Principal struct {
	Type         string
	Identity     string
	Organization Organization
}

type ProfessionalAuthService struct{}
    ProfessionalAuthService stub for demo

func NewProfessionalAuthService(config ProfessionalConfig) (*ProfessionalAuthService, error)

func (s *ProfessionalAuthService) CreateToken(userID string, scopes []string, expiry time.Duration) (string, error)

func (s *ProfessionalAuthService) ValidateToken(token string) (*ProfessionalClaims, error)

type ProfessionalClaims struct {
	UserID    string
	Scopes    []string
	ExpiresAt time.Time
}

type ProfessionalConfig struct {
	Issuer            string
	Audience          string
	TokenExpiry       time.Duration
	ServiceID         string
	MeshID            string
	UseSecureDefaults bool
}
    ProfessionalConfig for professional interface demo

type SimpleAuthenticator struct {
	// Has unexported fields.
}
    SimpleAuthenticator is a basic implementation of Authenticator

func NewAuthenticator(tokenService *token.Service) *SimpleAuthenticator
    NewAuthenticator creates a new authenticator

func NewRFCCompliantService() *SimpleAuthenticator

func (a *SimpleAuthenticator) Authenticate(ctx context.Context, credentials *Credentials) (*AuthResult, error)
    Authenticate authenticates user credentials

func (a *SimpleAuthenticator) AuthorizePowerOfAttorney(ctx context.Context, req PowerOfAttorneyRequest) (*PowerOfAttorneyResponse, error)

func (a *SimpleAuthenticator) CreateAdvancedDelegation(ctx context.Context, req DelegationRequest) (*DelegationResponse, error)

func (a *SimpleAuthenticator) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error)
    RefreshToken refreshes an access token using a refresh token

func (a *SimpleAuthenticator) ValidateToken(ctx context.Context, tokenStr string) (*TokenClaims, error)
    ValidateToken validates a token and returns its claims

type TimeWindow struct{}

type TokenClaims struct {
	Subject   string    `json:"sub"`
	Issuer    string    `json:"iss"`
	Audience  string    `json:"aud"`
	ExpiresAt time.Time `json:"exp"`
	IssuedAt  time.Time `json:"iat"`
	NotBefore time.Time `json:"nbf"`
	Scope     string    `json:"scope,omitempty"`
	KeyID     string    `json:"kid,omitempty"`
}
    TokenClaims represents the claims in a token

type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Password string   `json:"password"` // In real implementation, this should be hashed
	Roles    []string `json:"roles"`
	Active   bool     `json:"active"`
}
    User represents a user in the system

type ValidityPeriod struct {
	Days int
}

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz
```go
package authz // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"


TYPES

type Action struct{ Name string }

type BasicEnforcer struct {
	// Has unexported fields.
}
    BasicEnforcer minimal structure for backward compatibility

func NewBasicEnforcer() *BasicEnforcer
    NewBasicEnforcer creates a new BasicEnforcer

func (e *BasicEnforcer) AddPolicy(ctx context.Context, policy *Policy) error
    AddPolicy adds a new policy

func (e *BasicEnforcer) Authorize(ctx context.Context, reqOrSubject interface{}, actionOrNil ...interface{}) (*Decision, error)
    Authorize authorizes a request (compatibility method)

func (e *BasicEnforcer) AuthorizeWithParams(ctx context.Context, subject Subject, action Action, resource Resource) (*Decision, error)
    AuthorizeWithParams is an alias for Authorize for compatibility

func (e *BasicEnforcer) Evaluate(ctx context.Context, req *Request) (*Decision, error)
    Evaluate evaluates a request (simple matching)

func (e *BasicEnforcer) RemovePolicy(ctx context.Context, policyID string) error
    RemovePolicy removes a policy

type Condition struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}
    Condition represents a policy condition

type Decision struct {
	Allow    bool              `json:"allow"`
	Allowed  bool              `json:"allowed"` // Compatibility field
	Policies []string          `json:"policies,omitempty"`
	Reason   string            `json:"reason,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
    Decision represents an authorization decision

type Effect string
    Effect represents the effect of a policy

const (
	// Allow grants access
	Allow Effect = "allow"
	// Deny denies access
	Deny Effect = "deny"
)
type MemoryAuthorizer struct {
	// Has unexported fields.
}
    MemoryAuthorizer implements Authorizer using in-memory policies

func NewMemoryAuthorizer() *MemoryAuthorizer
    NewMemoryAuthorizer creates a new in-memory authorizer

func (ma *MemoryAuthorizer) AddPolicy(policy Policy)
    AddPolicy adds a policy to the authorizer

func (ma *MemoryAuthorizer) Authorize(ctx context.Context, request Request) (Decision, error)
    Authorize makes an authorization decision

func (ma *MemoryAuthorizer) GetPermissions(ctx context.Context, subject string) ([]Permission, error)
    GetPermissions retrieves permissions for a subject

type Permission struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
	Granted  bool     `json:"granted"`
}
    Permission represents a permission

type Policy struct {
	ID         string            `json:"id"`
	Subject    string            `json:"subject"`
	Resource   string            `json:"resource"`
	Actions    []string          `json:"actions"`
	Effect     Effect            `json:"effect"`
	Conditions []Condition       `json:"conditions,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}
    Policy represents an authorization policy

type Request struct {
	Subject  string            `json:"subject"`
	Resource string            `json:"resource"`
	Action   string            `json:"action"`
	Context  map[string]string `json:"context,omitempty"`
}
    Request represents an authorization request

type Resource struct{ ID string }

type Subject struct{ ID string }
    Subject, Action, Resource simple legacy compatibility wrappers

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/circuit
```go
package circuit // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/circuit"


TYPES

type Breaker struct {
	// Has unexported fields.
}
    Breaker represents a circuit breaker

func NewBreaker(opts Options) *Breaker
    NewBreaker creates a new circuit breaker with the given options

func (cb *Breaker) Counts() Counts
    Counts returns the current counts

func (cb *Breaker) Execute(fn func() error) error
    Execute executes the given function within the circuit breaker

func (cb *Breaker) ExecuteContext(ctx context.Context, fn func() error) error
    ExecuteContext executes the given function within the circuit breaker with
    context

func (cb *Breaker) State() State
    State returns the current state of the circuit breaker

type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}
    Counts holds the statistics for a circuit breaker

type Options struct {
	MaxFailures      int           `json:"max_failures"`
	FailureThreshold int           `json:"failure_threshold"`
	Timeout          time.Duration `json:"timeout"`
	ResetTimeout     time.Duration `json:"reset_timeout"`
	ReadyToTrip      func(counts Counts) bool
	OnStateChange    func(name string, from State, to State)
	MaxRequests      uint32        `json:"max_requests"`
	HalfOpenLimit    int           `json:"half_open_limit"`
	Interval         time.Duration `json:"interval"`
}
    Options configures a circuit breaker

type State int
    State represents the circuit breaker state

const (
	Closed State = iota
	Open
	HalfOpen
)
```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/common
```go
package common // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/common"

Package common provides common types and utilities for the GAuth implementation

TYPES

type Config struct {
	Debug       bool          `json:"debug" yaml:"debug"`
	LogLevel    string        `json:"log_level" yaml:"log_level"`
	Environment string        `json:"environment" yaml:"environment"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
}
    Config represents common configuration options

func DefaultConfig() *Config
    DefaultConfig returns a default configuration

type EventType string
    EventType represents different types of events in the GAuth system

const (
	// TokenCreated represents a token creation event
	TokenCreated EventType = "token.created"
	// TokenRevoked represents a token revocation event
	TokenRevoked EventType = "token.revoked"
	// TokenValidated represents a token validation event
	TokenValidated EventType = "token.validated"
	// AuthorizationGranted represents an authorization grant event
	AuthorizationGranted EventType = "authorization.granted"
	// AuthorizationDenied represents an authorization denial event
	AuthorizationDenied EventType = "authorization.denied"

	// Delegation events (RFC 0111)
	DelegationCreated   EventType = "delegation.created"
	DelegationRevoked   EventType = "delegation.revoked"
	DelegationValidated EventType = "delegation.validated"
)
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}
    Logger interface for common logging

type RateLimitConfig struct {
	RequestsPerSecond int `json:"requests_per_second" yaml:"requests_per_second"`
	BurstSize         int `json:"burst_size" yaml:"burst_size"`
	WindowSize        int `json:"window_size" yaml:"window_size"`
}
    RateLimitConfig represents rate limiting configuration

func DefaultRateLimitConfig() RateLimitConfig
    DefaultRateLimitConfig returns the default rate limiting configuration

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
    Response represents a common response structure

func NewErrorResponse(err error, message string) *Response
    NewErrorResponse creates an error response

func NewSuccessResponse(data interface{}, message string) *Response
    NewSuccessResponse creates a successful response

type SimpleLogger struct{}
    SimpleLogger is a basic logger implementation

func NewSimpleLogger() *SimpleLogger
    NewSimpleLogger creates a new simple logger

func (l *SimpleLogger) Debug(args ...interface{})

func (l *SimpleLogger) Debugf(format string, args ...interface{})

func (l *SimpleLogger) Error(args ...interface{})

func (l *SimpleLogger) Errorf(format string, args ...interface{})

func (l *SimpleLogger) Info(args ...interface{})

func (l *SimpleLogger) Infof(format string, args ...interface{})

func (l *SimpleLogger) Warn(args ...interface{})

func (l *SimpleLogger) Warnf(format string, args ...interface{})

type Utils struct{}
    Utils contains common utility functions

func NewUtils() *Utils
    NewUtils creates a new utils instance

func (u *Utils) GenerateID() string
    GenerateID generates a simple ID (placeholder implementation)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
    ValidationError represents a validation error

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Reason string            `json:"reason,omitempty"`
	Errors []ValidationError `json:"errors,omitempty"`
}
    ValidationResult represents the result of a validation operation

func (vr *ValidationResult) AddError(field, message string)
    AddError adds a validation error to the result

func (vr ValidationResult) IsValid() bool
    IsValid returns true if the validation result is valid

type Validator interface {
	Validate(interface{}) error
}
    Validator interface for common validation

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/errors
```go
package errors // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/errors"


VARIABLES

var (
	ErrUnauthenticated = New(ErrCodeUnauthenticated, "authentication required")
	ErrUnauthorized    = New(ErrCodeUnauthorized, "insufficient permissions")
	ErrNotFound        = New(ErrCodeNotFound, "resource not found")
	ErrInternal        = New(ErrCodeInternal, "internal server error")
	ErrRateLimit       = New(ErrCodeRateLimit, "rate limit exceeded")
	ErrTimeout         = New(ErrCodeTimeout, "operation timed out")
	// Legacy/example exported errors for compatibility
	ErrInvalidToken      = New(ErrCodeInvalidToken, "invalid token")
	ErrInsufficientScope = New(ErrCodeInsufficientScope, "insufficient scope")
	ErrRateLimited       = ErrRateLimit
)
    Additional predefined errors for common cases


FUNCTIONS

func GetRetryAfter(err error) time.Duration
    GetRetryAfter extracts retry-after duration from error details

func IsCode(err error, code ErrorCode) bool
    IsCode checks if the error has a specific code

func IsRateLimitError(err error) bool
    IsRateLimitError checks if an error is a rate limit error

func Middleware() func(http.Handler) http.Handler
    Middleware creates an error handling middleware function


TYPES

type Error = GAuthError
    Error is a compatibility type alias expected by legacy examples.

type ErrorCode string
    ErrorCode represents different types of errors

const (
	// Source constants for compatibility with examples
	SourceToken         = "token"
	SourceAuthorization = "authorization"
	SourceRateLimiting  = "rate_limiting"
	SourceStorage       = "storage" // legacy example expects this
	// Authentication errors
	ErrCodeUnauthenticated ErrorCode = "UNAUTHENTICATED"
	ErrCodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	ErrCodeInvalidToken    ErrorCode = "INVALID_TOKEN"
	ErrCodeExpiredToken    ErrorCode = "EXPIRED_TOKEN"
	// Legacy alias expectations
	ErrServerError  ErrorCode = "INTERNAL_ERROR" // alias used in examples
	ErrTokenExpired ErrorCode = "EXPIRED_TOKEN"  // alias used in examples

	// Validation errors
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrCodeMissingField   ErrorCode = "MISSING_FIELD"

	// System errors
	ErrCodeInternal  ErrorCode = "INTERNAL_ERROR"
	ErrCodeNotFound  ErrorCode = "NOT_FOUND"
	ErrCodeConflict  ErrorCode = "CONFLICT"
	ErrCodeRateLimit ErrorCode = "RATE_LIMIT"
	ErrCodeTimeout   ErrorCode = "TIMEOUT"

	// Legacy/example specific codes
	ErrCodeInsufficientScope ErrorCode = "INSUFFICIENT_SCOPE"

	// Network errors
	ErrCodeNetworkError ErrorCode = "NETWORK_ERROR"
	ErrCodeServiceDown  ErrorCode = "SERVICE_DOWN"
)
func GetCode(err error) ErrorCode
    GetCode extracts the error code from an error

type ErrorDetails struct {
	Timestamp      time.Time              `json:"timestamp"`
	AdditionalInfo map[string]interface{} `json:"additional_info,omitempty"`
	RequestID      string                 `json:"request_id,omitempty"`
	UserID         string                 `json:"user_id,omitempty"`
	ClientID       string                 `json:"client_id,omitempty"`
	HTTPMethod     string                 `json:"http_method,omitempty"`
	HTTPPath       string                 `json:"http_path,omitempty"`
	HTTPStatusCode int                    `json:"http_status_code,omitempty"` // legacy compatibility
}
    ErrorDetails represents structured error details

type GAuthError struct {
	Code      ErrorCode     `json:"code"`
	Message   string        `json:"message"`
	Details   *ErrorDetails `json:"details,omitempty"`
	Source    string        `json:"source,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	File      string        `json:"file,omitempty"`
	Line      int           `json:"line,omitempty"`
}
    GAuthError represents a structured error

func New(codeOrError interface{}, message string) *GAuthError
    New creates a new GAuthError

func Newf(code ErrorCode, format string, args ...interface{}) *GAuthError
    Newf creates a new GAuthError with formatted message

func Wrap(err error, code ErrorCode, message string) *GAuthError
    Wrap wraps an existing error with additional context

func (e *GAuthError) AddInfo(k string, v interface{}) *GAuthError

func (e *GAuthError) Error() string
    Error implements the error interface

func (e *GAuthError) WithCause(cause error) *GAuthError
    WithCause adds a cause to the error

func (e *GAuthError) WithDetails(details interface{}) *GAuthError
    WithDetails adds details to the error

func (e *GAuthError) WithFields(fields interface{}) *GAuthError
    WithFields adds fields to the error (accepts both map[string]interface{} and
    map[string]string)

func (e *GAuthError) WithHTTPInfo(parts ...interface{}) *GAuthError

func (e *GAuthError) WithRequestInfo(parts ...interface{}) *GAuthError

func (e *GAuthError) WithSource(source string) *GAuthError
    --- Compatibility chain methods (no-op enrichers used by examples) ---

type MultiError struct {
	Errors []error `json:"errors"`
}
    MultiError holds multiple errors

func NewMultiError() *MultiError
    NewMultiError creates a new multi-error

func (m *MultiError) Add(err error)
    Add adds an error to the multi-error

func (m *MultiError) Error() string
    Error implements the error interface

func (m *MultiError) HasErrors() bool
    HasErrors returns true if there are any errors

type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value,omitempty"`
	Message string      `json:"message"`
}
    ValidationError represents a validation error with field-specific
    information

func NewValidationError(field, message string) *ValidationError
    NewValidationError creates a new validation error

func (v *ValidationError) Error() string
    Error implements the error interface

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/events
```go
package events // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/events"

Package events provides an event system for the GAuth framework

TYPES

type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Subject   string    `json:"subject"`
	Resource  string    `json:"resource"`
	Metadata  *Metadata `json:"metadata"`
	Error     string    `json:"error,omitempty"`
}
    Event represents a system event

type EventBus struct {
	// Has unexported fields.
}
    EventBus manages event distribution

func NewEventBus() *EventBus
    NewEventBus creates a new event bus

func (eb *EventBus) Publish(event Event)
    Publish sends an event to all registered handlers

func (eb *EventBus) Subscribe(handler EventHandler)
    Subscribe adds an event handler to the bus

type EventHandler interface {
	Handle(event Event)
}
    EventHandler defines the interface for event handlers

type EventPublisher struct {
	// Has unexported fields.
}
    EventPublisher manages event publishing and subscription

func NewEventPublisher() *EventPublisher
    NewEventPublisher creates a new event publisher

func (p *EventPublisher) Publish(event Event)
    Publish sends an event to all subscribed handlers

func (p *EventPublisher) Subscribe(handler EventHandler)
    Subscribe adds an event handler

type EventType string
    EventType represents the type of event

const (
	EventTypeToken  EventType = "token"
	EventTypeAuth   EventType = "auth"
	EventTypeSystem EventType = "system"
	EventTypeAudit  EventType = "audit"
)
type Metadata struct {
	// Has unexported fields.
}

func NewMetadata() *Metadata
    NewMetadata creates a new metadata instance

func (m *Metadata) Get(key string) (*Value, bool)
    Get returns the Value and existence

func (m *Metadata) GetBool(key string) (bool, bool)
    GetBool returns a bool value if present

func (m *Metadata) GetFloat(key string) (float64, bool)
    GetFloat returns a float64 value if present

func (m *Metadata) GetInt(key string) (int, bool)
    GetInt returns an int value if present

func (m *Metadata) GetString(key string) (string, bool)
    GetString returns a string value if present

func (m *Metadata) GetTime(key string) (time.Time, bool)
    GetTime returns a time value if present

func (m *Metadata) Has(key string) bool
    Has checks if a key exists

func (m *Metadata) Keys() []string
    Keys returns all metadata keys

func (m *Metadata) SetBool(key string, value bool)
    SetBool sets a boolean value

func (m *Metadata) SetFloat(key string, value float64)
    SetFloat sets a float value

func (m *Metadata) SetInt(key string, value int)
    SetInt sets an integer value

func (m *Metadata) SetString(key, value string)
    SetString sets a string value

func (m *Metadata) SetStringSlice(key string, value []string)
    SetStringSlice sets a slice of strings value

func (m *Metadata) SetTime(key string, value time.Time)
    SetTime sets a time value

func (m *Metadata) Size() int
    Size returns the number of metadata fields

type SimpleDispatcher struct {
	// Has unexported fields.
}
    SimpleDispatcher for event handling by type

func NewSimpleDispatcher() *SimpleDispatcher

func (d *SimpleDispatcher) Dispatch(event Event)

func (d *SimpleDispatcher) RegisterHandler(eventType EventType, handler EventHandler)

type Value struct {
	Type  ValueType
	Value interface{}
}
    Value represents a typed metadata value

type ValueType string
    Metadata provides structured metadata with helper methods ValueType and
    Value struct for typed metadata

const (
	ValueTypeString  ValueType = "string"
	ValueTypeInt     ValueType = "int"
	ValueTypeFloat   ValueType = "float"
	ValueTypeBool    ValueType = "bool"
	ValueTypeTime    ValueType = "time"
	ValueTypeSlice   ValueType = "slice"
	ValueTypeUnknown ValueType = "unknown"
)
```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth
```go
package gauth // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"


VARIABLES

var (
	ErrInvalidToken  = &GAuthError{Code: "invalid_token", Message: "Invalid token"}
	ErrUnauthorized  = &GAuthError{Code: "unauthorized", Message: "Unauthorized access"}
	ErrTokenExpired  = &GAuthError{Code: "token_expired", Message: "Token has expired"}
	ErrInvalidGrant  = &GAuthError{Code: "invalid_grant", Message: "Invalid authorization grant"}
	ErrInvalidClient = &GAuthError{Code: "invalid_client", Message: "Invalid client credentials"}
)
    Error definitions


TYPES

type AuthorizationGrant struct {
	GrantID      string
	Scope        []string
	ValidUntil   time.Time
	Restrictions interface{}
	ClientID     string // Add ClientID field
}
    AuthorizationGrant represents an authorization grant

type AuthorizationRequest struct {
	ClientID string
	Scopes   []string
}
    AuthorizationRequest represents an authorization request

type Config struct {
	AuthServerURL     string
	ClientID          string
	ClientSecret      string
	SigningKey        string // NEW: distinct HMAC signing key (educational)
	Scopes            []string
	AccessTokenExpiry time.Duration
	RateLimit         interface{} // Placeholder for rate limit config
}
    Config represents the configuration for GAuth

type GAuth interface {
	InitiateAuthorization(req AuthorizationRequest) (*AuthorizationGrant, error)
	RequestToken(req TokenRequest) (*TokenResponse, error)
	ValidateToken(token string) (*TokenValidationResult, error)
	Close() error
}
    GAuth represents the main GAuth interface

type GAuthError struct {
	Code    string
	Message string
}
    GAuthError represents a GAuth error

func (e *GAuthError) Error() string

type PowerAdministrationPoint struct {
	GAuth       GAuth
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
    PowerAdministrationPoint represents a power administration point

func NewPowerAdministrationPoint(id, name, description string) *PowerAdministrationPoint
    NewPowerAdministrationPoint creates a new power administration point

func (p *PowerAdministrationPoint) InvalidateToken(token string) error
    InvalidateToken invalidates a token

type RateLimit struct {
	RequestsPerSecond int
	BurstSize         int
	Window            time.Duration
}
    RateLimit represents rate limit configuration

type ResourceServer struct {
	// Has unexported fields.
}
    ResourceServer represents a resource server

func NewResourceServer(name string, service *Service) *ResourceServer
    NewResourceServer creates a new resource server

func (rs *ResourceServer) ProcessTransaction(transaction TransactionDetails, token string) (string, error)
    ProcessTransaction processes a transaction

func (rs *ResourceServer) SetRateLimit(args ...interface{})
    SetRateLimit sets rate limiting for the resource server with multiple
    parameter support

type Restriction struct {
	Type   string
	Value  interface{}
	Scopes []string
}
    Restriction represents authorization restrictions

type Service struct {
	// Has unexported fields.
}
    Service represents the main GAuth service

func New(config Config) (*Service, error)
    New creates a new Service instance

func (g *Service) Close() error
    Close closes the Service and cleans up resources

func (g *Service) InitiateAuthorization(req AuthorizationRequest) (*AuthorizationGrant, error)
    InitiateAuthorization initiates an authorization flow

func (g *Service) InvalidateToken(token string) error
    InvalidateToken invalidates/revokes a token

func (g *Service) RequestToken(req TokenRequest) (*TokenResponse, error)
    RequestToken requests a token using an authorization grant

func (g *Service) ValidateToken(token string) (*TokenValidationResult, error)
    ValidateToken validates a token and returns client information

type TokenRequest struct {
	GrantID      string
	Scope        []string
	Restrictions interface{}
	Context      interface{}
}
    TokenRequest represents a token request

type TokenResponse struct {
	Token      string
	Scope      []string
	ValidUntil time.Time
}
    TokenResponse represents a token response

type TokenValidationResult struct {
	ClientID string
	Scope    []string
	Valid    bool
}
    TokenValidationResult represents the result of token validation

type TransactionDetails struct {
	ID             string
	Type           TransactionType
	Status         TransactionStatus
	ClientID       string
	ResourceID     string
	Scopes         []string
	Amount         float64
	Currency       string
	Timestamp      time.Time
	Source         string
	Destination    string
	Description    string
	CustomMetadata map[string]interface{} // Add CustomMetadata field
}
    TransactionDetails represents transaction details

type TransactionStatus string
    TransactionStatus represents transaction status

const (
	TransactionPending   TransactionStatus = "pending"
	TransactionCompleted TransactionStatus = "completed"
	TransactionFailed    TransactionStatus = "failed"
	TransactionCancelled TransactionStatus = "cancelled"
)
type TransactionType string
    TransactionType represents different types of transactions

const (
	PaymentTransaction    TransactionType = "payment"
	TransferTransaction   TransactionType = "transfer"
	WithdrawalTransaction TransactionType = "withdrawal"
	DepositTransaction    TransactionType = "deposit"
)
```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa
```go
package poa // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"

Package poa provides Power-of-Attorney functionality This is a compatibility
alias for the rfc0111 package

CONSTANTS

const (
	PrincipalTypeOrganization = "Organization"
	OrgTypeNonProfit          = "NonProfit"
	ClientTypeLLM             = "LLM"
	RepresentationSole        = "Sole"
	SignatureSingle           = "Single"
	SectorInformationComm     = IndustrySector("InformationComm")
	SectorProfessional        = IndustrySector("Professional")
	SectorFinancialInsurance  = IndustrySector("FinancialInsurance")
	GeoTypeNational           = "National"
	GeoTypeRegional           = "Regional"
	TransactionLoan           = Transaction("Loan")
	TransactionPurchase       = Transaction("Purchase")
	DecisionFinancial         = Decision("Financial")
	DecisionStrategic         = Decision("Strategic")
	DecisionInfoSharing       = Decision("InfoSharing")
	ActionResearching         = NonPhysicalAction("Researching")
	ActionBrainstorming       = NonPhysicalAction("Brainstorming")
)
    Example constants for demo compatibility


FUNCTIONS

func CreateRFC0115CompliantConfig() interface{}
    Stub functions for RFC-0115 demo compatibility

func ValidatePoADefinition(def PoADefinition) error
    ValidatePoADefinition performs minimal structural validation for the
    RFC-0115 example.

func ValidateRFC0115Compliance(config interface{}) error

TYPES

type Attestation struct {
	AttestedBy    string                 `json:"attested_by"`
	AttestedAt    time.Time              `json:"attested_at"`
	Evidence      map[string]interface{} `json:"evidence"`
	Confidence    float64                `json:"confidence"`
	ValidityScore float64                `json:"validity_score"`
}
    Attestation represents attestation information

func CreateAttestation(attestedBy string, evidence map[string]interface{}) *Attestation
    CreateAttestation creates an attestation with evidence

type AuthorizationScope struct {
	AuthorizationType AuthorizationType
	ApplicableSectors []IndustrySector
	ApplicableRegions []GeographicScope
	AuthorizedActions AuthorizedActions
}

type AuthorizationType struct {
	RepresentationType string
	Restrictions       []string
	SubProxyAuthority  bool
	SignatureType      string
}

type AuthorizedActions struct {
	Transactions       []Transaction
	Decisions          []Decision
	NonPhysicalActions []NonPhysicalAction
}

type AuthorizedClient struct {
	Type              string
	Identity          string
	Version           string
	OperationalStatus string
}

type ClientOwnerInfo struct {
	Name                      string
	RegisteredPowerOfAttorney bool
	CommercialRegisterEntry   bool
}

type ConflictResolution struct {
	ArbitrationJurisdiction string
}

type DeathIncapacityRules struct {
	ContinuationOnDeath    bool
	IncapacityInstructions string
}

type Decision string

type Delegation struct {
	DelegatedBy string    `json:"delegated_by"`
	DelegatedTo string    `json:"delegated_to"`
	DelegatedAt time.Time `json:"delegated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Scope       []string  `json:"scope"`
	Constraints []string  `json:"constraints,omitempty"`
	Revocable   bool      `json:"revocable"`
}
    Delegation represents delegation information

func CreateDelegationAttestation(delegatedBy, delegatedTo string, scope []string) *Delegation
    CreateDelegationAttestation creates a delegation attestation

type DelegationRequest struct {
	DelegatedBy string        `json:"delegated_by"`
	Scope       []string      `json:"scope"`
	Duration    time.Duration `json:"duration"`
	Constraints []string      `json:"constraints,omitempty"`
}
    DelegationRequest represents a delegation request

type FormalRequirements struct {
	NotarialCertification  bool
	IDVerificationRequired bool
	DigitalSignatures      bool
}

type GeographicScope struct {
	Type       string
	Identifier string
}

type IndustrySector string

type JurisdictionLaw struct {
	Language            string
	GoverningLaw        string
	PlaceOfJurisdiction string
	AttachedDocuments   []string
}

type MemoryService struct {
	// Has unexported fields.
}
    MemoryService implements the PoA service using in-memory storage

func NewMemoryService() *MemoryService
    NewMemoryService creates a new memory-based PoA service

func (s *MemoryService) Issue(ctx context.Context, req *Request) (*ProofOfAuthorization, error)
    Issue issues a new proof of authorization

func (s *MemoryService) List(ctx context.Context, subject string) ([]*ProofOfAuthorization, error)
    List lists all PoAs for a subject

func (s *MemoryService) Revoke(ctx context.Context, poaID string) error
    Revoke revokes a proof of authorization

func (s *MemoryService) Validate(ctx context.Context, poa *ProofOfAuthorization) error
    Validate validates a proof of authorization

type NonPhysicalAction string

type Organization struct {
	Type                string `json:"type"`
	Name                string `json:"name"`
	RegisterEntry       string `json:"register_entry"`
	ManagingDirector    string `json:"managing_director"`
	RegisteredAuthority bool   `json:"registered_authority"`
}
    Organization contains registration info for the principal organization.

type POAStatus string
    Simplified local POA status constants (legacy compatibility subset)

const (
	POAStatusActive  POAStatus = "active"
	POAStatusRevoked POAStatus = "revoked"
	POAStatusExpired POAStatus = "expired"
	POAStatusPending POAStatus = "pending"
)
type Parties struct {
	Principal        Principal        `json:"principal"`
	Representative   *Representative  `json:"representative,omitempty"`
	AuthorizedClient AuthorizedClient `json:"authorized_client"`
}
    Parties encapsulates principal, representative, and authorized client.

type PoADefinition struct {
	Parties       Parties            `json:"parties"`
	Authorization AuthorizationScope `json:"authorization"`
	Requirements  Requirements       `json:"requirements"`
}
    PoADefinition aggregates all sections of a Power-of-Attorney definition.

type PowerLimits struct {
	PowerLevels        []string
	InteractionBounds  []string
	ToolLimitations    []string
	QuantumResistance  bool
	ExplicitExclusions []string
}

type Principal struct {
	Type         string        `json:"type"`
	Identity     string        `json:"identity"`
	Organization *Organization `json:"organization,omitempty"`
}
    Principal represents the principal party (organization or individual)

type ProofOfAuthorization struct {
	ID          string                 `json:"id"`
	Subject     string                 `json:"subject"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	Issuer      string                 `json:"issuer"`
	IssuedAt    time.Time              `json:"issued_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Scope       []string               `json:"scope"`
	Delegation  *Delegation            `json:"delegation,omitempty"`
	Attestation *Attestation           `json:"attestation,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
    ProofOfAuthorization represents a proof of authorization token

type Representative struct {
	ClientOwner *ClientOwnerInfo `json:"client_owner,omitempty"`
}
    Representative contains client owner info linking to authorization.

type Request struct {
	Subject    string                 `json:"subject"`
	Resource   string                 `json:"resource"`
	Action     string                 `json:"action"`
	Scope      []string               `json:"scope,omitempty"`
	Delegation *DelegationRequest     `json:"delegation,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}
    Request represents a PoA request

type Requirements struct {
	ValidityPeriod       ValidityPeriod
	FormalRequirements   FormalRequirements
	PowerLimits          PowerLimits
	RightsObligations    RightsObligations
	SpecialConditions    SpecialConditions
	DeathIncapacityRules DeathIncapacityRules
	SecurityCompliance   SecurityCompliance
	JurisdictionLaw      JurisdictionLaw
	ConflictResolution   ConflictResolution
}

type RightsObligations struct {
	ReportingDuties   []string
	LiabilityRules    []string
	CompensationRules []string
}

type SecurityCompliance struct {
	CommunicationProtocols []string
	SecurityProperties     []string
	ComplianceInfo         []string
	UpdateMechanism        string
}

type Service interface {
	Issue(ctx context.Context, req *Request) (*ProofOfAuthorization, error)
	Validate(ctx context.Context, poa *ProofOfAuthorization) error
	Revoke(ctx context.Context, poaID string) error
	List(ctx context.Context, subject string) ([]*ProofOfAuthorization, error)
}
    Service defines the PoA service interface

type SpecialConditions struct {
	ConditionalEffectiveness []string
	ImmediateNotification    []string
}

type Transaction string

type ValidityPeriod struct {
	StartTime             time.Time
	EndTime               time.Time
	AutoRenewalConditions []string
	TerminationConditions []string
}

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rate
```go
package rate // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rate"

Package rate provides rate limiting functionality

VARIABLES

var ErrLimitExceeded = fmt.Errorf("rate limit exceeded")
    ErrLimitExceeded is returned when rate limit is exceeded


FUNCTIONS

func Demo() error
    Demo demonstrates rate limiting functionality

func WrapTokenBucket(config Config, fn func() error) func() error
    WrapTokenBucket wraps a function with token bucket rate limiting


TYPES

type Algorithm int
    Algorithm represents different rate limiting algorithms

const (
	// TokenBucket represents the token bucket algorithm
	TokenBucket Algorithm = iota
	// SlidingWindow represents the sliding window algorithm
	SlidingWindow
	// FixedWindow represents the fixed window algorithm
	FixedWindow
)
func (a Algorithm) String() string
    String returns a string representation of the algorithm

type Config struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	BurstSize         int           `json:"burst_size"`
	WindowSize        time.Duration `json:"window_size"`
}
    Config represents rate limiting configuration

func DefaultConfig() Config
    DefaultConfig returns the default rate limiting configuration

type EnhancedConfig struct {
	Config
	Rate   int           `json:"rate"`   // Alias for RequestsPerSecond
	Window time.Duration `json:"window"` // Alias for WindowSize
}
    Enhanced Config with additional fields for monitoring example compatibility

func NewEnhancedConfig(rate int, window time.Duration) EnhancedConfig
    NewEnhancedConfig creates an enhanced config with aliases

type Limiter struct {
	// Has unexported fields.
}
    Limiter represents a rate limiter

func NewLimiter(config Config) *Limiter
    NewLimiter creates a new rate limiter

func NewLimiterFromConfig(config *Config) *Limiter
    NewLimiterFromConfig allows creating a Limiter from a Config struct

func (l *Limiter) Allow() bool
    Allow checks if a request is allowed under the rate limit

func (l *Limiter) AllowClient(ctx context.Context, clientID string) error
    AllowClient checks if a request is allowed for a specific client ID This
    is a simple implementation that uses the same bucket for all clients In a
    production system, you would maintain per-client buckets

func (l *Limiter) Close() error
    Close closes the rate limiter and cleans up resources

func (l *Limiter) GetRemainingRequests(clientID string) int
    GetRemainingRequests returns the number of remaining requests in the current
    window

func (l *Limiter) Reset(clientID string)
    Reset resets the rate limiter for a specific client

func (l *Limiter) Wait(ctx context.Context) error
    Wait waits until a request is allowed

type TokenBucketWrapper struct {
	// Has unexported fields.
}

func NewTokenBucket(config EnhancedConfig) *TokenBucketWrapper
    NewTokenBucket creates a new token bucket rate limiter

func (tb *TokenBucketWrapper) Allow() error
    Allow checks if a request should be allowed

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/ratelimit
```go
package ratelimit // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/ratelimit"


FUNCTIONS

func WrapTokenBucket(fn func() error, config Config) func() error
    WrapTokenBucket wraps a function with rate limiting


TYPES

type Algorithm string
    Algorithm represents different rate limiting algorithms

const (
	TokenBucket   Algorithm = "token_bucket"
	LeakyBucket   Algorithm = "leaky_bucket"
	FixedWindow   Algorithm = "fixed_window"
	SlidingWindow Algorithm = "sliding_window"
)
type Config struct {
	Algorithm Algorithm     `json:"algorithm"`
	Rate      int           `json:"rate"`   // requests per period
	Period    time.Duration `json:"period"` // time period
	Burst     int           `json:"burst"`  // maximum burst size
	KeyFunc   func() string `json:"-"`      // function to generate keys
}
    Config represents rate limiter configuration

func DefaultConfig() Config
    DefaultConfig returns a default rate limiter configuration

type Limiter interface {
	Allow(key string) bool
	AllowN(key string, n int) bool
	Wait(ctx context.Context, key string) error
	WaitN(ctx context.Context, key string, n int) error
	Reset(key string)
	Close() error
}
    Limiter defines the rate limiter interface

func NewLimiter(config Config) Limiter
    NewLimiter creates a new rate limiter

type RateLimitError struct {
	Key   string
	Limit int
}
    RateLimitError represents a rate limit exceeded error

func (e *RateLimitError) Error() string

type TokenBucketLimiter struct {
	// Has unexported fields.
}
    TokenBucketLimiter implements a token bucket rate limiter

func (l *TokenBucketLimiter) Allow(key string) bool
    Allow checks if a request is allowed

func (l *TokenBucketLimiter) AllowN(key string, n int) bool
    AllowN checks if n requests are allowed

func (l *TokenBucketLimiter) Close() error
    Close closes the rate limiter

func (l *TokenBucketLimiter) Reset(key string)
    Reset resets the rate limiter for a specific key

func (l *TokenBucketLimiter) Wait(ctx context.Context, key string) error
    Wait waits until a request can be allowed

func (l *TokenBucketLimiter) WaitN(ctx context.Context, key string, n int) error
    WaitN waits until n requests can be allowed

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/resilience
```go
package resilience // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/resilience"


FUNCTIONS

func NewRateLimiter(requestsPerSecond, burstSize int) interface{ Allow() bool }
    NewRateLimiter returns a simple token bucket rate limiter


TYPES

type Bulkhead struct {
	// Has unexported fields.
}
    Bulkhead type for test compatibility

func NewBulkhead(maxConcurrent int) *Bulkhead

func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error

type BulkheadConfig struct {
	MaxConcurrentRequests int
}

type CircuitBreakerConfig struct {
	Threshold     int
	ResetTimeout  time.Duration
	OnStateChange func(name string, from, to interface{})
}
    CircuitBreakerConfig configures circuit breaker behavior

type CircuitState string
    CircuitState represents the state of a circuit breaker (for demo
    compatibility)

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half-open"
)
type Composite struct {
	// Has unexported fields.
}
    Minimal stub for Composite to allow compilation

func NewComposite(opts CompositeOptions) *Composite

func (c *Composite) Execute(ctx context.Context, fn func() error) error
    Execute executes a function with composite resilience patterns applied
    (stub)

type CompositeOptions struct {
	CircuitOptions interface{}
	MaxConcurrent  int
	RetryStrategy  RetryStrategy
	RateLimit      int
	BurstSize      int
}

type PatternOption func(*Patterns)
    Minimal stub for PatternOption

func WithBulkhead(maxConcurrentRequests int) PatternOption
    WithBulkhead returns a PatternOption for bulkhead

func WithCircuitBreaker(threshold int, resetTimeout time.Duration, onStateChange func(name string, from, to interface{})) PatternOption
    WithCircuitBreaker returns a PatternOption for circuit breaker

func WithRateLimit(requestsPerSecond, burstSize int, onLimit func()) PatternOption
    WithRateLimit returns a PatternOption for rate limiting

func WithRetry(maxAttempts int, initialInterval, maxInterval time.Duration) PatternOption
    WithRetry returns a PatternOption for retry

type Patterns struct {
	// Has unexported fields.
}
    Patterns combines multiple resilience patterns

func Combine(patterns ...*Patterns) *Patterns
    Combine returns a Patterns that applies all given patterns in order

func NewCircuitBreaker(config interface{}) *Patterns
    NewCircuitBreaker stub

func NewPatterns(name string, opts ...PatternOption) *Patterns
    NewPatterns creates a Patterns struct with options applied

func NewRetry(config interface{}) *Patterns
    NewRetry with retry logic for integration test compatibility

func NewTimeout(config interface{}) *Patterns
    NewTimeout stub

func (p *Patterns) Execute(ctx context.Context, fn func() error) error
    Execute executes a function with retry pattern applied

type RateLimitConfig struct {
	RequestsPerSecond int
	BurstSize         int
	OnLimit           func()
}
    Minimal config stubs for test compatibility

type RetryConfig struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
}

type RetryStrategy struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}
    RetryStrategy stub for test compatibility

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc
```go
package rfc // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc"


FUNCTIONS

func DemoRFC0111PowerOfAttorney() error
    DemoRFC0111PowerOfAttorney demonstrates an RFC 0111 style issuance +
    revocation cycle.

func GetSupportedRFCs() []string
func TestRFC0115Features() error
    TestRFC0115Features exercises basic token store operations for RFC 0115
    style features.

func ValidateCombinedRFCConfig(config *CombinedRFCConfig) error
    ValidateCombinedRFCConfig validates a combined RFC configuration

func ValidateCompliance(code string) bool
    ValidateCompliance performs a simple supported-RFC check.

func ValidateRFC0111Flow(svc *gauth.Service, issuedToken string) error
    ValidateRFC0111Flow performs a minimal validation sequence on a gauth
    Service. NOTE: The caller must have already issued a token to supply here;
    this helper focuses on validation + revocation semantics only. (Examples
    perform full grant/token issuance flows.)


TYPES

type Authorization struct {
	ApplicableRegions []GeographicRegion `json:"applicable_regions"`
	ApplicableSectors []string           `json:"applicable_sectors"`
}
    Authorization represents authorization scope

type AuthorizedClient struct {
	Identity          string `json:"identity"`
	Type              string `json:"type"`
	OperationalStatus string `json:"operational_status"`
}
    AuthorizedClient represents an authorized client

type CombinedRFCConfig struct {
	RFC0111          *RFC0111Config         `json:"rfc_0111"`
	RFC0115          *RFC0115Config         `json:"rfc_0115"`
	IntegrationLevel string                 `json:"integration_level"`
	CombinedVersion  string                 `json:"combined_version"`
	Compatibility    map[string]interface{} `json:"compatibility"`
	Metadata         map[string]interface{} `json:"metadata"`
} // CreateCombinedRFCConfig creates a new combined RFC configuration
    CombinedRFCConfig represents configuration for combined RFC compliance

func CreateCombinedRFCConfig() *CombinedRFCConfig

type ComplianceInfo struct {
	RFC0111 RFC0111Summary
	RFC0115 RFC0115Summary
	RFC0150 RFC0150Summary
}

func GetComplianceInfo() ComplianceInfo

type Exclusion struct {
	Prohibited      bool `json:"prohibited"`
	LicenseRequired bool `json:"license_required"`
}
    Exclusion represents an exclusion with its properties

type GAuthContext struct {
	PPArchitectureRole  string `json:"pp_architecture_role"`
	ExclusionsCompliant bool   `json:"exclusions_compliant"`
	AIGovernanceLevel   string `json:"ai_governance_level"`
}
    GAuthContext represents GAuth integration context

type GeographicRegion struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
}
    GeographicRegion represents a geographic region

type Organization struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	RegisterEntry string `json:"register_entry"`
}
    Organization represents an organization entity

type PoAParties struct {
	Principal        Principal        `json:"principal"`
	AuthorizedClient AuthorizedClient `json:"authorized_client"`
}
    PoAParties represents the parties involved in PoA

type PolicyAdministrationPoint struct {
	ClientOwnerAuthorizer string `json:"client_owner_authorizer"`
}
    PolicyAdministrationPoint represents PAP configuration

type PolicyDecisionPoint struct {
	PrimaryPDP string `json:"primary_pdp"`
}
    PolicyDecisionPoint represents PDP configuration

type PolicyEnforcementPoint struct {
	SupplySide PowerComponent `json:"supply_side"`
	DemandSide PowerComponent `json:"demand_side"`
}
    PolicyEnforcementPoint represents PEP configuration

type PolicyInformationPoint struct {
	AuthorizationServer string `json:"authorization_server"`
}
    PolicyInformationPoint represents PIP configuration

type PolicyVerificationPoint struct {
	TrustServiceProvider string `json:"trust_service_provider"`
}
    PolicyVerificationPoint represents PVP configuration

type PowerComponent struct {
	Entity string `json:"entity"`
	Status string `json:"status"`
}
    PowerComponent represents a power-related component with entity and status

type Principal struct {
	Identity     string        `json:"identity"`
	Type         string        `json:"type"`
	Organization *Organization `json:"organization,omitempty"`
}
    Principal represents a principal party

type RFC0111Client struct {
	Type           RFC0111ClientType      `json:"type"`
	Identity       string                 `json:"identity"`
	AutonomyLevel  string                 `json:"autonomy_level"`
	AICapabilities []string               `json:"ai_capabilities"`
	RequestTypes   []string               `json:"request_types"`
	ComplianceMode string                 `json:"compliance_mode"`
	Config         *CombinedRFCConfig     `json:"config,omitempty"`
	Endpoint       string                 `json:"endpoint,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
} // NewRFC0111Client creates a new RFC 0111 client
    RFC0111Client represents a client for RFC 0111 operations

func NewRFC0111Client(config *CombinedRFCConfig, endpoint string) *RFC0111Client

func (c *RFC0111Client) Initialize() error
    Initialize initializes the RFC 0111 client

func (c *RFC0111Client) ProcessRequest(request interface{}) error
    ProcessRequest processes a request using RFC 0111 protocols

func (c *RFC0111Client) ValidateToken(token string) error
    ValidateToken validates a token using RFC 0111 rules

type RFC0111ClientType string
    RFC0111ClientType represents different types of RFC 0111 clients

const (
	RFC0111ClientTypeDigitalAgent  RFC0111ClientType = "digital_agent"
	RFC0111ClientTypeAgenticAI     RFC0111ClientType = "agentic_ai"
	RFC0111ClientTypeHumanoidRobot RFC0111ClientType = "humanoid_robot"
)
type RFC0111Config struct {
	Enabled        bool                  `json:"enabled"`
	Exclusions     RFC0111Exclusions     `json:"exclusions"`
	PPArchitecture RFC0111PPArchitecture `json:"pp_architecture"`
}
    RFC0111Config represents RFC 0111 specific configuration

type RFC0111Exclusions struct {
	Web3Blockchain     Exclusion `json:"web3_blockchain"`
	AIOperators        Exclusion `json:"ai_operators"`
	DNABasedIdentities Exclusion `json:"dna_based_identities"`
	DecentralizedAuth  Exclusion `json:"decentralized_auth"`
	EnforcementLevel   string    `json:"enforcement_level"`
}
    RFC0111Exclusions represents exclusions for RFC 0111

type RFC0111PPArchitecture struct {
	PEP PolicyEnforcementPoint    `json:"pep"` // Policy Enforcement Point
	PDP PolicyDecisionPoint       `json:"pdp"` // Policy Decision Point
	PIP PolicyInformationPoint    `json:"pip"` // Policy Information Point
	PAP PolicyAdministrationPoint `json:"pap"` // Policy Administration Point
	PVP PolicyVerificationPoint   `json:"pvp"` // Policy Verification Point
}
    RFC0111PPArchitecture represents the PP architecture for RFC 0111

type RFC0111Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}
    Summary types (lightweight – do not duplicate full config domain models)

func GetRFC0111Compliance() RFC0111Summary

type RFC0115Config struct {
	Enabled       bool   `json:"enabled"`
	PoADefinition string `json:"poa_definition"`
}
    RFC0115Config represents RFC 0115 specific configuration

type RFC0115PoADefinition struct {
	Definition    string                 `json:"definition"`
	Attestation   string                 `json:"attestation"`
	Verification  map[string]interface{} `json:"verification"`
	Parties       PoAParties             `json:"parties"`
	Authorization Authorization          `json:"authorization"`
	GAuthContext  GAuthContext           `json:"gauth_context"`
}
    RFC0115PoADefinition represents the PoA definition for RFC 0115

func CreateDefaultPoADefinition(definition string) RFC0115PoADefinition
    CreateDefaultPoADefinition creates a default PoA definition with sample data

type RFC0115Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}

func GetRFC0115Compliance() RFC0115Summary

type RFC0150Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}

func GetRFC0150Compliance() RFC0150Summary

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111
```go
package rfc0111 // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"


FUNCTIONS

func Demo() error
    Demo demonstrates RFC 0111 power-of-attorney functionality

func ValidateRFC0111Compliance(cfg *RFC0111Config) error
    ValidateRFC0111Compliance performs minimal semantic checks required by the
    example. It intentionally does NOT try to replicate deeper domain logic –
    the goal is to confirm that mandatory exclusions are enforced and that key
    numeric / duration parameters are sensible.


TYPES

type DelegationRequest struct {
	Grantor      string            `json:"grantor"`
	Grantee      string            `json:"grantee"`
	Scope        []string          `json:"scope"`
	Restrictions map[string]string `json:"restrictions"`
	Duration     time.Duration     `json:"duration"`
}
    DelegationRequest represents a request to create a POA

type DelegationResponse struct {
	POA       PowerOfAttorney `json:"power_of_attorney"`
	AuthToken string          `json:"auth_token"`
	ExpiresAt time.Time       `json:"expires_at"`
}
    DelegationResponse represents the response to a delegation request

type GAuth10Framework struct {
	AuthServer string
	Clients    []string
}
    GAuth10Framework is a placeholder for the framework struct

func (f *GAuth10Framework) GetStatus() map[string]interface{}
    GetStatus returns the current status of the framework

func (f *GAuth10Framework) ToJSON() ([]byte, error)
    ToJSON serializes the framework to JSON

type POAStatus string
    POAStatus represents the status of a PowerOfAttorney

const (
	POAStatusActive  POAStatus = "active"
	POAStatusRevoked POAStatus = "revoked"
	POAStatusExpired POAStatus = "expired"
)
type PowerOfAttorney struct {
	ID           string            `json:"id"`
	Grantor      string            `json:"grantor"`
	Grantee      string            `json:"grantee"`
	Scope        []string          `json:"scope"`
	Restrictions map[string]string `json:"restrictions"`
	ValidFrom    time.Time         `json:"valid_from"`
	ValidUntil   time.Time         `json:"valid_until"`
	Status       POAStatus         `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}
    PowerOfAttorney represents a delegation of authority

type RFC0111Config struct {
	AuthorizationServerURL    string        `json:"authorization_server_url"`
	TrustServiceProvider      string        `json:"trust_service_provider"`
	RequireNotarization       bool          `json:"require_notarization"`
	MaxDelegationDepth        int           `json:"max_delegation_depth"`
	DefaultTokenValidity      time.Duration `json:"default_token_validity"`
	AuditingEnabled           bool          `json:"auditing_enabled"`
	ComplianceTrackingEnabled bool          `json:"compliance_tracking_enabled"`

	// Mandatory open‑source exclusions as per the narrative in the example.
	ExcludeWeb3          bool `json:"exclude_web3"`
	ExcludeAIOperators   bool `json:"exclude_ai_operators"`
	ExcludeDNAIdentities bool `json:"exclude_dna_identities"`
}
    RFC0111Config represents high-level configuration required by the official
    RFC-0111 implementation demo. All fields are intentionally kept exactly as
    referenced in the example for backwards compatibility.

type Service struct {
	// Has unexported fields.
}
    Service provides RFC 0111 power-of-attorney services

func NewService(auditLogger *audit.MemoryLogger, authorizer *authz.MemoryAuthorizer) *Service
    NewService creates a new RFC 0111 service

func (s *Service) CreateDelegation(req DelegationRequest) (*DelegationResponse, error)
    CreateDelegation creates a new power-of-attorney delegation

func (s *Service) GetDelegation(poaID string) (*PowerOfAttorney, error)
    GetDelegation retrieves a power-of-attorney by ID

func (s *Service) ListDelegations(userID string) ([]*PowerOfAttorney, error)
    ListDelegations lists all delegations for a user (as grantor or grantee)

func (s *Service) RevokeDelegation(poaID, revoker string) error
    RevokeDelegation revokes a power-of-attorney delegation

func (s *Service) ValidateDelegation(poaID, grantee, action string) error
    ValidateDelegation validates a power-of-attorney for a specific action

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/store
```go
package store // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/store"

Package store provides token storage functionality This is a compatibility alias
for the token package storage

TYPES

type Filter = token.Filter
    Re-export types from token package for compatibility

type MemoryStore = token.MemoryStore
    Re-export types from token package for compatibility

type Store = token.Store
    Re-export types from token package for compatibility

type Token = token.Token
    Re-export types from token package for compatibility

type TokenType = token.TokenType
    Re-export types from token package for compatibility

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token
```go
package token // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"


VARIABLES

var (
	ErrTokenExpired  = errors.New(errMsgExpired)
	ErrTokenNotFound = errors.New(errMsgNotFound)
	ErrInvalidToken  = errors.New(errMsgInvalid)
	ErrTokenRevoked  = errors.New(errMsgRevoked)
)
    Error variables


FUNCTIONS

func GenerateID() string
    GenerateID generates a new unique ID

func NewID() string
    NewID generates a new unique token ID

func ToBytes(_ interface{}) []byte
    --- BENCHMARK STRICT COMPATIBILITY PATCHES --- Patch: Accept JWTSigner and
    Algorithm as []byte and string for benchmark compatibility

func ToString(val interface{}) string

TYPES

type Algorithm string
    Algorithm enum for JWT algorithms

const (
	HS256 Algorithm = "HS256"
	RS256 Algorithm = "RS256"
)
type BenchmarkMetrics struct{}
    --- BENCHMARK STRICT COMPATIBILITY PATCHES ---

func NewMetrics(_ string) *BenchmarkMetrics

func (m *BenchmarkMetrics) ObserveEntry(...interface{})

func (m *BenchmarkMetrics) ObserveStorageOperation(...interface{})

type Blacklist interface {
	Add(ctx context.Context, tokenID string, reason string) error
	IsBlacklisted(ctx context.Context, tokenID string) bool
	Remove(ctx context.Context, tokenID string) error
}
    Blacklist interface for token blacklisting

type Config struct {
	DefaultTTL     time.Duration
	MaxTTL         time.Duration
	SigningKey     []byte
	Algorithm      Algorithm
	ValidateExpiry bool
	Store          Store
	SigningMethod  string
	ValidityPeriod time.Duration
}
    Config holds token service configuration

type DefaultQuerier struct {
	// Has unexported fields.
}
    DefaultQuerier provides default query implementation

func NewDefaultQuerier(store Store) *DefaultQuerier
    NewDefaultQuerier creates a new default querier

func (dq *DefaultQuerier) Count(ctx context.Context, filter Filter) (int, error)
    Count returns the count of tokens matching the filter

func (dq *DefaultQuerier) Query(ctx context.Context, filter Filter) ([]*Token, error)
    Query executes a query with the given filter

type DefaultValidationChain struct {
	// Has unexported fields.
}
    DefaultValidationChain provides default validation chain

func NewValidationChain(config ValidationConfig) *DefaultValidationChain

func (dvc *DefaultValidationChain) AddValidator(validator Validator)
    AddValidator adds a validator to the chain

func (dvc *DefaultValidationChain) Validate(ctx context.Context, token *Token) error
    Validate validates a token using the chain

type DeviceInfo struct {
	ID        string
	UserAgent string
	IPAddress string
	Platform  string
	Version   string
}
    DeviceInfo represents device information

type Filter struct {
	Metadata     map[string]string // Add Metadata field
	Scopes       []string
	Types        []TokenType // Add Types field
	IssuedAfter  time.Time
	IssuedBefore time.Time
	ExpiresAfter time.Time
	ExpiresBefor time.Time
	TokenType    string
	Subject      string
	Issuer       string
	ClientID     string
	Status       string
	Active       bool // Add Active field
}
    Filter represents filtering criteria for tokens Filter layout optimized for
    locality; slice/map first, then times, then scalars/bools.

type JWTSigner interface {
	Sign(token *Token) (string, error)
	Verify(tokenString string) (*Token, error)
}
    JWTSigner interface for JWT signing

func NewMockSigner() JWTSigner
    NewMockSigner returns a dummy JWTSigner for benchmarks

type Manager struct {
	// Has unexported fields.
}
    Manager provides token management operations

func NewManager(cfg ManagerConfig) *Manager
    NewManager creates a new token manager (compatibility shim for integration
    tests)

func (m *Manager) CompleteRotation() error
    CompleteRotation is a stub for integration test compatibility

func (m *Manager) CreateToken(ctx context.Context, claims map[string]interface{}, ttl time.Duration) (*Token, error)
    --- Integration test compatibility shims --- CreateToken creates a token
    from claims and ttl (integration test compatibility)

func (m *Manager) CreateTokenWithRefresh(ctx context.Context, claims map[string]interface{}, accessTTL, refreshTTL time.Duration) (*Token, string, error)
    CreateTokenWithRefresh is a stub for integration test compatibility

func (m *Manager) Issue(ctx context.Context, subject string, scopes []string, ttl time.Duration) (*Token, error)
    Manager.Issue compatibility method for (context.Context, string, []string,
    time.Duration)

func (m *Manager) RefreshToken(ctx context.Context, refreshToken string) (*Token, error)
    RefreshToken is a stub for integration test compatibility

func (m *Manager) RevokeToken(ctx context.Context, token *Token) error
    RevokeToken revokes a token (integration test compatibility)

func (m *Manager) RotateKey(keyID string, signingKey []byte) error
    RotateKey is a stub for integration test compatibility

func (m *Manager) ValidateToken(ctx context.Context, token *Token) (map[string]interface{}, error)
    ValidateToken validates a token (integration test compatibility)

type ManagerConfig struct {
	Store      Store
	Monitor    *Monitor // integration test compatibility
	SigningKey []byte
	Issuer     string
	KeyID      string
}
    ManagerConfig provides configuration for the token manager (compatibility
    shim for integration tests) ManagerConfig field order optimized (group
    pointer/slice types first) to reduce padding.

type MemoryBlacklist struct {
	// Has unexported fields.
}
    MemoryBlacklist provides in-memory blacklist implementation

func NewBlacklist() *MemoryBlacklist
    NewBlacklist creates a new memory blacklist

func (mb *MemoryBlacklist) Add(ctx context.Context, tokenID string, reason string) error
    Add adds a token to the blacklist

func (mb *MemoryBlacklist) IsBlacklisted(ctx context.Context, tokenID string) bool
    IsBlacklisted checks if a token is blacklisted

func (mb *MemoryBlacklist) Remove(ctx context.Context, tokenID string) error
    Remove removes a token from the blacklist

type MemoryStore struct {
	// Has unexported fields.
}
    MemoryStore provides in-memory token storage implementation

func NewMemoryStore(ttls ...time.Duration) *MemoryStore
    NewMemoryStore creates a new memory store with optional TTL

func (ms *MemoryStore) Close() error
    Close closes the store

func (ms *MemoryStore) Delete(ctx context.Context, tokenID string) error
    Delete removes a token

func (ms *MemoryStore) Get(ctx context.Context, tokenID string) (*Token, error)
    Get retrieves a token by ID

func (ms *MemoryStore) List(ctx context.Context, filter Filter) ([]*Token, error)
    List returns tokens matching the filter (Filter by value now)

func (ms *MemoryStore) MarkRevoked(ctx context.Context, tokenID, reason string) error
    Keep the original revocation marking logic but also remove

func (ms *MemoryStore) Revoke(ctx context.Context, tokenID, reason string) error
    Revoke marks a token as revoked

func (ms *MemoryStore) Rotate(ctx context.Context, oldToken *Token, newToken *Token) error
    Rotate creates a new token pair (old, new)

func (ms *MemoryStore) Save(ctx context.Context, key string, token *Token) error
    Save stores a token with a custom key (3-parameter version)

func (ms *MemoryStore) Validate(ctx context.Context, token *Token) error
    Validate checks if a token is valid

type Metadata struct {
	AppData    map[string]interface{}
	Labels     map[string]string   // Add Labels field
	Attributes map[string][]string // Add Attributes field
	Device     *DeviceInfo         // Add Device field
	Scopes     []string
	Tags       []string // Add Tags field
	IssuedAt   time.Time
	ExpiresAt  time.Time
	ClientID   string
	AppID      string // Add AppID field
	AppVersion string // Add AppVersion field
}
    Metadata holds token metadata Metadata layout optimized for reduced padding
    (maps/slices/pointers first, then times, then strings).

type Monitor struct {
	// Has unexported fields.
}
    Monitor is a stub struct for integration test compatibility

func NewMonitor() *Monitor
    NewMonitor is a stub for integration test compatibility

func (m *Monitor) GetStats() MonitorStats
    GetStats is a stub for integration test compatibility

func (m *Monitor) IncCreated()

func (m *Monitor) IncRevoked()

func (m *Monitor) IncValidated()

type MonitorStats struct {
	TokensCreated   uint64
	TokensValidated uint64
	TokensRevoked   uint64
}
    MonitorStats is a struct for integration test compatibility

type Querier interface {
	Query(ctx context.Context, filter Filter) ([]*Token, error) // Filter by value
	Count(ctx context.Context, filter Filter) (int, error)      // Filter by value
}
    Querier interface for token queries

type RevocationStatus struct {
	RevokedAt time.Time
	RevokedBy string
	Reason    string
}
    RevocationStatus represents the revocation status of a token

type Service struct {
	// Has unexported fields.
}
    Service provides token service operations

func NewService(store Store, blacklist Blacklist, config *Config) *Service
    NewService creates a new token service

func NewServiceForBenchmarks(config Config, store *MemoryStore) *Service
    NewServiceForBenchmarks creates a service instance for benchmarking This
    function provides the exact signature expected by the benchmark tests

func (s *Service) Issue(ctx context.Context, subject string, scopes []string, ttl time.Duration) (*Token, error)
    Issue is a stub for benchmark compatibility

func (s *Service) Validate(ctx context.Context, token *Token) error
    --- BENCHMARK STRICT COMPATIBILITY PATCHES --- Validate stub for benchmark
    compatibility

type SimpleJWTSigner struct {
	// Has unexported fields.
}
    SimpleJWTSigner provides simple JWT signing

func NewJWTSigner(key []byte, algorithm Algorithm) *SimpleJWTSigner
    NewJWTSigner creates a new JWT signer

func (sjs *SimpleJWTSigner) Sign(token *Token) (string, error)
    Sign signs a token and returns a JWT string

func (sjs *SimpleJWTSigner) Verify(tokenString string) (*Token, error)
    Verify verifies a JWT string and returns the token

type Store interface {
	Save(ctx context.Context, key string, token *Token) error // 3-parameter version as main
	Get(ctx context.Context, tokenID string) (*Token, error)
	List(ctx context.Context, filter Filter) ([]*Token, error) // Accept Filter by value not pointer
	Delete(ctx context.Context, tokenID string) error
	Close() error
}
    Store interface for token storage - using the 3-parameter signature as
    primary

type Token struct {
	Metadata         *Metadata
	RevocationStatus *RevocationStatus
	Scopes           []string
	Audience         []string // Add Audience field
	ID               string
	Value            string
	Subject          string
	Issuer           string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	NotBefore        time.Time // Add NotBefore field
	Algorithm        Algorithm // Add Algorithm field
	Type             TokenType // Use TokenType instead of string
	TokenType        TokenType
}
    Token represents a token in the GAuth system Token layout optimized:
    pointer/slice fields grouped, times grouped, smaller scalars last.

type TokenType string
    Token type constants

const (
	Access  TokenType = "access"
	Refresh TokenType = "refresh"
	ID      TokenType = "id"
)
type Type = TokenType
    Type alias for backward compatibility (some examples use token.Type)

type ValidationChain interface {
	Validate(ctx context.Context, token *Token) error
	AddValidator(validator Validator)
}
    ValidationChain interface for token validation

type ValidationConfig struct {
	CheckExpiry    bool
	CheckBlacklist bool
	CheckSignature bool
	MaxAge         time.Duration
}
    ValidationConfig holds validation configuration

type ValidationError struct {
	Code    string
	Message string
}
    ValidationError represents a validation error

func (ve *ValidationError) Error() string

type Validator interface {
	Validate(ctx context.Context, token *Token) error
}
    Validator interface for token validation

```

## github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web
```go
package web // import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web"

Package web: common status and error string constants for server responses.

CONSTANTS

const (
	StatusTimeout             = "timeout"
	StatusInvalidJurisdiction = "invalid jurisdiction"
	StatusDisallowedScope     = "disallowed scope"
)
const (
	TokenStatusNotFound       = "not_found"
	TokenStatusValid          = "valid"
	TokenStatusAlreadyRevoked = "already_revoked"
	TokenStatusRevoked        = "revoked"
)
    ===================== TOKEN STORE IMPLEMENTATION =====================


TYPES

type AuditEntry struct {
	ID       string    `json:"id"`
	At       time.Time `json:"at"`
	Actor    string    `json:"actor"`
	Action   string    `json:"action"`
	Resource string    `json:"resource"`
	Outcome  string    `json:"outcome"`
	Meta     any       `json:"meta,omitempty"`
}
    ===================== AUDIT LOG IMPLEMENTATION =====================
    Lightweight in-memory append-only audit log for demo purposes.

type AuditLog struct {
	// Has unexported fields.
}

func NewAuditLog(capacity int) *AuditLog

func (l *AuditLog) Append(e *AuditEntry)

func (l *AuditLog) List(limit int) []*AuditEntry

func (l *AuditLog) Subscribe() chan *AuditEntry

func (l *AuditLog) Unsubscribe(ch chan *AuditEntry)

type BetaServer struct {
	// Has unexported fields.
}
    BetaServer is the primary exported server for the beta demonstration UI/API.

    Deprecated: The legacy name EducationalServer is retained as a type alias
    for backward compatibility. Prefer BetaServer moving forward.

func NewBetaServer(port string) *BetaServer
    NewBetaServer creates a new BetaServer instance.

func NewEducationalServer(port string) *BetaServer
    NewEducationalServer is deprecated; use NewBetaServer. Retained for callers
    not yet migrated. Deprecated: use NewBetaServer.

func (s *BetaServer) Run() error
    Run starts an HTTP server and blocks until a termination signal is received.
    If the provided port already includes a leading ':', keep it; otherwise
    default to :8080.

type EducationalServer = BetaServer
    Backward compatibility type alias

type Event struct {
	ID   string    `json:"id"`
	At   time.Time `json:"at"`
	Type string    `json:"type"`
	Data any       `json:"data"`
}
    ===================== EVENT HUB IMPLEMENTATION =====================

type EventHub struct {
	// Has unexported fields.
}

func NewEventHub(capacity int) *EventHub

func (h *EventHub) Emit(e *Event)

func (h *EventHub) List(limit int) []*Event

func (h *EventHub) Subscribe() chan *Event

func (h *EventHub) Unsubscribe(ch chan *Event)

type ExampleJob struct {
	ID         string
	ExampleID  string
	State      JobState
	Output     string
	Error      string
	CreatedAt  time.Time
	StartedAt  time.Time
	FinishedAt time.Time
	Logs       []string // incremental log lines for SSE
}
    ExampleJob represents execution metadata.

type ExampleMeta struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Group            string `json:"group"`
	EstimatedSeconds int    `json:"estimated_seconds"`
}
    ExampleMeta defines catalog metadata for an example.

type JobManager struct {
	// Has unexported fields.
}
    JobManager stores jobs in memory.

func NewJobManager(capacity int) *JobManager
    NewJobManager creates manager with capacity (0 => unlimited).

func (jm *JobManager) AddJob(j *ExampleJob)

func (jm *JobManager) AppendLog(id, line string)
    AppendLog appends a log line to a job if it exists.

func (jm *JobManager) GetJob(id string) (*ExampleJob, bool)

func (jm *JobManager) GetLogs(id string) []string
    GetLogs returns a copy of logs for a job.

func (jm *JobManager) ListJobs(state *JobState, limit int) []*ExampleJob
    ListJobs returns jobs filtered by optional state and limited.

func (jm *JobManager) SetJobState(id string, st JobState, out, errStr string)

type JobState string
    JobState represents state of an example job.

const (
	JobQueued  JobState = "queued"
	JobRunning JobState = "running"
	JobDone    JobState = "done"
	JobFailed  JobState = "failed"
	JobTimeout JobState = "timeout"
)
type Token struct {
	ID        string     `json:"id"`
	Value     string     `json:"token"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	Meta      any        `json:"meta,omitempty"`
}

type TokenStore struct {
	// Has unexported fields.
}

func NewTokenStore(capacity int) *TokenStore

func (ts *TokenStore) Create(ttlSeconds int, meta any) *Token

func (ts *TokenStore) Metrics() gin.H

func (ts *TokenStore) Revoke(id string) string

func (ts *TokenStore) Validate(idOrVal string) (string, *Token)

```

