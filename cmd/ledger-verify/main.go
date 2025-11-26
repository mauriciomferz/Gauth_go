package main

import (
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	internalCrypto "github.com/mauriciomferz/Gauth_go/internal/crypto"
	"github.com/mauriciomferz/Gauth_go/internal/notary"
)

type verifyOutput struct {
	Success     bool   `json:"success"`
	Path        string `json:"path"`
	Entries     int    `json:"entries"`
	Mismatches  int    `json:"mismatches"`
	InvalidSigs int    `json:"invalid_signatures"`
	HeadHash    string `json:"head_hash"`
	Strict      bool   `json:"strict"`
	Error       string `json:"error,omitempty"`
}

func main() {
	strict := flag.Bool("strict", false, "treat unsigned entries as invalid")
	outJSON := flag.Bool("json", true, "emit JSON output")
	flag.Parse()
	if flag.NArg() < 1 {
		emit(verifyOutput{Success: false, Error: "ledger_path_required"}, *outJSON)
		os.Exit(2)
	}
	path := flag.Arg(0)
	led := notary.NewRotationLedger(path)
	if err := led.Load(); err != nil {
		emit(verifyOutput{Success: false, Path: path, Error: fmt.Sprintf("load_failed:%v", err)}, *outJSON)
		os.Exit(1)
	}
	entries := led.Entries()
	// Build pub resolver from global registry (active + historical)
	pubs := map[string]ed25519.PublicKey{}
	if internalCrypto.GlobalEdDSARegistry != nil {
		for _, k := range internalCrypto.GlobalEdDSARegistry.ListCurrent() {
			if len(k.Public) == ed25519.PublicKeySize {
				kid := fmt.Sprintf("ed25519:%x", k.Public[:8])
				pubs[kid] = k.Public
			}
		}
	}
	resolver := func(kid string) ed25519.PublicKey { return pubs[kid] }
	mismatches, invalid := notary.VerifyRotationLedger(entries, *strict, resolver)
	out := verifyOutput{Success: mismatches == 0 && invalid == 0, Path: path, Entries: len(entries), Mismatches: mismatches, InvalidSigs: invalid, HeadHash: led.HeadHash(), Strict: *strict}
	emit(out, *outJSON)
	if !out.Success {
		os.Exit(1)
	}
}

func emit(v verifyOutput, jsonOut bool) {
	if jsonOut {
		enc, _ := json.Marshal(v)
		fmt.Println(string(enc))
	} else {
		if v.Success {
			fmt.Printf("Ledger OK entries=%d head=%s\n", v.Entries, v.HeadHash)
		} else {
			fmt.Printf("Ledger FAIL mismatches=%d invalid_sigs=%d error=%s\n", v.Mismatches, v.InvalidSigs, v.Error)
		}
	}
}
