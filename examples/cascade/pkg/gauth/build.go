//go:build !windows
// +build !windows

// Package gauth provides authentication integration for cascade services
// This file ensures proper build constraints and package visibility
package gauth

// Build verification - this ensures the package builds correctly in CI/CD
var _ interface{} = (*ServiceAuth)(nil)
var _ interface{} = (*GAuth)(nil)
