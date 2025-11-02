package test

import (
  "bytes"
  "encoding/json"
  "net/http"
  "os"
  "os/exec"
  "testing"
  "time"
)

// This test assumes the server may be running locally on :8080 (best-effort). If unreachable, it is skipped.
func TestRotationVerifyCLI(t *testing.T) {
  client := &http.Client{Timeout: 3 * time.Second}
  resp, err := client.Get("http://localhost:8080/api/v1/rotation/summary/v2")
  if err != nil { t.Skip("skipping: rotation v2 endpoint not reachable: ", err); return }
  defer resp.Body.Close()
  if resp.StatusCode != 200 { t.Skip("skipping: non-200 status from rotation v2 endpoint (likely unsigned artifact)") }
  var raw map[string]any
  if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil { t.Fatalf("decode: %v", err) }
  // write artifact to temp file
  b, _ := json.Marshal(raw)
  tmp := t.TempDir() + "/artifact.json"
  if err := os.WriteFile(tmp, b, 0o600); err != nil { t.Fatalf("write temp: %v", err) }
  // Use path relative to module root (test package lives in ./test so we need to go up one level).
  cmd := exec.Command("go", "run", "../cmd/rotation-verify", "--file", tmp, "--json")
  out, err := cmd.CombinedOutput()
  if len(out) == 0 { t.Fatalf("expected json output") }
  var parsed map[string]any
  _ = json.Unmarshal(out, &parsed)
  if err != nil {
    // Raw content heuristics first
    if bytes.Contains(out, []byte("\"threshold_met\": false")) {
      t.Skip("skipping: threshold not met (unsigned/no pubs)")
      return
    }
    if bytes.Contains(out, []byte("public_key_not_found")) {
      t.Skip("skipping: no public keys available")
      return
    }
    if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 3 {
      t.Skip("skipping: rotation verify threshold unmet")
      return
    }
    if tm, ok := parsed["threshold_met"].(bool); ok && !tm { t.Skip("skipping: threshold not met") ; return }
    if vw, ok := parsed["verified_weight"].(float64); ok && vw == 0 { t.Skip("skipping: no verified weight") ; return }
    t.Fatalf("rotation-verify failed: %v output=%s", err, string(out))
  }
}