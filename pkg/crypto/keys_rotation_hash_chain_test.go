package crypto

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"sort"
	"testing"
	"time"
)

// TestRotationLogHashChain ensures each rotation log entry correctly references prior hash and that recomputed hashes match.
func TestRotationLogHashChain(t *testing.T) {
	// Use temp file for log
	tmp, err := os.CreateTemp(t.TempDir(), "rotation_log_*.jsonl")
	if err != nil {
		t.Fatalf("temp file err: %v", err)
	}
	path := tmp.Name()
	if cerr := tmp.Close(); cerr != nil {
		t.Fatalf("close temp: %v", cerr)
	}
	t.Setenv("AGENTAUTH_EDDSA_ROTATION_LOG", path)

	// Short TTL to force multiple rotations quickly
	m, err := NewManager(200 * time.Millisecond)
	if err != nil {
		t.Fatalf("manager err: %v", err)
	}
	defer m.Stop()
	// Perform several rotations
	for i := 0; i < 5; i++ {
		time.Sleep(210 * time.Millisecond)
		if _, rotErr := m.Rotate(); rotErr != nil {
			t.Fatalf("rotate err: %v", rotErr)
		}
	}
	// Read log file
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log err: %v", err)
	}
	lines := bytes.Split(data, []byte{'\n'})
	var prevHash string
	for idx, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var rec map[string]any
		if jErr := json.Unmarshal(line, &rec); jErr != nil {
			t.Fatalf("json parse line %d: %v", idx, jErr)
		}
		// Extract stored hash
		storedHash, ok := rec["hash"].(string)
		if !ok || storedHash == "" {
			t.Fatalf("hash field missing or invalid line %d", idx)
		}
		// Remove hash field for recomputation
		delete(rec, "hash")
		// Remove signature and public_key fields for canonical hash recomputation
		delete(rec, "signature")
		delete(rec, "public_key")
		// Keep prev_hash for integrity check (optional on first entry)
		chainPrev := ""
		if ph, ok := rec["prev_hash"].(string); ok {
			chainPrev = ph
		}
		// Recompute
		// Marshal with sorted keys for canonical JSON
		keys := make([]string, 0, len(rec))
		for k := range rec {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf := bytes.NewBuffer(nil)
		buf.WriteByte('{')
		for i, k := range keys {
			v, _ := json.Marshal(rec[k])
			buf.WriteString("\"")
			buf.WriteString(k)
			buf.WriteString("\":")
			buf.Write(v)
			if i < len(keys)-1 {
				buf.WriteByte(',')
			}
		}
		buf.WriteByte('}')
		h := sha256.Sum256(buf.Bytes())
		recomputed := base64.RawURLEncoding.EncodeToString(h[:])
		if storedHash != recomputed {
			t.Errorf("hash mismatch line %d stored=%s recomputed=%s", idx, storedHash, recomputed)
			t.Logf("Stored JSON: %s", string(line))
			t.Logf("Recomputed JSON: %s", buf.String())
		}
		if idx > 0 { // after first line
			if chainPrev != prevHash {
				t.Fatalf("prev_hash mismatch line %d expected %s got %s", idx, prevHash, chainPrev)
			}
		}
		prevHash = storedHash
	}
	if prevHash == "" {
		t.Fatalf("no hash entries found")
	}
}
