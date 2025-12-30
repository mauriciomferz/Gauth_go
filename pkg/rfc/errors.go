package rfc

import (
	"github.com/mauriciomferz/AgentAuth/pkg/rfc/errs"
)

// Code aliases from errs package
type ErrorCode = errs.ErrorCode

const (
	ErrNotFound                = errs.ErrNotFound
	ErrUnauthorized            = errs.ErrUnauthorized
	ErrRevoked                 = errs.ErrRevoked
	ErrExpired                 = errs.ErrExpired
	ErrScopeViolation          = errs.ErrScopeViolation
	ErrRestrictionExceeded     = errs.ErrRestrictionExceeded
	ErrInvalidRequest          = errs.ErrInvalidRequest
	ErrIntegrityFailure        = errs.ErrIntegrityFailure
	ErrInternal                = errs.ErrInternal
	ErrReplay                  = errs.ErrReplay
	ErrDelegationDepthExceeded = errs.ErrDelegationDepthExceeded
	ErrConfiguration           = errs.ErrConfiguration
)

// RFCError alias
type RFCError = errs.RFCError

// New creates an RFCError (convenience wrapper)
func New(code ErrorCode, msg string) error { return errs.New(code, msg) }
