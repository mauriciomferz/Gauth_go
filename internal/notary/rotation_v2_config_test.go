package notary

import (
	"os"
	"testing"
	"time"
)

func TestLoadWeightsConfigValid(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "weights-*.json")
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	content := `{"schema_version":1,"active_key_set_id":"set","threshold_weight":3,"signers":[{"id":"b","alg":"ED25519","weight":1},{"id":"a","alg":"ED25519","weight":2}],"algorithm_suite":["ed25519"]}`
	if _, err2 := tmp.WriteString(content); err2 != nil {
		t.Fatalf("write: %v", err)
	}
	tmp.Close()
	cfg, err := LoadWeightsConfig(tmp.Name())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.ThresholdWeight != 3 {
		t.Fatalf("threshold mismatch")
	}
	art, err := BuildArtifactFromConfig(cfg, "prev", time.Unix(0, 0))
	if err != nil {
		t.Fatalf("artifact: %v", err)
	}
	// Signer order should be a then b after digest build
	if len(art.Signers) != 2 || art.Signers[0].ID != "a" || art.Signers[1].ID != "b" {
		t.Fatalf("unexpected signer order: %#v", art.Signers)
	}
	if art.CanonicalDigest == "" {
		t.Fatalf("digest empty")
	}
}

func TestLoadWeightsConfigDuplicate(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "weights-dupe-*.json")
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	content := `{"schema_version":1,"active_key_set_id":"set","threshold_weight":3,"signers":[{"id":"a","alg":"ED25519","weight":1},{"id":"a","alg":"ED25519","weight":2}],"algorithm_suite":["ed25519"]}`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	tmp.Close()
	if _, err := LoadWeightsConfig(tmp.Name()); err == nil {
		t.Fatalf("expected duplicate id error")
	}
}
