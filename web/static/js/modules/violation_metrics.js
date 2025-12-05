// violation_metrics.js - exposes token validation violation counters & anomaly rates
import { backoffFetchJSON } from './refresh.js';

export function initViolationMetricsPanel(){
  const panel = document.getElementById('violation-metrics-panel');
  if(!panel) return;
  const statusEl = document.getElementById('violation-metrics-status');
  const totalEl = document.getElementById('violation-total');
  const rate60El = document.getElementById('violation-rate-60');
  const rate300El = document.getElementById('violation-rate-300');
  const surgeEl = document.getElementById('violation-surge-flag');
  const tbody = document.getElementById('violation-counters-tbody');
  let attempts=0;
  async function load(){
    attempts++;
    statusEl.textContent='Loading…';
    try {
      const data = await backoffFetchJSON('/api/v1/beta/metrics/violations', attempts);
      if(!data.success){ statusEl.textContent='Unavailable'; return; }
      totalEl.textContent = data.total ?? 0;
      const counters = data.counters || {};
      const categories = data.categories || Object.keys(counters);
      tbody.innerHTML = categories.map(cat => {
        const v = counters[cat] ?? 0;
        return `<tr><td>${cat}</td><td class="text-right font-mono">${v}</td></tr>`;
      }).join('') || '<tr><td colspan="2" class="text-center text-gray-400">No data</td></tr>';
      if(data.anomaly){
        rate60El.textContent = (data.anomaly.rate_per_minute_60s ?? 0).toFixed(2);
        rate300El.textContent = (data.anomaly.rate_per_minute_300s ?? 0).toFixed(2);
        const surge = !!data.anomaly.surge_60s;
        surgeEl.textContent = surge ? 'Surge' : 'Normal';
        surgeEl.className = surge ? 'px-2 py-0.5 rounded bg-red-600 text-white text-xs' : 'px-2 py-0.5 rounded bg-green-600 text-white text-xs';
      }
      statusEl.textContent='Updated '+new Date().toLocaleTimeString();
    } catch(err){ statusEl.textContent='Error'; console.warn('[violation-metrics] fetch error', err); }
  }
  load();
  setInterval(load, 10000);
}
