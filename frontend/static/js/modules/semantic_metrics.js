// semantic_metrics.js - shows AAP-001 semantic counters & computed 60s rates
import { backoffFetchJSON } from './refresh.js';

export function initSemanticMetricsPanel(){
  const panel = document.getElementById('semantic-metrics-panel');
  if(!panel) return;
  const statusEl = document.getElementById('semantic-metrics-status');
  const tbody = document.getElementById('semantic-counters-tbody');
  const rateBadge = document.getElementById('semantic-rate-badge');
  let attempts=0;
  async function load(){
    attempts++;
    statusEl.textContent='Loading…';
    try {
      const data = await backoffFetchJSON('/api/v1/beta/metrics/poa/semantics', attempts);
      if(!data.success){ statusEl.textContent='Unavailable'; return; }
      const counters = data.counters || {};
      tbody.innerHTML = Object.keys(counters).sort().map(k=>`<tr><td>${k}</td><td class='text-right font-mono'>${counters[k]}</td></tr>`).join('') || '<tr><td colspan="2" class="text-center text-gray-400 py-2">Empty</td></tr>';
      if(data.rates){
        const r = data.rates;
        const sampleKey = r.scope_violation !== undefined ? 'scope_violation' : Object.keys(r)[0];
        const val = sampleKey ? r[sampleKey] : 0;
        rateBadge.textContent = sampleKey ? `${sampleKey} 60s rate: ${val.toFixed(2)}/min` : 'No rates';
      }
      statusEl.textContent='Updated '+new Date().toLocaleTimeString();
    } catch(err){ statusEl.textContent='Error'; console.warn('[semantic-metrics] fetch error', err); }
  }
  load();
  setInterval(load, 12000);
}
