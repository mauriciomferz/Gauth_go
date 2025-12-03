// governance.js - capability registry hashes, audit preview, lifecycle timeline
import { backoffFetchJSON } from './refresh.js';

export function initGovernancePanels(){
  initCapabilities();
  initAuditPreview();
  initLifecycleTimeline();
}

function initCapabilities(){
  const capPanel = document.getElementById('capability-governance-panel');
  if(!capPanel) return;
  const hashEl = document.getElementById('cap-registry-hash');
  const prevEl = document.getElementById('cap-registry-prev-hash');
  const changedEl = document.getElementById('cap-registry-changed-at');
  const listEl = document.getElementById('cap-registry-list');
  async function load(){
    try {
      const data = await backoffFetchJSON('/api/v1/beta/capabilities');
      if(!data.success){ return; }
      hashEl.textContent = data.capability_registry_hash || '—';
      prevEl.textContent = data.capability_registry_prev_hash || '—';
      changedEl.textContent = data.capability_registry_last_changed_at || '—';
      const caps = (data.capabilities||[]).map(c=>`<li class='cap-item'><span class='cap-id'>${c.id}</span><span class='cap-version text-gray-500 ml-2'>v${c.version}</span>${c.stable?'<span class="ml-2 text-green-600 text-xs">stable</span>':''}</li>`).join('');
      listEl.innerHTML = caps || '<li class="text-gray-400">No capabilities</li>';
    } catch(err){ console.warn('[capabilities] load error', err); }
  }
  load();
  setInterval(load, 10000);
}

function initAuditPreview(){
  const auditBody = document.getElementById('audit-preview-tbody');
  const auditStatus = document.getElementById('audit-preview-status');
  const btnJSON = document.getElementById('export-audit-json');
  const btnCSV = document.getElementById('export-audit-csv');
  let latestEntries = [];
  if(!auditBody) return;
  async function load(){
    try {
      auditStatus.textContent = 'Loading…';
      const data = await backoffFetchJSON('/api/v1/audit/logs?limit=20');
      if(!data.success){ auditStatus.textContent='Unavailable'; return; }
      const entries = data.entries||[];
      auditBody.innerHTML = entries.map(e=>`<tr><td>${e.id}</td><td>${e.action}</td><td>${e.resource||''}</td><td>${e.outcome}</td><td>${e.reason||''}</td><td>${e.at}</td></tr>`).join('') || `<tr><td colspan='6' class='text-center text-gray-400'>No audit entries</td></tr>`;
      auditStatus.textContent = `Updated ${new Date().toLocaleTimeString()}`;
      latestEntries = entries;
    } catch(err){ auditStatus.textContent='Error'; console.warn('[audit] load error', err); }
  }
  function exportAuditJSON(){
    const blob = new Blob([JSON.stringify({generated_at:new Date().toISOString(), entries:latestEntries}, null, 2)], {type:'application/json'});
    const a=document.createElement('a'); a.href=URL.createObjectURL(blob); a.download=`audit_logs_${Date.now()}.json`; a.click();
  }
  function exportAuditCSV(){
    const rows=['id,action,resource,outcome,reason,at'];
    latestEntries.forEach(e=>rows.push([e.id,e.action,e.resource||'',e.outcome,e.reason||'',e.at].map(s=>`"${s}"`).join(',')));
    const blob=new Blob([rows.join('\n')], {type:'text/csv'}); const a=document.createElement('a'); a.href=URL.createObjectURL(blob); a.download=`audit_logs_${Date.now()}.csv`; a.click();
  }
  if(btnJSON) btnJSON.addEventListener('click', exportAuditJSON);
  if(btnCSV) btnCSV.addEventListener('click', exportAuditCSV);
  load();
  setInterval(load, 7000);
}

function initLifecycleTimeline(){
  const lcBody = document.getElementById('lifecycle-timeline-tbody');
  const lcStatus = document.getElementById('lifecycle-timeline-status');
  const filterType = document.getElementById('lifecycle-filter-type');
  const filterID = document.getElementById('lifecycle-filter-id');
  const refreshBtn = document.getElementById('lifecycle-refresh-btn');
  const btnJSON = document.getElementById('export-lifecycle-json');
  const btnCSV = document.getElementById('export-lifecycle-csv');
  let latestEvents = [];
  if(!lcBody) return;
  async function load(){
    const params = new URLSearchParams();
    if(filterType && filterType.value) params.set('entity_type', filterType.value);
    if(filterID && filterID.value) params.set('entity_id', filterID.value);
    const url = '/api/v1/beta/lifecycle/timeline'+(params.toString()?('?' + params.toString()):'');
    lcStatus.textContent='Loading…';
    try {
      const data = await backoffFetchJSON(url);
      if(!data.success){ lcStatus.textContent='Unavailable'; return; }
      const events = data.events||[];
      lcBody.innerHTML = events.map(ev=>`<tr><td>${ev.entity_type}</td><td>${ev.entity_id}</td><td>${ev.old_status}</td><td>${ev.new_status}</td><td>${ev.outcome}</td><td>${ev.reason}</td><td>${ev.latency_ns}</td><td>${ev.at}</td></tr>`).join('') || `<tr><td colspan='8' class='text-center text-gray-400'>No events</td></tr>`;
      lcStatus.textContent = `Updated ${new Date().toLocaleTimeString()}`;
      latestEvents = events;
    } catch(err){ lcStatus.textContent='Error'; console.warn('[lifecycle] load error', err); }
  }
  if(refreshBtn){ refreshBtn.addEventListener('click', load); }
  function exportLifecycleJSON(){
    const blob = new Blob([JSON.stringify({generated_at:new Date().toISOString(), events:latestEvents}, null, 2)], {type:'application/json'});
    const a=document.createElement('a'); a.href=URL.createObjectURL(blob); a.download=`lifecycle_timeline_${Date.now()}.json`; a.click();
  }
  function exportLifecycleCSV(){
    const rows=['entity_type,entity_id,old_status,new_status,outcome,reason,latency_ns,at'];
    latestEvents.forEach(ev=>rows.push([ev.entity_type,ev.entity_id,ev.old_status,ev.new_status,ev.outcome,ev.reason,ev.latency_ns,ev.at].map(s=>`"${s}"`).join(',')));
    const blob=new Blob([rows.join('\n')], {type:'text/csv'}); const a=document.createElement('a'); a.href=URL.createObjectURL(blob); a.download=`lifecycle_timeline_${Date.now()}.csv`; a.click();
  }
  if(btnJSON) btnJSON.addEventListener('click', exportLifecycleJSON);
  if(btnCSV) btnCSV.addEventListener('click', exportLifecycleCSV);
  load();
  setInterval(load, 6000);
}
