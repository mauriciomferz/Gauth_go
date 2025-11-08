// Package main provides a CLI for generating and verifying receipt chain snapshot artifacts
// used in prototype notarization flows. Supports generation with optional previous hash
// chaining and verification mode emitting structured JSON results.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
)

// CLI snapshot generator.
// Example:
//
//	snapshot -receipts=./receipts.json -prev="<prev_hash>" -out=./snapshot.json -pretty
//
// If -out omitted, writes JSON to stdout.
// Exits non-zero on error.
func main() {
	// Mode flags
	verifyMode := flag.Bool("verify", false, "Verify an existing snapshot instead of generating one")
	snapshotFile := flag.String("snapshot", "", "Path to snapshot JSON file for verification")
	// Generation flags
	receiptsPath := flag.String("receipts", "", "Path to receipt chain persistence file")
	prevHash := flag.String("prev", "", "Previous snapshot hash for chaining (optional, generation only)")
	outPath := flag.String("out", "", "Output file path (optional; stdout if empty, generation only)")
	pretty := flag.Bool("pretty", false, "Pretty-print JSON output")
	showMeta := flag.Bool("meta", false, "Include simple generation metadata comment on stdout (ignored if writing file)")
	flag.Parse()

	if *receiptsPath == "" {
		fmt.Fprintln(os.Stderr, "error: -receipts path required")
		os.Exit(2)
	}
	rs := notary.NewReceiptStore(*receiptsPath)
	if err := rs.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "error: load receipts: %v\n", err)
		os.Exit(3)
	}

	if *verifyMode {
		if *snapshotFile == "" {
			fmt.Fprintln(os.Stderr, "error: -snapshot path required in verify mode")
			os.Exit(2)
		}
		f, err := os.Open(*snapshotFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: open snapshot: %v\n", err)
			os.Exit(8)
		}
		b, err := io.ReadAll(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: read snapshot: %v\n", err)
			f.Close()
			os.Exit(9)
		}
		var snap notary.Snapshot
		if err2 := json.Unmarshal(b, &snap); err2 != nil {
			fmt.Fprintf(os.Stderr, "error: parse snapshot JSON: %v\n", err)
			f.Close()
			os.Exit(10)
		}
		res, err := notary.VerifySnapshot(rs, snap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: verify snapshot: %v\n", err)
			f.Close()
			os.Exit(11)
		}
		f.Close()
		out := struct {
			Snapshot notary.Snapshot                   `json:"snapshot"`
			Result   notary.SnapshotVerificationResult `json:"result"`
		}{Snapshot: snap, Result: res}
		var enc []byte
		if *pretty {
			enc, err = json.MarshalIndent(out, "", "  ")
		} else {
			enc, err = json.Marshal(out)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: marshal result: %v\n", err)
			os.Exit(12)
		}
		fmt.Println(string(enc))
		if res.Valid {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	// Generation path
	snap, err := notary.GenerateSnapshot(rs, *prevHash)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: generate snapshot: %v\n", err)
		os.Exit(4)
	}
	var data []byte
	if *pretty {
		data, err = json.MarshalIndent(snap, "", "  ")
	} else {
		data, err = json.Marshal(snap)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: marshal snapshot: %v\n", err)
		os.Exit(5)
	}
	if *outPath != "" {
		tmp := *outPath + ".tmp"
		if err := os.WriteFile(tmp, data, 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "error: write temp file: %v\n", err)
			os.Exit(6)
		}
		if err := os.Rename(tmp, *outPath); err != nil {
			fmt.Fprintf(os.Stderr, "error: rename temp file: %v\n", err)
			os.Exit(7)
		}
		return
	}
	if *showMeta {
		fmt.Printf("// generated_at=%s receipt_count=%d merkle_enabled=%t\n", time.Now().UTC().Format(time.RFC3339Nano), snap.ReceiptCount, snap.MerkleRoot != "")
	}
	fmt.Println(string(data))
}
