//go:build go1.18

package gauth_aap_001

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/token"
)

// FuzzParseEnvelope tests the robustness of envelope unmarshaling and basic field validation.
func FuzzParseEnvelope(f *testing.F) {
	// Seed with valid V1 and V2 JSON
	v1, _ := json.Marshal(token.Envelope{Version: "v1", DelegationID: "did1", Grantor: "alice", Grantee: "bob"})
	v2, _ := json.Marshal(token.EnvelopeV2{Version: "poa/v1-env2", DelegationID: "did2", Grantor: "carol", Grantee: "dave", CanonicalDigest: "abc"})
	f.Add(v1)
	f.Add(v2)
	f.Add([]byte(`{"ver":"v1","delegation_id":"","grantor":"alice"}`))           // missing field
	f.Add([]byte(`{"ver":"poa/v1-env2","delegation_id":"did","grantee":"bob"}`)) // missing grantor

	f.Fuzz(func(t *testing.T, data []byte) {
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			return // Malformed JSON is expected/handled
		}

		ver, _ := raw["ver"].(string)
		useV2 := strings.HasSuffix(ver, "env2")

		if useV2 {
			var env2 token.EnvelopeV2
			if err := json.Unmarshal(data, &env2); err != nil {
				return
			}
			// Basic validation logic from VerifyToken
			if env2.DelegationID == "" || env2.Grantor == "" || env2.Grantee == "" {
				// expected validation failure
				return
			}
		} else {
			var env token.Envelope
			if err := json.Unmarshal(data, &env); err != nil {
				return
			}
			// Basic validation logic from VerifyToken
			if env.DelegationID == "" || env.Grantor == "" || env.Grantee == "" {
				// expected validation failure
				return
			}
		}
	})
}
