package verification

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ⚠️ SECURITY NOTICE ⚠️
//
// This package implements TIME-DELAYED ACTIVATION to prevent CRITICAL-5 vulnerability.
//
// Defense Principle: 24-Hour Cancellation Window
//   - PoA created but NOT immediately active
//   - Principal receives multi-channel notifications
//   - If key was stolen, Principal can revoke before activation
//   - Attacker cannot immediately drain funds
//
// Real-World Precedent:
//   - Bank wire transfers: 24-hour cancellation window
//   - Smart contract timelocks: Compound, Uniswap governance
//   - Cold wallet delays: Hardware wallets require manual confirmation

// TimelockPoA implements time-delayed PoA activation with cancellation window.
type TimelockPoA struct {
	registry         PoARegistry
	notifier         MultiChannelNotifier
	defaultDelay     time.Duration
	pendingPoAs      sync.Map // map[string]*PendingPoA
	activationTimers sync.Map // map[string]*time.Timer
}

// PoARegistry interface for storing and retrieving PoAs.
type PoARegistry interface {
	Store(ctx context.Context, poa *PoAData) error
	Get(ctx context.Context, poaID string) (*PoAData, error)
	UpdateStatus(ctx context.Context, poaID string, status PoAStatus) error
}

// MultiChannelNotifier sends notifications via multiple channels.
type MultiChannelNotifier interface {
	SendNotification(ctx context.Context, recipient PrincipalContact, subject, message string) error
}

// PoAData represents a Power of Attorney with metadata.
type PoAData struct {
	ID             string
	Issuer         string
	Grantee        string
	Scope          string
	Constraints    interface{} // SemanticAllowList from constraints package
	CreatedAt      time.Time
	ActivationTime time.Time
	ExpiresAt      time.Time
	Status         PoAStatus
	Signature      []byte
	Principal      PrincipalContact
}

// PoAStatus represents the current status of a PoA.
type PoAStatus string

const (
	PoAStatusPending   PoAStatus = "PENDING"   // Created, waiting for activation
	PoAStatusActive    PoAStatus = "ACTIVE"    // Activated and usable
	PoAStatusCancelled PoAStatus = "CANCELLED" // Cancelled before activation
	PoAStatusRevoked   PoAStatus = "REVOKED"   // Revoked after activation
	PoAStatusExpired   PoAStatus = "EXPIRED"   // Expired naturally
)

// PendingPoA tracks a PoA awaiting activation.
type PendingPoA struct {
	PoA           *PoAData
	CancelURL     string
	NotifiedAt    time.Time
	RemindersSent int
}

// NewTimelockPoA creates a new timelock PoA manager.
func NewTimelockPoA(registry PoARegistry, notifier MultiChannelNotifier, defaultDelay time.Duration) *TimelockPoA {
	if defaultDelay == 0 {
		defaultDelay = 24 * time.Hour // Default: 24-hour delay
	}

	return &TimelockPoA{
		registry:     registry,
		notifier:     notifier,
		defaultDelay: defaultDelay,
	}
}

// CreateWithDelay creates a PoA with time-delayed activation.
// Returns the PoA ID and cancellation URL.
func (t *TimelockPoA) CreateWithDelay(ctx context.Context, poa *PoAData) (string, string, error) {
	// Set activation time
	poa.ActivationTime = time.Now().Add(t.defaultDelay)
	poa.Status = PoAStatusPending
	poa.CreatedAt = time.Now()

	// Store in registry
	if err := t.registry.Store(ctx, poa); err != nil {
		return "", "", fmt.Errorf("failed to store PoA: %w", err)
	}

	// Generate cancellation URL
	cancelURL := fmt.Sprintf("https://agentauth.example.com/cancel/%s", poa.ID)

	// Track as pending
	pending := &PendingPoA{
		PoA:        poa,
		CancelURL:  cancelURL,
		NotifiedAt: time.Now(),
	}
	t.pendingPoAs.Store(poa.ID, pending)

	// Send initial notification
	if err := t.sendActivationNotification(ctx, pending); err != nil {
		return "", "", fmt.Errorf("failed to send notification: %w", err)
	}

	// Schedule activation
	timer := time.AfterFunc(t.defaultDelay, func() {
		if err := t.activatePoA(context.Background(), poa.ID); err != nil {
			// Log error but don't block timer callback
			_ = err
		}
	})
	t.activationTimers.Store(poa.ID, timer)

	// Schedule reminder at 12 hours (halfway point)
	reminderDelay := t.defaultDelay / 2
	time.AfterFunc(reminderDelay, func() {
		if err := t.sendReminderNotification(context.Background(), poa.ID); err != nil {
			// Log error but don't block timer callback
			_ = err
		}
	})

	return poa.ID, cancelURL, nil
}

// CancelPoA cancels a pending PoA before activation.
func (t *TimelockPoA) CancelPoA(ctx context.Context, poaID string) error {
	// Retrieve pending PoA
	value, ok := t.pendingPoAs.Load(poaID)
	if !ok {
		return fmt.Errorf("PoA not found or already activated: %s", poaID)
	}

	pending := value.(*PendingPoA)

	// Check if already activated
	if pending.PoA.Status != PoAStatusPending {
		return fmt.Errorf("PoA cannot be cancelled (status: %s)", pending.PoA.Status)
	}

	// Cancel activation timer
	if timerValue, ok := t.activationTimers.Load(poaID); ok {
		timer := timerValue.(*time.Timer)
		timer.Stop()
		t.activationTimers.Delete(poaID)
	}

	// Update status
	if err := t.registry.UpdateStatus(ctx, poaID, PoAStatusCancelled); err != nil {
		return fmt.Errorf("failed to update PoA status: %w", err)
	}

	// Update local state
	pending.PoA.Status = PoAStatusCancelled
	t.pendingPoAs.Delete(poaID)

	// Send cancellation confirmation
	subject := "AgentAuth: Power of Attorney Cancelled"
	message := fmt.Sprintf(`
Your Power of Attorney (ID: %s) has been successfully cancelled.

If you did not request this cancellation, your account may be compromised.
Please contact security@agentauth.example.com immediately.

Cancelled at: %s
`, poaID, time.Now().Format(time.RFC3339))

	if err := t.notifier.SendNotification(ctx, pending.PoA.Principal, subject, message); err != nil {
		return fmt.Errorf("failed to send cancellation notification: %w", err)
	}

	return nil
}

// activatePoA activates a pending PoA after the delay period.
func (t *TimelockPoA) activatePoA(ctx context.Context, poaID string) error {
	// Retrieve pending PoA
	value, ok := t.pendingPoAs.Load(poaID)
	if !ok {
		return fmt.Errorf("PoA not found: %s", poaID)
	}

	pending := value.(*PendingPoA)

	// Update status to active
	if err := t.registry.UpdateStatus(ctx, poaID, PoAStatusActive); err != nil {
		return fmt.Errorf("failed to activate PoA: %w", err)
	}

	// Update local state
	pending.PoA.Status = PoAStatusActive
	t.pendingPoAs.Delete(poaID)
	t.activationTimers.Delete(poaID)

	// Send activation confirmation
	subject := "AgentAuth: Power of Attorney Activated"
	message := fmt.Sprintf(`
Your Power of Attorney (ID: %s) is now ACTIVE.

Grantee: %s
Scope: %s
Activated at: %s
Expires at: %s

You can revoke this PoA at any time: https://agentauth.example.com/revoke/%s

Monitor activity: https://agentauth.example.com/activity/%s
`, poaID, pending.PoA.Grantee, pending.PoA.Scope,
		time.Now().Format(time.RFC3339),
		pending.PoA.ExpiresAt.Format(time.RFC3339),
		poaID, poaID)

	if err := t.notifier.SendNotification(ctx, pending.PoA.Principal, subject, message); err != nil {
		return fmt.Errorf("failed to send activation notification: %w", err)
	}

	return nil
}

// sendActivationNotification sends initial notification about pending PoA.
func (t *TimelockPoA) sendActivationNotification(ctx context.Context, pending *PendingPoA) error {
	hoursUntilActivation := time.Until(pending.PoA.ActivationTime).Hours()

	subject := "🔔 AgentAuth: New Power of Attorney Created"
	message := fmt.Sprintf(`
IMPORTANT: A new Power of Attorney has been created for your account.

⏰ Activation Time: %s (%.0f hours from now)

Details:
- PoA ID: %s
- Grantee: %s
- Scope: %s
- Created: %s

🚨 IF YOU DID NOT AUTHORIZE THIS:
Cancel immediately: %s

This is a security feature. If your private key was stolen, this 24-hour 
delay gives you time to cancel the fraudulent PoA before it becomes active.

Questions? Contact security@agentauth.example.com
`,
		pending.PoA.ActivationTime.Format(time.RFC3339),
		hoursUntilActivation,
		pending.PoA.ID,
		pending.PoA.Grantee,
		pending.PoA.Scope,
		pending.PoA.CreatedAt.Format(time.RFC3339),
		pending.CancelURL)

	return t.notifier.SendNotification(ctx, pending.PoA.Principal, subject, message)
}

// sendReminderNotification sends a reminder about pending PoA activation.
func (t *TimelockPoA) sendReminderNotification(ctx context.Context, poaID string) error {
	value, ok := t.pendingPoAs.Load(poaID)
	if !ok {
		return nil // Already activated or cancelled
	}

	pending := value.(*PendingPoA)
	pending.RemindersSent++

	hoursUntilActivation := time.Until(pending.PoA.ActivationTime).Hours()

	subject := "⏰ AgentAuth Reminder: PoA Activates Soon"
	message := fmt.Sprintf(`
Reminder: Your Power of Attorney will activate in approximately %.0f hours.

PoA ID: %s
Grantee: %s
Activation Time: %s

To cancel: %s

If this was you, no action is needed.
`,
		hoursUntilActivation,
		pending.PoA.ID,
		pending.PoA.Grantee,
		pending.PoA.ActivationTime.Format(time.RFC3339),
		pending.CancelURL)

	return t.notifier.SendNotification(ctx, pending.PoA.Principal, subject, message)
}

// GetPendingPoAs returns all pending PoAs for a principal.
func (t *TimelockPoA) GetPendingPoAs(principalEmail string) []*PendingPoA {
	var pending []*PendingPoA

	t.pendingPoAs.Range(func(key, value interface{}) bool {
		p := value.(*PendingPoA)
		if p.PoA.Principal.Email == principalEmail {
			pending = append(pending, p)
		}
		return true
	})

	return pending
}

// TimeUntilActivation returns the time remaining until PoA activation.
func (p *PendingPoA) TimeUntilActivation() time.Duration {
	return time.Until(p.PoA.ActivationTime)
}

// CanBeCancelled returns true if the PoA can still be cancelled.
func (p *PendingPoA) CanBeCancelled() bool {
	return p.PoA.Status == PoAStatusPending && time.Now().Before(p.PoA.ActivationTime)
}
