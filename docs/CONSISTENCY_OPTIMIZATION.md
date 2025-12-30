---
title: Consistency Optimization
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Consistency Proof V2 Optimization Roadmap

## Goal
Reduce `VerifyConsistencyProofV2` start-root verification from O(n) (full tree rebuild) to O(log n) using provided prefix subtree decomposition (`prefix_roots`, `prefix_sizes`).

## Background
The Merkle accumulator uses copy-up promotion for odd node counts: an unpaired last node is promoted upward unchanged. This differs from RFC6962's complete binary expansion semantics and affects reconstruction when concatenating perfect subtree roots.

The current proof exposes `prefix_roots` for maximal power-of-two blocks covering the first `start_length` leaves. Example: start_length=13 -> blocks 8,4,1.

Naively treating these block roots as leaves of a meta-tree and applying copy-up promotion over them does not always reproduce the canonical start root. The canonical tree shape depends on internal pairing boundaries across leaf indices, not just block boundaries.

## Challenges
1. Copy-up promotion breaks simple associative combining of block roots.
2. Binary decomposition yields strictly descending block sizes (e.g., 8,4,1) — no adjacent equal-sized blocks to merge naturally.
3. Canonical parent formation order (left-to-right leaf pairing) may interleave nodes across block boundaries before higher-level combination.

## Proposed Algorithm (Frontier Stack Merge)
Maintain a frontier stack of (rootDigest, size) representing perfect subtrees processed from left to right across the first `start_length` leaves.

Procedure:
1. Initialize empty stack.
2. For each prefix block (root R, size S) in order:
   a. Push (R, S) onto stack.
   b. While top two entries have equal size: pop both, combine parent = H(left.root + right.root), push (parent, size*2).
3. After ingesting all blocks, stack will contain entries with strictly decreasing sizes (due to descending prefix sizes). We must perform a final fold: repeatedly combine the two oldest entries where the left size is greater than the right, but the right subtree logically sits at the boundary required by the global tree shape.

Refinement:
Instead of direct fold, we simulate the global tree height-wise:
- Track a virtual leaf index cursor advancing by block sizes.
- For each block root inserted, attempt upward merges only when the cursor after insertion aligns to an even boundary at that level (i.e., `cursor % (2*blockSize) == 0`). This mirrors how a complete parent over two adjacent perfect subtrees would have formed during original leaf-by-leaf construction.

Pseudocode sketch:
```
cursor = 0
stack = [] // each element: {digest, size}
for each (R,S) in prefixBlocks:
  stack.push({R,S})
  cursor += S
  // attempt merges
  while len(stack) >= 2:
      a = stack[len(stack)-2]; b = stack[len(stack)-1]
      if a.size == b.size && cursor % (2*a.size) == 0:
          parent = HashNode(a.digest, b.digest)
          stack = stack[:len(stack)-2]
          stack.push({parent, a.size*2})
      else:
          break
// Final reduction: perform ordered combining applying promotion when no sibling available.
root = reduceCopyUp(stack)
```
`reduceCopyUp` builds a tree over the remaining stack entries from left to right using copy-up semantics, but only combining pairs whose sizes align at the lowest common level. This requires expanding smaller right subtree into its internal structure conceptually; however because each right subtree is perfect, we can safely combine when sizes match or treat unmatched as promoted upward until a match occurs.

## Complexity
The algorithm performs at most one merge per level per block insertion → O(k) where k = number of prefix blocks (<= log2(start_length)).

## Validation Strategy
1. Property tests comparing result against full rebuild across random sizes (1..N) for N up to large (e.g., 50k) using generated event hashes.
2. Edge cases: powers of two, size = power + 1, size = (2^a + 2^b + 2^c) with gaps, and sizes triggering multiple consecutive merges.
3. Tamper tests altering one prefix root should change reconstruction result but still fail overall verification.

## Incremental Plan
- Phase 1: Implement algorithm returning reconstructed start root; add test harness comparing to full rebuild for small sizes.
- Phase 2: Fuzz/property driven tests (random sequences) ensuring equality.
- Phase 3: Switch verification to algorithm (feature flag off by default initially).
- Phase 4: Remove full rebuild path once confident; keep prefix integrity cross-check.

## Open Questions
- Do we need to include an explicit `prefix_alignment` array to accelerate alignment checks? (Likely no; cursor arithmetic suffices.)
- Should we expose reconstructed start root separately to clients? (Probably not; they can recompute or trust path.)

## Future Extensions
- Add `delta_leaves_count` to allow verifying expected growth separate from end_length.
- Provide combined inclusion + consistency multi-proof for batch auditing.
- Offer streaming verification API returning intermediate reconstructed subtree hashes for external monitors.

## Status
Prefix fields generated and validated. Fast reconstruction implemented only for the trivial power-of-two case (single prefix block) behind feature flag `AGENTAUTH_CONSISTENCY_V2_FAST=1`. General multi-block reconstruction deferred due to correctness mismatch observed in early prototype. This document remains the design reference for the future full implementation.
