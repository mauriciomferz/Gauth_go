// rotation_v2.js - fetch & render Rotation V2 weighted multi-sig artifact summary
// Expected backend endpoint: /api/v1/rotation/summary/v2
// Sample response shape (approx):
// {
//   "success": true,
//   "artifact": { "version":2, "active_key_set_id":"ks1", "previous_artifact_hash":"...",
//       "threshold_weight":100, "signers":[{"kid":"k1","alg":"ED25519","weight":40,"sig":"..."}],
//       "algorithm_suite":["ED25519"], "canonical_digest":"...", "generated_at":"2025-01-01T00:00:00Z" },
//   "verified_weight":120,
//   "verified_weight_by_alg": { "ED25519":120 },
//   "failures": ["missing_pub_key:k2"],
//   "continuity": {"previous_hash":"abc","latest_hash":"def"}
// }

async function fetchJSONWithRetry(url, attempt){
  const a = attempt || 1;
  try {
    const res = await fetch(url, {headers:{'Accept':'application/json'}});
    if(!res.ok) throw new Error('HTTP '+res.status);
    return await res.json();
  } catch(e){
    if(a < 3){
      await new Promise(r=>setTimeout(r, 500 * Math.pow(2,a-1)));
      return fetchJSONWithRetry(url, a+1);
    }
    throw e;
  }
}

function setText(id, value){ const el = document.getElementById(id); if(el) el.textContent = value; }
function formatSig(sig){ if(!sig) return '—'; return sig.length > 20 ? sig.slice(0,20)+'…' : sig; }

function renderStatusBadge(verifiedWeight, threshold){
  const el = document.getElementById('rotv2-threshold-status');
  if(!el) return;
  if(!threshold || threshold <=0){ el.textContent = 'n/a'; el.className='badge badge-neutral'; return; }
  if(verifiedWeight >= threshold){
    el.textContent = 'Threshold Met';
    el.className = 'badge badge-ok';
  } else {
    el.textContent = 'Partial';
    el.className = 'badge badge-partial';
  }
}

async function verifySignersClient(artifact){
  if(!artifact || !artifact.signers) return {total: artifact ? artifact.signers.length : 0, ok: 0, results: []};
  const payload = artifact.canonical_digest ? new TextEncoder().encode('GAUTH_ROTATION_V2:'+artifact.canonical_digest) : null;
  if(!payload) return {total: artifact.signers.length, ok: 0, results: []};
  const subtle = (window.crypto && window.crypto.subtle);
  const results = [];
  let okCount = 0;
  for(const s of artifact.signers){
    const id = s.id || s.kid || s.signer;
    const alg = (s.alg||'').toUpperCase();
    const sigB64 = s.signature || s.sig || s.Sig;
    const pubB64 = s.public; // expecting base64url encoded public key bytes
    let status = 'skipped';
    if(alg === 'ED25519' && subtle && pubB64 && sigB64){
      try {
        const pubRaw = Uint8Array.from(atob(pubB64.replace(/-/g,'+').replace(/_/g,'/')), c=>c.charCodeAt(0));
        const key = await subtle.importKey('raw', pubRaw, {name:'Ed25519'}, false, ['verify']);
        const sigRaw = Uint8Array.from(atob(sigB64.replace(/-/g,'+').replace(/_/g,'/')), c=>c.charCodeAt(0));
        const valid = await subtle.verify({name:'Ed25519'}, key, sigRaw, payload);
        status = valid ? 'ok' : 'fail';
      } catch(e){ status = 'error'; }
    }
    if(status === 'ok') okCount++;
    results.push({id, alg, status});
  }
  return {total: artifact.signers.length, ok: okCount, results};
}

function renderAlgWeights(map){
  const tbody = document.querySelector('#rotv2-alg-weights tbody');
  if(!tbody) return;
  tbody.innerHTML='';
  if(!map || Object.keys(map).length===0){
    const tr = document.createElement('tr');
    tr.innerHTML = '<td colspan="2" style="padding:2px 4px; opacity:.6;">No weights</td>';
    tbody.appendChild(tr);
    return;
  }
  Object.keys(map).sort().forEach(alg => {
    const tr = document.createElement('tr');
    tr.innerHTML = `<td style="padding:2px 4px;">${alg}</td><td style="padding:2px 4px; text-align:right;">${map[alg]}</td>`;
    tbody.appendChild(tr);
  });
}

function renderSigners(signers){
  const tbody = document.querySelector('#rotv2-signers tbody');
  if(!tbody) return;
  tbody.innerHTML='';
  if(!signers || signers.length === 0){
    const tr = document.createElement('tr');
    tr.innerHTML = '<td colspan="4" style="padding:2px 4px; opacity:.6;">No signers</td>';
    tbody.appendChild(tr);
    return;
  }
  signers.forEach(s => {
    const id = s.id || s.kid || s.signer || s.Kid;
    const alg = s.alg || s.Alg;
    const weight = s.weight || s.Weight || 0;
    const sig = s.signature || s.sig || s.Sig;
    const tr = document.createElement('tr');
    tr.innerHTML = `<td style="padding:2px 4px; font-family:monospace;">${id}</td>`+
      `<td style=\"padding:2px 4px;\">${alg}</td>`+
      `<td style=\"padding:2px 4px; text-align:right;\">${weight}</td>`+
      `<td style=\"padding:2px 4px; font-family:monospace;\">${formatSig(sig)}</td>`;
    tbody.appendChild(tr);
  });
}

function renderFailures(failures){
  const wrap = document.getElementById('rotv2-failures-wrapper');
  const list = document.getElementById('rotv2-failures');
  if(!wrap || !list) return;
  if(!failures || failures.length === 0){ wrap.style.display='none'; list.innerHTML=''; return; }
  wrap.style.display='block';
  list.innerHTML='';
  failures.slice(0,50).forEach(f => { const li=document.createElement('li'); li.textContent=f; list.appendChild(li); });
  if(failures.length>50){ const li=document.createElement('li'); li.textContent=`… ${failures.length-50} more`; list.appendChild(li);}  
}

export async function initRotationV2Panel(){
  const statusEl = document.getElementById('rotation-v2-status');
  if(!statusEl) return; // panel not present
  let lastData = null;
  function attachActions(){
    const btnDigest = document.getElementById('rotv2-copy-latest');
    const btnPrev = document.getElementById('rotv2-copy-prev');
    const btnDownload = document.getElementById('rotv2-download');
    if(btnDigest){
      btnDigest.onclick = async () => {
        if(!lastData || !lastData.artifact) return;
        const text = lastData.artifact.canonical_digest || '';
        try { await navigator.clipboard.writeText(text); btnDigest.textContent='Copied!'; setTimeout(()=>btnDigest.textContent='Copy Digest',1200);} catch(_){ btnDigest.textContent='Copy Failed'; setTimeout(()=>btnDigest.textContent='Copy Digest',1400); }
      };
    }
    if(btnPrev){
      btnPrev.onclick = async () => {
        if(!lastData || !lastData.artifact) return;
        const text = lastData.artifact.previous_artifact_hash || '';
        try { await navigator.clipboard.writeText(text); btnPrev.textContent='Copied!'; setTimeout(()=>btnPrev.textContent='Copy Prev',1200);} catch(_){ btnPrev.textContent='Copy Failed'; setTimeout(()=>btnPrev.textContent='Copy Prev',1400); }
      };
    }
    if(btnDownload){
      btnDownload.onclick = () => {
        if(!lastData) return;
        const blob = new Blob([JSON.stringify(lastData, null, 2)], {type:'application/json'});
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url; a.download = 'rotation_v2_artifact.json';
        document.body.appendChild(a); a.click();
        setTimeout(()=>{ URL.revokeObjectURL(url); a.remove(); }, 1000);
      };
    }
  }
  attachActions();
  async function load(){
    try {
      statusEl.textContent='Loading…';
      const data = await fetchJSONWithRetry('/api/v1/rotation/summary/v2');
      if(!data || data.success === false){ statusEl.textContent='Unavailable'; return; }
      lastData = data;
      const art = data.artifact || {};
      setText('rotv2-verified-weight', data.verified_weight != null ? data.verified_weight : '0');
      setText('rotv2-threshold', art.threshold_weight != null ? art.threshold_weight : '—');
      setText('rotv2-prev-hash', data.continuity && data.continuity.previous_hash || art.previous_artifact_hash || '—');
      setText('rotv2-latest-hash', data.continuity && data.continuity.latest_hash || art.canonical_digest || '—');
      setText('rotv2-keyset-id', art.active_key_set_id || '—');
      setText('rotv2-generated-at', art.generated_at || '—');
      setText('rotv2-alg-suite', (art.algorithm_suite && art.algorithm_suite.join(',')) || '—');
      renderStatusBadge(data.verified_weight || 0, art.threshold_weight || 0);
      renderAlgWeights(data.verified_weight_by_alg || {});
      renderSigners(art.signers || []);
      renderFailures(data.failures || []);
      // Client-side verification (best effort; depends on embedded public keys)
      const verify = await verifySignersClient(art);
      // augment table rows with verification badges
      const tbody = document.querySelector('#rotv2-signers tbody');
      if(tbody && verify.results.length){
        const rows = tbody.querySelectorAll('tr');
        verify.results.forEach((r, idx)=>{
          const row = rows[idx];
          if(!row) return;
          const cell = document.createElement('td');
          cell.style.padding = '2px 4px';
          cell.style.textAlign = 'center';
          if(r.status === 'ok'){ cell.innerHTML = '<span class="verify-ok">✓</span>'; }
          else if(r.status === 'fail'){ cell.innerHTML = '<span class="verify-fail" title="verification failed">✗</span>'; }
          else if(r.status === 'error'){ cell.innerHTML = '<span class="verify-fail" title="error verifying">!</span>'; }
          else { cell.innerHTML = '<span style="opacity:.4;">—</span>'; }
          row.appendChild(cell);
        });
        // add header cell if not present
        const theadRow = document.querySelector('#rotv2-signers thead tr');
        if(theadRow && theadRow.children.length === 4){
          const th = document.createElement('th');
            th.style.padding='2px 4px'; th.textContent='Verified';
          theadRow.appendChild(th);
        }
      }
      statusEl.textContent = 'Updated '+new Date().toLocaleTimeString();
    } catch(e){
      statusEl.textContent='Error';
      console.warn('[rotation-v2] load failed', e);
    }
  }
  load();
  setInterval(load, 45000);
}
