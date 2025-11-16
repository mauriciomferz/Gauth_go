package external

import (
	"context"
	"testing"
	"time"
)

// Test Brazil Connector - CPF Validation
func TestBrazilCPFValidation(t *testing.T) {
	config := &BrazilConnectorConfig{
		GovBrURL:       "https://sso.staging.acesso.gov.br",
		GovBrClientID:  "test_client",
		GovBrSecret:    "test_secret",
		ReceitaURL:     "https://api.receita.fazenda.gov.br",
		DETRANURL:      "https://api.detran.gov.br",
		RequestTimeout: 30 * time.Second,
	}

	connector, err := NewBrazilIdentityConnector(config)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}
	defer connector.Close()

	tests := []struct {
		name      string
		cpf       string
		wantValid bool
		wantError bool
	}{
		{"Valid CPF format", "12345678909", true, false},
		{"Invalid format - too short", "123456789", false, false},
		{"Invalid format - too long", "123456789012", false, false},
		{"Invalid format - letters", "12345678A09", false, false},
		{"All same digits", "11111111111", false, false},
		{"Empty CPF", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CPFRequest{
				CPF:  tt.cpf,
				Name: "Test User",
			}

			resp, err := connector.ValidateCPF(context.Background(), req)
			
			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if resp != nil && resp.Valid != tt.wantValid {
				t.Errorf("Expected Valid=%v, got %v", tt.wantValid, resp.Valid)
			}
		})
	}
}

// Test Canada Connector - SIN Validation (Luhn)
func TestCanadaSINValidation(t *testing.T) {
	config := &CanadaConnectorConfig{
		ServiceCanadaURL:      "https://api.servicecanada.gc.ca",
		ServiceCanadaKey:      "test_key",
		ProvincialServicesURL: "https://api.provincial.gc.ca",
		IRCCURL:               "https://api.ircc.gc.ca",
		RequestTimeout:        30 * time.Second,
	}

	connector, err := NewCanadaIdentityConnector(config)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}
	defer connector.Close()

	tests := []struct {
		name      string
		sin       string
		wantValid bool
		sinType   string
	}{
		{"Valid business SIN", "046454286", true, "Business"},  // SINs starting with 0 are Business
		{"Valid temporary SIN", "900000001", true, "Temporary"},
		{"Invalid Luhn check", "123456789", false, ""},
		{"Invalid format - too short", "12345678", false, ""},
		{"Invalid format - letters", "04645428A", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &SINRequest{
				SIN:         tt.sin,
				Name:        "John Doe",
				DateOfBirth: "1990-01-15",
			}

			resp, err := connector.ValidateSIN(context.Background(), req)
			
			if err != nil && tt.wantValid {
				t.Errorf("Unexpected error: %v", err)
			}
			if resp != nil && resp.Valid != tt.wantValid {
				t.Errorf("Expected Valid=%v, got %v", tt.wantValid, resp.Valid)
			}
			if resp != nil && tt.sinType != "" && resp.Type != tt.sinType {
				t.Errorf("Expected Type=%s, got %s", tt.sinType, resp.Type)
			}
		})
	}
}

// Test Mexico Connector - CURP Validation
func TestMexicoCURPValidation(t *testing.T) {
	config := &MexicoConnectorConfig{
		RENAPOURL:      "https://renapo.gob.mx/api",
		RENAPOAPIKey:   "test_key",
		SATURL:         "https://api.sat.gob.mx",
		INEURL:         "https://api.ine.mx",
		RequestTimeout: 30 * time.Second,
	}

	connector, err := NewMexicoIdentityConnector(config)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}
	defer connector.Close()

	tests := []struct {
		name      string
		curp      string
		wantValid bool
	}{
		{"Format validation", "GOTJ901015HDFRRL09", false}, // Invalid check digit but tests format parsing
		{"Invalid format - too short", "GOTJ901015HDFRRL0", false},
		{"Invalid format - too long", "GOTJ901015HDFRRL099", false},
		{"Format with lowercase", "gotj901015hdfrrl09", false}, // Uppercased but invalid check digit
		{"Invalid gender code", "GOTJ901015XDFRRL09", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CURPRequest{
				CURP: tt.curp,
				Name: "Test User",
			}

			resp, err := connector.ValidateCURP(context.Background(), req)
			
			if err != nil && tt.wantValid {
				t.Errorf("Unexpected error: %v", err)
			}
			if resp != nil && resp.Valid != tt.wantValid {
				t.Errorf("Expected Valid=%v, got %v (error: %s)", tt.wantValid, resp.Valid, resp.Error)
			}
		})
	}
}

// Test South Africa Connector - ID Number Validation (Luhn)
func TestSouthAfricaIDValidation(t *testing.T) {
	config := &SouthAfricaConnectorConfig{
		DHA_URL:        "https://api.dha.gov.za",
		DHA_APIKey:     "test_key",
		NATISURL:       "https://api.natis.gov.za",
		RequestTimeout: 30 * time.Second,
	}

	connector, err := NewSouthAfricaIdentityConnector(config)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}
	defer connector.Close()

	tests := []struct {
		name       string
		idNumber   string
		wantValid  bool
		wantGender string
		wantCitizen string
	}{
		{"Valid ID - Male Citizen", "9001085800083", true, "Male", "SA Citizen"},
		{"Valid ID - Female", "9001084800084", true, "Female", "SA Citizen"},
		{"Invalid format - too short", "900108580008", false, "", ""},
		{"Invalid Luhn check", "9001085800088", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &IDNumberRequest{
				IDNumber: tt.idNumber,
				Name:     "Test User",
			}

			resp, err := connector.ValidateIDNumber(context.Background(), req)
			
			if err != nil && tt.wantValid {
				t.Errorf("Unexpected error: %v", err)
			}
			if resp != nil {
				if resp.Valid != tt.wantValid {
					t.Errorf("Expected Valid=%v, got %v", tt.wantValid, resp.Valid)
				}
				if tt.wantGender != "" && resp.Gender != tt.wantGender {
					t.Errorf("Expected Gender=%s, got %s", tt.wantGender, resp.Gender)
				}
			}
		})
	}
}

// Test Nigeria Connector - NIN/BVN Validation
func TestNigeriaNINValidation(t *testing.T) {
	config := &NigeriaConnectorConfig{
		NIMCURL:        "https://api.nimc.gov.ng",
		NIMCAPIKey:     "test_key",
		BVNURL:         "https://api.bvn.ng",
		FRSCURL:        "https://api.frsc.gov.ng",
		RequestTimeout: 30 * time.Second,
	}

	connector, err := NewNigeriaIdentityConnector(config)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}
	defer connector.Close()

	tests := []struct {
		name      string
		nin       string
		wantValid bool
	}{
		{"Valid NIN format", "12345678901", true},
		{"Invalid format - too short", "1234567890", false},
		{"Invalid format - too long", "123456789012", false},
		{"Invalid format - letters", "1234567890A", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &NINRequest{
				NIN:       tt.nin,
				FirstName: "John",
				Surname:   "Doe",
			}

			resp, err := connector.ValidateNIN(context.Background(), req)
			
			if err != nil && tt.wantValid {
				t.Errorf("Unexpected error: %v", err)
			}
			if resp != nil && resp.Valid != tt.wantValid {
				t.Errorf("Expected Valid=%v, got %v", tt.wantValid, resp.Valid)
			}
		})
	}
}

// Test Kenya Connector - National ID Validation
func TestKenyaNationalIDValidation(t *testing.T) {
	config := &KenyaConnectorConfig{
		IPRSURL:        "https://api.iprs.go.ke",
		IPRSAPIKey:     "test_key",
		NTSAURL:        "https://api.ntsa.go.ke",
		HudumaURL:      "https://api.huduma.go.ke",
		RequestTimeout: 30 * time.Second,
	}

	connector, err := NewKenyaIdentityConnector(config)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}
	defer connector.Close()

	tests := []struct {
		name      string
		idNumber  string
		wantValid bool
	}{
		{"Valid 8-digit ID", "12345678", true},
		{"Valid 7-digit ID", "1234567", true},
		{"Invalid format - too short", "123456", false},
		{"Invalid format - too long", "123456789", false},
		{"Invalid format - letters", "1234567A", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &NationalIDRequest{
				IDNumber:  tt.idNumber,
				FirstName: "John",
				Surname:   "Doe",
			}

			resp, err := connector.ValidateNationalID(context.Background(), req)
			
			if err != nil && tt.wantValid {
				t.Errorf("Unexpected error: %v", err)
			}
			if resp != nil && resp.Valid != tt.wantValid {
				t.Errorf("Expected Valid=%v, got %v", tt.wantValid, resp.Valid)
			}
		})
	}
}

// Benchmark tests
func BenchmarkBrazilCPFValidation(b *testing.B) {
	config := &BrazilConnectorConfig{
		GovBrURL:       "https://sso.staging.acesso.gov.br",
		GovBrClientID:  "test",
		GovBrSecret:    "test",
		ReceitaURL:     "https://api.receita.fazenda.gov.br",
		DETRANURL:      "https://api.detran.gov.br",
		RequestTimeout: 30 * time.Second,
	}
	
	connector, _ := NewBrazilIdentityConnector(config)
	defer connector.Close()
	
	req := &CPFRequest{
		CPF:  "12345678909",
		Name: "Test User",
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = connector.ValidateCPF(ctx, req)
	}
}

func BenchmarkCanadaSINLuhn(b *testing.B) {
	config := &CanadaConnectorConfig{
		ServiceCanadaURL:      "https://api.servicecanada.gc.ca",
		ServiceCanadaKey:      "test",
		ProvincialServicesURL: "https://api.provincial.gc.ca",
		IRCCURL:               "https://api.ircc.gc.ca",
		RequestTimeout:        30 * time.Second,
	}
	
	connector, _ := NewCanadaIdentityConnector(config)
	defer connector.Close()
	
	req := &SINRequest{
		SIN:         "046454286",
		Name:        "John Doe",
		DateOfBirth: "1990-01-15",
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = connector.ValidateSIN(ctx, req)
	}
}

func BenchmarkSouthAfricaIDLuhn(b *testing.B) {
	config := &SouthAfricaConnectorConfig{
		DHA_URL:        "https://api.dha.gov.za",
		DHA_APIKey:     "test",
		NATISURL:       "https://api.natis.gov.za",
		RequestTimeout: 30 * time.Second,
	}
	
	connector, _ := NewSouthAfricaIdentityConnector(config)
	defer connector.Close()
	
	req := &IDNumberRequest{
		IDNumber: "9001085800083",
		Name:     "Test User",
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = connector.ValidateIDNumber(ctx, req)
	}
}
