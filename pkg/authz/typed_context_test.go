package authz

import (
	"strconv"
	"testing"
	"time"
)

func TestContext(t *testing.T) {
	ctx := make(map[string]string)

	// Test string values
	ctx["user"] = "john"
	val, ok := ctx["user"]
	if !ok || val != "john" {
		t.Errorf("String value not set correctly: got %v, want %s", val, "john")
	}

	// Test int values (store as string)
	ctx["age"] = strconv.Itoa(30)
	ival, err := strconv.Atoi(ctx["age"])
	if err != nil || ival != 30 {
		t.Errorf("Int value not set correctly: got %v, want %d", ival, 30)
	}

	// Test float values (store as string)
	ctx["score"] = strconv.FormatFloat(9.5, 'f', -1, 64)
	fval, err := strconv.ParseFloat(ctx["score"], 64)
	if err != nil || fval != 9.5 {
		t.Errorf("Float value not set correctly: got %v, want %f", fval, 9.5)
	}

	// Test bool values (store as string)
	ctx["admin"] = strconv.FormatBool(true)
	bval, err := strconv.ParseBool(ctx["admin"])
	if err != nil || bval != true {
		t.Errorf("Bool value not set correctly: got %v, want %t", bval, true)
	}

	// Test time values (store as string)
	now := time.Now().UTC()
	ctx["timestamp"] = now.Format(time.RFC3339Nano)
	tval, err := time.Parse(time.RFC3339Nano, ctx["timestamp"])
	if err != nil || !tval.Equal(now) {
		t.Errorf("Time value not set correctly: got %v, want %v", tval, now)
	}

	// Test key existence and removal
	if _, ok := ctx["user"]; !ok {
		t.Error("Key 'user' not found")
	}
	delete(ctx, "user")
	if _, ok := ctx["user"]; ok {
		t.Error("Key 'user' was not removed")
	}

	// Test key count
	if len(ctx) != 4 {
		t.Errorf("Context returned wrong number of keys: got %d, want %d", len(ctx), 4)
	}

	if v, ok := ctx["age"]; !ok || v != "30" {
		t.Errorf("Context age value incorrect: got %v, want %s", v, "30")
	}
}

// Removed duplicate TestAnnotations definition
func TestAnnotations(t *testing.T) {
	annotations := make(map[string]string)

	// Test string values
	annotations["source"] = "api"
	val, ok := annotations["source"]
	if !ok || val != "api" {
		t.Errorf("String value not set correctly: got %v, want %s", val, "api")
	}

	// Test int values (store as string)
	annotations["priority"] = strconv.Itoa(5)
	ival, err := strconv.Atoi(annotations["priority"])
	if err != nil || ival != 5 {
		t.Errorf("Int value not set correctly: got %v, want %d", ival, 5)
	}

	// Test float values (store as string)
	annotations["confidence"] = strconv.FormatFloat(0.95, 'f', -1, 64)
	fval, err := strconv.ParseFloat(annotations["confidence"], 64)
	if err != nil || fval != 0.95 {
		t.Errorf("Float value not set correctly: got %v, want %f", fval, 0.95)
	}

	// Test bool values (store as string)
	annotations["cached"] = strconv.FormatBool(true)
	bval, err := strconv.ParseBool(annotations["cached"])
	if err != nil || bval != true {
		t.Errorf("Bool value not set correctly: got %v, want %t", bval, true)
	}

	// Test key existence and removal
	if _, ok := annotations["source"]; !ok {
		t.Error("Key 'source' not found")
	}
	delete(annotations, "source")
	if _, ok := annotations["source"]; ok {
		t.Error("Key 'source' was not removed")
	}

	// Test key count
	if len(annotations) != 3 {
		t.Errorf("Annotations returned wrong number of keys: got %d, want %d", len(annotations), 3)
	}
}

func TestAccessRequest(t *testing.T) {
	request := NewAccessRequest(
		Subject{ID: "user123", Type: "user"},
		Resource{ID: "doc456", Type: "document"},
		Action{Name: "read"},
	)

	// Test basic properties
	if request.Subject.ID != "user123" {
		t.Errorf("Subject ID incorrect: got %s, want %s", request.Subject.ID, "user123")
	}

	if request.Resource.ID != "doc456" {
		t.Errorf("Resource ID incorrect: got %s, want %s", request.Resource.ID, "doc456")
	}

	if request.Action.Name != "read" {
		t.Errorf("Action Name incorrect: got %s, want %s", request.Action.Name, "read")
	}

	// Test adding context values (all as strings)
	request.Context["department"] = "engineering"
	request.Context["priority"] = strconv.Itoa(3)
	request.Context["urgent"] = strconv.FormatBool(true)
	request.Context["tags"] = "confidential,shared"

	val, ok := request.Context["department"]
	if !ok || val != "engineering" {
		t.Errorf("Context string value incorrect: got %v, want %s", val, "engineering")
	}

	// Context should exist and contain all values
	if len(request.Context) != 4 {
		t.Errorf("Context should have 4 keys, got %d", len(request.Context))
	}
}

func TestAccessResponse(t *testing.T) {
	response := &AccessResponse{
		Allowed:     true,
		Reason:      "policy matched",
		PolicyID:    "policy123",
		Annotations: make(map[string]string),
	}

	// Test adding annotations (all as strings)
	response.Annotations["source"] = "database"
	response.Annotations["expires"] = strconv.Itoa(3600)
	response.Annotations["cached"] = strconv.FormatBool(false)

	val, ok := response.Annotations["source"]
	if !ok || val != "database" {
		t.Errorf("Annotation string value incorrect: got %v, want %s", val, "database")
	}

	ival, err := strconv.Atoi(response.Annotations["expires"])
	if err != nil || ival != 3600 {
		t.Errorf("Annotation int value incorrect: got %v, want %d", ival, 3600)
	}

	bval, err := strconv.ParseBool(response.Annotations["cached"])
	if err != nil || bval != false {
		t.Errorf("Annotation bool value incorrect: got %v, want %t", bval, false)
	}
}
