package web

import (
	"time"
)

// policyAuditorAdapter adapts AuditLog to policy.Auditor interface
type policyAuditorAdapter struct {
	log *AuditLog
}

func (a *policyAuditorAdapter) Append(entry interface{}) {
	if a.log == nil {
		return
	}
	m, ok := entry.(map[string]interface{})
	if !ok {
		return
	}
	ae := &AuditEntry{
		ID: randomNonce(6),
		At: time.Now(),
	}
	if v, ok := m["at"].(time.Time); ok {
		ae.At = v
	}
	if v, ok := m["actor"].(string); ok {
		ae.Actor = v
	}
	if v, ok := m["action"].(string); ok {
		ae.Action = v
	}
	if v, ok := m["resource"].(string); ok {
		ae.Resource = v
	}
	if v, ok := m["outcome"].(string); ok {
		ae.Outcome = v
	}
	if v, ok := m["meta"]; ok {
		ae.Meta = v
	}
	a.log.Append(ae)
}

func (a *policyAuditorAdapter) List(limit int) []interface{} {
	if a.log == nil {
		return nil
	}
	entries := a.log.List(limit)
	res := make([]interface{}, len(entries))
	for i, e := range entries {
		res[i] = e
	}
	return res
}
