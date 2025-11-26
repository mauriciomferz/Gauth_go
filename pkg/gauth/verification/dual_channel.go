package verification

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Challenge represents a verification challenge sent via dual channels.
type Challenge struct {
	ID        string
	Code      string
	CreatedAt time.Time
	ExpiresAt time.Time
	PoAID     string
	Principal PrincipalContact
	Confirmed bool
}

// PrincipalContact contains contact information for out-of-band verification.
type PrincipalContact struct {
	PhoneNumber string // E.164 format: +1234567890
	Email       string
	Name        string
}

// DualChannelVerifier requires out-of-band confirmation via SMS + Email.
type DualChannelVerifier struct {
	smsGateway   SMSGateway
	emailService EmailService
	challenges   sync.Map // map[string]*Challenge (thread-safe)
	codeLength   int
	expiryTime   time.Duration
}

// SMSGateway interface for sending SMS messages.
type SMSGateway interface {
	SendSMS(ctx context.Context, phoneNumber, message string) error
}

// EmailService interface for sending email messages.
type EmailService interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// NewDualChannelVerifier creates a new dual-channel verifier.
func NewDualChannelVerifier(sms SMSGateway, email EmailService) *DualChannelVerifier {
	return &DualChannelVerifier{
		smsGateway:   sms,
		emailService: email,
		codeLength:   8,
		expiryTime:   5 * time.Minute,
	}
}

// RequestVerification initiates dual-channel verification for PoA creation.
func (d *DualChannelVerifier) RequestVerification(ctx context.Context, poaID string, principal PrincipalContact) (string, error) {
	code, err := generateSecureCode(d.codeLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate challenge code: %w", err)
	}

	challengeID := generateChallengeID()
	challenge := &Challenge{
		ID:        challengeID,
		Code:      code,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(d.expiryTime),
		PoAID:     poaID,
		Principal: principal,
		Confirmed: false,
	}

	d.challenges.Store(challengeID, challenge)

	smsMessage := fmt.Sprintf("GAuth Security: Confirm Power of Attorney creation with code: %s (expires in 5 min)", code)
	if err := d.smsGateway.SendSMS(ctx, principal.PhoneNumber, smsMessage); err != nil {
		d.challenges.Delete(challengeID)
		return "", fmt.Errorf("failed to send SMS: %w", err)
	}

	emailSubject := "GAuth: Confirm Power of Attorney Creation"
	emailBody := fmt.Sprintf("Verification Code: %s\nThis code expires in 5 minutes.\nPoA ID: %s", code, poaID)

	if err := d.emailService.SendEmail(ctx, principal.Email, emailSubject, emailBody); err != nil {
		d.challenges.Delete(challengeID)
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	time.AfterFunc(d.expiryTime, func() {
		d.challenges.Delete(challengeID)
	})

	return challengeID, nil
}

// ConfirmVerification verifies the user-provided code matches the challenge.
func (d *DualChannelVerifier) ConfirmVerification(challengeID, userCode string) error {
	value, ok := d.challenges.Load(challengeID)
	if !ok {
		return fmt.Errorf("challenge not found or expired")
	}

	challenge := value.(*Challenge)

	if time.Now().After(challenge.ExpiresAt) {
		d.challenges.Delete(challengeID)
		return fmt.Errorf("challenge expired")
	}

	expectedCode := normalizeCode(challenge.Code)
	providedCode := normalizeCode(userCode)

	if subtle.ConstantTimeCompare([]byte(expectedCode), []byte(providedCode)) != 1 {
		return fmt.Errorf("invalid verification code")
	}

	challenge.Confirmed = true
	d.challenges.Store(challengeID, challenge)
	d.challenges.Delete(challengeID)

	return nil
}

// IsConfirmed checks if a challenge has been successfully confirmed.
func (d *DualChannelVerifier) IsConfirmed(challengeID string) bool {
	value, ok := d.challenges.Load(challengeID)
	if !ok {
		return false
	}
	challenge := value.(*Challenge)
	return challenge.Confirmed
}

// generateSecureCode generates a cryptographically secure random code.
func generateSecureCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	code := base32.StdEncoding.EncodeToString(bytes)
	code = code[:length]

	if length >= 4 {
		mid := length / 2
		code = code[:mid] + "-" + code[mid:]
	}

	return code, nil
}

// generateChallengeID generates a unique challenge identifier.
func generateChallengeID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("ch_%x", bytes)
}

// normalizeCode removes formatting characters for comparison.
func normalizeCode(code string) string {
	code = strings.ToUpper(code)
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return code
}

// MaskPhoneNumber masks a phone number for display.
func MaskPhoneNumber(phone string) string {
	if len(phone) <= 7 {
		return "***"
	}
	return phone[:4] + "***" + phone[len(phone)-4:]
}

// MaskEmail masks an email address for display.
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || len(parts[0]) <= 2 {
		return "***@***"
	}
	user := parts[0]
	return user[:1] + "***" + user[len(user)-1:] + "@" + parts[1]
}
