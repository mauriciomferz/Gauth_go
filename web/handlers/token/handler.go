package token

import (
	"context"
	"math/rand"
	"os"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

// Helper interfaces to avoid circular dependencies with web package
type Auditor interface {
	LogAction(actor, action, resource, outcome string)
}

type Emitter interface {
	EmitTokenCreated(id string)
	EmitTokenStatusChanged(id, old, new, reason string)
}

type PrimaryAuth interface {
	ValidateToken(token string) (any, error)
	ViolationSnapshot() map[string]uint64
}

type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

type Span interface {
	SetTag(key string, value any)
	End()
}

type LifecycleRecorder interface {
	RecordEvent(entityType, entityID, oldStatus, newStatus, outcome, reason string, latencyNS int64)
}

// JWKSETagUpdater allows updating the server's JWKS ETag
type JWKSETagUpdater interface {
	UpdateJWKSETag(etag string)
}

// CapabilityEnforcer abstracts capability checking
type CapabilityEnforcer func(action string, claims map[string]any) (bool, []string)

// Handler manages token operations
type Handler struct {
	Store        *Store
	Replay       *ReplayNonceStore
	Auditor      Auditor
	Emitter      Emitter
	PrimaryAuth  PrimaryAuth
	Tracer       Tracer
	TracerRatio  float64
	CapEnforcer  CapabilityEnforcer
	Metrics      metrics.Metrics
	Lifecycle    LifecycleRecorder
	KeyProvider  crypto.KeyProvider
	ETagUpdater  JWKSETagUpdater
	GAuthService gauth.GAuth

	// Configs
	UseJWTLib    bool
	JWTAlg       string
	JWTKid       string
	JWTRotation  string
	JWTSecret    string // for HMAC fallback
	Issuer       string
	ReplayStrict bool
	ClockSkew    time.Duration
}

func NewHandler(store *Store, replay *ReplayNonceStore, auditor Auditor, emitter Emitter, primaryAuth PrimaryAuth, tracer Tracer, capEnforcer CapabilityEnforcer, m metrics.Metrics, lifecycle LifecycleRecorder, kp crypto.KeyProvider) *Handler {
	// Defaults/Env loading could be here or passed in.
	// For now, load envs that are "static" here, or assume caller sets them.
	// But apiTokenCreate heavily used os.Getenv. Let's load them for convenience.
	// We can update them if needed.

	h := &Handler{
		Store:       store,
		Replay:      replay,
		Auditor:     auditor,
		Emitter:     emitter,
		PrimaryAuth: primaryAuth,
		Tracer:      tracer,
		TracerRatio: 0, // Default to 0? Caller should set.
		CapEnforcer: capEnforcer,
		Metrics:     m,
		Lifecycle:   lifecycle,
		KeyProvider: kp,

		UseJWTLib:    os.Getenv("GAUTH_USE_JWT_LIB") == "1",
		JWTAlg:       os.Getenv("GAUTH_JWT_ALG"),
		JWTKid:       os.Getenv("GAUTH_JWT_KID"),
		JWTRotation:  os.Getenv("GAUTH_JWT_ROTATION_DAYS"),
		JWTSecret:    os.Getenv("GAUTH_JWT_SECRET"),
		Issuer:       os.Getenv("GAUTH_ISSUER"),
		ReplayStrict: os.Getenv("GAUTH_REPLAY_STRICT") == "1",
		ClockSkew:    0,
	}

	if h.JWTAlg == "" {
		h.JWTAlg = "RS256" // Default
	}
	if h.JWTKid == "" {
		h.JWTKid = "demo-rsa" // Default fallback
	}
	// Clock skew loading
	if skew := os.Getenv("GAUTH_JWT_CLOCK_SKEW_SECONDS"); skew != "" {
		if dur, err := time.ParseDuration(skew + "s"); err == nil {
			h.ClockSkew = dur
		}
	}

	return h
}

// Helper for random sampling
func (h *Handler) ShouldTrace() bool {
	if h.Tracer == nil {
		return false
	}
	if h.TracerRatio <= 0 {
		// Treat <= 0 as "trace everything" to match legacy behavior
		return true
	}
	//nolint:gosec // weak random acceptable
	return rand.Float64() < h.TracerRatio
}
