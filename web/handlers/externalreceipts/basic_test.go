package externalreceipts_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	anchorint "github.com/mauriciomferz/AgentAuth/internal/anchor"
	externalreceipts "github.com/mauriciomferz/AgentAuth/web/handlers/externalreceipts"
)

// Clean test implementation replacing corrupted content.
type stubStore struct {
	entries []anchorint.StoredExternalAnchorReceipt
}

func (s *stubStore) Latest() anchorint.StoredExternalAnchorReceipt {
	if len(s.entries) == 0 {
		return anchorint.StoredExternalAnchorReceipt{}
	}
	return s.entries[len(s.entries)-1]
}
func (s *stubStore) Entries() []anchorint.StoredExternalAnchorReceipt { return s.entries }

type depsImpl struct {
	store         *stubStore
	integrity     string
	lastVerifyAge uint64
}

// ExternalReceiptStore returns nil interface when underlying pointer is nil to allow handler unconfigured detection.
func (d *depsImpl) ExternalReceiptStore() any {
	if d.store == nil {
		return nil
	}
	return d.store
}
func (d *depsImpl) ExternalReceiptStoreLatest() anchorint.StoredExternalAnchorReceipt {
	if d.store == nil {
		return anchorint.StoredExternalAnchorReceipt{}
	}
	return d.store.Latest()
}
func (d *depsImpl) ExternalReceiptStoreEntries() []anchorint.StoredExternalAnchorReceipt {
	if d.store == nil {
		return []anchorint.StoredExternalAnchorReceipt{}
	}
	return d.store.Entries()
}
func (d *depsImpl) ExternalReceiptIntegrityStatus() string            { return d.integrity }
func (d *depsImpl) ExternalReceiptLastVerify() time.Time              { return time.Time{} }
func (d *depsImpl) SetExternalAnchorReceiptsIntegrity(s string)       { d.integrity = s }
func (d *depsImpl) SetExternalAnchorReceiptsLastVerifyAge(age uint64) { d.lastVerifyAge = age }

func TestExternalReceiptsLatestUnconfigured(t *testing.T) {
	r := gin.New()
	deps := &depsImpl{store: nil}
	externalreceipts.RegisterChain(r.Group("/api/v1/beta"), deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/capabilities/anchor/external/receipts/latest", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool `json:"success"`
		Configured bool `json:"configured"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || resp.Configured {
		t.Fatalf("expected configured=false got %+v", resp)
	}
}

func TestExternalReceiptsVerifyEmpty(t *testing.T) {
	r := gin.New()
	deps := &depsImpl{store: &stubStore{entries: []anchorint.StoredExternalAnchorReceipt{}}}
	externalreceipts.RegisterChain(r.Group("/api/v1/beta"), deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/capabilities/anchor/external/receipts/verify", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Integrity  string `json:"integrity"`
		Total      int    `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || !resp.Configured || resp.Integrity != "empty" || resp.Total != 0 {
		t.Fatalf("unexpected resp %+v", resp)
	}
}

// helper to build a stored receipt with computed chain hash based on previous hash and base fields
func buildReceipt(prev string, hash string, provider string, version int, latency float64) anchorint.StoredExternalAnchorReceipt {
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	base := struct {
		Hash           string  `json:"hash"`
		Timestamp      string  `json:"timestamp"`
		Provider       string  `json:"provider"`
		Version        int     `json:"version"`
		LatencySeconds float64 `json:"latency_seconds"`
		PrevHash       string  `json:"prev_hash"`
	}{Hash: hash, Timestamp: ts, Provider: provider, Version: version, LatencySeconds: latency, PrevHash: prev}
	enc, _ := json.Marshal(base)
	h := sha256.Sum256(append([]byte(prev), enc...))
	chain := fmt.Sprintf("%x", h[:])
	return anchorint.StoredExternalAnchorReceipt{ExternalAnchorReceipt: anchorint.ExternalAnchorReceipt{Hash: hash, Timestamp: ts, Provider: provider, Version: version, LatencySeconds: latency}, PrevHash: prev, ChainHash: chain}
}

func TestExternalReceiptsVerifyOK(t *testing.T) {
	first := buildReceipt("", "h1", "prov", 1, 0.01)
	second := buildReceipt(first.ChainHash, "h2", "prov", 1, 0.02)
	r := gin.New()
	deps := &depsImpl{store: &stubStore{entries: []anchorint.StoredExternalAnchorReceipt{first, second}}}
	externalreceipts.RegisterChain(r.Group("/api/v1/beta"), deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/capabilities/anchor/external/receipts/verify", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Integrity  string `json:"integrity"`
		Total      int    `json:"total"`
		ChainHead  string `json:"chain_head"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || !resp.Configured || resp.Integrity != "ok" || resp.Total != 2 || resp.ChainHead == "" {
		t.Fatalf("unexpected resp %+v", resp)
	}
}

func TestExternalReceiptsVerifyMismatch(t *testing.T) {
	first := buildReceipt("", "h1", "prov", 1, 0.01)
	second := buildReceipt(first.ChainHash, "h2", "prov", 1, 0.02)
	// Force mismatch by altering chain hash
	second.ChainHash = "deadbeef"
	r := gin.New()
	deps := &depsImpl{store: &stubStore{entries: []anchorint.StoredExternalAnchorReceipt{first, second}}}
	externalreceipts.RegisterChain(r.Group("/api/v1/beta"), deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/capabilities/anchor/external/receipts/verify", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool           `json:"success"`
		Configured bool           `json:"configured"`
		Integrity  string         `json:"integrity"`
		Total      int            `json:"total"`
		Details    map[string]any `json:"details"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || !resp.Configured || resp.Integrity != "mismatch" || resp.Total != 2 || resp.Details == nil {
		t.Fatalf("unexpected resp %+v", resp)
	}
}

func TestExternalReceiptsChainListing(t *testing.T) {
	first := buildReceipt("", "h1", "prov", 1, 0.01)
	second := buildReceipt(first.ChainHash, "h2", "prov", 1, 0.02)
	r := gin.New()
	deps := &depsImpl{store: &stubStore{entries: []anchorint.StoredExternalAnchorReceipt{first, second}}}
	externalreceipts.RegisterChain(r.Group("/api/v1/beta"), deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/capabilities/anchor/external/receipts", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool             `json:"success"`
		Configured bool             `json:"configured"`
		Total      int              `json:"total"`
		Entries    []map[string]any `json:"entries"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || !resp.Configured || resp.Total != 2 || len(resp.Entries) != 2 {
		t.Fatalf("unexpected resp %+v", resp)
	}
}

func TestExternalReceiptsLatestEmpty(t *testing.T) {
	r := gin.New()
	deps := &depsImpl{store: &stubStore{entries: []anchorint.StoredExternalAnchorReceipt{}}}
	externalreceipts.RegisterChain(r.Group("/api/v1/beta"), deps)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/beta/capabilities/anchor/external/receipts/latest", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success    bool `json:"success"`
		Configured bool `json:"configured"`
		Empty      bool `json:"empty"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || !resp.Configured || !resp.Empty {
		t.Fatalf("unexpected resp %+v", resp)
	}
}
