// app.js - lightweight dashboard logic
import { initRotationV2Panel } from './rotation_v2.js';
async function fetchJSON(url){
  const res = await fetch(url, { headers:{'Accept':'application/json'} });
  if(!res.ok) throw new Error('HTTP '+res.status);
  return res.json();
}
function safeText(el, v){ if(el) el.textContent = v; }

async function loadRevocation(){
  const statusEl = document.getElementById('revocation-status');
  try {
    const data = await fetchJSON('/api/v1/token/revocation/head');
    safeText(document.getElementById('revocation-head'), JSON.stringify(data, null, 2));
    safeText(statusEl, 'Updated '+new Date().toLocaleTimeString());
  } catch(e){ safeText(statusEl, 'Error'); }
}

async function loadRotation(){
  const statusEl = document.getElementById('rotation-status');
  try {
    const data = await fetchJSON('/api/v1/rotation/summary');
    safeText(document.getElementById('rotation-head-hash'), data.head_hash || data.threshold || '—');
    safeText(document.getElementById('rotation-aggregate-hash'), data.aggregate_hash || '—');
    safeText(document.getElementById('rotation-chain-length'), data.chain_length || '—');
    safeText(document.getElementById('rotation-signature-kid'), data.active_key_id || '—');
    safeText(statusEl, 'Updated '+new Date().toLocaleTimeString());
  } catch(e){ safeText(statusEl, 'Error'); }
}

async function loadCapability(){
  const statusEl = document.getElementById('capability-status');
  try {
    const data = await fetchJSON('/api/v1/capability/anchor/latest');
    safeText(document.getElementById('cap-registry-hash'), data.registry_hash || '—');
    safeText(document.getElementById('cap-prev-hash'), data.previous_hash || '—');
    safeText(document.getElementById('cap-anchored-at'), data.anchored_at || '—');
    safeText(document.getElementById('cap-provider'), data.provider || '—');
    safeText(statusEl, 'Updated '+new Date().toLocaleTimeString());
  } catch(e){ safeText(statusEl, 'Error'); }
}

async function loadErrors(){
  try {
    const data = await fetchJSON('/api/v1/errors/catalog');
    const tbody = document.querySelector('#error-table tbody');
    tbody.innerHTML = '';
    data.entries.forEach(e => {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td>${e.code}</td><td>${e.http_status}</td><td>${e.category}</td><td>${e.severity}</td><td>${e.retryable}</td><td>${e.description}</td>`;
      tbody.appendChild(tr);
    });
  } catch(e){ /* silent */ }
}

async function loadAlgorithms(){
  try {
    const data = await fetchJSON('/api/v1/beta/discovery');
    const list = document.getElementById('alg-list');
    list.innerHTML = '';
    (data.algorithms||[]).forEach(a => { const li = document.createElement('li'); li.textContent = a; list.appendChild(li); });
  } catch(e){ /* silent */ }
}

async function loadOpenAPISpec(){
  try {
    const res = await fetch('/api/openapi/gauth.yaml');
    if(!res.ok) throw new Error('spec missing');
    const text = await res.text();
    const el = document.getElementById('openapi-container');
    el.textContent = text;
  } catch(e){ const el = document.getElementById('openapi-container'); el.textContent = 'Failed to load spec'; }
}

function schedule(){
  loadRevocation(); loadRotation(); loadCapability(); loadErrors(); loadAlgorithms(); loadOpenAPISpec();
  setInterval(loadRevocation, 30000);
  setInterval(loadRotation, 45000);
  setInterval(loadCapability, 60000);
  setInterval(loadErrors, 120000);
  setInterval(loadAlgorithms, 120000);
  initRotationV2Panel();
}

window.addEventListener('DOMContentLoaded', schedule);
