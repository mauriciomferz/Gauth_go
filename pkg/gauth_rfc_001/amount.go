package gauth_rfc_001

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Amount represents a structured monetary value with currency (sec13.item2 P2).
// Examples: "100.00 USD", "50.50 EUR", "1000.00", "1,234.56 GBP"
type Amount struct {
	Value    float64 // Numeric value
	Currency string  // ISO 4217 currency code (e.g., "USD", "EUR", "GBP") or empty
}

// ParseAmount parses a structured amount string into Amount struct.
// Supported formats:
//   - "100.00" (value only, no currency)
//   - "100.00 USD" (value with currency)
//   - "1,234.56 EUR" (value with comma separators and currency)
//   - "1234.56" (value without separators)
//
// Currency codes are validated against a subset of ISO 4217.
func ParseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Amount{}, fmt.Errorf("empty amount string")
	}

	// Remove comma thousand separators
	s = strings.ReplaceAll(s, ",", "")

	// Pattern: optional negative sign, digits, optional decimal, optional currency
	// Examples: "100", "100.00", "100.00 USD", "-50.50 EUR"
	pattern := regexp.MustCompile(`^(-?\d+(?:\.\d+)?)\s*([A-Z]{3})?$`)
	matches := pattern.FindStringSubmatch(s)
	if matches == nil {
		return Amount{}, fmt.Errorf("invalid amount format: %s", s)
	}

	valueStr := matches[1]
	currency := matches[2]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return Amount{}, fmt.Errorf("invalid numeric value: %s (%w)", valueStr, err)
	}

	// Validate currency if provided
	if currency != "" && !isValidCurrency(currency) {
		return Amount{}, fmt.Errorf("invalid currency code: %s", currency)
	}

	return Amount{Value: value, Currency: currency}, nil
}

// String returns the canonical string representation (e.g., "100.00 USD").
func (a Amount) String() string {
	if a.Currency != "" {
		return fmt.Sprintf("%.2f %s", a.Value, a.Currency)
	}
	return fmt.Sprintf("%.2f", a.Value)
}

// isValidCurrency validates against a subset of common ISO 4217 currency codes.
// Extend this list as needed for production use.
func isValidCurrency(code string) bool {
	validCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true, "CNY": true,
		"AUD": true, "CAD": true, "CHF": true, "SEK": true, "NZD": true,
		"MXN": true, "SGD": true, "HKD": true, "NOK": true, "KRW": true,
		"TRY": true, "INR": true, "RUB": true, "BRL": true, "ZAR": true,
	}
	return validCurrencies[code]
}

// CompareAmounts checks if two amounts are compatible and returns comparison result.
// Returns:
//   - error if currencies mismatch (when both are specified)
//   - 0 if a == b
//   - -1 if a < b
//   - 1 if a > b
func CompareAmounts(a, b Amount) (int, error) {
	// Check currency compatibility
	if a.Currency != "" && b.Currency != "" && a.Currency != b.Currency {
		return 0, fmt.Errorf("currency mismatch: %s vs %s", a.Currency, b.Currency)
	}

	if a.Value == b.Value {
		return 0, nil
	}
	if a.Value < b.Value {
		return -1, nil
	}
	return 1, nil
}

// EnforceMaxAmount validates that requested amount does not exceed limit.
// Returns error if limit is exceeded or currencies are incompatible.
func EnforceMaxAmount(requested, limit Amount) error {
	cmp, err := CompareAmounts(requested, limit)
	if err != nil {
		return err
	}
	if cmp > 0 {
		return fmt.Errorf("amount %s exceeds limit %s", requested.String(), limit.String())
	}
	return nil
}
