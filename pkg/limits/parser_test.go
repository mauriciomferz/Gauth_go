package limits

import (
	"testing"
	"time"
)

func TestParseRateLimit(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    RateLimit
		wantErr bool
	}{
		// Valid formats
		{"basic minute", "100/minute", RateLimit{100, time.Minute}, false},
		{"basic hour", "1000/hour", RateLimit{1000, time.Hour}, false},
		{"basic day", "50000/day", RateLimit{50000, 24 * time.Hour}, false},
		{"basic week", "10000/week", RateLimit{10000, 7 * 24 * time.Hour}, false},
		{"basic month", "1000000/month", RateLimit{1000000, 30 * 24 * time.Hour}, false},

		// Plural forms
		{"plural minutes", "100/minutes", RateLimit{100, time.Minute}, false},
		{"plural hours", "1000/hours", RateLimit{1000, time.Hour}, false},
		{"plural days", "50000/days", RateLimit{50000, 24 * time.Hour}, false},

		// Abbreviations
		{"abbrev min", "100/min", RateLimit{100, time.Minute}, false},
		{"abbrev hr", "1000/hr", RateLimit{1000, time.Hour}, false},
		{"abbrev h", "1000/h", RateLimit{1000, time.Hour}, false},
		{"abbrev d", "50000/d", RateLimit{50000, 24 * time.Hour}, false},
		{"abbrev w", "10000/w", RateLimit{10000, 7 * 24 * time.Hour}, false},
		{"abbrev mo", "1000000/mo", RateLimit{1000000, 30 * 24 * time.Hour}, false},

		// K/M multipliers
		{"5K minute", "5K/minute", RateLimit{5000, time.Minute}, false},
		{"10k hour (lowercase)", "10k/hour", RateLimit{10000, time.Hour}, false},
		{"1M day", "1M/day", RateLimit{1000000, 24 * time.Hour}, false},
		{"2m month (lowercase)", "2m/month", RateLimit{2000000, 30 * 24 * time.Hour}, false},

		// Decimal multipliers
		{"1.5K hour", "1.5K/hour", RateLimit{1500, time.Hour}, false},
		{"0.5M day", "0.5M/day", RateLimit{500000, 24 * time.Hour}, false},
		{"2.5k minute", "2.5k/minute", RateLimit{2500, time.Minute}, false},

		// Whitespace handling
		{"spaces before", "  100/minute", RateLimit{100, time.Minute}, false},
		{"spaces after", "100/minute  ", RateLimit{100, time.Minute}, false},
		{"spaces around slash", "100 / minute", RateLimit{100, time.Minute}, false},
		{"spaces everywhere", "  100  /  minute  ", RateLimit{100, time.Minute}, false},

		// Case insensitivity
		{"uppercase period", "100/MINUTE", RateLimit{100, time.Minute}, false},
		{"mixed case period", "100/Hour", RateLimit{100, time.Hour}, false},
		{"uppercase K", "5K/day", RateLimit{5000, 24 * time.Hour}, false},
		{"uppercase M", "1M/month", RateLimit{1000000, 30 * 24 * time.Hour}, false},

		// Error cases
		{"empty string", "", RateLimit{}, true},
		{"missing slash", "100minute", RateLimit{}, true},
		{"missing value", "/minute", RateLimit{}, true},
		{"missing period", "100/", RateLimit{}, true},
		{"invalid number", "abc/minute", RateLimit{}, true},
		{"invalid period", "100/year", RateLimit{}, true},
		{"negative number", "-100/minute", RateLimit{}, true},
		{"invalid multiplier", "100X/minute", RateLimit{}, true},
		{"multiple slashes", "100/minute/hour", RateLimit{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRateLimit(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRateLimit(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && (got.Limit != tt.want.Limit || got.Period != tt.want.Period) {
				t.Errorf("ParseRateLimit(%q) = {%d, %v}, want {%d, %v}", tt.input, got.Limit, got.Period, tt.want.Limit, tt.want.Period)
			}
		})
	}
}

func TestFormatPeriod(t *testing.T) {
	tests := []struct {
		period time.Duration
		want   string
	}{
		{time.Minute, "minute"},
		{time.Hour, "hour"},
		{24 * time.Hour, "day"},
		{7 * 24 * time.Hour, "week"},
		{30 * 24 * time.Hour, "month"},
		{2 * time.Minute, "2m0s"}, // fallback to Duration.String()
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatPeriod(tt.period)
			if got != tt.want {
				t.Errorf("FormatPeriod(%v) = %q, want %q", tt.period, got, tt.want)
			}
		})
	}
}

// TestParseNumericWithMultiplier tests the internal numeric parsing logic
func TestParseNumericWithMultiplier(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"100", 100, false},
		{"1000", 1000, false},
		{"5K", 5000, false},
		{"5k", 5000, false},
		{"10K", 10000, false},
		{"1M", 1000000, false},
		{"1m", 1000000, false},
		{"2M", 2000000, false},
		{"1.5K", 1500, false},
		{"0.5M", 500000, false},
		{"2.5K", 2500, false},
		{"0.25M", 250000, false},
		{"  100  ", 100, false},
		{"abc", 0, true},
		{"", 0, true},
		{"-100", 0, true},
		{"100X", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseNumericWithMultiplier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNumericWithMultiplier(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseNumericWithMultiplier(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestParsePeriod tests the internal period parsing logic
func TestParsePeriod(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"minute", time.Minute, false},
		{"minutes", time.Minute, false},
		{"min", time.Minute, false},
		{"hour", time.Hour, false},
		{"hours", time.Hour, false},
		{"hr", time.Hour, false},
		{"h", time.Hour, false},
		{"day", 24 * time.Hour, false},
		{"days", 24 * time.Hour, false},
		{"d", 24 * time.Hour, false},
		{"week", 7 * 24 * time.Hour, false},
		{"weeks", 7 * 24 * time.Hour, false},
		{"wk", 7 * 24 * time.Hour, false},
		{"w", 7 * 24 * time.Hour, false},
		{"month", 30 * 24 * time.Hour, false},
		{"months", 30 * 24 * time.Hour, false},
		{"mo", 30 * 24 * time.Hour, false},
		{"MINUTE", time.Minute, false},
		{"Hour", time.Hour, false},
		{"  day  ", 24 * time.Hour, false},
		{"year", 0, true},
		{"second", 0, true},
		{"", 0, true},
		{"unknown", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parsePeriod(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePeriod(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parsePeriod(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
