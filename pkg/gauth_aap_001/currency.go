package gauth_aap_001

import (
	"fmt"
	"strings"
)

// CurrencyConverter handles currency validation and conversion
type CurrencyConverter interface {
	// Convert converts an amount from one currency to another
	Convert(amount float64, from, to string) (float64, error)
	// IsValidCurrency checks if a currency code is supported
	IsValidCurrency(code string) bool
}

// StaticCurrencyConverter implements CurrencyConverter with fixed rates
type StaticCurrencyConverter struct {
	BaseCurrency string
	Rates        map[string]float64 // Rate relative to BaseCurrency (e.g. EUR/USD = 1.1)
}

// NewStaticCurrencyConverter creates a converter with default rates
func NewStaticCurrencyConverter() *StaticCurrencyConverter {
	return &StaticCurrencyConverter{
		BaseCurrency: "USD",
		Rates: map[string]float64{
			"USD": 1.0,
			"EUR": 1.1,   // 1 EUR = 1.1 USD
			"GBP": 1.3,   // 1 GBP = 1.3 USD
			"JPY": 0.007, // 1 JPY = 0.007 USD
		},
	}
}

// Convert converts an amount from source to target currency
func (c *StaticCurrencyConverter) Convert(amount float64, from, to string) (float64, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)

	if !c.IsValidCurrency(from) {
		return 0, fmt.Errorf("unsupported source currency: %s", from)
	}
	if !c.IsValidCurrency(to) {
		return 0, fmt.Errorf("unsupported target currency: %s", to)
	}

	if from == to {
		return amount, nil
	}

	// Convert to base currency first
	fromRate := c.Rates[from]
	baseAmount := amount * fromRate

	// Convert from base currency to target
	toRate := c.Rates[to]
	if toRate == 0 {
		return 0, fmt.Errorf("invalid rate for target currency: %s", to)
	}

	return baseAmount / toRate, nil
}

// IsValidCurrency checks if the currency is in the rates map
func (c *StaticCurrencyConverter) IsValidCurrency(code string) bool {
	_, exists := c.Rates[strings.ToUpper(code)]
	return exists
}

// MockCurrencyConverter for testing
type MockCurrencyConverter struct {
	ValidCurrencies map[string]bool
	ConversionFunc  func(amount float64, from, to string) (float64, error)
}

func NewMockCurrencyConverter() *MockCurrencyConverter {
	return &MockCurrencyConverter{
		ValidCurrencies: make(map[string]bool),
	}
}

func (m *MockCurrencyConverter) Convert(amount float64, from, to string) (float64, error) {
	if m.ConversionFunc != nil {
		return m.ConversionFunc(amount, from, to)
	}
	return amount, nil // 1:1 default
}

func (m *MockCurrencyConverter) IsValidCurrency(code string) bool {
	return m.ValidCurrencies[strings.ToUpper(code)]
}
