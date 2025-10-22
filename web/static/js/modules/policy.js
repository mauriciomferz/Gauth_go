// policy.js - Experimental policy engine UI integration
import { addConsoleOutput, escapeHtml } from "./console.js";

const POLICY_CONSOLE_ID = 'policy-output';

function out(msg, level='info') {
  addConsoleOutput(POLICY_CONSOLE_ID, msg, level);
}

async function fetchJSON(url) {
  const r = await fetch(url);
  if(!r.ok) throw new Error(`${r.status} ${r.statusText}`);
  return r.json();
}

export async function loadProvenance() {
  try {
    out('# Fetching provenance...');
    const data = await fetchJSON('/api/v1/beta/policy/provenance');
    out(`head_hash: ${data.head_hash || '<empty>'}`,'success');
    out(`verified: ${data.verified} ${data.verification_error? 'err='+data.verification_error:''}`,'info');
    out(`chain length: ${data.chain?.length||0}`,'info');
  } catch(e){ out(`✗ provenance error: ${e.message}`,'error'); }
}

export async function loadChainPage(offset=0, limit=10) {
  try {
    out(`# Chain page offset=${offset} limit=${limit}`);
    const data = await fetchJSON(`/api/v1/beta/policy/chain?offset=${offset}&limit=${limit}`);
    (data.hashes||[]).forEach((h,i)=> out(`${offset+i}. ${h}`,'info'));
    out(`returned=${data.returned} total=${data.total} verified=${data.chain_verified}`,'success');
  } catch(e){ out(`✗ chain page error: ${e.message}`,'error'); }
}

export async function checkConsistency() {
  try {
    out('# Checking audit consistency...');
    const data = await fetchJSON('/api/v1/beta/policy/audit-consistency');
    out(`consistent=${data.consistent} chain_verified=${data.chain_verified}`,'info');
  } catch(e){ out(`✗ consistency error: ${e.message}`,'error'); }
}

export async function evaluatePolicy() {
  try {
    const subject = document.getElementById('pol-subject')?.value || 'alice@example.com';
    const action = document.getElementById('pol-action')?.value || 'read';
    const resource = document.getElementById('pol-resource')?.value || 'report:finance';
    let attrs = {};
    try { const raw = document.getElementById('pol-attrs')?.value; if(raw) attrs = JSON.parse(raw); } catch {}
    out(`# Evaluate ${subject} ${action} ${resource}`);
    const r = await fetch('/api/v1/beta/policy/evaluate', {
      method:'POST', headers:{'Content-Type':'application/json'},
      body: JSON.stringify({ subject, action, resource, attrs })
    });
    const data = await r.json();
    if(!data.success){ out(`✗ eval failed: ${data.message||'error'}`,'error'); return; }
    const allow = data.allow === true && data.deny !== true;
    const badge = `<span class='inline-block px-2 py-0.5 rounded text-xs font-semibold ${allow? 'bg-green-600 text-white':'bg-red-600 text-white'}' aria-label='${allow? 'Decision: Allow':'Decision: Deny'}'>${allow? 'ALLOW':'DENY'}</span>`;
    const reason = escapeHtml((data.reason||'').substring(0,180));
    const matched = Array.isArray(data.matched)? data.matched.map(r=>escapeHtml(r)).join(',') : '';
    const deniedBy = Array.isArray(data.denied_by)? data.denied_by.map(r=>escapeHtml(r)).join(',') : '';
    out(`${badge} reason=${reason}`,'success');
    if(matched) out(`matched: ${matched}`,'info');
    if(deniedBy) out(`denied_by: ${deniedBy}`,'warning');
    out(`bundle_hash=${escapeHtml(data.bundle_hash||'')} chain_head=${escapeHtml(data.chain_head||'')} policy_version=${escapeHtml(String(data.policy_version||'-'))}`,'info');
  } catch(e){ out(`✗ evaluate error: ${e.message}`,'error'); }
}

export async function submitBundle() {
  try {
    const txt = document.getElementById('pol-bundle-json')?.value;
    if(!txt){ out('✗ no bundle JSON provided','error'); return; }
    let payload; try { payload = JSON.parse(txt); } catch(e){ out('✗ invalid JSON','error'); return; }
    out('# Submitting bundle...');
    const r = await fetch('/api/v1/beta/policy/bundles', {
      method:'POST', headers:{'Content-Type':'application/json', 'X-Admin-Token': (document.getElementById('pol-admin-token')?.value||'') },
      body: JSON.stringify(payload)
    });
    const data = await r.json();
    if(!data.success){ out(`✗ submit failed: ${data.message||'error'}`,'error'); return; }
    out(`✓ bundle appended head=${data.head_hash} active_version=${data.policy_version||'?'} revisions_total=${data.revisions_total||'?'} active_version_gauge=${data.active_version||'?'} (rollback cleared)`,'success');
    refreshPolicyGovernance();
  } catch(e){ out(`✗ submit error: ${e.message}`,'error'); }
}

// Governance helpers
async function loadGovernanceMetrics(){
  try {
    const r = await fetch('/api/v1/beta/policy/metrics');
    if(!r.ok) throw new Error(r.status);
    const data = await r.json();
    const elRev = document.getElementById('pg-revisions-total');
    const elAct = document.getElementById('pg-active-version');
    if(elRev) elRev.textContent = data.revisions_total ?? '0';
    if(elAct) elAct.textContent = data.active_version ?? '-';
  } catch(e){ /* silent */ }
}

async function loadChainHead(){
  try {
    const r = await fetch('/api/v1/beta/policy/provenance');
    if(!r.ok) throw new Error(r.status);
    const data = await r.json();
    const elHead = document.getElementById('pg-head-hash');
    const elVerified = document.getElementById('pg-chain-verified');
    if(elHead) elHead.textContent = data.head_hash || '—';
    if(elVerified){
      // Use unified badge coloring
      updateVerificationBadge(!!data.verified);
    }
  } catch(e){ /* silent */ }
}

async function performRollback(){
  const verInput = document.getElementById('pg-rollback-version');
  const token = document.getElementById('pol-admin-token')?.value||'';
  const v = verInput?.value.trim();
  if(!v){ out('✗ rollback: version required','error'); return; }
  out(`# rollback -> version ${v}`);
  try {
    const r = await fetch(`/api/v1/beta/policy/rollback?version=${encodeURIComponent(v)}`, { method:'POST', headers:{'X-Admin-Token': token} });
    const data = await r.json();
    if(!data.success){ out(`✗ rollback failed: ${data.message||'error'}`,'error'); return; }
    out(`✓ rollback active_version=${data.active_version} head_hash=${data.head_hash}`,'success');
    refreshPolicyGovernance();
  } catch(e){ out(`✗ rollback error: ${e.message}`,'error'); }
}

  // --- Diff Panel Integration (beta) ---
  export async function loadPolicyDiff(fromVersion, toVersion) {
    try {
      const params = [];
      if (fromVersion) params.push(`from=${fromVersion}`);
      if (toVersion) params.push(`to=${toVersion}`);
      const qs = params.length ? `?${params.join('&')}` : '';
      const res = await fetch(`/api/v1/beta/policy/diff${qs}`);
      const data = await res.json();
      if (!data.success) {
        console.warn('diff error', data.message);
        return null;
      }
      return data.diff;
    } catch (e) {
      console.error('diff fetch failed', e);
      return null;
    }
  }

  export function renderDiff(diff) {
    if (!diff) return;
    const panel = document.getElementById('pg-diff-panel');
    if (!panel) return;
    const fmt = (arr) => arr.map(p => p.id || p.ID).join(', ') || '∅';
    panel.innerHTML = `
      <div class="pg-diff-summary">
        <div><strong>From</strong> v${diff.from_version} (${diff.from_hash.slice(0,8)}) → <strong>To</strong> v${diff.to_version} (${diff.to_hash.slice(0,8)})</div>
        <div>Added: ${fmt(diff.added)}</div>
        <div>Removed: ${fmt(diff.removed)}</div>
        <div>Changed: ${diff.changed.map(c => c.id).join(', ') || '∅'}</div>
      </div>`;
  }

  // updateVerificationBadge colors badge based on verification status boolean.
  export function updateVerificationBadge(verified) {
    const el = document.getElementById('pg-chain-verified');
    if (!el) return;
    el.textContent = verified ? 'verified' : 'verification-failed';
    el.classList.remove('pg-badge-ok','pg-badge-fail');
    el.classList.add(verified ? 'pg-badge-ok' : 'pg-badge-fail');
  }
export function refreshPolicyGovernance(){
  loadGovernanceMetrics();
  loadChainHead();
}

function initGovernanceUI(){
  document.querySelectorAll('[data-action="policy-rollback"]').forEach(b=> b.addEventListener('click', performRollback));
  refreshPolicyGovernance();
  setInterval(refreshPolicyGovernance, 15000);
}
        // Inline current policy version badge near decision for easier UX
        const ver = escapeHtml(String(data.policy_version||'-'));
        out(`bundle_hash=${escapeHtml(data.bundle_hash||'')} chain_head=${escapeHtml(data.chain_head||'')} <span class='pg-version-badge' aria-label='policy version'>v${ver}</span>`,'info');
export function policyInit(){
  document.querySelectorAll('[data-action="policy-provenance"]').forEach(b=> b.addEventListener('click', ()=>loadProvenance()));
  document.querySelectorAll('[data-action="policy-chain-page"]').forEach(b=> b.addEventListener('click', ()=>{
    const off = parseInt(document.getElementById('pol-chain-offset')?.value||'0',10);
    const lim = parseInt(document.getElementById('pol-chain-limit')?.value||'10',10);
    loadChainPage(off, lim);
  }));
  document.querySelectorAll('[data-action="policy-consistency"]').forEach(b=> b.addEventListener('click', checkConsistency));
  document.querySelectorAll('[data-action="policy-evaluate"]').forEach(b=> b.addEventListener('click', evaluatePolicy));
  document.querySelectorAll('[data-action="policy-submit-bundle"]').forEach(b=> b.addEventListener('click', submitBundle));
  initGovernanceUI();
  // Initial timeline load + interval
  loadTimeline();
  setInterval(loadTimeline, 15000);
}

window.GAuth = window.GAuth || {}; Object.assign(window.GAuth, { loadProvenance, loadChainPage, checkConsistency, evaluatePolicy, submitBundle, refreshPolicyGovernance });

// Timeline fetch & render
async function loadTimeline(){
  try {
    const r = await fetch('/api/v1/beta/policy/timeline');
    if(!r.ok) throw new Error(r.status);
    const data = await r.json();
    const el = document.getElementById('pg-timeline');
    if(!el) return;
    if(!data.success){ el.innerHTML = '<div class="text-red-600">(timeline error)</div>'; return; }
    if(!Array.isArray(data.timeline) || data.timeline.length === 0){ el.innerHTML = '<div class="text-gray-400">(empty chain)</div>'; return; }
    el.innerHTML = data.timeline.map(item => {
      const active = item.active;
      const badge = active ? '<span class="inline-block px-1 rounded bg-green-100 text-green-700" title="Active">A</span>' : '';
      const rb = (!active && data.rolled_back && item.version === data.active_version) ? '<span class="inline-block px-1 rounded bg-yellow-100 text-yellow-700" title="Rollback Active">R</span>' : '';
      return `<div>${badge}${rb} v${item.version} ${item.short_hash} <span class='text-gray-500'>${item.created}</span></div>`;
    }).join('');
  } catch(e){
    const el = document.getElementById('pg-timeline');
    if(el) el.innerHTML = '<div class="text-red-600">(timeline load failed)</div>';
  }
}
