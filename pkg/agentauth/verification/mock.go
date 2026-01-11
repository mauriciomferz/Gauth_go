package verification

import (
	"context"
	"fmt"
)

// ⚠️ PRODUCTION PLACEHOLDER ⚠️
//
// This is a MOCK implementation for testing purposes only.
// DO NOT use in production without implementing real SMS/Email gateways.
//
// Production alternatives:
//   - SMS: Twilio, AWS SNS, MessageBird
//   - Email: SendGrid, AWS SES, Mailgun
//
// Security Requirements:
//   - Use TLS/SSL for all gateway connections
//   - Implement rate limiting (prevent SMS/email flooding)
//   - Log all verification attempts for audit trail
//   - Monitor for suspicious patterns (mass verification requests)

// MockSMSGateway is a mock SMS gateway for testing.
type MockSMSGateway struct {
	sentMessages []SMSMessage
}

// SMSMessage represents a sent SMS message.
type SMSMessage struct {
	PhoneNumber string
	Message     string
	SentAt      string
}

// NewMockSMSGateway creates a new mock SMS gateway.
func NewMockSMSGateway() *MockSMSGateway {
	return &MockSMSGateway{
		sentMessages: make([]SMSMessage, 0),
	}
}

// SendSMS simulates sending an SMS message.
func (m *MockSMSGateway) SendSMS(ctx context.Context, phoneNumber, message string) error {
	// Validate phone number format (basic E.164 check)
	if len(phoneNumber) < 10 || phoneNumber[0] != '+' {
		return fmt.Errorf("invalid phone number format: %s (must be E.164 format, e.g., +1234567890)", phoneNumber)
	}

	// Store message
	m.sentMessages = append(m.sentMessages, SMSMessage{
		PhoneNumber: phoneNumber,
		Message:     message,
		SentAt:      ctx.Value("timestamp").(string),
	})

	// Simulate successful send
	fmt.Printf("[MOCK SMS] To: %s\nMessage: %s\n\n", phoneNumber, message)
	return nil
}

// GetSentMessages returns all sent messages (for testing).
func (m *MockSMSGateway) GetSentMessages() []SMSMessage {
	return m.sentMessages
}

// MockEmailService is a mock email service for testing.
type MockEmailService struct {
	sentEmails []EmailMessage
}

// EmailMessage represents a sent email message.
type EmailMessage struct {
	To      string
	Subject string
	Body    string
	SentAt  string
}

// NewMockEmailService creates a new mock email service.
func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		sentEmails: make([]EmailMessage, 0),
	}
}

// SendEmail simulates sending an email message.
func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	// Validate email format (basic check)
	if !isValidEmail(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	// Store email
	m.sentEmails = append(m.sentEmails, EmailMessage{
		To:      to,
		Subject: subject,
		Body:    body,
		SentAt:  ctx.Value("timestamp").(string),
	})

	// Simulate successful send
	fmt.Printf("[MOCK EMAIL] To: %s\nSubject: %s\nBody:\n%s\n\n", to, subject, body)
	return nil
}

// GetSentEmails returns all sent emails (for testing).
func (m *MockEmailService) GetSentEmails() []EmailMessage {
	return m.sentEmails
}

// isValidEmail performs basic email validation.
func isValidEmail(email string) bool {
	// Very basic check - production should use proper email validation library
	return len(email) > 3 && containsChar(email, '@') && containsChar(email, '.')
}

// containsChar checks if string contains a character.
func containsChar(s string, c rune) bool {
	for _, char := range s {
		if char == c {
			return true
		}
	}
	return false
}

// MockMultiChannelNotifier sends notifications via mock channels.
type MockMultiChannelNotifier struct {
	smsGateway   SMSGateway
	emailService EmailService
}

// NewMockMultiChannelNotifier creates a new mock multi-channel notifier.
func NewMockMultiChannelNotifier(sms SMSGateway, email EmailService) *MockMultiChannelNotifier {
	return &MockMultiChannelNotifier{
		smsGateway:   sms,
		emailService: email,
	}
}

// SendNotification sends a notification via both SMS and Email.
func (m *MockMultiChannelNotifier) SendNotification(
	ctx context.Context,
	recipient PrincipalContact,
	subject, message string,
) error {
	// Send via SMS
	smsMessage := fmt.Sprintf("%s: %s", subject, message)
	if err := m.smsGateway.SendSMS(ctx, recipient.PhoneNumber, smsMessage); err != nil {
		return fmt.Errorf("failed to send SMS notification: %w", err)
	}

	// Send via Email
	if err := m.emailService.SendEmail(ctx, recipient.Email, subject, message); err != nil {
		return fmt.Errorf("failed to send email notification: %w", err)
	}

	return nil
}

// MockPoARegistry is a mock in-memory PoA registry for testing.
type MockPoARegistry struct {
	poas map[string]*PoAData
}

// NewMockPoARegistry creates a new mock PoA registry.
func NewMockPoARegistry() *MockPoARegistry {
	return &MockPoARegistry{
		poas: make(map[string]*PoAData),
	}
}

// Store stores a PoA in the registry.
func (m *MockPoARegistry) Store(ctx context.Context, poa *PoAData) error {
	if poa.ID == "" {
		return fmt.Errorf("PoA ID cannot be empty")
	}
	m.poas[poa.ID] = poa
	return nil
}

// Get retrieves a PoA from the registry.
func (m *MockPoARegistry) Get(ctx context.Context, poaID string) (*PoAData, error) {
	poa, ok := m.poas[poaID]
	if !ok {
		return nil, fmt.Errorf("PoA not found: %s", poaID)
	}
	return poa, nil
}

// UpdateStatus updates the status of a PoA.
func (m *MockPoARegistry) UpdateStatus(ctx context.Context, poaID string, status PoAStatus) error {
	poa, ok := m.poas[poaID]
	if !ok {
		return fmt.Errorf("PoA not found: %s", poaID)
	}
	poa.Status = status
	return nil
}

// GetAll returns all PoAs (for testing).
func (m *MockPoARegistry) GetAll() []*PoAData {
	poas := make([]*PoAData, 0, len(m.poas))
	for _, poa := range m.poas {
		poas = append(poas, poa)
	}
	return poas
}
