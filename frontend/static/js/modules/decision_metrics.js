// decision_metrics.js - renders decision metrics endpoint data
import { backoffFetchJSON } from './refresh.js';

export function initDecisionMetricsPanel(){
  const panel = document.getElementById('decision-metrics-panel');
  if(!panel) return;
  const statusEl = document.getElementById('decision-metrics-status');
  const countsBody = document.getElementById('decision-counts-tbody');
  const reasonsBody = document.getElementById('decision-reasons-tbody');
  let attempts = 0;
  async function load(){
    attempts++;
    statusEl.textContent = 'Loading…';
    try {
      const data = await backoffFetchJSON('/api/v1/beta/metrics/decisions', attempts);
      if(!data.success){ statusEl.textContent='Unavailable'; return; }
      const dec = data.decisions || {}; const counts = dec.counts||[]; const reasons = dec.reasons||[];
      countsBody.innerHTML = counts.map(e=>`<tr><td>${e.action}</td><td class='truncate max-w-[160px]'>${e.resource}</td><td>${e.outcome}</td><td>${e.count}</td></tr>`).join('') || `<tr><td colspan='4' class='text-center text-gray-400'>No decisions yet</td></tr>`;
      reasonsBody.innerHTML = reasons.map(e=>`<tr><td>${e.action}</td><td class='truncate max-w-[160px]'>${e.resource}</td><td>${e.outcome}</td><td><span class='reason-badge reason-${e.reason}'>${e.reason}</span></td><td>${e.count}</td></tr>`).join('') || `<tr><td colspan='5' class='text-center text-gray-400'>No reasons recorded</td></tr>`;
      statusEl.textContent = `Updated ${new Date().toLocaleTimeString()}`;
      latestData = {counts, reasons};
    } catch(err){ statusEl.textContent = 'Error'; console.warn('[decision-metrics] fetch error', err); }
  }
  // hold latest for export
  let latestData = {counts:[], reasons:[]};
  function exportJSON(){
    const blob = new Blob([JSON.stringify({generated_at:new Date().toISOString(), ...latestData}, null, 2)], {type:'application/json'});
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = `decision_metrics_${Date.now()}.json`;
    a.click();
  }
  function exportCSV(){
    const rows = ['type,action,resource,outcome,reason,count'];
    latestData.counts.forEach(c=>rows.push(['count', c.action, c.resource, c.outcome, '', c.count].map(s=>`"${s}"`).join(',')));
    latestData.reasons.forEach(r=>rows.push(['reason', r.action, r.resource, r.outcome, r.reason, r.count].map(s=>`"${s}"`).join(',')));
    const blob = new Blob([rows.join('\n')], {type:'text/csv'});
    const a = document.createElement('a'); a.href=URL.createObjectURL(blob); a.download=`decision_metrics_${Date.now()}.csv`; a.click();
  }
  const btnJSON = document.getElementById('export-decisions-json');
  const btnCSV = document.getElementById('export-decisions-csv');
  if(btnJSON) btnJSON.addEventListener('click', exportJSON);
  if(btnCSV) btnCSV.addEventListener('click', exportCSV);
  load();
  setInterval(load, 5000);
}
