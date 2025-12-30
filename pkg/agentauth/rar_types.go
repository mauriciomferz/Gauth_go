package agentauth

// AuthorizationDetail represents a structured authorization object as per RFC 9396.
// It allows for fine-grained, resource-specific permissions.
type AuthorizationDetail struct {
	// Type of the authorization data (e.g., "payment_initiation", "patient_record_access")
	Type string `json:"type"`

	// Actions authorized on the resource
	Actions []string `json:"actions,omitempty"`

	// Locations (Resource/API endpoints) where the access is valid
	Locations []string `json:"locations,omitempty"`

	// DataTypes specific to the resource (e.g., "lab_results", "meta_data")
	DataTypes []string `json:"datatypes,omitempty"`

	// Identifier for specific resource instance
	Identifier string `json:"identifier,omitempty"`

	// Privileges or roles associated with the access
	Privileges []string `json:"privileges,omitempty"`

	// InstructedAmount for financial transactions (RFC 9396 example extension)
	InstructedAmount *Amount `json:"instructedAmount,omitempty"`

	// CreditorAccount for payment contexts
	CreditorAccount *Account `json:"creditorAccount,omitempty"`

	// Catch-all for other extensions
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Amount represents a monetary value with currency.
type Amount struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"` // String to avoid float precision issues
}

// Account represents a financial account identifier.
type Account struct {
	IBAN string `json:"iban,omitempty"`
	BIC  string `json:"bic,omitempty"`
}

// MarshalJSON custom marshaler to flatten metadata if needed, or standard default.
// Using standard default for now.
