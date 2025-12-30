// Revocation Transparency Panel Module
// Polls revocation chain status endpoints and exposes Merkle proof generation.
// Data sources:
//  - /api/v1/.well-known/agentauth (revocation_support object) for rich metadata (tree head snapshot)
//  - /api/v1/token/revocation/head (fast head + length + aggregate + verified)
//  - /api/v1/token/revocation/root (merkle_root + length)
//  - /api/v1/token/revocation/proof (on-demand; id | index | hash)
//  - /api/v1/token/revocation/consistency (future UI; placeholder link)
// Accessible updates via aria-live regions. Resilient fetch with simple backoff.

import { backoffFetchJSON } from './refresh.js';

const HEAD_ENDPOINT = '/api/v1/token/revocation/head';
const ROOT_ENDPOINT = '/api/v1/token/revocation/root';
const WELL_KNOWN = '/api/v1/.well-known/agentauth';
const PROOF_ENDPOINT = '/api/v1/token/revocation/proof';
const CONSISTENCY_ENDPOINT_BASE = '/api/v1/token/revocation/consistency';

// Poll cadence (ms)
const POLL_INTERVAL = 20000; // 20s
const WELL_KNOWN_INTERVAL = 60000; // 60s (metadata slower moving)

let lastTreeHeadTimestamp = null;
let lastProof = null; // cached JSON response from last inclusion proof fetch
let lastLeafDigest = null; // derived leaf digest (if event_hash present)
let verifying = false;
let lastConsistency = null; // cached consistency proof response
let wellKnownAvailable = true; // Track if endpoint exists

export function initRevocationTransparency() {
  const contentEl = document.getElementById('revocation-transparency-content');
  if (!contentEl) return;

  attachProofHandler();
  attachConsistencyHandler();
  attachVerifyHandler();
  // Initial immediate polls
  pollHead();
  pollRoot();
  pollWellKnown();
  // Schedule recurring polls
  setInterval(pollHead, POLL_INTERVAL);
  setInterval(pollRoot, POLL_INTERVAL);
  setInterval(() => {
    if (wellKnownAvailable) pollWellKnown();
  }, WELL_KNOWN_INTERVAL);
}

async function pollHead() {
  try {
    const data = await backoffFetchJSON(HEAD_ENDPOINT);
    if (!data || !data.success) return;
    setText('rev-chain-head', data.revocation_chain_head || '—');
    setText('rev-chain-length', data.revocation_chain_length ?? '—');
    setText('rev-chain-aggregate', data.revocation_chain_aggregate || '');
    renderVerified(data.verified);
  } catch (e) {
    setText('rev-chain-head', '(error)');
  }
}

async function pollRoot() {
  try {
    const data = await backoffFetchJSON(ROOT_ENDPOINT);
    if (!data || !data.success) return;
    setText('rev-merkle-root', data.merkle_root || '—');
    // length from root endpoint may mirror chain length (store if present)
    if (data.length !== undefined) {
      setText('rev-tree-heads-count', data.length);
    }
  } catch (e) {
    setText('rev-merkle-root', '(error)');
  }
}

async function pollWellKnown() {
  try {
    const data = await backoffFetchJSON(WELL_KNOWN);
    if (!data) {
      // Endpoint doesn't exist (404) - stop polling
      wellKnownAvailable = false;
      console.log('📭 .well-known/agentauth endpoint not available - polling disabled');
      return;
    }
    if (!data.revocation_support) return;
    const rs = data.revocation_support;
    // Basic chain meta (prefer richer snapshot if available)
    if (rs.revocation_chain_head) setText('rev-chain-head', rs.revocation_chain_head);
    if (rs.revocation_chain_length !== undefined) setText('rev-chain-length', rs.revocation_chain_length);
    if (typeof rs.revocation_chain_aggregate === 'string') setText('rev-chain-aggregate', rs.revocation_chain_aggregate);
    if (typeof rs.revocation_chain_verified === 'boolean') renderVerified(rs.revocation_chain_verified);
    if (typeof rs.merkle_root === 'string') setText('rev-merkle-root', rs.merkle_root);
    if (typeof rs.proof_endpoints !== 'undefined') setText('rev-proof-endpoints', Array.isArray(rs.proof_endpoints) ? rs.proof_endpoints.join(', ') : rs.proof_endpoints);
    if (rs.consistency_endpoint) setText('rev-consistency-endpoint', rs.consistency_endpoint);
    if (typeof rs.tree_heads_count === 'number') setText('rev-tree-heads-count', rs.tree_heads_count);

    // Latest tree head snapshot (object) contains timestamp & signature meta
    if (typeof rs.latest_tree_head === 'object' && rs.latest_tree_head !== null) {
      const th = rs.latest_tree_head;
      if (th.timestamp) {
        lastTreeHeadTimestamp = Date.parse(th.timestamp);
        setText('rev-tree-head-timestamp', th.timestamp);
      }
      if (th.threshold !== undefined) setText('rev-threshold', th.threshold);
      if (th.satisfied_weight !== undefined) setText('rev-satisfied-weight', th.satisfied_weight);
      if (th.weights_total !== undefined) setText('rev-total-weight', th.weights_total);
      updateTreeHeadAge();
    }

    // Staleness heuristic: if tree head age > 2 minutes mark stale
    updateStaleness();
  } catch (e) {
    // Defer; intermittent errors okay
  }
}

function updateTreeHeadAge() {
  if (!lastTreeHeadTimestamp) return;
  const ageSec = Math.floor((Date.now() - lastTreeHeadTimestamp) / 1000);
  setText('rev-tree-head-age', `age: ${ageSec}s`);
}

function updateStaleness() {
  if (!lastTreeHeadTimestamp) return;
  const ageSec = Math.floor((Date.now() - lastTreeHeadTimestamp) / 1000);
  const staleEl = document.getElementById('rev-staleness');
  const badge = document.getElementById('rev-stale-flag');
  if (!staleEl || !badge) return;
  staleEl.classList.remove('hidden');
  if (ageSec > 120) {
    badge.textContent = 'stale';
    badge.className = 'px-2 py-0.5 rounded text-xs font-semibold bg-red-100 text-red-700';
  } else if (ageSec > 60) {
    badge.textContent = 'warming';
    badge.className = 'px-2 py-0.5 rounded text-xs font-semibold bg-yellow-100 text-yellow-700';
  } else {
    badge.textContent = 'fresh';
    badge.className = 'px-2 py-0.5 rounded text-xs font-semibold bg-green-100 text-green-700';
  }
}

function renderVerified(ok) {
  const el = document.getElementById('rev-chain-verified');
  if (!el) return;
  if (ok) {
    el.textContent = 'verified';
    el.className = 'px-2 py-0.5 rounded text-xs font-semibold bg-green-100 text-green-700';
  } else {
    el.textContent = 'unverified';
    el.className = 'px-2 py-0.5 rounded text-xs font-semibold bg-red-100 text-red-700';
  }
}

function setText(id, value) {
  const el = document.getElementById(id);
  if (el) el.textContent = value;
}

function attachProofHandler() {
  // Initialize proof fetcher UI
  initProofFetcher();
}

function attachConsistencyHandler() {
  const btn = document.getElementById('rev-consistency-btn');
  const startInput = document.getElementById('rev-consistency-start');
  const targetInput = document.getElementById('rev-consistency-target');
  const resultEl = document.getElementById('rev-consistency-result');
  const verifyBtn = document.getElementById('rev-consistency-verify-btn');
  if (!btn || !startInput || !targetInput || !resultEl) return;
  btn.addEventListener('click', async () => {
    const start = parseInt(startInput.value, 10) || 0;
    let target = parseInt(targetInput.value, 10) || 0;
    // target length of 0 signals latest
    const qp = new URLSearchParams();
    qp.set('start', String(start));
    if (target > 0) qp.set('target_length', String(target));
    const url = `${CONSISTENCY_ENDPOINT_BASE}?${qp.toString()}`;
    resultEl.textContent = 'Fetching consistency…';
    try {
      const data = await backoffFetchJSON(url);
      if (!data || !data.success) {
        resultEl.textContent = 'Consistency unavailable';
        lastConsistency = null;
        verifyBtn?.setAttribute('disabled','disabled');
        return;
      }
      lastConsistency = data;
      // Heuristic: enable verify if proof has start_length & end_length or start_root/end_root style fields
      if (data.proof && data.latest_tree_head) {
        verifyBtn?.removeAttribute('disabled');
      } else {
        verifyBtn?.setAttribute('disabled','disabled');
      }
      resultEl.textContent = JSON.stringify(data, null, 2);
    } catch (e) {
      resultEl.textContent = `Error: ${e.message || e}`;
      lastConsistency = null;
      verifyBtn?.setAttribute('disabled','disabled');
    }
  });
}

function attachVerifyHandler() {
  const btn = document.getElementById('rev-verify-btn');
  const resultEl = document.getElementById('rev-verify-result');
  const statusEl = document.getElementById('rev-verify-status');
  const consistencyBtn = document.getElementById('rev-consistency-verify-btn');
  const consistencyResultEl = document.getElementById('rev-consistency-verify-result');
  if (!btn || !resultEl || !statusEl) return;
  btn.addEventListener('click', () => {
    if (!lastProof || !lastLeafDigest) {
      resultEl.textContent = '(no cached proof)';
      return;
    }
    if (verifying) return; // guard double clicks
    verifying = true;
    statusEl.textContent = 'verifying';
    try {
      const merkleRootEl = document.getElementById('rev-merkle-root');
      const expectedRoot = merkleRootEl ? (merkleRootEl.textContent || '').trim() : '';
      const proofSteps = (lastProof.proof || []).map(step => ({ sibling: step.sibling || step.Sibling, position: step.position || step.Position }));
      const recomputed = computeRootFromProof(lastLeafDigest, proofSteps);
      const ok = expectedRoot && recomputed === expectedRoot;
      resultEl.textContent = JSON.stringify({
        expected_root: expectedRoot,
        recomputed_root: recomputed,
        match: ok,
        leaf_digest: lastLeafDigest,
        steps: proofSteps.length
      }, null, 2);
      statusEl.textContent = ok ? 'verified' : 'mismatch';
      statusEl.className = ok ? 'text-[10px] text-green-600 font-semibold' : 'text-[10px] text-red-600 font-semibold';
    } catch (e) {
      resultEl.textContent = `Error: ${e.message || e}`;
      statusEl.textContent = 'error';
      statusEl.className = 'text-[10px] text-red-600 font-semibold';
    } finally {
      verifying = false;
    }
  });

  if (consistencyBtn && consistencyResultEl) {
    consistencyBtn.addEventListener('click', () => {
      if (!lastConsistency) {
        consistencyResultEl.textContent = '(no consistency proof)';
        return;
      }
      // Basic append-only check: ensure latest_tree_head.chain_length >= start_length and proof length plausible.
      try {
        const th = lastConsistency.latest_tree_head || {};
        const chainLen = th.chain_length ?? th.length ?? 0;
        // server consistency proof schema (prototype): {proof:{start_length,end_length,start_root,end_root,new_leaves:[]}, latest_tree_head:{chain_length,merkle_root,...}}
        const proof = lastConsistency.proof || {};
        const startLen = proof.start_length ?? 0;
        const endLen = proof.end_length ?? chainLen;
        const newLeaves = Array.isArray(proof.new_leaves) ? proof.new_leaves : [];
        const conditions = [];
        conditions.push(['length_non_decreasing', endLen >= startLen]);
        conditions.push(['end_matches_latest_or_target', chainLen === endLen]);
        conditions.push(['new_leaves_count_match', (endLen - startLen) === newLeaves.length]);
        // Evaluate
        const allOk = conditions.every(c => c[1]);
        consistencyResultEl.textContent = JSON.stringify({
          start_length: startLen,
          end_length: endLen,
          latest_chain_length: chainLen,
          new_leaves: newLeaves.length,
          checks: conditions.map(c => ({name: c[0], ok: c[1]})),
          append_only_verified: allOk
        }, null, 2);
        // Visual hint (re-use small status element if present)
        const verifyStatus = document.getElementById('rev-verify-status');
        if (verifyStatus) {
          verifyStatus.textContent = allOk ? 'verified' : 'mismatch';
          verifyStatus.className = allOk ? 'text-[10px] text-green-600 font-semibold' : 'text-[10px] text-red-600 font-semibold';
        }
      } catch (e) {
        consistencyResultEl.textContent = `Error: ${e.message || e}`;
      }
    });
  }
}

// Derive leaf digest from proof response; server may include event_hash or leaf_digest directly.
function deriveLeafDigest(data) {
  if (!data) return null;
  if (data.leaf_digest) return data.leaf_digest;
  if (data.event_hash && /^[a-f0-9]{16,}$/i.test(data.event_hash) {
    // mirror Go LeafDigestForEventHash: SHA256("AGENTAUTH_MERKLE_LEAF:" + event_hash) hex
    const enc = new TextEncoder();
    const prefix = enc.encode('AGENTAUTH_MERKLE_LEAF:');
    const ev = enc.encode(data.event_hash);
    const combined = new Uint8Array(prefix.length + ev.length);
    combined.set(prefix, 0); combined.set(ev, prefix.length);
    const hashBuf = sha256Bytes(combined);
    return toHex(hashBuf);
  }
  return null;
}

// Compute root from leaf digest + ordered proof steps (array of {sibling, position})
function computeRootFromProof(leafDigest, steps) {
  if (!leafDigest || !Array.isArray(steps) return '';
  let cur = leafDigest;
  for (const step of steps) {
    const sibling = step.sibling;
    const pos = step.position; // 'R' or 'L'
    if (!sibling || (pos !== 'R' && pos !== 'L') return '';
    if (pos === 'R') {
      cur = merkleParent(cur, sibling);
    } else {
      cur = merkleParent(sibling, cur);
    }
  }
  return cur;
}

// Merkle parent digest: SHA256("AGENTAUTH_MERKLE_NODE:" + left + right)
function merkleParent(leftHex, rightHex) {
  const enc = new TextEncoder();
  const prefix = enc.encode('AGENTAUTH_MERKLE_NODE:');
  const left = enc.encode(leftHex);
  const right = enc.encode(rightHex);
  const combined = new Uint8Array(prefix.length + left.length + right.length);
  combined.set(prefix, 0); combined.set(left, prefix.length); combined.set(right, prefix.length + left.length);
  const hashBuf = sha256Bytes(combined);
  return toHex(hashBuf);
}

// Lightweight SHA-256 using Web Crypto if available, fallback to js library (requires subtle digest synchronous wrapper)
function sha256Bytes(data) {
  if (window.crypto && window.crypto.subtle) {
    // WebCrypto is async; but verification path can be async simplified by blocking (we'll fake sync via deasync pattern). Simpler: use synchronous implementation.
    // For now implement a tiny sync SHA-256 (precomputed) using built-in in browsers isn't exposed sync; fallback minimal implementation omitted for brevity.
  }
  // Minimal pure JS SHA-256 (optimized for small inputs) adapted from public domain implementation.
  return sha256PureJS(data);
}

function sha256PureJS(messageBytes) {
  // Convert to array of 32-bit words
  const h = [0x6a09e667,0xbb67ae85,0x3c6ef372,0xa54ff53a,0x510e527f,0x9b05688c,0x1f83d9ab,0x5be0cd19];
  const k = [
    0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,
    0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,
    0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,
    0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,
    0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,
    0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,
    0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,
    0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2
  ];
  // Pre-processing (padding)
  const l = messageBytes.length;
  const withOne = new Uint8Array(l+1); withOne.set(messageBytes,0); withOne[l]=0x80;
  let paddedLength = withOne.length;
  while ((paddedLength * 8) % 512 !== 448) paddedLength++;
  const padded = new Uint8Array(paddedLength + 8);
  padded.set(withOne,0);
  const bitLen = l * 8;
  for (let i=0;i<8;i++) padded[padded.length-1-i] = bitLen >>> (8*i) & 0xff;
  // Process blocks
  const w = new Uint32Array(64);
  for (let i=0;i<padded.length;i+=64) {
    for (let j=0;j<16;j++) {
      w[j] = (padded[i+4*j]<<24)|(padded[i+4*j+1]<<16)|(padded[i+4*j+2]<<8)|(padded[i+4*j+3]);
    }
    for (let j=16;j<64;j++) {
      const s0 = rightRotate(w[j-15],7)^rightRotate(w[j-15],18)^(w[j-15]>>>3);
      const s1 = rightRotate(w[j-2],17)^rightRotate(w[j-2],19)^(w[j-2]>>>10);
      w[j] = (w[j-16]+s0+w[j-7]+s1)>>>0;
    }
    let a=h[0],b=h[1],c=h[2],d=h[3],e=h[4],f=h[5],g=h[6],hh=h[7];
    for (let j=0;j<64;j++) {
      const S1 = rightRotate(e,6)^rightRotate(e,11)^rightRotate(e,25);
      const ch = (e & f) ^ (~e & g);
      const temp1 = (hh + S1 + ch + k[j] + w[j])>>>0;
      const S0 = rightRotate(a,2)^rightRotate(a,13)^rightRotate(a,22);
      const maj = (a & b) ^ (a & c) ^ (b & c);
      const temp2 = (S0 + maj)>>>0;
      hh=g; g=f; f=e; e=(d + temp1)>>>0; d=c; c=b; b=a; a=(temp1 + temp2)>>>0;
    }
    h[0]=(h[0]+a)>>>0; h[1]=(h[1]+b)>>>0; h[2]=(h[2]+c)>>>0; h[3]=(h[3]+d)>>>0;
    h[4]=(h[4]+e)>>>0; h[5]=(h[5]+f)>>>0; h[6]=(h[6]+g)>>>0; h[7]=(h[7]+hh)>>>0;
  }
  const out = new Uint8Array(32);
  for (let i=0;i<8;i++) {
    out[4*i] = h[i]>>>24; out[4*i+1] = h[i]>>>16 & 0xff; out[4*i+2] = h[i]>>>8 & 0xff; out[4*i+3] = h[i] & 0xff;
  }
  return out;
  function rightRotate(x,n){ return (x>>>n)|(x<<(32-n)); }
}

function toHex(buf) {
  return Array.from(buf).map(b=>b.toString(16).padStart(2,'0')).join('');
}

function initProofFetcher() {
  const btn = document.getElementById('rev-proof-btn');
  const input = document.getElementById('rev-proof-input');
  const result = document.getElementById('rev-proof-result');
  if (!btn || !input || !result) return;
  btn.addEventListener('click', async () => {
    const raw = (input.value || '').trim();
    if (!raw) {
      result.textContent = '(enter id | index | hash)';
      return;
    }
    // Determine query param key heuristically: numeric -> index, hex length >= 16 -> hash, else id
    let qp = 'id';
    if (/^\d+$/.test(raw) qp = 'index';
    else if (/^[a-fA-F0-9]{16,}$/.test(raw) qp = 'hash';
    const url = `${PROOF_ENDPOINT}?${qp}=${encodeURIComponent(raw)}`;
    result.textContent = 'Fetching proof…';
    try {
      const data = await backoffFetchJSON(url);
      if (!data || !data.success) {
        result.textContent = 'Proof unavailable';
        return;
      }
      // Pretty print proof
      result.textContent = JSON.stringify(data, null, 2);
    } catch (e) {
      result.textContent = `Error: ${e.message || e}`;
    }
  });
}

// Periodically update age badge more smoothly without waiting for next metadata poll
setInterval(updateTreeHeadAge, 5000);
setInterval(updateStaleness, 15000);
