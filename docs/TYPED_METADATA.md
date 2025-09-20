
# Event System with Type-Safe Metadata

---

## RFC 0111 Compliance & Legal Notice

GAuth implements the GiFo-RfC 0111 (GAuth) standard for AI power-of-attorney, delegation, and auditability. All protocol roles, flows, and exclusions are respected. See https://gimelfoundation.com for the full RFC.

**Exclusions:** GAuth MUST NOT include Web3, DNA-based identity, or decentralized auth logic. See RFC 0111 Section 2.

**Licensing:** Code is subject to the Gimel Foundation's Legal Provisions Relating to GiFo Documents. See LICENSE, Apache 2.0, and referenced licenses for OAuth, OpenID Connect, and MCP.

**P*P Roles:** GAuth implements Power*Point roles (PEP, PDP, PIP, PAP, PVP) as defined in RFC 0111. See the Architecture Guide for details.

---

This document explains how to use the improved type-safe event system in GAuth.

## Overview

The event system has been enhanced to use strongly typed metadata instead of the generic `map[string]interface{}`. This provides:

1. Type safety - Catch type errors at compile time
2. Clear intent - Self-documenting code with explicit types
3. Better tooling - IDE auto-completion and static analysis
4. Improved performance - Avoid reflection and type assertions when possible

## Creating Events with Typed Metadata

```go
// Create event with strongly typed metadata
event := events.NewEvent().
    WithType(events.EventTypeAuth).
    WithActionEnum(events.ActionLogin).
    WithStatusEnum(events.StatusSuccess).
    WithSubject("user123").
    WithStringMetadata("source_ip", "192.168.1.1").
    WithIntMetadata("attempt", 1).
    WithBoolMetadata("remember_me", true).
    WithTimeMetadata("login_time", time.Now())
```

## Reading Typed Metadata

```go
// Access specific typed values
if ip, exists := event.Metadata.GetString("source_ip"); exists {
    fmt.Printf("Connection from: %s\n", ip)
}

if attempt, exists := event.Metadata.GetInt("attempt"); exists {
    fmt.Printf("Attempt #%d\n", attempt)
}

if rememberMe, exists := event.Metadata.GetBool("remember_me"); exists {
    fmt.Printf("Remember me: %v\n", rememberMe)
}

// Get a timestamp value
if loginTime, exists := event.Metadata.Get("login_time"); exists {
    if ts, err := loginTime.ToTime(); err == nil {
        fmt.Printf("Login time: %s\n", ts.Format(time.RFC3339))
    }
}
```

## Converting Legacy Metadata

If you have existing code using the old map[string]interface{} style metadata, you can convert it:

```go
// Legacy untyped metadata
legacyMetadata := map[string]interface{}{
    "session_id": "sess_12345",
    "ttl":        3600,
    "active":     true,
}

// Convert to typed metadata
event := events.NewEvent()
event = event.MergeMetadata(legacyMetadata)
```

## Creating Read-Only Metadata

For values that shouldn't be changed after creation:

```go
metadata := events.NewMetadata()
metadata.SetReadOnly("origin", events.NewStringValue("trusted_service"))

// This will not change the value
metadata.SetString("origin", "attacker")
```

## Custom Metadata Types

```go
// Define a custom metadata value
customValue := events.NewStringValue("custom_data")
event = event.WithTypedMetadata("custom_field", customValue)
```

## Working with Event Handlers

Event handlers have been updated to work with typed metadata. Here's how a handler might process metadata:

```go
func (h *CustomHandler) Handle(event events.Event) {
    if event.Metadata != nil {
        // Process specific metadata types
        if sourceIp, exists := event.Metadata.GetString("source_ip"); exists {
            h.trackIpAccess(sourceIp)
        }
        
        if attempt, exists := event.Metadata.GetInt("attempt"); exists && attempt > 3 {
            h.flagSuspiciousActivity(event.Subject)
        }
    }
}
```