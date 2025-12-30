package delegation

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

// ReconstructStartRootFromPrefixBlocks attempts to derive the canonical start root using only the
// prefix subtree decomposition provided in ConsistencyProofV2. The current implementation FALLS BACK
// (returns empty string) because the optimized algorithm is not yet finalized. A non-empty return value
// indicates successful reconstruction.
//
// Planned algorithm described in docs/CONSISTENCY_OPTIMIZATION.md.
func ReconstructStartRootFromPrefixBlocks(prefixRoots []string, prefixSizes []int, expectedStartLength int, prefixBridges []string) string {
	if len(prefixRoots) == 0 || len(prefixRoots) != len(prefixSizes) {
		return ""
	}
	sum := 0
	for _, s := range prefixSizes {
		if s <= 0 || (s&(s-1)) != 0 {
			return ""
		}
		sum += s
	}
	if sum != expectedStartLength {
		return ""
	}
	if os.Getenv("AGENTAUTH_CONSISTENCY_V2_FAST") != "1" {
		return ""
	}
	if len(prefixRoots) == 1 {
		return prefixRoots[0]
	}
	// Reconstruct using right-to-left reduction sequence mirrored from proof generation.
	type seg struct {
		h  string
		sz int
	}
	blocks := make([]seg, len(prefixRoots))
	for i := range prefixRoots {
		blocks[i] = seg{h: prefixRoots[i], sz: prefixSizes[i]}
	}
	bridgeIdx := 0
	for len(blocks) > 1 {
		last := blocks[len(blocks)-1]
		prev := blocks[len(blocks)-2]
		parent := sha256.Sum256(append(append([]byte("AGENTAUTH_MERKLE_NODE:"), []byte(prev.h)...), []byte(last.h)...))
		phex := hex.EncodeToString(parent[:])
		if len(prefixBridges) > 0 {
			if bridgeIdx >= len(prefixBridges) || prefixBridges[bridgeIdx] != phex {
				return ""
			}
			bridgeIdx++
		}
		// replace prev, drop last
		blocks[len(blocks)-2] = seg{h: phex, sz: prev.sz + last.sz}
		blocks = blocks[:len(blocks)-1]
		// collapse equal-sized from right
		for len(blocks) >= 2 {
			a := blocks[len(blocks)-2]
			b := blocks[len(blocks)-1]
			if a.sz != b.sz {
				break
			}
			parent2 := sha256.Sum256(append(append([]byte("AGENTAUTH_MERKLE_NODE:"), []byte(a.h)...), []byte(b.h)...))
			phex2 := hex.EncodeToString(parent2[:])
			if len(prefixBridges) > 0 {
				if bridgeIdx >= len(prefixBridges) || prefixBridges[bridgeIdx] != phex2 {
					return ""
				}
				bridgeIdx++
			}
			blocks[len(blocks)-2] = seg{h: phex2, sz: a.sz * 2}
			blocks = blocks[:len(blocks)-1]
		}
	}
	if len(prefixBridges) > 0 && bridgeIdx != len(prefixBridges) {
		return ""
	}
	return blocks[0].h
}
