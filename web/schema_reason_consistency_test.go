package web

// Test to ensure the JSON Schemas reason enums remain consistent with internal taxonomy.
// It parses the schema files under docs/schema and extracts the union of all `reason` enum values,
// then compares with the canonical slice defined here (mirrors server_clean.go constants).
// Any drift causes test failure, acting as a guardrail for future edits.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// canonicalReasons reflects the authoritative set used in server_clean.go (line with taxonomy comment).
var canonicalReasons = []string{
	"status_change",
	"invalid_transition",
	"init",
	"noop",
	"maintenance",
	"rate_limited",
	"policy_violation",
	"not_found",
	"unsupported_status",
	"invalid_payload",
}

// schemaReasonFiles to inspect (exclude capabilities_registry since it lacks reasons enumeration).
var schemaReasonFiles = []string{
	"decision_metrics.schema.json",
	"lifecycle_timeline.schema.json",
	"audit_logs.schema.json",
}

// helper to walk a generic JSON structure and collect all enum arrays keyed by property name "reason".
func collectEnums(v interface{}, out *map[string]struct{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		// If this node has properties.reason.enum we capture it
		if props, ok := val["properties"].(map[string]interface{}); ok {
			if reasonNode, ok := props["reason"].(map[string]interface{}); ok {
				if enumVals, ok := reasonNode["enum"].([]interface{}); ok {
					for _, e := range enumVals {
						if s, ok2 := e.(string); ok2 {
							(*out)[s] = struct{}{}
						}
					}
				}
			}
		}
		for _, child := range val {
			collectEnums(child, out)
		}
	case []interface{}:
		for _, child := range val {
			collectEnums(child, out)
		}
	}
}

func TestSchemaReasonConsistency(t *testing.T) {
	root := filepath.Join("..", "docs", "schema")
	found := map[string]struct{}{}
	for _, fname := range schemaReasonFiles {
		b, err := os.ReadFile(filepath.Join(root, fname))
		if err != nil {
			t.Fatalf("read schema %s: %v", fname, err)
		}
		var any interface{}
		if err := json.Unmarshal(b, &any); err != nil {
			t.Fatalf("unmarshal schema %s: %v", fname, err)
		}
		collectEnums(any, &found)
	}

	// Build maps for comparison
	canonical := map[string]struct{}{}
	for _, r := range canonicalReasons {
		canonical[r] = struct{}{}
	}

	// Detect missing in schemas
	var missingInSchemas []string
	for r := range canonical {
		if _, ok := found[r]; !ok {
			missingInSchemas = append(missingInSchemas, r)
		}
	}

	// Detect extras present only in schemas
	var extraInSchemas []string
	for r := range found {
		if _, ok := canonical[r]; !ok {
			extraInSchemas = append(extraInSchemas, r)
		}
	}

	if len(missingInSchemas) > 0 || len(extraInSchemas) > 0 {
		t.Fatalf("reason taxonomy drift detected. Missing in schemas: %v | Extra in schemas: %v (canonical=%v)", missingInSchemas, extraInSchemas, canonicalReasons)
	}
}
