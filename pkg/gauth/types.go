package gauth

import (
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/config"
)

// Metrics defines the minimal instrumentation surface for the basic GAuth token service.
// We intentionally keep this interface tiny to allow easy adaptation to different telemetry
// backends (Prometheus, OpenTelemetry, in-memory test collector, etc.) without importing
// heavy dependencies here.
type Metrics interface {
	IncTokensIssued()
	IncTokenValidations()
	IncTokenValidationFailures()
}

// NoopMetrics is a do-nothing implementation used when instrumentation is not supplied.
var NoopMetrics Metrics = noopMetrics{}

type noopMetrics struct{}

func (n noopMetrics) IncTokensIssued()            {}
func (n noopMetrics) IncTokenValidations()        {}
func (n noopMetrics) IncTokenValidationFailures() {}

// Config represents the configuration for GAuth
type Config struct {
	AuthServerURL     string
	ClientID          string
	ClientSecret      string
	SigningKey        string // NEW: distinct HMAC signing key (beta demo; replace for production)
	Scopes            []string
	AccessTokenExpiry time.Duration
	RateLimit         interface{} // Placeholder for rate limit config
	Audience          []string    // Accepted audiences (optional)

	// Centralized App Config
	AppConfig *config.Config
}

// AuthorizationRequest represents an authorization request
type AuthorizationRequest struct {
	ClientID string
	Scopes   []string
}

// AuthorizationGrant represents an authorization grant
type AuthorizationGrant struct {
	GrantID      string
	Scope        []string
	ValidUntil   time.Time
	Restrictions interface{}
	ClientID     string // Add ClientID field
}

// TokenRequest represents a token request
type TokenRequest struct {
	GrantID      string
	Scope        []string
	Restrictions interface{}
	Context      interface{}
	// RFC 9396
	AuthorizationDetails []AuthorizationDetail `json:"authorization_details,omitempty"`
}

// TokenResponse represents a token response
type TokenResponse struct {
	Token      string
	Scope      []string
	ValidUntil time.Time
}

// TokenValidationResult represents the result of token validation
type TokenValidationResult struct {
	ClientID string
	Scope    []string
	Valid    bool
}

// TransactionType represents different types of transactions
type TransactionType string

const (
	PaymentTransaction    TransactionType = "payment"
	TransferTransaction   TransactionType = "transfer"
	WithdrawalTransaction TransactionType = "withdrawal"
	DepositTransaction    TransactionType = "deposit"
)

// TransactionStatus represents transaction status
type TransactionStatus string

const (
	TransactionPending   TransactionStatus = "pending"
	TransactionCompleted TransactionStatus = "completed"
	TransactionFailed    TransactionStatus = "failed"
	// TransactionCanceled represents a transaction that was canceled by the user.
	TransactionCanceled TransactionStatus = "canceled"
	// TransactionCancelled is kept as a backwards-compatible alias (deprecated).
	// Deprecated: use TransactionCanceled.
	TransactionCancelled = TransactionCanceled
)

// TransactionDetails represents transaction details
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

// Restriction represents authorization restrictions
type Restriction struct {
	Type   string
	Value  interface{}
	Scopes []string
}
