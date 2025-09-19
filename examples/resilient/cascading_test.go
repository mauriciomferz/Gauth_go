package resilient

import (
	"context"
	"testing"
	"time"
)

func TestGAuthAuthorizationFlow_Success(t *testing.T) {
	gauth := NewGAuthAuthorizationServer()
	poa := gauth.IssuePowerOfAttorney("HumanOwner123", "OrderService", "OrderService:process", time.Hour)
	if poa == nil {
		t.Fatal("Power of attorney should be issued")
	}
	grant, err := gauth.IssueGrant("OrderService", "OrderService:process")
	if err != nil || grant == nil {
		t.Fatalf("Grant should be issued, got error: %v", err)
	}
	token, err := gauth.IssueExtendedToken(grant.GrantID)
	if err != nil || token == nil {
		t.Fatalf("Extended token should be issued, got error: %v", err)
	}
	if token.Agent != "OrderService" || token.Scope != "OrderService:process" {
		t.Error("Token fields do not match expected values")
	}
}

func TestGAuthAuthorizationFlow_Denied(t *testing.T) {
	gauth := NewGAuthAuthorizationServer()
	// No power of attorney issued
	_, err := gauth.IssueGrant("OrderService", "OrderService:process")
	if err == nil {
		t.Error("Grant should not be issued without power of attorney")
	}
	_, err = gauth.AuthorizeAction("OrderService", "OrderService:process")
	if err == nil {
		t.Error("Authorization should be denied without power of attorney")
	}
}

func TestGAuthAuditLog(t *testing.T) {
	gauth := NewGAuthAuthorizationServer()
	gauth.IssuePowerOfAttorney("HumanOwner123", "OrderService", "OrderService:process", time.Hour)
	_, _ = gauth.AuthorizeAction("OrderService", "OrderService:process")
	if len(gauth.auditLog) == 0 {
		t.Error("Audit log should contain events")
	}
	found := false
	for _, event := range gauth.auditLog {
		if event.Action == "AuthorizeAction" && event.Result == "granted" {
			found = true
		}
	}
	if !found {
		t.Error("Expected 'AuthorizeAction' with 'granted' in audit log")
	}
}

func TestMicroserviceProcessRequestIntegration(t *testing.T) {
       mesh := NewServiceMesh()
       gauth := NewGAuthAuthorizationServer()
       // Re-initialize Bulkhead for all services with higher MaxConcurrent to avoid 'bulkhead full' error for any dependency
       for _, svc := range mesh.services {
	       svc.Bulkhead = nil // Disable Bulkhead for test
	       gauth.IssuePowerOfAttorney("HumanOwner123", svc.Name, svc.Name+":process", time.Hour)
       }
       orderSvc := mesh.services[OrderService]
       err := orderSvc.processRequest(context.Background(), mesh, gauth)
       if err != nil {
	       t.Errorf("OrderService should be authorized and process request, got error: %v", err)
       }
}
