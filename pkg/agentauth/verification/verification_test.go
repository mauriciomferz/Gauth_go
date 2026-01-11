package verification

import (
	"context"
	"strings"
	"testing"
	"time"
)

type timestampCtxKey struct{}

func TestDualChannelVerifier_RequestVerification(t *testing.T) {
	ctx := context.WithValue(context.Background(), timestampCtxKey{}, time.Now().Format(time.RFC3339))

	sms := NewMockSMSGateway()
	email := NewMockEmailService()
	verifier := NewDualChannelVerifier(sms, email)

	principal := PrincipalContact{
		PhoneNumber: "+1234567890",
		Email:       "principal@example.com",
		Name:        "Test Principal",
	}

	challengeID, err := verifier.RequestVerification(ctx, "poa_test_123", principal)
	if err != nil {
		t.Fatalf("RequestVerification failed: %v", err)
	}

	if challengeID == "" {
		t.Fatal("Expected non-empty challengeID")
	}

	sentSMS := sms.GetSentMessages()
	if len(sentSMS) != 1 {
		t.Fatalf("Expected 1 SMS message, got %d", len(sentSMS))
	}

	sentEmails := email.GetSentEmails()
	if len(sentEmails) != 1 {
		t.Fatalf("Expected 1 email, got %d", len(sentEmails))
	}
}

func TestDualChannelVerifier_ConfirmVerification(t *testing.T) {
	ctx := context.WithValue(context.Background(), timestampCtxKey{}, time.Now().Format(time.RFC3339))

	sms := NewMockSMSGateway()
	email := NewMockEmailService()
	verifier := NewDualChannelVerifier(sms, email)

	principal := PrincipalContact{
		PhoneNumber: "+1234567890",
		Email:       "principal@example.com",
		Name:        "Test Principal",
	}

	challengeID, err := verifier.RequestVerification(ctx, "poa_test_123", principal)
	if err != nil {
		t.Fatalf("RequestVerification failed: %v", err)
	}

	sentSMS := sms.GetSentMessages()
	smsText := sentSMS[0].Message
	codeStart := strings.Index(smsText, "code: ") + 6
	codeEnd := strings.Index(smsText[codeStart:], " ")
	if codeEnd == -1 {
		codeEnd = len(smsText) - codeStart
	}
	code := smsText[codeStart : codeStart+codeEnd]

	err = verifier.ConfirmVerification(challengeID, code)
	if err != nil {
		t.Errorf("ConfirmVerification failed with correct code: %v", err)
	}
}

func TestDualChannelVerifier_InvalidCode(t *testing.T) {
	ctx := context.WithValue(context.Background(), timestampCtxKey{}, time.Now().Format(time.RFC3339))

	sms := NewMockSMSGateway()
	email := NewMockEmailService()
	verifier := NewDualChannelVerifier(sms, email)

	principal := PrincipalContact{
		PhoneNumber: "+1234567890",
		Email:       "principal@example.com",
		Name:        "Test Principal",
	}

	challengeID, err := verifier.RequestVerification(ctx, "poa_test_123", principal)
	if err != nil {
		t.Fatalf("RequestVerification failed: %v", err)
	}

	err = verifier.ConfirmVerification(challengeID, "WRONG-CODE")
	if err == nil {
		t.Error("Expected error with invalid code, got nil")
	}
}

func TestMaskPhoneNumber(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{
			name:     "Standard US number",
			phone:    "+1234567890",
			expected: "+123***7890",
		},
		{
			name:     "Short number",
			phone:    "+123",
			expected: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskPhoneNumber(tt.phone)
			if result != tt.expected {
				t.Errorf("MaskPhoneNumber(%s) = %s, want %s", tt.phone, result, tt.expected)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "Standard email",
			email:    "user@example.com",
			expected: "u***r@example.com",
		},
		{
			name:     "Short username",
			email:    "ab@example.com",
			expected: "***@***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskEmail(tt.email)
			if result != tt.expected {
				t.Errorf("MaskEmail(%s) = %s, want %s", tt.email, result, tt.expected)
			}
		})
	}
}

func TestTimelockPoA_CreateWithDelay(t *testing.T) {
	ctx := context.WithValue(context.Background(), timestampCtxKey{}, time.Now().Format(time.RFC3339))

	registry := NewMockPoARegistry()
	sms := NewMockSMSGateway()
	email := NewMockEmailService()
	notifier := NewMockMultiChannelNotifier(sms, email)

	timelock := NewTimelockPoA(registry, notifier, 1*time.Hour)

	poa := &PoAData{
		ID:      "poa_test_123",
		Issuer:  "0xPrincipal123",
		Grantee: "0xAgent456",
		Scope:   "Trading on Uniswap V3",
		Principal: PrincipalContact{
			PhoneNumber: "+1234567890",
			Email:       "principal@example.com",
			Name:        "Test Principal",
		},
	}

	poaID, cancelURL, err := timelock.CreateWithDelay(ctx, poa)
	if err != nil {
		t.Fatalf("CreateWithDelay failed: %v", err)
	}

	if poaID == "" {
		t.Error("Expected non-empty poaID")
	}

	if !strings.Contains(cancelURL, poaID) {
		t.Errorf("Cancel URL should contain PoA ID: %s", cancelURL)
	}

	storedPoA, err := registry.Get(ctx, poaID)
	if err != nil {
		t.Fatalf("Failed to retrieve stored PoA: %v", err)
	}

	if storedPoA.Status != PoAStatusPending {
		t.Errorf("Expected status PENDING, got %s", storedPoA.Status)
	}
}

func TestTimelockPoA_CancelPoA(t *testing.T) {
	ctx := context.WithValue(context.Background(), timestampCtxKey{}, time.Now().Format(time.RFC3339))

	registry := NewMockPoARegistry()
	sms := NewMockSMSGateway()
	email := NewMockEmailService()
	notifier := NewMockMultiChannelNotifier(sms, email)

	timelock := NewTimelockPoA(registry, notifier, 1*time.Hour)

	poa := &PoAData{
		ID:      "poa_test_123",
		Issuer:  "0xPrincipal123",
		Grantee: "0xAgent456",
		Scope:   "Trading on Uniswap V3",
		Principal: PrincipalContact{
			PhoneNumber: "+1234567890",
			Email:       "principal@example.com",
			Name:        "Test Principal",
		},
	}

	poaID, _, err := timelock.CreateWithDelay(ctx, poa)
	if err != nil {
		t.Fatalf("CreateWithDelay failed: %v", err)
	}

	err = timelock.CancelPoA(ctx, poaID)
	if err != nil {
		t.Fatalf("CancelPoA failed: %v", err)
	}

	storedPoA, err := registry.Get(ctx, poaID)
	if err != nil {
		t.Fatalf("Failed to retrieve stored PoA: %v", err)
	}

	if storedPoA.Status != PoAStatusCancelled {
		t.Errorf("Expected status CANCELLED, got %s", storedPoA.Status)
	}
}

func TestGenerateSecureCode(t *testing.T) {
	code1, err := generateSecureCode(8)
	if err != nil {
		t.Fatalf("generateSecureCode failed: %v", err)
	}

	if len(code1) != 9 {
		t.Errorf("Expected code length 9 (with hyphen), got %d", len(code1))
	}

	if !strings.Contains(code1, "-") {
		t.Error("Expected code to contain hyphen")
	}

	code2, err := generateSecureCode(8)
	if err != nil {
		t.Fatalf("generateSecureCode failed: %v", err)
	}

	if code1 == code2 {
		t.Error("Expected different codes on subsequent calls")
	}
}
