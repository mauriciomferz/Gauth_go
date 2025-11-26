package gauth_rfc_001

import (
	"testing"
)

func TestParseAmount_ValidFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantVal  float64
		wantCurr string
		wantErr  bool
	}{
		{"plain value", "100.00", 100.00, "", false},
		{"value with currency", "100.00 USD", 100.00, "USD", false},
		{"with commas", "1,234.56 EUR", 1234.56, "EUR", false},
		{"no decimals", "500", 500.00, "", false},
		{"no decimals with currency", "500 GBP", 500.00, "GBP", false},
		{"negative value", "-50.50", -50.50, "", false},
		{"negative with currency", "-50.50 EUR", -50.50, "EUR", false},
		{"large value", "999999.99 JPY", 999999.99, "JPY", false},
		{"small decimal", "0.01 USD", 0.01, "USD", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amt, err := ParseAmount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseAmount(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if amt.Value != tt.wantVal {
					t.Errorf("ParseAmount(%q) value = %.2f, want %.2f", tt.input, amt.Value, tt.wantVal)
				}
				if amt.Currency != tt.wantCurr {
					t.Errorf("ParseAmount(%q) currency = %q, want %q", tt.input, amt.Currency, tt.wantCurr)
				}
			}
		})
	}
}

func TestParseAmount_InvalidFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"no number", "USD"},
		{"invalid currency", "100.00 XXX"},
		{"multiple currencies", "100 USD EUR"},
		{"text only", "hundred dollars"},
		{"invalid separator", "100.00.00"},
		{"currency before value", "USD 100"},
		{"two-letter currency", "100 US"},
		{"four-letter currency", "100 USDD"},
		{"lowercase currency", "100 usd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAmount(tt.input)
			if err == nil {
				t.Errorf("ParseAmount(%q) expected error, got nil", tt.input)
			}
		})
	}
}

func TestAmount_String(t *testing.T) {
	tests := []struct {
		name string
		amt  Amount
		want string
	}{
		{"value only", Amount{Value: 100.00}, "100.00"},
		{"with currency", Amount{Value: 100.00, Currency: "USD"}, "100.00 USD"},
		{"negative", Amount{Value: -50.50, Currency: "EUR"}, "-50.50 EUR"},
		{"zero", Amount{Value: 0, Currency: "GBP"}, "0.00 GBP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.amt.String()
			if got != tt.want {
				t.Errorf("Amount.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompareAmounts(t *testing.T) {
	tests := []struct {
		name    string
		a       Amount
		b       Amount
		want    int
		wantErr bool
	}{
		{"equal values no currency", Amount{Value: 100}, Amount{Value: 100}, 0, false},
		{"equal with currency", Amount{Value: 100, Currency: "USD"}, Amount{Value: 100, Currency: "USD"}, 0, false},
		{"a less than b", Amount{Value: 50, Currency: "USD"}, Amount{Value: 100, Currency: "USD"}, -1, false},
		{"a greater than b", Amount{Value: 150, Currency: "EUR"}, Amount{Value: 100, Currency: "EUR"}, 1, false},
		{"currency mismatch", Amount{Value: 100, Currency: "USD"}, Amount{Value: 100, Currency: "EUR"}, 0, true},
		{"one has currency, other doesn't", Amount{Value: 100}, Amount{Value: 100, Currency: "USD"}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareAmounts(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CompareAmounts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CompareAmounts() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEnforceMaxAmount(t *testing.T) {
	tests := []struct {
		name      string
		requested Amount
		limit     Amount
		wantErr   bool
	}{
		{"within limit", Amount{Value: 50, Currency: "USD"}, Amount{Value: 100, Currency: "USD"}, false},
		{"at limit", Amount{Value: 100, Currency: "USD"}, Amount{Value: 100, Currency: "USD"}, false},
		{"exceeds limit", Amount{Value: 150, Currency: "USD"}, Amount{Value: 100, Currency: "USD"}, true},
		{"currency mismatch", Amount{Value: 50, Currency: "USD"}, Amount{Value: 100, Currency: "EUR"}, true},
		{"no currency ok", Amount{Value: 50}, Amount{Value: 100}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnforceMaxAmount(tt.requested, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnforceMaxAmount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseAmount_RoundTrip verifies parsing and serialization are consistent.
func TestParseAmount_RoundTrip(t *testing.T) {
	inputs := []string{"100.00 USD", "50.50 EUR", "1234.56 GBP", "999.99"}
	for _, input := range inputs {
		amt, err := ParseAmount(input)
		if err != nil {
			t.Fatalf("ParseAmount(%q) failed: %v", input, err)
		}
		serialized := amt.String()
		amt2, err := ParseAmount(serialized)
		if err != nil {
			t.Fatalf("ParseAmount(%q) round-trip failed: %v", serialized, err)
		}
		if amt2.Value != amt.Value || amt2.Currency != amt.Currency {
			t.Errorf("round-trip mismatch: %v -> %s -> %v", amt, serialized, amt2)
		}
	}
}
