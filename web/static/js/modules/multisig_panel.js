// multisig_panel.js - visualize multi-signature status of latest signed tree head
// Fetches revocation head endpoint (or consistency latest_tree_head) and renders signer weights, threshold satisfaction.
import { verifyEd25519 } from './eddsa_fallback.js';

async function fetchLatestTreeHead() {
  try {
    const res = await fetch('/api/v1/token/revocation/head');
    if(!res.ok) return null;
    const data = await res.json();
    // Expect latest_tree_head field or fall back to constructing from fields if provided differently.
    if(data.latest_tree_head) return data.latest_tree_head;
    // Fallback legacy shape: attempt to build minimal object
    if(data.merkle_root) {
      return {
        merkle_root: data.merkle_root,
        chain_length: data.revocation_chain_length,
        aggregate_hash: data.aggregate_hash || '',
        signatures: data.signatures || [],
        threshold: data.threshold || 1,
        weights_total: data.weights_total || (data.signatures ? data.signatures.length : 0),
        satisfied_weight: data.satisfied_weight || (data.signatures ? data.signatures.length : 0)
      };
    }
    return null;
  } catch(e){
    console.warn('[multisig] fetch latest tree head failed', e);
    return null;
  }
}

async function fetchEdDSAKeys(){
  try {
    const res = await fetch('/api/v1/crypto/eddsa/keys');
    if(!res.ok) return [];
    const data = await res.json();
    return data.keys || [];
  } catch(e){
    console.warn('[multisig] fetch keys failed', e);
    return [];
  }
}

// Reconstruct canonical tree head bytes (Version-aware) matching server-side signableTreeHeadBytes
function canonicalTreeHeadBytes(sth){
  const version = sth.version || sth.Version || 1;
  const ts = (sth.timestamp || sth.Timestamp || '').replace(/Z$/, 'Z'); // server uses RFC3339
  if(version >= 2){
    const payload = {
      version: version,
      merkle_root: sth.merkle_root || sth.MerkleRoot || '',
      chain_length: sth.chain_length || sth.ChainLength || 0,
      aggregate_hash: sth.aggregate_hash || sth.AggregateHash || '',
      timestamp: ts,
      threshold: sth.threshold || sth.Threshold || 0,
      weights_total: sth.weights_total || sth.WeightsTotal || 0
    };
    return new TextEncoder().encode(JSON.stringify(payload));
  }
  const payload = {
    version: version,
    merkle_root: sth.merkle_root || sth.MerkleRoot || '',
    chain_length: sth.chain_length || sth.ChainLength || 0,
    aggregate_hash: sth.aggregate_hash || sth.AggregateHash || '',
    timestamp: ts
  };
  return new TextEncoder().encode(JSON.stringify(payload));
}

async function verifySignatures(sth){
  if(!sth) return {verified: false, entries: []};
  const sigs = sth.signatures || sth.Signatures || [];
  if(sigs.length === 0) return {verified: false, entries: []};
  const keys = await fetchEdDSAKeys();
  const keyMap = {};
  for(const k of keys){ keyMap[k.kid] = k.public_b64; }
  const payload = canonicalTreeHeadBytes(sth);
  const results = [];
  let allValid = true;
  for(const s of sigs){
    const kid = s.Kid || s.kid;
    const sigB64 = s.Sig || s.sig;
    let ok = false, errMsg = '';
    try {
      const pubB64 = keyMap[kid];
      if(!pubB64) { errMsg = 'missing_public_key'; allValid = false; } else {
        const pubRaw = Uint8Array.from(atob(pubB64.replace(/-/g,'+').replace(/_/g,'/')), c=>c.charCodeAt(0));
        // If WebCrypto Ed25519 not available fall back to naive placeholder (treat as unverifiable)
  const subtle = (window.crypto && window.crypto.subtle);
  if(subtle && subtle.importKey){
          // Ed25519 import (Node 20+ & modern browsers: name 'Ed25519') - may not yet be widely available; wrap in try
          let cryptoValid = false;
          try {
            const keyObj = await subtle.importKey('raw', pubRaw, {name: 'Ed25519'}, false, ['verify']);
            const sigRaw = Uint8Array.from(atob(sigB64.replace(/-/g,'+').replace(/_/g,'/')), c=>c.charCodeAt(0));
            cryptoValid = await subtle.verify({name:'Ed25519'}, keyObj, sigRaw, payload);
          } catch(importErr){
            errMsg = 'webcrypto_unsupported';
          }
          ok = cryptoValid;
          if(!cryptoValid && !errMsg) errMsg = 'verify_failed';
          if(!cryptoValid) allValid = false;
        } else {
          // Fallback attempt (currently stubbed returns false until real implementation added)
          const sigRaw = Uint8Array.from(atob(sigB64.replace(/-/g,'+').replace(/_/g,'/')), c=>c.charCodeAt(0));
          const fallbackOk = verifyEd25519(pubRaw, payload, sigRaw);
          if(fallbackOk){
            ok = true;
          } else {
            errMsg = 'fallback_unverified';
            allValid = false;
          }
        }
      }
    } catch(e){
      errMsg = 'exception';
      allValid = false;
    }
    results.push({kid, weight: s.Weight || s.weight || 1, ok, error: errMsg});
  }
  return {verified: allValid, entries: results};
}

function renderMultiSig(sth, container, verify){
  if(!container) return;
  container.innerHTML = '';
  if(!sth){
    container.innerHTML = '<div class="text-sm text-gray-500">No signed tree head yet.</div>';
    return;
  }
  const threshold = sth.threshold || sth.Threshold || 1;
  const satisfied = sth.satisfied_weight || sth.SatisfiedWeight || 0;
  const weightsTotal = sth.weights_total || sth.WeightsTotal || 0;
  const sigs = sth.signatures || sth.Signatures || [];
  const version = sth.version || sth.Version || 1;
  const chainLength = sth.chain_length || sth.ChainLength || 0;
  const merkleRoot = sth.merkle_root || sth.MerkleRoot || '';
  const remaining = Math.max(0, threshold - satisfied);

  const statusBadge = satisfied >= threshold ? '<span class="px-2 py-1 rounded bg-green-200 text-green-800 text-xs" aria-label="threshold satisfied">Threshold Met</span>' : '<span class="px-2 py-1 rounded bg-yellow-200 text-yellow-800 text-xs" aria-label="threshold not met">Partial</span>';
  const verifiedCount = verify && verify.entries ? verify.entries.filter(e=>e.ok).length : 0;
  const verifyBadge = verify ? (verify.verified ? `<span class=\"ml-2 px-2 py-1 rounded bg-green-100 text-green-700 text-xs\" aria-label=\"all signatures verified\">Verified ${verifiedCount}/${(sigs.length||0)}</span>` : `<span class=\"ml-2 px-2 py-1 rounded bg-red-100 text-red-700 text-xs\" aria-label=\"some signatures failed verification\">Verified ${verifiedCount}/${(sigs.length||0)}</span>`) : '';
  const header = `<div class=\"flex items-center justify-between mb-2\"><div class=\"font-semibold\">Signed Tree Head (v${version})</div><div class=\"flex items-center\">${statusBadge}${verifyBadge}</div></div>`;
  const meta = `<div class="text-xs mb-2 space-y-1">
    <div><span class="font-mono text-gray-700">Root:</span> <span class="break-all font-mono">${merkleRoot || '(empty)'}</span></div>
    <div>Chain Length: <span class="font-semibold">${chainLength}</span></div>
    <div>Threshold: <span class="font-semibold">${threshold}</span> | Satisfied: <span class="font-semibold">${satisfied}</span> / Total: <span class="font-semibold">${weightsTotal}</span></div>
    ${satisfied < threshold ? `<div class="text-amber-700">Need <span class="font-semibold">${remaining}</span> more weight to satisfy threshold.</div>` : '<div class="text-green-700">Multi-sig threshold achieved.</div>'}
  </div>`;
  let table = '<table class="w-full text-xs border border-gray-300"><thead><tr class="bg-gray-100"><th class="p-1 text-left">Signer (kid)</th><th class="p-1">Alg</th><th class="p-1">Weight</th><th class="p-1">Signature (b64url)</th><th class="p-1">Verified</th></tr></thead><tbody>';
  if(sigs.length === 0){
    table += '<tr><td colspan="5" class="p-2 text-center text-gray-500">No signatures</td></tr>';
  } else {
    const verifyMap = {};
    if(verify && verify.entries){ for(const r of verify.entries){ verifyMap[r.kid] = r; } }
    for(const s of sigs){
      const kid = s.Kid || s.kid;
      const vr = verifyMap[kid];
      let cell = '<span class="text-gray-500">n/a</span>';
      if(vr){
        if(vr.ok){ cell = '<span class="text-green-700">ok</span>'; } else { cell = `<span class="text-red-600" title="${vr.error}">fail</span>`; }
      }
      table += `<tr class="border-t"><td class="p-1 font-mono">${kid}</td><td class="p-1">${s.Alg || s.alg}</td><td class="p-1">${s.Weight || s.weight || 1}</td><td class="p-1 break-all font-mono">${s.Sig || s.sig}</td><td class="p-1">${cell}</td></tr>`;
    }
  }
  table += '</tbody></table>';
  const verifyBanner = verify ? (verify.verified ? '<div class="mt-2 text-xs text-green-700">All signatures verified client-side.</div>' : '<div class="mt-2 text-xs text-red-600">One or more signatures could not be verified.</div>') : '';
  container.innerHTML = header + meta + table + verifyBanner;
}

export async function initMultiSigPanel(){
  const container = document.getElementById('multisig-panel');
  if(!container) return; // panel not present
  container.innerHTML = '<div class="text-xs text-gray-500">Loading multi-sig status...</div>';
  const sth = await fetchLatestTreeHead();
  const verify = await verifySignatures(sth);
  renderMultiSig(sth, container, verify);
  // Poll occasionally (simple interval) to reflect newly signed heads
  let pollMs = 15000;
  if(typeof window !== 'undefined'){ pollMs = window.__GAUTH_MULTISIG_POLL || pollMs; }
  setInterval(async ()=>{
    const updated = await fetchLatestTreeHead();
    const verify = await verifySignatures(updated);
    renderMultiSig(updated, container, verify);
  }, pollMs);
}
