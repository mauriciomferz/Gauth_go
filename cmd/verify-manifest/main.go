// Command verify-manifest verifies the signed policy manifest emitted by the GAuth server.
// It fetches (or reads from file) the /api/v1/policy/manifest JSON, reconstructs the
// canonical unsigned portion, recomputes the sha256 hash, and verifies the Ed25519 signature
// using domain separation prefix "GAUTH_POLICY_MANIFEST:".
//
// Usage examples:
//
//	go run ./cmd/verify-manifest                         # fetch default URL and require --public-key
//	go run ./cmd/verify-manifest -url http://localhost:8080/api/v1/policy/manifest -public-key <b64raw>
//	go run ./cmd/verify-manifest -file manifest.json -public-key <b64raw> -print-canonical
//	go run ./cmd/verify-manifest -public-key-file ./pub.key -json
//
// Public key format: base64.RawURLEncoding or base64.StdEncoding of 32-byte Ed25519 public key.
//
// Exit codes:
//
//	0 success (verified)
//	1 verification failure (hash/signature mismatch or structural error)
//	2 usage / input error (missing public key or fetch failure)
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Canonical struct definitions (mirrors web/policy_manifest.go)
type manifestCanonical struct {
	SchemaVersion         int             `json:"schema_version"`
	Capabilities          []manifestCap   `json:"capabilities"`
	ActionMatrix          []manifestAction `json:"action_matrix"`
	RegistryHash          string          `json:"registry_hash"`
	RegistryPrevHash      string          `json:"registry_prev_hash,omitempty"`
	RegistryLastChangedAt string          `json:"registry_last_changed_at,omitempty"`
	CapabilityCount       int             `json:"capability_count"`
	ActionCount           int             `json:"action_count"`
}

type manifestCap struct {
	ID              string   `json:"id"`
	Version         string   `json:"version"`
	Stable          bool     `json:"stable"`
	DeprecatedAfter string   `json:"deprecated_after,omitempty"`
	SunsetAfter     string   `json:"sunset_after,omitempty"`
	Versions        []string `json:"versions,omitempty"`
}

type manifestAction struct {
	Action   string   `json:"action"`
	Required []string `json:"required"`
}

func decodePublicKey(b64 string) (ed25519.PublicKey, error) {
	// Try raw URL first then std
	data, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		data, err = base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, err
		}
	}
	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: %d", len(data))
	}
	return ed25519.PublicKey(data), nil
}

func fetchManifest(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, string(b))
	}
	return io.ReadAll(resp.Body)
}

func reconstructCanonical(payload map[string]any) (manifestCanonical, []byte, string, error) {
	// Convert capabilities (preserve order as provided; server already sorts)
	capsAny, ok := payload["capabilities"].([]any)
	if !ok {
		return manifestCanonical{}, nil, "", errors.New("capabilities missing or wrong type")
	}
	caps := make([]manifestCap, 0, len(capsAny))
	for _, c := range capsAny {
		cm := c.(map[string]any)
		mc := manifestCap{ID: cm["id"].(string), Version: cm["version"].(string), Stable: cm["stable"].(bool)}
		if v, ok := cm["deprecated_after"].(string); ok && v != "" { mc.DeprecatedAfter = v }
		if v, ok := cm["sunset_after"].(string); ok && v != "" { mc.SunsetAfter = v }
		if vv, ok := cm["versions"].([]any); ok && len(vv) > 0 {
			list := make([]string, 0, len(vv))
			for _, x := range vv { list = append(list, x.(string)) }
			mc.Versions = list
		}
		caps = append(caps, mc)
	}
	actsAny, ok := payload["action_matrix"].([]any)
	if !ok { return manifestCanonical{}, nil, "", errors.New("action_matrix missing or wrong type") }
	acts := make([]manifestAction, 0, len(actsAny))
	for _, a := range actsAny {
		am := a.(map[string]any)
		reqList := []string{}
		if rr, ok := am["required"].([]any); ok {
			for _, r := range rr { reqList = append(reqList, r.(string)) }
		}
		acts = append(acts, manifestAction{Action: am["action"].(string), Required: reqList})
	}
	canon := manifestCanonical{
		SchemaVersion: int(payload["schema_version"].(float64)),
		Capabilities: caps,
		ActionMatrix: acts,
		RegistryHash: payload["registry_hash"].(string),
		CapabilityCount: int(payload["capability_count"].(float64)),
		ActionCount: int(payload["action_count"].(float64)),
	}
	if v, ok := payload["registry_prev_hash"].(string); ok && v != "" { canon.RegistryPrevHash = v }
	if v, ok := payload["registry_last_changed_at"].(string); ok && v != "" { canon.RegistryLastChangedAt = v }
	raw, err := json.Marshal(canon)
	if err != nil { return manifestCanonical{}, nil, "", err }
	sum := sha256.Sum256(raw)
	h := fmt.Sprintf("sha256:%x", sum[:])
	return canon, raw, h, nil
}

func main() {
	url := flag.String("url", "http://localhost:8080/api/v1/policy/manifest", "Manifest endpoint URL")
	file := flag.String("file", "", "Read manifest JSON from file instead of HTTP")
	pubKeyStr := flag.String("public-key", "", "Ed25519 public key (base64 raw or std)")
	pubKeyFile := flag.String("public-key-file", "", "Path to file containing base64 public key")
	expectKid := flag.String("expect-kid", "", "Expected sig_kid (optional)")
	printCanon := flag.Bool("print-canonical", false, "Print reconstructed canonical JSON to stdout")
	jsonOut := flag.Bool("json", false, "Emit machine-friendly JSON result")
	flag.Parse()

	if *pubKeyStr == "" && *pubKeyFile == "" {
		fmt.Fprintln(os.Stderr, "missing --public-key or --public-key-file")
		os.Exit(2)
	}
	var pubKey ed25519.PublicKey
	if *pubKeyFile != "" {
		b, err := os.ReadFile(*pubKeyFile)
		if err != nil { fmt.Fprintf(os.Stderr, "read public key file: %v\n", err); os.Exit(2) }
		pubKeyStrVal := string(b)
		pubKeyStrVal = trimWhitespace(pubKeyStrVal)
		pk, err := decodePublicKey(pubKeyStrVal)
		if err != nil { fmt.Fprintf(os.Stderr, "decode public key file: %v\n", err); os.Exit(2) }
		pubKey = pk
	} else {
		pk, err := decodePublicKey(*pubKeyStr)
		if err != nil { fmt.Fprintf(os.Stderr, "decode public key: %v\n", err); os.Exit(2) }
		pubKey = pk
	}

	var data []byte
	var err error
	if *file != "" {
		data, err = os.ReadFile(*file)
		if err != nil { fmt.Fprintf(os.Stderr, "read file: %v\n", err); os.Exit(2) }
	} else {
		data, err = fetchManifest(*url)
		if err != nil { fmt.Fprintf(os.Stderr, "fetch manifest: %v\n", err); os.Exit(2) }
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		fmt.Fprintf(os.Stderr, "decode manifest: %v\n", err)
		os.Exit(2)
	}

	manifestHashField, ok := payload["manifest_hash"].(string)
	if !ok || manifestHashField == "" { fmt.Fprintln(os.Stderr, "manifest_hash missing"); os.Exit(1) }
	sigB64, ok := payload["signature"].(string)
	if !ok || sigB64 == "" { fmt.Fprintln(os.Stderr, "signature missing"); os.Exit(1) }
	kid, _ := payload["sig_kid"].(string)
	if *expectKid != "" && kid != *expectKid { fmt.Fprintf(os.Stderr, "kid mismatch expected=%s got=%s\n", *expectKid, kid); os.Exit(1) }

	canon, rawCanon, recomputedHash, err := reconstructCanonical(payload)
	if err != nil { fmt.Fprintf(os.Stderr, "canonical reconstruction error: %v\n", err); os.Exit(1) }
	if recomputedHash != manifestHashField {
		emitResult(*jsonOut, false, "hash_mismatch", map[string]any{"expected": manifestHashField, "got": recomputedHash})
		os.Exit(1)
	}
	// Verify signature
	sigBytes, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil { sigBytes, err = base64.StdEncoding.DecodeString(sigB64) }
	if err != nil { emitResult(*jsonOut, false, "signature_decode_error", map[string]any{"error": err.Error()}); os.Exit(1) }
	msg := append([]byte("GAUTH_POLICY_MANIFEST:"), rawCanon...)
	if !ed25519.Verify(pubKey, msg, sigBytes) {
		emitResult(*jsonOut, false, "signature_invalid", map[string]any{"kid": kid})
		os.Exit(1)
	}

	if *printCanon {
		fmt.Println(string(rawCanon))
	}
	emitResult(*jsonOut, true, "ok", map[string]any{"kid": kid, "manifest_hash": manifestHashField, "capabilities": canon.CapabilityCount, "actions": canon.ActionCount})
	os.Exit(0)
}

func trimWhitespace(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' { continue }
		out = append(out, r)
	}
	return string(out)
}

func emitResult(jsonMode bool, success bool, status string, extra map[string]any) {
	if jsonMode {
		obj := map[string]any{"success": success, "status": status}
		for k, v := range extra { obj[k] = v }
		b, _ := json.Marshal(obj)
		fmt.Println(string(b))
		return
	}
	if success {
		fmt.Printf("[verify-manifest] SUCCESS status=%s details=%v\n", status, extra)
	} else {
		fmt.Printf("[verify-manifest] FAILURE status=%s details=%v\n", status, extra)
	}
}
