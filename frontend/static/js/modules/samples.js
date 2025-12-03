// NOTE: Removed dynamic self-import of main.js to prevent early initialization
// before DOMContentLoaded which caused some buttons (e.g. token metrics) to miss
// their event bindings. The bundler entrypoint (main.js) now handles init.

import { escapeHtml } from "./console.js";
import { attachLogStream, stopManualJobStream, restoreCachedJob, cancelSampleStream } from "./joblogs.js";
import { showToast } from "./toast.js";

// Job polling UI logic
function renderJobTable(jobs) {
	const tbody = document.getElementById('job-table-body');
	if (!tbody) return;
	if (!jobs.length) {
		tbody.innerHTML = '<tr><td colspan="6" class="text-center text-gray-400 py-4">No jobs found.</td></tr>';
		return;
	}
	tbody.innerHTML = jobs.map(job => {
		const stateColor = job.state==='done' ? 'text-green-600' : job.state==='failed' ? 'text-red-600' : job.state==='timeout' ? 'text-orange-600' : job.state==='running' ? 'text-blue-600' : 'text-gray-600';
		return `<tr>
			<td class="px-3 py-2 font-mono text-xs">${escapeHtml(job.id)}</td>
			<td class="px-3 py-2">${escapeHtml(job.example_id||'')}</td>
			<td class="px-3 py-2 ${stateColor}">${escapeHtml(job.state||'')}</td>
			<td class="px-3 py-2">${job.started_at ? escapeHtml(job.started_at) : '-'}</td>
			<td class="px-3 py-2">${job.finished_at && job.started_at ? ((new Date(job.finished_at)-new Date(job.started_at))/1000).toFixed(1)+'s' : '-'}</td>
			<td class="px-3 py-2"><button class="bg-blue-100 hover:bg-blue-200 text-blue-800 text-xs px-2 py-1 rounded" data-rerun-sample="${escapeHtml(job.example_id)}">Re-run</button></td>
		</tr>`;
	}).join('');
}

function pollJobs() {
	fetch('/api/v1/beta/examples/run/jobs')
		.then(r => r.json())
		.then(j => {
			if (!j.success || !Array.isArray(j.jobs)) throw new Error(j.message||'invalid jobs');
			renderJobTable(j.jobs);
			document.getElementById('job-panel-msg').textContent = '';
		})
		.catch(err => {
			document.getElementById('job-panel-msg').textContent = 'Failed to load jobs: '+err.message;
		});
}

function startJobPolling() {
	pollJobs();
	window._jobPollInterval = setInterval(pollJobs, 2000);
}

function stopJobPolling() {
	if (window._jobPollInterval) clearInterval(window._jobPollInterval);
}

let currentSampleStreamJob = null;
let catalogCache = [];
let filteredCatalog = [];

function renderSamplesList(examples) {
	const samplesList = document.getElementById("samples-list");
	if (!samplesList) return;
	if (!examples.length) {
		samplesList.innerHTML = "<div class=\"text-gray-400 p-4\">No examples found.</div>";
		return;
	}
	samplesList.innerHTML = examples.map(ex => {
    const badges = [
      ex.isAdvanced ? '<span class="ml-2 text-xs bg-purple-600 text-white px-1.5 py-0.5 rounded">ADV</span>' : '',
      ex.hasNegative ? '<span class="ml-1 text-xs bg-red-600 text-white px-1.5 py-0.5 rounded">NEG</span>' : ''
    ].join('');
    return `<button class="sample-btn bg-blue-50 hover:bg-blue-100 border border-blue-200 text-blue-800 text-sm font-medium px-3 py-2 m-1 rounded shadow-sm flex items-center"
      data-action="run-sample" data-sample-id="${ex.id}">
      <i class="fas fa-flask mr-1 text-blue-500"></i><span>${escapeHtml(ex.title || ex.id)}</span>${badges}
    </button>`;
  }).join("");
}

function runSample(id) {
	const samplesOutput = document.getElementById("samples-output");
	if (!samplesOutput) return Promise.reject(new Error('missing samples-output'));
	stopManualJobStream();
	if (currentSampleStreamJob) currentSampleStreamJob = null;
	samplesOutput.innerHTML = `<span class='text-gray-400'>[Queued ${id}]</span><br><span class='text-yellow-400'>Submitting job...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
	return fetch("/api/v1/beta/examples/run", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id }) })
		.then(res => res.json())
		.then(data => {
			if (!data.success) {
				samplesOutput.innerHTML = `<span class='text-red-400'>Failed to queue job: ${escapeHtml(data.message || "unknown error")}</span>`;
				throw new Error(data.message || 'queue failed');
			}
			const jobId = data.job_id; currentSampleStreamJob = jobId;
			samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} queued for ${id}]</span><br><span class='text-yellow-400'>State: ${data.state}</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
			attachLogStream(jobId, id);
			return { jobId, id };
		})
		.catch(err => { throw err; });
}

// Wait for a specific job to reach terminal state
function waitForJobCompletion(jobId, timeoutMs = 30000, intervalMs = 600) {
	const start = Date.now();
	return new Promise((resolve, reject) => {
		const tick = () => {
			fetch('/api/v1/beta/examples/run/jobs')
				.then(r => r.json())
				.then(j => {
					if (!j.success || !Array.isArray(j.jobs)) throw new Error('jobs fetch failed');
					const job = j.jobs.find(x => x.id === jobId);
					if (job && ['done','failed','timeout','canceled'].includes(job.state)) {
						return resolve(job);
					}
					if (Date.now() - start > timeoutMs) return reject(new Error('job timeout'));
					setTimeout(tick, intervalMs);
				})
				.catch(err => {
					if (Date.now() - start > timeoutMs) return reject(err);
					setTimeout(tick, intervalMs);
				});
		};
		tick();
	});
}

let _allSamplesCancel = false;

function cancelAllSamples() {
	_allSamplesCancel = true;
	const btn = document.getElementById('cancel-all-samples-btn');
	if (btn) btn.classList.add('hidden');
	showToast('All-samples run canceled','warning');
}

function updateAllSamplesProgress(done, total) {
	const span = document.getElementById('all-samples-progress');
	if (span) span.textContent = `${done}/${total} completed`;
}

// Improved dynamic runner for entire catalog
function runAllSamples() {
  if (!catalogCache.length) return showToast('Catalog empty','warning');
  if (window.__runningAllSamples) return showToast('Already running','info');
  window.__runningAllSamples = true; _allSamplesCancel = false;
  const cancelBtn = document.getElementById('cancel-all-samples-btn');
  if (cancelBtn) cancelBtn.classList.remove('hidden');
  showToast(`Running all ${catalogCache.length} samples`, 'info', 2500);
  const total = catalogCache.length;
  let idx = 0;
  const runNext = () => {
    if (_allSamplesCancel) { window.__runningAllSamples = false; updateAllSamplesProgress(idx,total); return; }
    if (idx >= total) {
      showToast('All samples completed','success');
      window.__runningAllSamples = false;
      if (cancelBtn) cancelBtn.classList.add('hidden');
      updateAllSamplesProgress(total,total);
      return;
    }
    const ex = catalogCache[idx];
    runSample(ex.id)
      .then(({ jobId }) => waitForJobCompletion(jobId).catch(()=>({state:'unknown'})))
      .then(() => {
        idx++;
        updateAllSamplesProgress(idx,total);
        runNext();
      })
      .catch(err => {
        showToast(`Sample ${ex.id} error: ${err.message}`,'error',3000);
        idx++;
        updateAllSamplesProgress(idx,total);
        runNext();
      });
  };
  updateAllSamplesProgress(0,total);
  runNext();
}

function runAllBasics() {
  if (window.__runningBasics) return showToast('Basics run already active','info');
  const basics = catalogCache.filter(ex => (ex.Group||ex.group)==='basics');
  if (!basics.length) return showToast('No basics found','warning');
  window.__runningBasics = true; let idx = 0;
  showToast(`Running ${basics.length} basics`, 'info', 2000);
  const step = () => {
    if (idx >= basics.length) { showToast('All basics completed','success'); window.__runningBasics = false; return; }
    const ex = basics[idx];
    runSample(ex.id).catch(()=>{});
    idx++;
    setTimeout(step, 1400);
  };
  step();
}

function runAdvancedSuite() {
  if (window.__runningAdvanced) return showToast('Advanced run already active','info');
  const adv = catalogCache.filter(ex => (ex.Group||ex.group)==='advanced');
  if (!adv.length) return showToast('No advanced samples found','warning');
  window.__runningAdvanced = true; let idx = 0;
  showToast(`Running ${adv.length} advanced samples`, 'info', 2000);
  const step = () => {
    if (idx >= adv.length) { showToast('Advanced suite completed','success'); window.__runningAdvanced = false; return; }
    const ex = adv[idx];
    runSample(ex.id).catch(()=>{});
    idx++;
    setTimeout(step, 1700);
  };
  step();
}

function viewExample(exampleId) {
  // TODO: Implement example viewing logic (modal, panel, or redirect)
  alert(`View Example: ${exampleId}`);
}

function fetchCatalog() {
  const list = document.getElementById('samples-list');
  if (list) list.innerHTML = '<div class="p-4 text-sm text-blue-600 animate-pulse">Loading catalog...</div>';
  return fetch('/api/v1/beta/examples/catalog')
    .then(r => r.json())
    .then(j => {
      if (!j.success || !Array.isArray(j.examples)) throw new Error(j.message || 'invalid catalog');
      catalogCache = j.examples;
			filteredCatalog = [...catalogCache];
			renderSamplesList(filteredCatalog);
      bindSampleButtons();
			wireFilters();
			showToast(`Loaded ${catalogCache.length} examples`, 'success', 2500);
    })
    .catch(err => {
      if (list) list.innerHTML = `<div class='p-4 text-red-600 text-sm'>Failed to load catalog: ${escapeHtml(err.message)}</div>`;
			showToast('Failed to load samples catalog','error');
    });
}

function bindSampleButtons(){
	document.querySelectorAll("[data-action=\"run-sample\"]").forEach(el => el.addEventListener("click", e => runSample(el.getAttribute("data-sample-id"))));
	document.querySelectorAll("[data-action=\"run-all-basics\"]").forEach(el => el.addEventListener("click", runAllBasics));
	document.querySelectorAll("[data-action=\"run-advanced-suite\"]").forEach(el => el.addEventListener("click", runAdvancedSuite));
	document.querySelectorAll('[data-action="cancel-all-samples"]').forEach(el => el.addEventListener('click', cancelAllSamples));
	document.querySelectorAll('[data-action="run-all-samples"]').forEach(el => el.addEventListener('click', runAllSamples));
	document.querySelectorAll('[data-action="view-example"]').forEach(el => el.addEventListener('click', e => {
		const ex = el.getAttribute('data-example');
		if (ex) viewExample(ex);
	}));
}

function observeTabSwitchCancellation(){
  const tabs = document.querySelectorAll('.tab-button[data-tab]');
	tabs.forEach(t => t.addEventListener('click', () => {
		const target = t.getAttribute('data-tab');
		// When leaving samples-demo tab: stop streams
		if (!t.classList.contains('active') && target === 'samples-demo') {
			stopManualJobStream();
			cancelSampleStream();
		}
		// Entering samples-demo tab: attempt to restore last job console
		if (target === 'samples-demo' && currentSampleStreamJob) {
			restoreCachedJob(currentSampleStreamJob);
		}
	}));
}

function applyFilters(){
	const q = (document.getElementById('samples-search')?.value || '').trim().toLowerCase();
	const adv = document.getElementById('filter-advanced')?.checked;
	const neg = document.getElementById('filter-negative')?.checked;
	const basics = document.getElementById('filter-basics')?.checked;
	// Persist filter state
	try {
		localStorage.setItem('samples-search', q);
		localStorage.setItem('samples-adv', adv?'1':'0');
		localStorage.setItem('samples-neg', neg?'1':'0');
		localStorage.setItem('samples-basics', basics?'1':'0');
	} catch {}
	filteredCatalog = catalogCache.filter(ex => {
		const title = (ex.title||'').toLowerCase();
		const id = (ex.id||'').toLowerCase();
		if (q && !(title.includes(q)||id.includes(q))) return false;
		const group = ex.Group || ex.group || '';
		if (!adv && group==='advanced') return false;
		if (!neg && (group==='negative'||group==='neg'||group==='failure')) return false;
		if (!basics && group==='basics') return false;
		return true;
	});
	renderSamplesList(filteredCatalog);
	bindSampleButtons();
}

function wireFilters(){
	// Restore filter state from localStorage
	try {
		const q = localStorage.getItem('samples-search')||'';
		const adv = localStorage.getItem('samples-adv')==='1';
		const neg = localStorage.getItem('samples-neg')==='1';
		const basics = localStorage.getItem('samples-basics')!=='0';
		const qEl = document.getElementById('samples-search');
		const advEl = document.getElementById('filter-advanced');
		const negEl = document.getElementById('filter-negative');
		const basicsEl = document.getElementById('filter-basics');
		if (qEl) qEl.value = q;
		if (advEl) advEl.checked = adv;
		if (negEl) negEl.checked = neg;
		if (basicsEl) basicsEl.checked = basics;
	} catch {}
	['samples-search','filter-advanced','filter-negative','filter-basics'].forEach(id=>{
		const el=document.getElementById(id); if(!el) return; el.addEventListener('input', applyFilters); el.addEventListener('change', applyFilters);
	});
	// Initial filter apply after restore
	setTimeout(applyFilters, 10);
}

export function samplesInit(){
  fetchCatalog();
  observeTabSwitchCancellation();
  
  // Event delegation for re-run buttons in job table
  document.addEventListener('click', function(e) {
    const rerunBtn = e.target.closest('[data-rerun-sample]');
    if (rerunBtn) {
      const exampleId = rerunBtn.getAttribute('data-rerun-sample');
      if (exampleId && window.GAuth.runSample) {
        window.GAuth.runSample(exampleId);
      }
    }
  });
  
  // Start polling immediately if samples tab is default active
  const active = document.querySelector('.tab-button.active');
  if (active && active.getAttribute('data-tab') === 'samples-demo') startJobPolling();
  const tabs = document.querySelectorAll('.tab-button[data-tab]');
  tabs.forEach(t => t.addEventListener('click', () => {
    const target = t.getAttribute('data-tab');
    if (target === 'samples-demo') startJobPolling();
    else stopJobPolling();
  }));
}

// Legacy global bridge (for compatibility)
window.GAuth = window.GAuth || {}; Object.assign(window.GAuth, { runSample, runAllBasics, runAdvancedSuite, runAllSamples, cancelAllSamples, viewExample, samplesInit });
// Removed automatic cancellation/autorun to give user full control
