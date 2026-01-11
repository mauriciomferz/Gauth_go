package agentauth_aap_001

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/aap"
)

// MockLimitStore implements LimitStore for testing
type MockLimitStore struct {
	Usage map[string]float64
}

func NewMockLimitStore() *MockLimitStore {
	return &MockLimitStore{
		Usage: make(map[string]float64),
	}
}

func (m *MockLimitStore) GetPeriodUsage(delegationID, periodKey string) (float64, error) {
	key := fmt.Sprintf("%s:%s", delegationID, periodKey)
	return m.Usage[key], nil
}

func (m *MockLimitStore) IncrementPeriodUsage(delegationID, periodKey string, amount float64) error {
	key := fmt.Sprintf("%s:%s", delegationID, periodKey)
	m.Usage[key] += amount
	return nil
}

func (m *MockLimitStore) ResetPeriodUsage(delegationID, periodKey string) error {
	return nil
}

func (m *MockLimitStore) GetDailyUsage(delegationID, date string) (float64, error) {
	return m.GetPeriodUsage(delegationID, date)
}

func (m *MockLimitStore) IncrementDailyUsage(delegationID, date string, amount float64) error {
	return m.IncrementPeriodUsage(delegationID, date, amount)
}

func (m *MockLimitStore) ResetDailyUsage(delegationID, date string) error {
	return nil
}

func (m *MockLimitStore) ExportDailyLimits(ctx context.Context) (map[string]map[string]float64, error) {
	return nil, nil
}

func TestFinancialLimits_MultiPeriod(t *testing.T) {
	now := time.Now()
	today := now.Format("2006-01-02")
	year, week := now.ISOWeek()
	thisWeek := fmt.Sprintf("%d-W%02d", year, week)
	thisMonth := now.Format("2006-01")

	tests := []struct {
		name         string
		restrictions map[string]string
		setupUsage   func(*MockLimitStore, string)
		expectError  bool
		errorSubstr  string
	}{
		{
			name: "Daily Limit Exceeded",
			restrictions: map[string]string{
				"max_daily_amount": "100",
			},
			setupUsage: func(s *MockLimitStore, id string) {
				_ = s.IncrementPeriodUsage(id, today, 150)
			},
			expectError: true,
			errorSubstr: "daily limit exceeded",
		},
		{
			name: "Daily Limit OK",
			restrictions: map[string]string{
				"max_daily_amount": "100",
			},
			setupUsage: func(s *MockLimitStore, id string) {
				_ = s.IncrementPeriodUsage(id, today, 50)
			},
			expectError: false,
		},
		{
			name: "Weekly Limit Exceeded",
			restrictions: map[string]string{
				"max_weekly_amount": "500",
			},
			setupUsage: func(s *MockLimitStore, id string) {
				_ = s.IncrementPeriodUsage(id, thisWeek, 600)
			},
			expectError: true,
			errorSubstr: "weekly limit exceeded",
		},
		{
			name: "Weekly Limit OK",
			restrictions: map[string]string{
				"max_weekly_amount": "500",
			},
			setupUsage: func(s *MockLimitStore, id string) {
				_ = s.IncrementPeriodUsage(id, thisWeek, 100)
			},
			expectError: false,
		},
		{
			name: "Monthly Limit Exceeded",
			restrictions: map[string]string{
				"max_monthly_amount": "2000",
			},
			setupUsage: func(s *MockLimitStore, id string) {
				_ = s.IncrementPeriodUsage(id, thisMonth, 2100)
			},
			expectError: true,
			errorSubstr: "monthly limit exceeded",
		},
		{
			name: "All Limits OK",
			restrictions: map[string]string{
				"max_daily_amount":   "100",
				"max_weekly_amount":  "500",
				"max_monthly_amount": "2000",
			},
			setupUsage: func(s *MockLimitStore, id string) {
				_ = s.IncrementPeriodUsage(id, today, 50)
				_ = s.IncrementPeriodUsage(id, thisWeek, 200)
				_ = s.IncrementPeriodUsage(id, thisMonth, 1000)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockLimitStore()
			validator := NewEnhancedPoAValidator(WithLimitStore(store))

			poa := &PowerOfAttorney{
				ID:           "test-poa",
				Grantor:      "alice",
				Grantee:      "bob",
				Scope:        []string{"transaction:payment"},
				Restrictions: tt.restrictions,
				ValidFrom:    now,
				ValidUntil:   now.Add(24 * time.Hour),
			}
			// Add requires financial restrictions
			poa.Restrictions["currency"] = "USD"
			poa.Restrictions["max_amount"] = "10"

			if tt.setupUsage != nil {
				tt.setupUsage(store, poa.ID)
			}

			err := validator.Validate(poa)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else {
					if e, ok := err.(aap.RFCError); !ok {
						t.Errorf("Expected aap.Error, got %T", err)
					} else if len(tt.errorSubstr) > 0 {
						if !strings.Contains(e.Error(), tt.errorSubstr) {
							t.Errorf("Expected error containing %q, got %q", tt.errorSubstr, err.Error())
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestFinancialLimits_CurrencyValidation(t *testing.T) {
	now := time.Now()
	store := NewMockLimitStore()
	mockConverter := NewMockCurrencyConverter()

	// Define supported currencies
	mockConverter.ConversionFunc = func(amount float64, from, to string) (float64, error) {
		return amount, nil
	}
	mockConverter.ValidCurrencies["USD"] = true
	mockConverter.ValidCurrencies["EUR"] = true

	// Note: Without converter, Default Basic Validator logic applies (simple regex check usually, but here we enforce check)
	// EnhancedValidator with converter enforces strict check

	tests := []struct {
		name        string
		currency    string
		converter   CurrencyConverter
		expectError bool
		errorSubstr string
	}{
		{
			name:        "Supported Currency",
			currency:    "USD",
			converter:   mockConverter,
			expectError: false,
		},
		{
			name:        "Unsupported Currency",
			currency:    "XYZ",
			converter:   mockConverter,
			expectError: true,
			errorSubstr: "unsupported currency code: XYZ",
		},
		{
			name:        "No Converter (Fallback)",
			currency:    "GBP", // Valid by regex but unknown to mock
			converter:   nil,
			expectError: false, // Fallback validation allows standard codes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewEnhancedPoAValidator(
				WithLimitStore(store),
				WithCurrencyConverter(tt.converter),
			)

			poa := &PowerOfAttorney{
				ID:      "test-currency-" + tt.name,
				Grantor: "alice",
				Grantee: "bob",
				Scope:   []string{"transaction:payment"},
				Restrictions: map[string]string{
					"currency":   tt.currency,
					"max_amount": "100",
				},
				ValidFrom:  now,
				ValidUntil: now.Add(24 * time.Hour),
			}

			err := validator.Validate(poa)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else {
					if e, ok := err.(aap.RFCError); !ok {
						t.Errorf("Expected aap.Error, got %T", err)
					} else if len(tt.errorSubstr) > 0 {
						if !strings.Contains(e.Error(), tt.errorSubstr) {
							t.Errorf("Expected error containing %q, got %q", tt.errorSubstr, err.Error())
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}
