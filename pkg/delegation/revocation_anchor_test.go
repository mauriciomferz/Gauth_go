package delegation

import (
	"testing"
)

type mockAnchor struct {
	lastSTH *SignedTreeHead
	called  bool
}

func (m *mockAnchor) OnRevocationAnchor(sth *SignedTreeHead) error {
	m.lastSTH = sth
	m.called = true
	return nil
}

func TestRevocationExternalAnchor_RFC111_C5(t *testing.T) {
	mock := &mockAnchor{}
	rc := NewRevocationChain(WithAnchorObserver(mock))

	// 1. Append event
	_, err := rc.Append(RevocationEvent{ID: "rev-ext-1", DelegationID: "del-ext-1"})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// 2. Sign Tree Head (triggers anchor)
	// Even without KeyProvider, SignTreeHead should produce an unsigned STH and trigger the anchor if not empty
	sth, err := rc.SignTreeHead()
	if err != nil {
		t.Fatalf("SignTreeHead failed: %v", err)
	}
	if sth == nil {
		t.Fatal("Expected STH")
	}

	// 3. Verify Anchor Observer was called
	if !mock.called {
		t.Fatal("External anchor observer was NOT called")
	}
	if mock.lastSTH == nil {
		t.Fatal("Observer received nil STH")
	}
	if mock.lastSTH.AggregateHash != sth.AggregateHash {
		t.Errorf("Observer received mismatched STH")
	}
}
