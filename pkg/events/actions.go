package events

import (
	"time"

	"github.com/google/uuid"
)

// EventAction represents specific actions within an event type
type EventAction string

// Auth event actions
const (
	// Authentication actions
	ActionLogin              EventAction = "login"
	ActionLogout             EventAction = "logout"
	ActionLoginFailed        EventAction = "login_failed"
	ActionPasswordChanged    EventAction = "password_changed"
	ActionPasswordReset      EventAction = "password_reset"
	ActionMultiFactorSuccess EventAction = "mfa_success"
	ActionMultiFactorFailed  EventAction = "mfa_failed"
	ActionSessionCreated     EventAction = "session_created"
	ActionSessionInvalidated EventAction = "session_invalidated"
	ActionSessionExpired     EventAction = "session_expired"
	ActionDeviceRegistered   EventAction = "device_registered"
	ActionDeviceUnregistered EventAction = "device_unregistered"
)

// Authorization event actions
const (
	// Authorization actions
	ActionAuthorizationGranted EventAction = "authorization_granted"
	ActionAuthorizationDenied  EventAction = "authorization_denied"
	ActionConsentGiven         EventAction = "consent_given"
	ActionConsentRevoked       EventAction = "consent_revoked"
	ActionScopeGranted         EventAction = "scope_granted"
	ActionScopeRevoked         EventAction = "scope_revoked"
	ActionRoleAssigned         EventAction = "role_assigned"
	ActionRoleRevoked          EventAction = "role_revoked"
	ActionPermissionGranted    EventAction = "permission_granted"
	ActionPermissionRevoked    EventAction = "permission_revoked"
)

// Token event actions
const (
	// Token actions
	ActionTokenIssued           EventAction = "token_issued"
	ActionTokenRefreshed        EventAction = "token_refreshed"
	ActionTokenRevoked          EventAction = "token_revoked"
	ActionTokenValidated        EventAction = "token_validated"
	ActionTokenValidationFailed EventAction = "token_validation_failed"
	ActionTokenExpired          EventAction = "token_expired"
	ActionTokenIntrospected     EventAction = "token_introspected"
)

// User activity event actions
const (
	// User activity actions
	ActionUserCreated            EventAction = "user_created"
	ActionUserUpdated            EventAction = "user_updated"
	ActionUserDeleted            EventAction = "user_deleted"
	ActionUserSuspended          EventAction = "user_suspended"
	ActionUserReinstated         EventAction = "user_reinstated"
	ActionUserProfileAccessed    EventAction = "user_profile_accessed"
	ActionUserPreferencesChanged EventAction = "user_preferences_changed"
)

// Delegation event actions
const (
	// Delegation actions (RFC111)
	ActionDelegationCreated      EventAction = "delegation_created"
	ActionDelegationUpdated      EventAction = "delegation_updated"
	ActionDelegationRevoked      EventAction = "delegation_revoked"
	ActionDelegationValidated    EventAction = "delegation_validated"
	ActionDelegationExercised    EventAction = "delegation_exercised"
	ActionDelegationChainCreated EventAction = "delegation_chain_created"
)

// System event actions
const (
	// System actions
	ActionSystemStartup        EventAction = "system_startup"
	ActionSystemShutdown       EventAction = "system_shutdown"
	ActionConfigChanged        EventAction = "config_changed"
	ActionKeyRotation          EventAction = "key_rotation"
	ActionBackupCreated        EventAction = "backup_created"
	ActionAlertTriggered       EventAction = "alert_triggered"
	ActionMaintenanceStarted   EventAction = "maintenance_started"
	ActionMaintenanceCompleted EventAction = "maintenance_completed"
)

// EventStatus represents the status of an event
type EventStatus string

// Event statuses
const (
	StatusSuccess EventStatus = "success"
	StatusFailure EventStatus = "failure"
	StatusError   EventStatus = "error"
	StatusWarning EventStatus = "warning"
	StatusInfo    EventStatus = "info"
	StatusPending EventStatus = "pending"
)

// CreateEvent creates a new event with default values
func CreateEvent() Event {
	return Event{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Metadata:  NewMetadata(),
	}
}

// Helper functions for creating typed events

// CreateAuthEvent creates a new authentication event
func CreateAuthEvent(action EventAction, status EventStatus) Event {
	e := CreateEvent()
	e.Type = EventTypeAuth
	e.Action = string(action)
	e.Status = string(status)
	return e
}

// CreateAuthzEvent creates a new authorization event
func CreateAuthzEvent(action EventAction, status EventStatus) Event {
	e := CreateEvent()
	e.Type = EventTypeAuthz
	e.Action = string(action)
	e.Status = string(status)
	return e
}

// CreateTokenEvent creates a new token event
func CreateTokenEvent(action EventAction, status EventStatus) Event {
	e := CreateEvent()
	e.Type = EventTypeToken
	e.Action = string(action)
	e.Status = string(status)
	return e
}

// CreateUserActivityEvent creates a new user activity event
func CreateUserActivityEvent(action EventAction, status EventStatus) Event {
	e := CreateEvent()
	e.Type = EventTypeUserActivity
	e.Action = string(action)
	e.Status = string(status)
	return e
}

// CreateAuditEvent creates a new audit event
func CreateAuditEvent(action EventAction, status EventStatus) Event {
	e := CreateEvent()
	e.Type = EventTypeAudit
	e.Action = string(action)
	e.Status = string(status)
	return e
}

// CreateSystemEvent creates a new system event
func CreateSystemEvent(action EventAction, status EventStatus) Event {
	e := CreateEvent()
	e.Type = EventTypeSystem
	e.Action = string(action)
	e.Status = string(status)
	return e
}

// NewAuthEvent creates a new authentication event (legacy name for CreateAuthEvent)
func NewAuthEvent(action EventAction, status EventStatus) Event {
	return CreateAuthEvent(action, status)
}

// NewAuthzEvent creates a new authorization event (legacy name for CreateAuthzEvent)
func NewAuthzEvent(action EventAction, status EventStatus) Event {
	return CreateAuthzEvent(action, status)
}

// NewTokenEvent creates a new token event (legacy name for CreateTokenEvent)
func NewTokenEvent(action EventAction, status EventStatus) Event {
	return CreateTokenEvent(action, status)
}

// NewUserActivityEvent creates a new user activity event (legacy name for CreateUserActivityEvent)
func NewUserActivityEvent(action EventAction, status EventStatus) Event {
	return CreateUserActivityEvent(action, status)
}

// NewAuditEvent creates a new audit event (legacy name for CreateAuditEvent)
func NewAuditEvent(action EventAction, status EventStatus) Event {
	return CreateAuditEvent(action, status)
}

// NewSystemEvent creates a new system event (legacy name for CreateSystemEvent)
func NewSystemEvent(action EventAction, status EventStatus) Event {
	return CreateSystemEvent(action, status)
}
