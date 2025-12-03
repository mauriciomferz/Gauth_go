// rotation_summary.js - displays rotation ledger summary & signature/anchor status
import { backoffFetchJSON } from './refresh.js';

export function initRotationSummaryPanel(){
  const panel = document.getElementById('rotation-summary-panel');
  if(!panel) return;
  const statusEl = document.getElementById('rotation-summary-status');
  const headEl = document.getElementById('rotation-head-hash');
  const aggEl = document.getElementById('rotation-aggregate-hash');
  const chainLenEl = document.getElementById('rotation-chain-length');
  const sigKidEl = document.getElementById('rotation-signature-kid');
  const sigHashEl = document.getElementById('rotation-signature-hash');
  const anchoredBadge = document.getElementById('rotation-anchored-badge');

  let attempts = 0;
  async function load(){
    attempts++;
    statusEl.textContent = 'Loading…';
    try {
      const data = await backoffFetchJSON('/api/v1/beta/rotations/summary', attempts);
      if(!data.success){ statusEl.textContent='Unavailable'; return; }
      if(!data.configured){ statusEl.textContent='Rotation ledger not configured'; return; }
      const sum = data.summary || {}; // {head_hash, aggregate_hash, chain_length, signature_kid, signature_hash}
      headEl.textContent = sum.head_hash || '—';
      aggEl.textContent = sum.aggregate_hash || '—';
      chainLenEl.textContent = (sum.chain_length ?? '0');
      sigKidEl.textContent = sum.signature_kid || '—';
      sigHashEl.textContent = (sum.signature_hash ? sum.signature_hash.slice(0,32)+'…' : '—');
      if(data.anchored){
        anchoredBadge.classList.remove('hidden');
        anchoredBadge.textContent = 'Anchored';
      } else {
        anchoredBadge.classList.add('hidden');
      }
      statusEl.textContent = 'Updated '+new Date().toLocaleTimeString();
    } catch(err){ statusEl.textContent='Error'; console.warn('[rotation-summary] fetch error', err); }
  }
  load();
  // refresh every 30s (rotation changes are infrequent)
  setInterval(load, 30000);
}
