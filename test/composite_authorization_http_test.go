package test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    web "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web"
)

func TestCompositeAuthorizationHTTP(t *testing.T) {
    srv := web.NewBetaServer("") // default port; in-memory metrics
    engine := srv.Engine()
    if engine == nil { t.Fatalf("engine nil") }
    // Initial GET should 404 on stable path
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/authorization/composite", nil)
    engine.ServeHTTP(w, req)
    if w.Code != 404 {
        t.Fatalf("expected 404 before activation, got %d body=%s", w.Code, w.Body.String())
    }
    // Activate artifact
    now := time.Now().UTC()
    vf := now.Add(1 * time.Hour).Format(time.RFC3339)
    vu := now.Add(2 * time.Hour).Format(time.RFC3339)
    exp := now.Add(4 * time.Hour).Format(time.RFC3339)
    payload := map[string]any{
        "ai_system_id": "ai_v1",
        "authorization_grant": map[string]any{"type": "general", "scope": []string{"financial_operations"}, "valid_from": vf, "valid_until": vu, "revocable": true},
        "powers_granted": map[string]any{"basic_powers": []string{"financial_operations"}},
        "decision_authority": map[string]any{"autonomous_decisions": []string{"routine_invoice_approval"}},
        "transaction_rights": map[string]any{"allowed_transaction_types": []string{"vendor_payments"}},
        "action_permissions": map[string]any{"system_actions": []string{"generate_reports"}},
        "dual_control_principle": map[string]any{"enabled": true},
        "authorization_cascade": map[string]any{"accountability_chain": []string{"ceo_001","cfo_001","ai_v1"}},
        "expires_at": exp,
    }
    buf, _ := json.Marshal(payload)
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("POST", "/api/v1/authorization/composite", bytes.NewReader(buf))
    req.Header.Set("Content-Type", "application/json")
    engine.ServeHTTP(w, req)
    if w.Code != 201 {
        t.Fatalf("expected 201 activation, got %d body=%s", w.Code, w.Body.String())
    }
    // Second activation with overlapping valid_from should conflict (409)
    w = httptest.NewRecorder()
    engine.ServeHTTP(w, req)
    if w.Code != 409 {
        t.Fatalf("expected 409 conflict, got %d body=%s", w.Code, w.Body.String())
    }
}
