// CSP-compliant event handler bindings for demo navigation and example viewing
document.addEventListener('DOMContentLoaded', function() {
    // Scroll to Demo button
    document.querySelectorAll('[data-action="scroll-to-demo"]').forEach(btn => {
        btn.addEventListener('click', scrollToDemo);
    });
    // View Example buttons
    document.querySelectorAll('[data-action="view-example"]').forEach(btn => {
        btn.addEventListener('click', function(e) {
            const ex = btn.getAttribute('data-example');
            if (ex) viewExample(ex);
        });
    });
});
// GAuth Beta Demo JavaScript

// Global state
let currentToken = null;
let demoState = {
    tokenCreated: false,
    subscriptionsActive: false,
    auditEntries: []
};

// Utility functions
function addConsoleOutput(containerId, message, type = 'info') {
    const container = document.getElementById(containerId);
    const timestamp = new Date().toISOString().split('T')[1].split('.')[0];
    const typeClass = type === 'success' ? 'console-success' :
                     type === 'error' ? 'console-error' :
                     type === 'warning' ? 'console-warning' :
                     'console-info';
    
    const line = document.createElement('div');
    line.className = `console-line ${typeClass}`;
    line.innerHTML = `<span class="text-gray-400">[${timestamp}]</span> ${message}`;
    
    container.appendChild(line);
    container.scrollTop = container.scrollHeight;
}
// ...existing code...

function clearConsoleOutput(containerId) {
    const container = document.getElementById(containerId);
    container.innerHTML = `
        <span class="text-gray-500"># ${containerId.replace('-output', '').toUpperCase()} Console</span><br>
        <span class="text-blue-400">gauth-${containerId.replace('-output', '')}></span> <span class="blinking-cursor">_</span>
    `;
}

function generateRandomId() {
    return Math.random().toString(36).substr(2, 9);
}

function simulateApiDelay() {
    return new Promise(resolve => setTimeout(resolve, Math.random() * 1000 + 500));
}

// Tab system
function showTab(e, tabId) {
    // Hide all tab contents
    const tabContents = document.querySelectorAll('.tab-content');
    tabContents.forEach(content => {
        content.style.display = 'none';
        content.classList.remove('active');
    });
    
    // Remove active class from all tab buttons
    const tabButtons = document.querySelectorAll('.tab-button');
    tabButtons.forEach(button => button.classList.remove('active'));
    
    // Show selected tab content
    const selectedTab = document.getElementById(tabId);
    if (selectedTab) {
        selectedTab.style.display = 'block';
        selectedTab.classList.add('active');
    }
    
    // Add active class to clicked button
    if (e && e.target) {
        e.target.classList.add('active');
    }
}

// DOM bindings without inline handlers (CSP friendly)
document.addEventListener('DOMContentLoaded', () => {
    // initialization block
    try {
        console.log('[GAuth Demo] DOMContentLoaded fired - initializing bindings');
        const bindReport = { tabs: 0, actions: 0 };
    // Tab buttons
        document.querySelectorAll('.tab-button[data-tab]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tabId = btn.getAttribute('data-tab');
                showTab(e, tabId);
            });
            bindReport.tabs++;
        });

        // Samples tab logic
        initSamplesTab();
// --- Samples Tab Logic ---
function initSamplesTab() {
    const runAdvancedSuiteBtn = document.getElementById('run-advanced-suite');
    const samplesList = document.getElementById('samples-list');
    const samplesSearch = document.getElementById('samples-search');
    const samplesOutput = document.getElementById('samples-output');
    let allExamples = [];
    let filteredExamples = [];
    const filterAdvanced = document.getElementById('filter-advanced');
    const filterNegative = document.getElementById('filter-negative');
    const filterBasics = document.getElementById('filter-basics');
    const runAllBasicsBtn = document.getElementById('run-all-basics');
    const poaMetricsPanel = document.getElementById('poa-metrics');
    const metricsElems = {
        req: document.getElementById('m-poa-req'),
        ok: document.getElementById('m-poa-ok'),
        jur: document.getElementById('m-poa-jur'),
        scope: document.getElementById('m-poa-scope'),
        missing: document.getElementById('m-poa-missing'),
        updated: document.getElementById('poa-metrics-updated')
    };

    if (!samplesList || !samplesSearch || !samplesOutput) return;

    // Fetch catalog
    fetch('/api/v1/beta/examples/catalog')
        .then(res => res.json())
        .then(data => {
            allExamples = (data.examples || []).map(e => enrichExample(e));
            filteredExamples = allExamples;
            renderSamplesList(filteredExamples);
            startMetricsPolling();
        })
        .catch(() => {
            samplesList.innerHTML = '<div class="text-red-500 p-4">Failed to load examples.</div>';
        });

    // Search filter
    function applyFilters() {
        const q = samplesSearch.value.toLowerCase();
        filteredExamples = allExamples.filter(ex => {
            if (q && !(ex.id.toLowerCase().includes(q) || (ex.title||'').toLowerCase().includes(q) || (ex.description||'').toLowerCase().includes(q))) return false;
            const isAdv = ex.isAdvanced;
            const hasNeg = ex.hasNegative;
            const isBasic = !isAdv && !hasNeg;
            if (!filterAdvanced.checked && isAdv) return false;
            if (!filterNegative.checked && hasNeg) return false;
            if (!filterBasics.checked && isBasic) return false;
            return true;
        });
        renderSamplesList(filteredExamples);
    }
    samplesSearch.addEventListener('input', applyFilters);
    [filterAdvanced, filterNegative, filterBasics].forEach(cb => cb && cb.addEventListener('change', applyFilters));

    function enrichExample(ex) {
        const title = (ex.title || '').toLowerCase();
        const desc = (ex.description || '').toLowerCase();
        ex.isAdvanced = /advanced/.test(title) || /advanced_poa/.test(ex.id);
        ex.hasNegative = /(negative|invalid|disallowed|missing)/.test(desc);
        return ex;
    }

    if (runAdvancedSuiteBtn) {
        runAdvancedSuiteBtn.addEventListener('click', () => runAdvancedSuite());
    }

    if (runAllBasicsBtn) {
        runAllBasicsBtn.addEventListener('click', () => runAllBasics());
    }

    function renderSamplesList(examples) {
        if (!examples.length) {
            samplesList.innerHTML = '<div class="text-gray-400 p-4">No examples found.</div>';
            return;
        }
        // Sort: featured categories first (gauth_protocol_basics), then alphabetical
        const featuredPrefix = 'gauth_protocol_basics';
        const sorted = [...examples].sort((a,b) => {
            const aFeat = a.id.startsWith(featuredPrefix) ? 0 : 1;
            const bFeat = b.id.startsWith(featuredPrefix) ? 0 : 1;
            if (aFeat !== bFeat) return aFeat - bFeat;
            return (a.title || a.id).localeCompare(b.title || b.id);
        });
        samplesList.innerHTML = sorted.map(ex => {
            const isAdvanced = /advanced/i.test(ex.title || '') || /advanced_poa/i.test(ex.id);
            const hasNegatives = /negative|invalid|disallowed|missing/i.test(ex.description || '');
            const badges = [
                isAdvanced ? '<span class="ml-2 inline-block bg-purple-100 text-purple-700 text-[10px] px-2 py-0.5 rounded">ADV</span>' : '',
                hasNegatives ? '<span class="ml-1 inline-block bg-red-100 text-red-700 text-[10px] px-2 py-0.5 rounded" title="Contains negative validation cases">NEG</span>' : ''
            ].join('');
            return `
            <div class="group border-b border-gray-100 px-4 py-2 hover:bg-gray-50 transition">
                <div class="flex items-start justify-between gap-4">
                    <div class="flex-1 min-w-0">
                        <div class="font-semibold text-gray-800 flex items-center">${escapeHtml(ex.title || ex.id)} ${badges}</div>
                        <div class="text-xs text-gray-500 truncate" title="${escapeHtml(ex.description || ex.id)}">${escapeHtml(ex.description || ex.id)}</div>
                        <div class="text-[10px] text-gray-400 mt-0.5">ID: ${escapeHtml(ex.id)}</div>
                    </div>
                    <div class="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition">
                        <button class="run-sample-btn bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-1 px-3 rounded shadow" data-sample-id="${ex.id}"><i class="fas fa-play mr-1"></i>Run</button>
                    </div>
                </div>
            </div>`;
        }).join('');
        samplesList.querySelectorAll('.run-sample-btn').forEach(btn => {
            btn.addEventListener('click', () => runSample(btn.getAttribute('data-sample-id')));
        });
    }

    function runSample(id) {
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Queued ${id}]</span><br><span class='text-yellow-400'>Submitting job...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
    fetch('/api/v1/beta/examples/run', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id })
        })
        .then(res => res.json())
        .then(data => {
            if (!data.success) {
                samplesOutput.innerHTML = `<span class='text-red-400'>Failed to queue job: ${escapeHtml(data.message || 'unknown error')}</span>`;
                return;
            }
            const jobId = data.job_id;
            samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} queued for ${id}]</span><br><span class='text-yellow-400'>State: ${data.state}</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            attachLogStream(jobId, id);
        })
        .catch(err => {
            samplesOutput.innerHTML = `<span class='text-red-400'>Error queuing job: ${escapeHtml(err.message)}`;
        });
    }

    // Attach SSE log stream for job (with reconnect + idle watchdog)
    function attachLogStream(jobId, exampleId) {
        let finished = false;
        let lastEventTs = Date.now();
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 5;
        let es;
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>Waiting for logs...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
        let logLines = [];
        let state = 'unknown';
        let output = '';
        let error = '';

        function startStream() {
            const primaryURL = `/api/v1/beta/examples/run/${jobId}/logs`;
            const fallbackURL = `/api/v1/educational/examples/run/${jobId}/logs`;
            function openES(url, triedFallback){
                es = new EventSource(url);
                es.onerror = () => {
                    if(!triedFallback){
                        es.close();
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[primary stream failed – trying deprecated path]</span>`;
                        openES(fallbackURL, true);
                    }
                };
            }
            openES(primaryURL, false);
            es.addEventListener('open', () => {
                lastEventTs = Date.now();
                reconnectAttempts = 0;
            });
            es.addEventListener('status', e => {
                lastEventTs = Date.now();
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                    updateSamplesOutput();
                } catch {}
            });
            es.addEventListener('log', e => {
                lastEventTs = Date.now();
                logLines.push(e.data);
                updateSamplesOutput();
            });
            es.addEventListener('done', e => {
                lastEventTs = Date.now();
                finished = true;
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                } catch {}
                updateSamplesOutput();
                es.close();
            });
            es.onerror = () => {
                if (!finished) {
                    es.close();
                    if (reconnectAttempts < maxReconnectAttempts) {
                        reconnectAttempts++;
                        const delay = Math.min(3000, 500 * reconnectAttempts);
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[reconnecting attempt ${reconnectAttempts} in ${delay}ms]</span>`;
                        setTimeout(startStream, delay);
                    } else {
                        samplesOutput.innerHTML += `<br><span class='text-red-400'>[stream error: max retries]</span>`;
                    }
                }
            };
        }
        startStream();

        const watchdog = setInterval(() => {
            if (finished) { clearInterval(watchdog); return; }
            const idle = Date.now() - lastEventTs;
            if (idle > 15000) {
                samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[idle – waiting for events]</span>`;
                lastEventTs = Date.now();
            }
        }, 5000);

        function updateSamplesOutput() {
            let html = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br>`;
            html += `<span class='text-yellow-400'>State: ${escapeHtml(state)}</span><br>`;
            if (logLines.length) {
                html += `<pre class='text-blue-300 whitespace-pre-wrap mt-2'>${escapeHtml(logLines.join('\n'))}</pre>`;
            }
            if (state === 'done') {
                html += `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'failed') {
                html += `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId})</span><br>`;
                if (error) html += `<span class='text-red-300'>Error: ${escapeHtml(error)}</span><br>`;
                if (output) html += `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'timeout') {
                html += `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            }
            html += `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            samplesOutput.innerHTML = html;
        }
    }
    }

    function pollJob(jobId, exampleId, startTs, chainCtx) {
    fetch(`/api/v1/beta/examples/run/${jobId}/status`)
            .then(r => r.json())
            .then(status => {
                if (!status.success) {
                    samplesOutput.innerHTML = `<span class='text-red-400'>Status error: ${escapeHtml(status.message || 'unknown')}</span>`;
                    return;
                }
                const job = status.job;
                const elapsed = ((Date.now() - startTs) / 1000).toFixed(2);
                if (job.state === 'queued' || job.state === 'running') {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>State: ${job.state} (elapsed ${elapsed}s)</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    setTimeout(() => pollJob(jobId, exampleId, startTs, chainCtx), 750);
                    return;
                }
                // Terminal states
                if (job.state === 'done') {
                    samplesOutput.innerHTML = `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId}) in ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '<span class="text-gray-400">(no output)</span>') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else if (job.state === 'failed') {
                    samplesOutput.innerHTML = `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.error ? `<span class='text-red-300'>Error: ${escapeHtml(job.error)}</span><br>` : '') +
                        (job.output ? `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed, error: job.error });
                        runNextInChain(chainCtx); // continue even on failure for educational completeness
                    }
                } else if (job.state === 'timeout') {
                    samplesOutput.innerHTML = `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>Job ${jobId} ended in state ${job.state}</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                }
            })
            .catch(err => {
                samplesOutput.innerHTML = `<span class='text-red-400'>Polling error: ${escapeHtml(err.message)}</span>`;
                if (chainCtx) {
                    chainCtx.results.push({ id: exampleId, state: 'error', error: err.message });
                    runNextInChain(chainCtx);
                }
            });
    }

    function runAllBasics() {
        const chain = ['gauth_protocol_basics:minimal_poa', 'gauth_protocol_basics:delegation', 'gauth_protocol_basics:token'];
        const ctx = { queue: chain, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running basic examples sequentially...</span>`;
        runNextInChain(ctx);
    }

    function runAdvancedSuite() {
        // Use all advanced and negative samples from the loaded catalog
        const advanced = allExamples.filter(ex => ex.isAdvanced);
        const negative = allExamples.filter(ex => ex.hasNegative);
        // Remove duplicates by id
        const ids = Array.from(new Set([...advanced, ...negative].map(ex => ex.id)));
        if (!ids.length) {
            samplesOutput.innerHTML = `<span class='text-yellow-400'>No advanced or negative samples found in catalog.</span>`;
            return;
        }
        const ctx = { queue: ids, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running advanced/negative examples sequentially...</span>`;
        runNextInChain(ctx);
    }
    function runNextInChain(ctx) {
        if (!ctx.queue.length) {
            // render summary
            const summary = ctx.results.map(r => `# ${r.id} -> ${r.state} (${r.elapsed||'-'}s)`).join('\n');
            samplesOutput.innerHTML = `<span class='text-green-400'>Composite run complete.</span><br><pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(summary)}</pre>`;
            // Show export buttons
            const exportDiv = document.getElementById('composite-export-buttons');
            if (exportDiv) {
                exportDiv.classList.remove('hidden');
                // Store summary in exportDiv for download handlers
                exportDiv.dataset.summary = JSON.stringify(ctx.results);
                exportDiv.dataset.summaryText = summary;
            }
            return;
        }
// Composite export download logic
document.addEventListener('DOMContentLoaded', function() {
    const exportDiv = document.getElementById('composite-export-buttons');
    if (!exportDiv) return;
    const btnJson = document.getElementById('download-composite-json');
    const btnCsv = document.getElementById('download-composite-csv');
    // Hide buttons initially
    exportDiv.classList.add('hidden');
    // Download JSON
    btnJson.addEventListener('click', function() {
        const summaryArr = exportDiv.dataset.summary;
        if (!summaryArr) return;
    fetch('/api/v1/beta/examples/composite/export/json', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: summaryArr
        })
        .then(r => {
            if (!r.ok) throw new Error('Export failed');
            return r.blob();
        })
        .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'composite_run_summary.json';
            document.body.appendChild(a);
            a.click();
            setTimeout(() => { document.body.removeChild(a); window.URL.revokeObjectURL(url); }, 100);
        })
        .catch(() => alert('Failed to download JSON export.'));
    });
    // Download CSV
    btnCsv.addEventListener('click', function() {
        const summaryArr = exportDiv.dataset.summary;
        if (!summaryArr) return;
    fetch('/api/v1/beta/examples/composite/export/csv', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: summaryArr
        })
        .then(r => {
            if (!r.ok) throw new Error('Export failed');
            return r.blob();
        })
        .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'composite_run_summary.csv';
            document.body.appendChild(a);
            a.click();
            setTimeout(() => { document.body.removeChild(a); window.URL.revokeObjectURL(url); }, 100);
        })
        .catch(() => alert('Failed to download CSV export.'));
    });
});
        const next = ctx.queue.shift();
    fetch('/api/v1/beta/examples/run', {
            method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: next })
        }).then(r => r.json()).then(data => {
            if (!data.success) {
                ctx.results.push({ id: next, state: 'queue_failed', error: data.message });
                runNextInChain(ctx);
                return;
            }
            pollJob(data.job_id, next, Date.now(), ctx);
        }).catch(err => {
            ctx.results.push({ id: next, state: 'error', error: err.message });
            runNextInChain(ctx);
        });
    }

    function startMetricsPolling() {
        if (!poaMetricsPanel) return;
        poaMetricsPanel.classList.remove('hidden');
        const fetchMetrics = () => {
            fetch('/api/v1/poa/metrics').then(r => r.json()).then(data => {
                if (!data.success) return;
                const m = data.metrics || {};
                metricsElems.req.textContent = m.total_requests;
                metricsElems.ok.textContent = m.total_success;
                metricsElems.jur.textContent = m.rejected_jurisdiction;
                metricsElems.scope.textContent = m.rejected_scope;
                metricsElems.missing.textContent = m.rejected_missing_fields;
                metricsElems.updated.textContent = new Date(data.timestamp).toLocaleTimeString();
                // Render POA duration sparkline
                const svg = document.getElementById('poa-duration-sparkline');
                const lastVal = document.getElementById('poa-duration-last');
                if (svg && m.recent_durations && Array.isArray(m.recent_durations) && m.recent_durations.length > 0) {
                    const values = m.recent_durations;
                    const w = 120, h = 24, pad = 2;
                    const min = Math.min(...values), max = Math.max(...values);
                    const range = max - min || 1;
                    const points = values.map((v, i) => {
                        const x = pad + i * ((w-2*pad)/(values.length-1||1));
                        const y = h - pad - ((v-min)/range)*(h-2*pad);
                        return `${x},${y}`;
                    }).join(' ');
                    svg.innerHTML = `<polyline fill="none" stroke="#2563eb" stroke-width="2" points="${points}" />`;
                    lastVal.textContent = values[values.length-1].toFixed(2);
                } else if (svg) {
                    svg.innerHTML = '';
                    if (lastVal) lastVal.textContent = '';
                }
            }).catch(()=>{});
        };
        fetchMetrics();
        setInterval(fetchMetrics, 4000);
    }

    function escapeHtml(str) {
        return String(str).replace(/[&<>"']/g, function(m) {
            return ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;','\'':'&#39;'})[m];
        });
    }
// END of main samples tab logic block

    // Default tab activation
    const activeBtn = document.querySelector('.tab-button.active[data-tab]') || document.querySelector('.tab-button[data-tab]');
    if (activeBtn) {
        showTab({ target: activeBtn }, activeBtn.getAttribute('data-tab'));
    }

    // Action buttons
    const actionMap = {
        'create-token': createToken,
        'validate-token': validateToken,
        'revoke-token': revokeToken,
        'check-authorization': checkAuthorization,
        'publish-event': publishEvent,
        'subscribe-events': subscribeEvents,
        'view-audit-log': viewAuditLog,
        'generate-report': generateReport
    };
        Object.entries(actionMap).forEach(([key, fn]) => {
            document.querySelectorAll(`[data-action="${key}"]`).forEach(el => {
                el.addEventListener('click', fn);
                bindReport.actions++;
            });
        });
        console.log(`[GAuth Demo] Initial binding complete:`, bindReport);
        if (bindReport.tabs === 0 || bindReport.actions === 0) {
            console.warn('[GAuth Demo] Warning: No bindings detected (tabs or actions). Scheduling retry...');
            setTimeout(rebindDemoHandlers, 500);
        }
    } catch (e) {
        console.error('[GAuth Demo] Initialization error:', e);
        setTimeout(rebindDemoHandlers, 750);
    }
});

// Fallback rebind function (in case of deferred HTML insertion or race conditions)
function rebindDemoHandlers(attempt = 1) {
    const MAX_ATTEMPTS = 5;
    const report = { tabs: 0, actions: 0, attempt };
    document.querySelectorAll('.tab-button[data-tab]').forEach(btn => {
        if (!btn.__gauthBound) {
            btn.addEventListener('click', (e) => {
                const tabId = btn.getAttribute('data-tab');
                showTab(e, tabId);
            });
            btn.__gauthBound = true;
            report.tabs++;
        }
    });
    const actionMap = {
        'create-token': createToken,
        'validate-token': validateToken,
        'revoke-token': revokeToken,
        'check-authorization': checkAuthorization,
        'publish-event': publishEvent,
        'subscribe-events': subscribeEvents,
        'view-audit-log': viewAuditLog,
        'generate-report': generateReport
    };
    Object.entries(actionMap).forEach(([key, fn]) => {
        document.querySelectorAll(`[data-action="${key}"]`).forEach(el => {
            if (!el.__gauthBound) {
                el.addEventListener('click', fn);
                el.__gauthBound = true;
                report.actions++;
            }
        });
    });
    console.log('[GAuth Demo] Rebind attempt', report);
    if ((report.tabs === 0 || report.actions === 0) && attempt < MAX_ATTEMPTS) {
        setTimeout(() => rebindDemoHandlers(attempt + 1), 600);
    } else if (attempt >= MAX_ATTEMPTS && (report.tabs === 0 || report.actions === 0)) {
        console.error('[GAuth Demo] Failed to bind some handlers after retries.');
    }
}

// Navigation
function scrollToDemo() {
    document.getElementById('demo').scrollIntoView({ 
        behavior: 'smooth' 
    });
}

// Token management functions have been migrated to modules/tokens.js
// createToken / validateToken / revokeToken now provided via GAuth namespace and ES module.

// Token Metrics Panel
async function fetchTokenMetrics() {
    try {
        const res = await fetch('/api/v1/token/metrics');
        const data = await res.json();
        if (!data.success) throw new Error('Failed to fetch token metrics');
        const m = data.metrics;
        const panel = document.getElementById('token-metrics-panel');
        if (panel) {
            panel.innerHTML = `<div class='text-xs text-gray-700'>
                <b>Created:</b> ${m.created} &nbsp; <b>Validated:</b> ${m.validated} &nbsp; <b>Revoked:</b> ${m.revoked} &nbsp; <b>Total:</b> ${m.total}
            </div>`;
        }
    } catch (e) {
        const panel = document.getElementById('token-metrics-panel');
        if (panel) panel.innerHTML = `<span class='text-red-500'>Failed to load token metrics</span>`;
    }
}

// Optionally, add event listeners for a metrics button
// Tab system
function showTab(e, tabId) {
    // Hide all tab contents
    const tabContents = document.querySelectorAll('.tab-content');
    tabContents.forEach(content => {
        content.style.display = 'none';
        content.classList.remove('active');
    });
    
    // Remove active class from all tab buttons
    const tabButtons = document.querySelectorAll('.tab-button');
    tabButtons.forEach(button => button.classList.remove('active'));
    
    // Show selected tab content
    const selectedTab = document.getElementById(tabId);
    if (selectedTab) {
        selectedTab.style.display = 'block';
        selectedTab.classList.add('active');
    }
    
    // Add active class to clicked button
    if (e && e.target) {
        e.target.classList.add('active');
    }
}

// DOM bindings without inline handlers (CSP friendly)
document.addEventListener('DOMContentLoaded', () => {
    try {
        console.log('[GAuth Demo] DOMContentLoaded fired - initializing bindings');
        const bindReport = { tabs: 0, actions: 0 };
    // Tab buttons
        document.querySelectorAll('.tab-button[data-tab]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tabId = btn.getAttribute('data-tab');
                showTab(e, tabId);
            });
            bindReport.tabs++;
        });

        // Samples tab logic
        initSamplesTab();
// --- Samples Tab Logic ---
function initSamplesTab() {
    const runAdvancedSuiteBtn = document.getElementById('run-advanced-suite');
    const samplesList = document.getElementById('samples-list');
    const samplesSearch = document.getElementById('samples-search');
    const samplesOutput = document.getElementById('samples-output');
    let allExamples = [];
    let filteredExamples = [];
    const filterAdvanced = document.getElementById('filter-advanced');
    const filterNegative = document.getElementById('filter-negative');
    const filterBasics = document.getElementById('filter-basics');
    const runAllBasicsBtn = document.getElementById('run-all-basics');
    const poaMetricsPanel = document.getElementById('poa-metrics');
    const metricsElems = {
        req: document.getElementById('m-poa-req'),
        ok: document.getElementById('m-poa-ok'),
        jur: document.getElementById('m-poa-jur'),
        scope: document.getElementById('m-poa-scope'),
        missing: document.getElementById('m-poa-missing'),
        updated: document.getElementById('poa-metrics-updated')
    };

    if (!samplesList || !samplesSearch || !samplesOutput) return;

    // Fetch catalog
    fetch('/api/v1/beta/examples/catalog')
        .then(res => res.json())
        .then(data => {
            allExamples = (data.examples || []).map(e => enrichExample(e));
            filteredExamples = allExamples;
            renderSamplesList(filteredExamples);
            startMetricsPolling();
        })
        .catch(() => {
            samplesList.innerHTML = '<div class="text-red-500 p-4">Failed to load examples.</div>';
        });

    // Search filter
    function applyFilters() {
        const q = samplesSearch.value.toLowerCase();
        filteredExamples = allExamples.filter(ex => {
            if (q && !(ex.id.toLowerCase().includes(q) || (ex.title||'').toLowerCase().includes(q) || (ex.description||'').toLowerCase().includes(q))) return false;
            const isAdv = ex.isAdvanced;
            const hasNeg = ex.hasNegative;
            const isBasic = !isAdv && !hasNeg;
            if (!filterAdvanced.checked && isAdv) return false;
            if (!filterNegative.checked && hasNeg) return false;
            if (!filterBasics.checked && isBasic) return false;
            return true;
        });
        renderSamplesList(filteredExamples);
    }
    samplesSearch.addEventListener('input', applyFilters);
    [filterAdvanced, filterNegative, filterBasics].forEach(cb => cb && cb.addEventListener('change', applyFilters));

    function enrichExample(ex) {
        const title = (ex.title || '').toLowerCase();
        const desc = (ex.description || '').toLowerCase();
        ex.isAdvanced = /advanced/.test(title) || /advanced_poa/.test(ex.id);
        ex.hasNegative = /(negative|invalid|disallowed|missing)/.test(desc);
        return ex;
    }

    if (runAdvancedSuiteBtn) {
        runAdvancedSuiteBtn.addEventListener('click', () => runAdvancedSuite());
    }

    if (runAllBasicsBtn) {
        runAllBasicsBtn.addEventListener('click', () => runAllBasics());
    }

    function renderSamplesList(examples) {
        if (!examples.length) {
            samplesList.innerHTML = '<div class="text-gray-400 p-4">No examples found.</div>';
            return;
        }
        // Sort: featured categories first (gauth_protocol_basics), then alphabetical
        const featuredPrefix = 'gauth_protocol_basics';
        const sorted = [...examples].sort((a,b) => {
            const aFeat = a.id.startsWith(featuredPrefix) ? 0 : 1;
            const bFeat = b.id.startsWith(featuredPrefix) ? 0 : 1;
            if (aFeat !== bFeat) return aFeat - bFeat;
            return (a.title || a.id).localeCompare(b.title || b.id);
        });
        samplesList.innerHTML = sorted.map(ex => {
            const isAdvanced = /advanced/i.test(ex.title || '') || /advanced_poa/i.test(ex.id);
            const hasNegatives = /negative|invalid|disallowed|missing/i.test(ex.description || '');
            const badges = [
                isAdvanced ? '<span class="ml-2 inline-block bg-purple-100 text-purple-700 text-[10px] px-2 py-0.5 rounded">ADV</span>' : '',
                hasNegatives ? '<span class="ml-1 inline-block bg-red-100 text-red-700 text-[10px] px-2 py-0.5 rounded" title="Contains negative validation cases">NEG</span>' : ''
            ].join('');
            return `
            <div class="group border-b border-gray-100 px-4 py-2 hover:bg-gray-50 transition">
                <div class="flex items-start justify-between gap-4">
                    <div class="flex-1 min-w-0">
                        <div class="font-semibold text-gray-800 flex items-center">${escapeHtml(ex.title || ex.id)} ${badges}</div>
                        <div class="text-xs text-gray-500 truncate" title="${escapeHtml(ex.description || ex.id)}">${escapeHtml(ex.description || ex.id)}</div>
                        <div class="text-[10px] text-gray-400 mt-0.5">ID: ${escapeHtml(ex.id)}</div>
                    </div>
                    <div class="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition">
                        <button class="run-sample-btn bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-1 px-3 rounded shadow" data-sample-id="${ex.id}"><i class="fas fa-play mr-1"></i>Run</button>
                    </div>
                </div>
            </div>`;
        }).join('');
        samplesList.querySelectorAll('.run-sample-btn').forEach(btn => {
            btn.addEventListener('click', () => runSample(btn.getAttribute('data-sample-id')));
        });
    }

    function runSample(id) {
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Queued ${id}]</span><br><span class='text-yellow-400'>Submitting job...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
    fetch('/api/v1/beta/examples/run', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id })
        })
        .then(res => res.json())
        .then(data => {
            if (!data.success) {
                samplesOutput.innerHTML = `<span class='text-red-400'>Failed to queue job: ${escapeHtml(data.message || 'unknown error')}</span>`;
                return;
            }
            const jobId = data.job_id;
            samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} queued for ${id}]</span><br><span class='text-yellow-400'>State: ${data.state}</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            attachLogStream(jobId, id);
        })
        .catch(err => {
            samplesOutput.innerHTML = `<span class='text-red-400'>Error queuing job: ${escapeHtml(err.message)}`;
        });
    }

    // Attach SSE log stream for job (with reconnect + idle watchdog)
    function attachLogStream(jobId, exampleId) {
        let finished = false;
        let lastEventTs = Date.now();
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 5;
        let es;
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>Waiting for logs...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
        let logLines = [];
        let state = 'unknown';
        let output = '';
        let error = '';

        function startStream() {
            const primaryURL = `/api/v1/beta/examples/run/${jobId}/logs`;
            const fallbackURL = `/api/v1/educational/examples/run/${jobId}/logs`;
            function openES(url, triedFallback){
                es = new EventSource(url);
                es.onerror = () => {
                    if(!triedFallback){
                        es.close();
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[primary stream failed – trying deprecated path]</span>`;
                        openES(fallbackURL, true);
                    }
                };
            }
            openES(primaryURL, false);
            es.addEventListener('open', () => {
                lastEventTs = Date.now();
                reconnectAttempts = 0;
            });
            es.addEventListener('status', e => {
                lastEventTs = Date.now();
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                    updateSamplesOutput();
                } catch {}
            });
            es.addEventListener('log', e => {
                lastEventTs = Date.now();
                logLines.push(e.data);
                updateSamplesOutput();
            });
            es.addEventListener('done', e => {
                lastEventTs = Date.now();
                finished = true;
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                } catch {}
                updateSamplesOutput();
                es.close();
            });
            es.onerror = () => {
                if (!finished) {
                    es.close();
                    if (reconnectAttempts < maxReconnectAttempts) {
                        reconnectAttempts++;
                        const delay = Math.min(3000, 500 * reconnectAttempts);
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[reconnecting attempt ${reconnectAttempts} in ${delay}ms]</span>`;
                        setTimeout(startStream, delay);
                    } else {
                        samplesOutput.innerHTML += `<br><span class='text-red-400'>[stream error: max retries]</span>`;
                    }
                }
            };
        }
        startStream();

        const watchdog = setInterval(() => {
            if (finished) { clearInterval(watchdog); return; }
            const idle = Date.now() - lastEventTs;
            if (idle > 15000) {
                samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[idle – waiting for events]</span>`;
                lastEventTs = Date.now();
            }
        }, 5000);

        function updateSamplesOutput() {
            let html = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br>`;
            html += `<span class='text-yellow-400'>State: ${escapeHtml(state)}</span><br>`;
            if (logLines.length) {
                html += `<pre class='text-blue-300 whitespace-pre-wrap mt-2'>${escapeHtml(logLines.join('\n'))}</pre>`;
            }
            if (state === 'done') {
                html += `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'failed') {
                html += `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId})</span><br>`;
                if (error) html += `<span class='text-red-300'>Error: ${escapeHtml(error)}</span><br>`;
                if (output) html += `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'timeout') {
                html += `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            }
            html += `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            samplesOutput.innerHTML = html;
        }
    }
    }

    function pollJob(jobId, exampleId, startTs, chainCtx) {
    fetch(`/api/v1/beta/examples/run/${jobId}/status`)
            .then(r => r.json())
            .then(status => {
                if (!status.success) {
                    samplesOutput.innerHTML = `<span class='text-red-400'>Status error: ${escapeHtml(status.message || 'unknown')}</span>`;
                    return;
                }
                const job = status.job;
                const elapsed = ((Date.now() - startTs) / 1000).toFixed(2);
                if (job.state === 'queued' || job.state === 'running') {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>State: ${job.state} (elapsed ${elapsed}s)</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    setTimeout(() => pollJob(jobId, exampleId, startTs, chainCtx), 750);
                    return;
                }
                // Terminal states
                if (job.state === 'done') {
                    samplesOutput.innerHTML = `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId}) in ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '<span class="text-gray-400">(no output)</span>') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else if (job.state === 'failed') {
                    samplesOutput.innerHTML = `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.error ? `<span class='text-red-300'>Error: ${escapeHtml(job.error)}</span><br>` : '') +
                        (job.output ? `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed, error: job.error });
                        runNextInChain(chainCtx); // continue even on failure for educational completeness
                    }
                } else if (job.state === 'timeout') {
                    samplesOutput.innerHTML = `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>Job ${jobId} ended in state ${job.state}</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                }
            })
            .catch(err => {
                samplesOutput.innerHTML = `<span class='text-red-400'>Polling error: ${escapeHtml(err.message)}</span>`;
                if (chainCtx) {
                    chainCtx.results.push({ id: exampleId, state: 'error', error: err.message });
                    runNextInChain(chainCtx);
                }
            });
    }

    function runAllBasics() {
        const chain = ['gauth_protocol_basics:minimal_poa', 'gauth_protocol_basics:delegation', 'gauth_protocol_basics:token'];
        const ctx = { queue: chain, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running basic examples sequentially...</span>`;
        runNextInChain(ctx);
    }

    function runAdvancedSuite() {
        // Use all advanced and negative samples from the loaded catalog
        const advanced = allExamples.filter(ex => ex.isAdvanced);
        const negative = allExamples.filter(ex => ex.hasNegative);
        // Remove duplicates by id
        const ids = Array.from(new Set([...advanced, ...negative].map(ex => ex.id)));
        if (!ids.length) {
            samplesOutput.innerHTML = `<span class='text-yellow-400'>No advanced or negative samples found in catalog.</span>`;
            return;
        }
        const ctx = { queue: ids, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running advanced/negative examples sequentially...</span>`;
        runNextInChain(ctx);
    }
    function runNextInChain(ctx) {
        if (!ctx.queue.length) {
            // render summary
            const summary = ctx.results.map(r => `# ${r.id} -> ${r.state} (${r.elapsed||'-'}s)`).join('\n');
            samplesOutput.innerHTML = `<span class='text-green-400'>Composite run complete.</span><br><pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(summary)}</pre>`;
            // Show export buttons
            const exportDiv = document.getElementById('composite-export-buttons');
            if (exportDiv) {
                exportDiv.classList.remove('hidden');
                // Store summary in exportDiv for download handlers
                exportDiv.dataset.summary = JSON.stringify(ctx.results);
                exportDiv.dataset.summaryText = summary;
            }
            return;
        }
// Composite export download logic
document.addEventListener('DOMContentLoaded', function() {
    const exportDiv = document.getElementById('composite-export-buttons');
    if (!exportDiv) return;
    const btnJson = document.getElementById('download-composite-json');
    const btnCsv = document.getElementById('download-composite-csv');
    // Hide buttons initially
    exportDiv.classList.add('hidden');
    // Download JSON
    btnJson.addEventListener('click', function() {
        const summaryArr = exportDiv.dataset.summary;
        if (!summaryArr) return;
    fetch('/api/v1/beta/examples/composite/export/json', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: summaryArr
        })
        .then(r => {
            if (!r.ok) throw new Error('Export failed');
            return r.blob();
        })
        .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'composite_run_summary.json';
            document.body.appendChild(a);
            a.click();
            setTimeout(() => { document.body.removeChild(a); window.URL.revokeObjectURL(url); }, 100);
        })
        .catch(() => alert('Failed to download JSON export.'));
    });
    // Download CSV
    btnCsv.addEventListener('click', function() {
        const summaryArr = exportDiv.dataset.summary;
        if (!summaryArr) return;
    fetch('/api/v1/beta/examples/composite/export/csv', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: summaryArr
        })
        .then(r => {
            if (!r.ok) throw new Error('Export failed');
            return r.blob();
        })
        .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'composite_run_summary.csv';
            document.body.appendChild(a);
            a.click();
            setTimeout(() => { document.body.removeChild(a); window.URL.revokeObjectURL(url); }, 100);
        })
        .catch(() => alert('Failed to download CSV export.'));
    });
});
        const next = ctx.queue.shift();
    fetch('/api/v1/beta/examples/run', {
            method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: next })
        }).then(r => r.json()).then(data => {
            if (!data.success) {
                ctx.results.push({ id: next, state: 'queue_failed', error: data.message });
                runNextInChain(ctx);
                return;
            }
            pollJob(data.job_id, next, Date.now(), ctx);
        }).catch(err => {
            ctx.results.push({ id: next, state: 'error', error: err.message });
            runNextInChain(ctx);
        });
    }

    function startMetricsPolling() {
        if (!poaMetricsPanel) return;
        poaMetricsPanel.classList.remove('hidden');
        const fetchMetrics = () => {
            fetch('/api/v1/poa/metrics').then(r => r.json()).then(data => {
                if (!data.success) return;
                const m = data.metrics || {};
                metricsElems.req.textContent = m.total_requests;
                metricsElems.ok.textContent = m.total_success;
                metricsElems.jur.textContent = m.rejected_jurisdiction;
                metricsElems.scope.textContent = m.rejected_scope;
                metricsElems.missing.textContent = m.rejected_missing_fields;
                metricsElems.updated.textContent = new Date(data.timestamp).toLocaleTimeString();
                // Render POA duration sparkline
                const svg = document.getElementById('poa-duration-sparkline');
                const lastVal = document.getElementById('poa-duration-last');
                if (svg && m.recent_durations && Array.isArray(m.recent_durations) && m.recent_durations.length > 0) {
                    const values = m.recent_durations;
                    const w = 120, h = 24, pad = 2;
                    const min = Math.min(...values), max = Math.max(...values);
                    const range = max - min || 1;
                    const points = values.map((v, i) => {
                        const x = pad + i * ((w-2*pad)/(values.length-1||1));
                        const y = h - pad - ((v-min)/range)*(h-2*pad);
                        return `${x},${y}`;
                    }).join(' ');
                    svg.innerHTML = `<polyline fill="none" stroke="#2563eb" stroke-width="2" points="${points}" />`;
                    lastVal.textContent = values[values.length-1].toFixed(2);
                } else if (svg) {
                    svg.innerHTML = '';
                    if (lastVal) lastVal.textContent = '';
                }
            }).catch(()=>{});
        };
        fetchMetrics();
        setInterval(fetchMetrics, 4000);
    }

    function escapeHtml(str) {
        return String(str).replace(/[&<>"']/g, function(m) {
            return ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;','\'':'&#39;'})[m];
        });
    }
// END of main samples tab logic block

    // Default tab activation
    const activeBtn = document.querySelector('.tab-button.active[data-tab]') || document.querySelector('.tab-button[data-tab]');
    if (activeBtn) {
        showTab({ target: activeBtn }, activeBtn.getAttribute('data-tab'));
    }

    // Action buttons
    const actionMap = {
        'create-token': createToken,
        'validate-token': validateToken,
        'revoke-token': revokeToken,
        'check-authorization': checkAuthorization,
        'publish-event': publishEvent,
        'subscribe-events': subscribeEvents,
        'view-audit-log': viewAuditLog,
        'generate-report': generateReport
    };
        Object.entries(actionMap).forEach(([key, fn]) => {
            document.querySelectorAll(`[data-action="${key}"]`).forEach(el => {
                el.addEventListener('click', fn);
                bindReport.actions++;
            });
        });
        console.log(`[GAuth Demo] Initial binding complete:`, bindReport);
        if (bindReport.tabs === 0 || bindReport.actions === 0) {
            console.warn('[GAuth Demo] Warning: No bindings detected (tabs or actions). Scheduling retry...');
            setTimeout(rebindDemoHandlers, 500);
        }
    } catch (e) {
        console.error('[GAuth Demo] Initialization error:', e);
        setTimeout(rebindDemoHandlers, 750);
    }
});

// Fallback rebind function (in case of deferred HTML insertion or race conditions)
function rebindDemoHandlers(attempt = 1) {
    const MAX_ATTEMPTS = 5;
    const report = { tabs: 0, actions: 0, attempt };
    document.querySelectorAll('.tab-button[data-tab]').forEach(btn => {
        if (!btn.__gauthBound) {
            btn.addEventListener('click', (e) => {
                const tabId = btn.getAttribute('data-tab');
                showTab(e, tabId);
            });
            btn.__gauthBound = true;
            report.tabs++;
        }
    });
    const actionMap = {
        'create-token': createToken,
        'validate-token': validateToken,
        'revoke-token': revokeToken,
        'check-authorization': checkAuthorization,
        'publish-event': publishEvent,
        'subscribe-events': subscribeEvents,
        'view-audit-log': viewAuditLog,
        'generate-report': generateReport
    };
    Object.entries(actionMap).forEach(([key, fn]) => {
        document.querySelectorAll(`[data-action="${key}"]`).forEach(el => {
            if (!el.__gauthBound) {
                el.addEventListener('click', fn);
                el.__gauthBound = true;
                report.actions++;
            }
        });
    });
    console.log('[GAuth Demo] Rebind attempt', report);
    if ((report.tabs === 0 || report.actions === 0) && attempt < MAX_ATTEMPTS) {
        setTimeout(() => rebindDemoHandlers(attempt + 1), 600);
    } else if (attempt >= MAX_ATTEMPTS && (report.tabs === 0 || report.actions === 0)) {
        console.error('[GAuth Demo] Failed to bind some handlers after retries.');
    }
}

// Navigation
function scrollToDemo() {
    document.getElementById('demo').scrollIntoView({ 
        behavior: 'smooth' 
    });
}

// Event and Audit Stream Control


document.addEventListener('DOMContentLoaded', function() {
    try {
        console.log('[GAuth Demo] DOMContentLoaded fired - initializing bindings');
        const bindReport = { tabs: 0, actions: 0 };
    // Tab buttons
        document.querySelectorAll('.tab-button[data-tab]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tabId = btn.getAttribute('data-tab');
                showTab(e, tabId);
            });
            bindReport.tabs++;
        });

        // Samples tab logic
        initSamplesTab();
// --- Samples Tab Logic ---
function initSamplesTab() {
    const runAdvancedSuiteBtn = document.getElementById('run-advanced-suite');
    const samplesList = document.getElementById('samples-list');
    const samplesSearch = document.getElementById('samples-search');
    const samplesOutput = document.getElementById('samples-output');
    let allExamples = [];
    let filteredExamples = [];
    const filterAdvanced = document.getElementById('filter-advanced');
    const filterNegative = document.getElementById('filter-negative');
    const filterBasics = document.getElementById('filter-basics');
    const runAllBasicsBtn = document.getElementById('run-all-basics');
    const poaMetricsPanel = document.getElementById('poa-metrics');
    const metricsElems = {
        req: document.getElementById('m-poa-req'),
        ok: document.getElementById('m-poa-ok'),
        jur: document.getElementById('m-poa-jur'),
        scope: document.getElementById('m-poa-scope'),
        missing: document.getElementById('m-poa-missing'),
        updated: document.getElementById('poa-metrics-updated')
    };

    if (!samplesList || !samplesSearch || !samplesOutput) return;

    // Fetch catalog
    fetch('/api/v1/beta/examples/catalog')
        .then(res => res.json())
        .then(data => {
            allExamples = (data.examples || []).map(e => enrichExample(e));
            filteredExamples = allExamples;
            renderSamplesList(filteredExamples);
            startMetricsPolling();
        })
        .catch(() => {
            samplesList.innerHTML = '<div class="text-red-500 p-4">Failed to load examples.</div>';
        });

    // Search filter
    function applyFilters() {
        const q = samplesSearch.value.toLowerCase();
        filteredExamples = allExamples.filter(ex => {
            if (q && !(ex.id.toLowerCase().includes(q) || (ex.title||'').toLowerCase().includes(q) || (ex.description||'').toLowerCase().includes(q))) return false;
            const isAdv = ex.isAdvanced;
            const hasNeg = ex.hasNegative;
            const isBasic = !isAdv && !hasNeg;
            if (!filterAdvanced.checked && isAdv) return false;
            if (!filterNegative.checked && hasNeg) return false;
            if (!filterBasics.checked && isBasic) return false;
            return true;
        });
        renderSamplesList(filteredExamples);
    }
    samplesSearch.addEventListener('input', applyFilters);
    [filterAdvanced, filterNegative, filterBasics].forEach(cb => cb && cb.addEventListener('change', applyFilters));

    function enrichExample(ex) {
        const title = (ex.title || '').toLowerCase();
        const desc = (ex.description || '').toLowerCase();
        ex.isAdvanced = /advanced/.test(title) || /advanced_poa/.test(ex.id);
        ex.hasNegative = /(negative|invalid|disallowed|missing)/.test(desc);
        return ex;
    }

    if (runAdvancedSuiteBtn) {
        runAdvancedSuiteBtn.addEventListener('click', () => runAdvancedSuite());
    }

    if (runAllBasicsBtn) {
        runAllBasicsBtn.addEventListener('click', () => runAllBasics());
    }

    function renderSamplesList(examples) {
        if (!examples.length) {
            samplesList.innerHTML = '<div class="text-gray-400 p-4">No examples found.</div>';
            return;
        }
        // Sort: featured categories first (gauth_protocol_basics), then alphabetical
        const featuredPrefix = 'gauth_protocol_basics';
        const sorted = [...examples].sort((a,b) => {
            const aFeat = a.id.startsWith(featuredPrefix) ? 0 : 1;
            const bFeat = b.id.startsWith(featuredPrefix) ? 0 : 1;
            if (aFeat !== bFeat) return aFeat - bFeat;
            return (a.title || a.id).localeCompare(b.title || b.id);
        });
        samplesList.innerHTML = sorted.map(ex => {
            const isAdvanced = /advanced/i.test(ex.title || '') || /advanced_poa/i.test(ex.id);
            const hasNegatives = /negative|invalid|disallowed|missing/i.test(ex.description || '');
            const badges = [
                isAdvanced ? '<span class="ml-2 inline-block bg-purple-100 text-purple-700 text-[10px] px-2 py-0.5 rounded">ADV</span>' : '',
                hasNegatives ? '<span class="ml-1 inline-block bg-red-100 text-red-700 text-[10px] px-2 py-0.5 rounded" title="Contains negative validation cases">NEG</span>' : ''
            ].join('');
            return `
            <div class="group border-b border-gray-100 px-4 py-2 hover:bg-gray-50 transition">
                <div class="flex items-start justify-between gap-4">
                    <div class="flex-1 min-w-0">
                        <div class="font-semibold text-gray-800 flex items-center">${escapeHtml(ex.title || ex.id)} ${badges}</div>
                        <div class="text-xs text-gray-500 truncate" title="${escapeHtml(ex.description || ex.id)}">${escapeHtml(ex.description || ex.id)}</div>
                        <div class="text-[10px] text-gray-400 mt-0.5">ID: ${escapeHtml(ex.id)}</div>
                    </div>
                    <div class="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition">
                        <button class="run-sample-btn bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-1 px-3 rounded shadow" data-sample-id="${ex.id}"><i class="fas fa-play mr-1"></i>Run</button>
                    </div>
                </div>
            </div>`;
        }).join('');
        samplesList.querySelectorAll('.run-sample-btn').forEach(btn => {
            btn.addEventListener('click', () => runSample(btn.getAttribute('data-sample-id')));
        });
    }

    function runSample(id) {
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Queued ${id}]</span><br><span class='text-yellow-400'>Submitting job...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
    fetch('/api/v1/beta/examples/run', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id })
        })
        .then(res => res.json())
        .then(data => {
            if (!data.success) {
                samplesOutput.innerHTML = `<span class='text-red-400'>Failed to queue job: ${escapeHtml(data.message || 'unknown error')}</span>`;
                return;
            }
            const jobId = data.job_id;
            samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} queued for ${id}]</span><br><span class='text-yellow-400'>State: ${data.state}</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            attachLogStream(jobId, id);
        })
        .catch(err => {
            samplesOutput.innerHTML = `<span class='text-red-400'>Error queuing job: ${escapeHtml(err.message)}`;
        });
    }

    // Attach SSE log stream for job (with reconnect + idle watchdog)
    function attachLogStream(jobId, exampleId) {
        let finished = false;
        let lastEventTs = Date.now();
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 5;
        let es;
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>Waiting for logs...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
        let logLines = [];
        let state = 'unknown';
        let output = '';
        let error = '';

        function startStream() {
            const primaryURL = `/api/v1/beta/examples/run/${jobId}/logs`;
            const fallbackURL = `/api/v1/educational/examples/run/${jobId}/logs`;
            function openES(url, triedFallback){
                es = new EventSource(url);
                es.onerror = () => {
                    if(!triedFallback){
                        es.close();
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[primary stream failed – trying deprecated path]</span>`;
                        openES(fallbackURL, true);
                    }
                };
            }
            openES(primaryURL, false);
            es.addEventListener('open', () => {
                lastEventTs = Date.now();
                reconnectAttempts = 0;
            });
            es.addEventListener('status', e => {
                lastEventTs = Date.now();
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                    updateSamplesOutput();
                } catch {}
            });
            es.addEventListener('log', e => {
                lastEventTs = Date.now();
                logLines.push(e.data);
                updateSamplesOutput();
            });
            es.addEventListener('done', e => {
                lastEventTs = Date.now();
                finished = true;
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                } catch {}
                updateSamplesOutput();
                es.close();
            });
            es.onerror = () => {
                if (!finished) {
                    es.close();
                    if (reconnectAttempts < maxReconnectAttempts) {
                        reconnectAttempts++;
                        const delay = Math.min(3000, 500 * reconnectAttempts);
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[reconnecting attempt ${reconnectAttempts} in ${delay}ms]</span>`;
                        setTimeout(startStream, delay);
                    } else {
                        samplesOutput.innerHTML += `<br><span class='text-red-400'>[stream error: max retries]</span>`;
                    }
                }
            };
        }
        startStream();



        const watchdog = setInterval(() => {
            if (finished) { clearInterval(watchdog); return; }
            const idle = Date.now() - lastEventTs;
            if (idle > 15000) {
                samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[idle – waiting for events]</span>`;
                lastEventTs = Date.now();
            }
        }, 5000);

        function updateSamplesOutput() {
            let html = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br>`;
            html += `<span class='text-yellow-400'>State: ${escapeHtml(state)}</span><br>`;
            if (logLines.length) {
                html += `<pre class='text-blue-300 whitespace-pre-wrap mt-2'>${escapeHtml(logLines.join('\n'))}</pre>`;
            }
            if (state === 'done') {
                html += `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'failed') {
                html += `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId})</span><br>`;
                if (error) html += `<span class='text-red-300'>Error: ${escapeHtml(error)}</span><br>`;
                if (output) html += `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'timeout') {
                html += `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            }
            html += `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            samplesOutput.innerHTML = html;
        }
    }
    }

    function pollJob(jobId, exampleId, startTs, chainCtx) {
    fetch(`/api/v1/beta/examples/run/${jobId}/status`)
            .then(r => r.json())
            .then(status => {
                if (!status.success) {
                    samplesOutput.innerHTML = `<span class='text-red-400'>Status error: ${escapeHtml(status.message || 'unknown')}</span>`;
                    return;
                }
                const job = status.job;
                const elapsed = ((Date.now() - startTs) / 1000).toFixed(2);
                if (job.state === 'queued' || job.state === 'running') {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>State: ${job.state} (elapsed ${elapsed}s)</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    setTimeout(() => pollJob(jobId, exampleId, startTs, chainCtx), 750);
                    return;
                }
                // Terminal states
                if (job.state === 'done') {
                    samplesOutput.innerHTML = `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId}) in ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '<span class="text-gray-400">(no output)</span>') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else if (job.state === 'failed') {
                    samplesOutput.innerHTML = `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.error ? `<span class='text-red-300'>Error: ${escapeHtml(job.error)}</span><br>` : '') +
                        (job.output ? `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed, error: job.error });
                        runNextInChain(chainCtx); // continue even on failure for educational completeness
                    }
                } else if (job.state === 'timeout') {
                    samplesOutput.innerHTML = `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>Job ${jobId} ended in state ${job.state}</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                }
            })
            .catch(err => {
                samplesOutput.innerHTML = `<span class='text-red-400'>Polling error: ${escapeHtml(err.message)}</span>`;
                if (chainCtx) {
                    chainCtx.results.push({ id: exampleId, state: 'error', error: err.message });
                    runNextInChain(chainCtx);
                }
            });
    }

    function runAllBasics() {
        const chain = ['gauth_protocol_basics:minimal_poa', 'gauth_protocol_basics:delegation', 'gauth_protocol_basics:token'];
        const ctx = { queue: chain, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running basic examples sequentially...</span>`;
        runNextInChain(ctx);
    }

    function runAdvancedSuite() {
        // Use all advanced and negative samples from the loaded catalog
        const advanced = allExamples.filter(ex => ex.isAdvanced);
        const negative = allExamples.filter(ex => ex.hasNegative);
        // Remove duplicates by id
        const ids = Array.from(new Set([...advanced, ...negative].map(ex => ex.id)));
        if (!ids.length) {
            samplesOutput.innerHTML = `<span class='text-yellow-400'>No advanced or negative samples found in catalog.</span>`;
            return;
        }
        const ctx = { queue: ids, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running advanced/negative examples sequentially...</span>`;
        runNextInChain(ctx);
    }
    function runNextInChain(ctx) {
        if (!ctx.queue.length) {
            // render summary
            const summary = ctx.results.map(r => `# ${r.id} -> ${r.state} (${r.elapsed||'-'}s)`).join('\n');
            samplesOutput.innerHTML = `<span class='text-green-400'>Composite run complete.</span><br><pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(summary)}</pre>`;
            // Show export buttons
            const exportDiv = document.getElementById('composite-export-buttons');
            if (exportDiv) {
                exportDiv.classList.remove('hidden');
                // Store summary in exportDiv for download handlers
                exportDiv.dataset.summary = JSON.stringify(ctx.results);
                exportDiv.dataset.summaryText = summary;
            }
            return;
        }
// Composite export download logic
document.addEventListener('DOMContentLoaded', function() {
    const exportDiv = document.getElementById('composite-export-buttons');
    if (!exportDiv) return;
    const btnJson = document.getElementById('download-composite-json');
    const btnCsv = document.getElementById('download-composite-csv');
    // Hide buttons initially
    exportDiv.classList.add('hidden');
    // Download JSON
    btnJson.addEventListener('click', function() {
        const summaryArr = exportDiv.dataset.summary;
        if (!summaryArr) return;
    fetch('/api/v1/beta/examples/composite/export/json', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: summaryArr
        })
        .then(r => {
            if (!r.ok) throw new Error('Export failed');
            return r.blob();
        })
        .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'composite_run_summary.json';
            document.body.appendChild(a);
            a.click();
            setTimeout(() => { document.body.removeChild(a); window.URL.revokeObjectURL(url); }, 100);
        })
        .catch(() => alert('Failed to download JSON export.'));
    });
    // Download CSV
    btnCsv.addEventListener('click', function() {
        const summaryArr = exportDiv.dataset.summary;
        if (!summaryArr) return;
    fetch('/api/v1/beta/examples/composite/export/csv', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: summaryArr
        })
        .then(r => {
            if (!r.ok) throw new Error('Export failed');
            return r.blob();
        })
        .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'composite_run_summary.csv';
            document.body.appendChild(a);
            a.click();
            setTimeout(() => { document.body.removeChild(a); window.URL.revokeObjectURL(url); }, 100);
        })
        .catch(() => alert('Failed to download CSV export.'));
    });
});
        const next = ctx.queue.shift();
    fetch('/api/v1/beta/examples/run', {
            method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: next })
        }).then(r => r.json()).then(data => {
            if (!data.success) {
                ctx.results.push({ id: next, state: 'queue_failed', error: data.message });
                runNextInChain(ctx);
                return;
            }
            pollJob(data.job_id, next, Date.now(), ctx);
        }).catch(err => {
            ctx.results.push({ id: next, state: 'error', error: err.message });
            runNextInChain(ctx);
        });
    }

    function startMetricsPolling() {
        if (!poaMetricsPanel) return;
        poaMetricsPanel.classList.remove('hidden');
        const fetchMetrics = () => {
            fetch('/api/v1/poa/metrics').then(r => r.json()).then(data => {
                if (!data.success) return;
                const m = data.metrics || {};
                metricsElems.req.textContent = m.total_requests;
                metricsElems.ok.textContent = m.total_success;
                metricsElems.jur.textContent = m.rejected_jurisdiction;
                metricsElems.scope.textContent = m.rejected_scope;
                metricsElems.missing.textContent = m.rejected_missing_fields;
                metricsElems.updated.textContent = new Date(data.timestamp).toLocaleTimeString();
                // Render POA duration sparkline
                const svg = document.getElementById('poa-duration-sparkline');
                const lastVal = document.getElementById('poa-duration-last');
                if (svg && m.recent_durations && Array.isArray(m.recent_durations) && m.recent_durations.length > 0) {
                    const values = m.recent_durations;
                    const w = 120, h = 24, pad = 2;
                    const min = Math.min(...values), max = Math.max(...values);
                    const range = max - min || 1;
                    const points = values.map((v, i) => {
                        const x = pad + i * ((w-2*pad)/(values.length-1||1));
                        const y = h - pad - ((v-min)/range)*(h-2*pad);
                        return `${x},${y}`;
                    }).join(' ');
                    svg.innerHTML = `<polyline fill="none" stroke="#2563eb" stroke-width="2" points="${points}" />`;
                    lastVal.textContent = values[values.length-1].toFixed(2);
                } else if (svg) {
                    svg.innerHTML = '';
                    if (lastVal) lastVal.textContent = '';
                }
            }).catch(()=>{});
        };
        fetchMetrics();
        setInterval(fetchMetrics, 4000);
    }

    function escapeHtml(str) {
        return String(str).replace(/[&<>"']/g, function(m) {
            return ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;','\'':'&#39;'})[m];
        });
    }
// END of main samples tab logic block

    // Default tab activation
    const activeBtn = document.querySelector('.tab-button.active[data-tab]') || document.querySelector('.tab-button[data-tab]');
    if (activeBtn) {
        showTab({ target: activeBtn }, activeBtn.getAttribute('data-tab'));
    }

    // Action buttons
    const actionMap = {
        'create-token': createToken,
        'validate-token': validateToken,
        'revoke-token': revokeToken,
        'check-authorization': checkAuthorization,
        'publish-event': publishEvent,
        'subscribe-events': subscribeEvents,
        'view-audit-log': viewAuditLog,
        'generate-report': generateReport
    };
        Object.entries(actionMap).forEach(([key, fn]) => {
            document.querySelectorAll(`[data-action="${key}"]`).forEach(el => {
                el.addEventListener('click', fn);
                bindReport.actions++;
            });
        });
        console.log(`[GAuth Demo] Initial binding complete:`, bindReport);
        if (bindReport.tabs === 0 || bindReport.actions === 0) {
            console.warn('[GAuth Demo] Warning: No bindings detected (tabs or actions). Scheduling retry...');
            setTimeout(rebindDemoHandlers, 500);
        }
    } catch (e) {
        console.error('[GAuth Demo] Initialization error:', e);
        setTimeout(rebindDemoHandlers, 750);
    }
});

// Fallback rebind function (in case of deferred HTML insertion or race conditions)
function rebindDemoHandlers(attempt = 1) {
    const MAX_ATTEMPTS = 5;
    const report = { tabs: 0, actions: 0, attempt };
    document.querySelectorAll('.tab-button[data-tab]').forEach(btn => {
        if (!btn.__gauthBound) {
            btn.addEventListener('click', (e) => {
                const tabId = btn.getAttribute('data-tab');
                showTab(e, tabId);
            });
            btn.__gauthBound = true;
            report.tabs++;
        }
    });
    const actionMap = {
        'create-token': createToken,
        'validate-token': validateToken,
        'revoke-token': revokeToken,
        'check-authorization': checkAuthorization,
        'publish-event': publishEvent,
        'subscribe-events': subscribeEvents,
        'view-audit-log': viewAuditLog,
        'generate-report': generateReport
    };
    Object.entries(actionMap).forEach(([key, fn]) => {
        document.querySelectorAll(`[data-action="${key}"]`).forEach(el => {
            if (!el.__gauthBound) {
                el.addEventListener('click', fn);
                el.__gauthBound = true;
                report.actions++;
            }
        });
    });
    console.log('[GAuth Demo] Rebind attempt', report);
    if ((report.tabs === 0 || report.actions === 0) && attempt < MAX_ATTEMPTS) {
        setTimeout(() => rebindDemoHandlers(attempt + 1), 600);
    } else if (attempt >= MAX_ATTEMPTS && (report.tabs === 0 || report.actions === 0)) {
        console.error('[GAuth Demo] Failed to bind some handlers after retries.');
    }
}

// Navigation
function scrollToDemo() {
    document.getElementById('demo').scrollIntoView({ 
        behavior: 'smooth' 
    });
}

// Event and Audit Stream Control


document.addEventListener('DOMContentLoaded', function() {
    try {
        console.log('[GAuth Demo] DOMContentLoaded fired - initializing bindings');
        const bindReport = { tabs: 0, actions: 0 };
    // Tab buttons
        document.querySelectorAll('.tab-button[data-tab]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tabId = btn.getAttribute('data-tab');
                showTab(e, tabId);
            });
            bindReport.tabs++;
        });

        // Samples tab logic
        initSamplesTab();
// --- Samples Tab Logic ---
function initSamplesTab() {
    const runAdvancedSuiteBtn = document.getElementById('run-advanced-suite');
    const samplesList = document.getElementById('samples-list');
    const samplesSearch = document.getElementById('samples-search');
    const samplesOutput = document.getElementById('samples-output');
    let allExamples = [];
    let filteredExamples = [];
    const filterAdvanced = document.getElementById('filter-advanced');
    const filterNegative = document.getElementById('filter-negative');
    const filterBasics = document.getElementById('filter-basics');
    const runAllBasicsBtn = document.getElementById('run-all-basics');
    const poaMetricsPanel = document.getElementById('poa-metrics');
    const metricsElems = {
        req: document.getElementById('m-poa-req'),
        ok: document.getElementById('m-poa-ok'),
        jur: document.getElementById('m-poa-jur'),
        scope: document.getElementById('m-poa-scope'),
        missing: document.getElementById('m-poa-missing'),
        updated: document.getElementById('poa-metrics-updated')
    };

    if (!samplesList || !samplesSearch || !samplesOutput) return;

    // Fetch catalog
    fetch('/api/v1/beta/examples/catalog')
        .then(res => res.json())
        .then(data => {
            allExamples = (data.examples || []).map(e => enrichExample(e));
            filteredExamples = allExamples;
            renderSamplesList(filteredExamples);
            startMetricsPolling();
        })
        .catch(() => {
            samplesList.innerHTML = '<div class="text-red-500 p-4">Failed to load examples.</div>';
        });

    // Search filter
    function applyFilters() {
        const q = samplesSearch.value.toLowerCase();
        filteredExamples = allExamples.filter(ex => {
            if (q && !(ex.id.toLowerCase().includes(q) || (ex.title||'').toLowerCase().includes(q) || (ex.description||'').toLowerCase().includes(q))) return false;
            const isAdv = ex.isAdvanced;
            const hasNeg = ex.hasNegative;
            const isBasic = !isAdv && !hasNeg;
            if (!filterAdvanced.checked && isAdv) return false;
            if (!filterNegative.checked && hasNeg) return false;
            if (!filterBasics.checked && isBasic) return false;
            return true;
        });
        renderSamplesList(filteredExamples);
    }
    samplesSearch.addEventListener('input', applyFilters);
    [filterAdvanced, filterNegative, filterBasics].forEach(cb => cb && cb.addEventListener('change', applyFilters));

    function enrichExample(ex) {
        const title = (ex.title || '').toLowerCase();
        const desc = (ex.description || '').toLowerCase();
        ex.isAdvanced = /advanced/.test(title) || /advanced_poa/.test(ex.id);
        ex.hasNegative = /(negative|invalid|disallowed|missing)/.test(desc);
        return ex;
    }

    if (runAdvancedSuiteBtn) {
        runAdvancedSuiteBtn.addEventListener('click', () => runAdvancedSuite());
    }

    if (runAllBasicsBtn) {
        runAllBasicsBtn.addEventListener('click', () => runAllBasics());
    }

    function renderSamplesList(examples) {
        if (!examples.length) {
            samplesList.innerHTML = '<div class="text-gray-400 p-4">No examples found.</div>';
            return;
        }
        // Sort: featured categories first (gauth_protocol_basics), then alphabetical
        const featuredPrefix = 'gauth_protocol_basics';
        const sorted = [...examples].sort((a,b) => {
            const aFeat = a.id.startsWith(featuredPrefix) ? 0 : 1;
            const bFeat = b.id.startsWith(featuredPrefix) ? 0 : 1;
            if (aFeat !== bFeat) return aFeat - bFeat;
            return (a.title || a.id).localeCompare(b.title || b.id);
        });
        samplesList.innerHTML = sorted.map(ex => {
            const isAdvanced = /advanced/i.test(ex.title || '') || /advanced_poa/i.test(ex.id);
            const hasNegatives = /negative|invalid|disallowed|missing/i.test(ex.description || '');
            const badges = [
                isAdvanced ? '<span class="ml-2 inline-block bg-purple-100 text-purple-700 text-[10px] px-2 py-0.5 rounded">ADV</span>' : '',
                hasNegatives ? '<span class="ml-1 inline-block bg-red-100 text-red-700 text-[10px] px-2 py-0.5 rounded" title="Contains negative validation cases">NEG</span>' : ''
            ].join('');
            return `
            <div class="group border-b border-gray-100 px-4 py-2 hover:bg-gray-50 transition">
                <div class="flex items-start justify-between gap-4">
                    <div class="flex-1 min-w-0">
                        <div class="font-semibold text-gray-800 flex items-center">${escapeHtml(ex.title || ex.id)} ${badges}</div>
                        <div class="text-xs text-gray-500 truncate" title="${escapeHtml(ex.description || ex.id)}">${escapeHtml(ex.description || ex.id)}</div>
                        <div class="text-[10px] text-gray-400 mt-0.5">ID: ${escapeHtml(ex.id)}</div>
                    </div>
                    <div class="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition">
                        <button class="run-sample-btn bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-1 px-3 rounded shadow" data-sample-id="${ex.id}"><i class="fas fa-play mr-1"></i>Run</button>
                    </div>
                </div>
            </div>`;
        }).join('');
        samplesList.querySelectorAll('.run-sample-btn').forEach(btn => {
            btn.addEventListener('click', () => runSample(btn.getAttribute('data-sample-id')));
        });
    }

    function runSample(id) {
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Queued ${id}]</span><br><span class='text-yellow-400'>Submitting job...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
    fetch('/api/v1/beta/examples/run', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id })
        })
        .then(res => res.json())
        .then(data => {
            if (!data.success) {
                samplesOutput.innerHTML = `<span class='text-red-400'>Failed to queue job: ${escapeHtml(data.message || 'unknown error')}</span>`;
                return;
            }
            const jobId = data.job_id;
            samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} queued for ${id}]</span><br><span class='text-yellow-400'>State: ${data.state}</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            attachLogStream(jobId, id);
        })
        .catch(err => {
            samplesOutput.innerHTML = `<span class='text-red-400'>Error queuing job: ${escapeHtml(err.message)}`;
        });
    }

    // Attach SSE log stream for job (with reconnect + idle watchdog)
    function attachLogStream(jobId, exampleId) {
        let finished = false;
        let lastEventTs = Date.now();
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 5;
        let es;
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>Waiting for logs...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
        let logLines = [];
        let state = 'unknown';
        let output = '';
        let error = '';

        function startStream() {
            const primaryURL = `/api/v1/beta/examples/run/${jobId}/logs`;
            const fallbackURL = `/api/v1/educational/examples/run/${jobId}/logs`;
            function openES(url, triedFallback){
                es = new EventSource(url);
                es.onerror = () => {
                    if(!triedFallback){
                        es.close();
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[primary stream failed – trying deprecated path]</span>`;
                        openES(fallbackURL, true);
                    }
                };
            }
            openES(primaryURL, false);
            es.addEventListener('open', () => {
                lastEventTs = Date.now();
                reconnectAttempts = 0;
            });
            es.addEventListener('status', e => {
                lastEventTs = Date.now();
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                    updateSamplesOutput();
                } catch {}
            });
            es.addEventListener('log', e => {
                lastEventTs = Date.now();
                logLines.push(e.data);
                updateSamplesOutput();
            });
            es.addEventListener('done', e => {
                lastEventTs = Date.now();
                finished = true;
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                } catch {}
                updateSamplesOutput();
                es.close();
            });
            es.onerror = () => {
                if (!finished) {
                    es.close();
                    if (reconnectAttempts < maxReconnectAttempts) {
                        reconnectAttempts++;
                        const delay = Math.min(3000, 500 * reconnectAttempts);
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[reconnecting attempt ${reconnectAttempts} in ${delay}ms]</span>`;
                        setTimeout(startStream, delay);
                    } else {
                        samplesOutput.innerHTML += `<br><span class='text-red-400'>[stream error: max retries]</span>`;
                    }
                }
            };
        }
        startStream();

        const watchdog = setInterval(() => {
            if (finished) { clearInterval(watchdog); return; }
            const idle = Date.now() - lastEventTs;
            if (idle > 15000) {
                samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[idle – waiting for events]</span>`;
                lastEventTs = Date.now();
            }
        }, 5000);

        function updateSamplesOutput() {
            let html = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br>`;
            html += `<span class='text-yellow-400'>State: ${escapeHtml(state)}</span><br>`;
            if (logLines.length) {
                html += `<pre class='text-blue-300 whitespace-pre-wrap mt-2'>${escapeHtml(logLines.join('\n'))}</pre>`;
            }
            if (state === 'done') {
                html += `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'failed') {
                html += `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId})</span><br>`;
                if (error) html += `<span class='text-red-300'>Error: ${escapeHtml(error)}</span><br>`;
                if (output) html += `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'timeout') {
                html += `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            }
            html += `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            samplesOutput.innerHTML = html;
        }
    }
    }

    function pollJob(jobId, exampleId, startTs, chainCtx) {
    fetch(`/api/v1/beta/examples/run/${jobId}/status`)
            .then(r => r.json())
            .then(status => {
                if (!status.success) {
                    samplesOutput.innerHTML = `<span class='text-red-400'>Status error: ${escapeHtml(status.message || 'unknown')}</span>`;
                    return;
                }
                const job = status.job;
                const elapsed = ((Date.now() - startTs) / 1000).toFixed(2);
                if (job.state === 'queued' || job.state === 'running') {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>State: ${job.state} (elapsed ${elapsed}s)</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    setTimeout(() => pollJob(jobId, exampleId, startTs, chainCtx), 750);
                    return;
                }
                // Terminal states
                if (job.state === 'done') {
                    samplesOutput.innerHTML = `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId}) in ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '<span class="text-gray-400">(no output)</span>') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else if (job.state === 'failed') {
                    samplesOutput.innerHTML = `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.error ? `<span class='text-red-300'>Error: ${escapeHtml(job.error)}</span><br>` : '') +
                        (job.output ? `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed, error: job.error });
                        runNextInChain(chainCtx); // continue even on failure for educational completeness
                    }
                } else if (job.state === 'timeout') {
                    samplesOutput.innerHTML = `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>Job ${jobId} ended in state ${job.state}</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                }
            })
            .catch(err => {
                samplesOutput.innerHTML = `<span class='text-red-400'>Polling error: ${escapeHtml(err.message)}</span>`;
                if (chainCtx) {
                    chainCtx.results.push({ id: exampleId, state: 'error', error: err.message });
                    runNextInChain(chainCtx);
                }
            });
    }

    function runAllBasics() {
        const chain = ['gauth_protocol_basics:minimal_poa', 'gauth_protocol_basics:delegation', 'gauth_protocol_basics:token'];
        const ctx = { queue: chain, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running basic examples sequentially...</span>`;
        runNextInChain(ctx);
    }

    function runAdvancedSuite() {
        // Use all advanced and negative samples from the loaded catalog
        const advanced = allExamples.filter(ex => ex.isAdvanced);
        const negative = allExamples.filter(ex => ex.hasNegative);
        // Remove duplicates by id
        const ids = Array.from(new Set([...advanced, ...negative].map(ex => ex.id)));
        if (!ids.length) {
            samplesOutput.innerHTML = `<span class='text-yellow-400'>No advanced or negative samples found in catalog.</span>`;
            return;
        }
        const ctx = { queue: ids, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running advanced/negative examples sequentially...</span>`;
        runNextInChain(ctx);
    }
    function runNextInChain(ctx) {
        if (!ctx.queue.length) {
            // render summary
            const summary = ctx.results.map(r => `# ${r.id} -> ${r.state} (${r.elapsed||'-'}s)`).join('\n');
            samplesOutput.innerHTML = `<span class='text-green-400'>Composite run complete.</span><br><pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(summary)}</pre>`;
            // Show export buttons
            const exportDiv = document.getElementById('composite-export-buttons');
            if (exportDiv) {
                exportDiv.classList.remove('hidden');
                // Store summary in exportDiv for download handlers
                exportDiv.dataset.summary = JSON.stringify(ctx.results);
                exportDiv.dataset.summaryText = summary;
            }
            return;
        }
// Composite export download logic
document.addEventListener('DOMContentLoaded', function() {
    const exportDiv = document.getElementById('composite-export-buttons');
    if (!exportDiv) return;
    const btnJson = document.getElementById('download-composite-json');
    const btnCsv = document.getElementById('download-composite-csv');
    // Hide buttons initially
    exportDiv.classList.add('hidden');
    // Download JSON
    btnJson.addEventListener('click', function() {
        const summaryArr = exportDiv.dataset.summary;
        if (!summaryArr) return;
    fetch('/api/v1/beta/examples/composite/export/json', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: summaryArr
        })
        .then(r => {
            if (!r.ok) throw new Error('Export failed');
            return r.blob();
        })
        .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'composite_run_summary.json';
            document.body.appendChild(a);
            a.click();
            setTimeout(() => { document.body.removeChild(a); window.URL.revokeObjectURL(url); }, 100);
        })
        .catch(() => alert('Failed to download JSON export.'));
    });
    // Download CSV
    btnCsv.addEventListener('click', function() {
        const summaryArr = exportDiv.dataset.summary;
        if (!summaryArr) return;
    fetch('/api/v1/beta/examples/composite/export/csv', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: summaryArr
        })
        .then(r => {
            if (!r.ok) throw new Error('Export failed');
            return r.blob();
        })
        .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'composite_run_summary.csv';
            document.body.appendChild(a);
            a.click();
            setTimeout(() => { document.body.removeChild(a); window.URL.revokeObjectURL(url); }, 100);
        })
        .catch(() => alert('Failed to download CSV export.'));
    });
});
        const next = ctx.queue.shift();
    fetch('/api/v1/beta/examples/run', {
            method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: next })
        }).then(r => r.json()).then(data => {
            if (!data.success) {
                ctx.results.push({ id: next, state: 'queue_failed', error: data.message });
                runNextInChain(ctx);
                return;
            }
            pollJob(data.job_id, next, Date.now(), ctx);
        }).catch(err => {
            ctx.results.push({ id: next, state: 'error', error: err.message });
            runNextInChain(ctx);
        });
    }

    function startMetricsPolling() {
        if (!poaMetricsPanel) return;
        poaMetricsPanel.classList.remove('hidden');
        const fetchMetrics = () => {
            fetch('/api/v1/poa/metrics').then(r => r.json()).then(data => {
                if (!data.success) return;
                const m = data.metrics || {};
                metricsElems.req.textContent = m.total_requests;
                metricsElems.ok.textContent = m.total_success;
                metricsElems.jur.textContent = m.rejected_jurisdiction;
                metricsElems.scope.textContent = m.rejected_scope;
                metricsElems.missing.textContent = m.rejected_missing_fields;
                metricsElems.updated.textContent = new Date(data.timestamp).toLocaleTimeString();
                // Render POA duration sparkline
                const svg = document.getElementById('poa-duration-sparkline');
                const lastVal = document.getElementById('poa-duration-last');
                if (svg && m.recent_durations && Array.isArray(m.recent_durations) && m.recent_durations.length > 0) {
                    const values = m.recent_durations;
                    const w = 120, h = 24, pad = 2;
                    const min = Math.min(...values), max = Math.max(...values);
                    const range = max - min || 1;
                    const points = values.map((v, i) => {
                        const x = pad + i * ((w-2*pad)/(values.length-1||1));
                        const y = h - pad - ((v-min)/range)*(h-2*pad);
                        return `${x},${y}`;
                    }).join(' ');
                    svg.innerHTML = `<polyline fill="none" stroke="#2563eb" stroke-width="2" points="${points}" />`;
                    lastVal.textContent = values[values.length-1].toFixed(2);
                } else if (svg) {
                    svg.innerHTML = '';
                    if (lastVal) lastVal.textContent = '';
                }
            }).catch(()=>{});
        };
        fetchMetrics();
        setInterval(fetchMetrics, 4000);
    }

    function escapeHtml(str) {
        return String(str).replace(/[&<>"']/g, function(m) {
            return ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;','\'':'&#39;'})[m];
        });
    }
// END of main samples tab logic block

    // Default tab activation
    const activeBtn = document.querySelector('.tab-button.active[data-tab]') || document.querySelector('.tab-button[data-tab]');
    if (activeBtn) {
        showTab({ target: activeBtn }, activeBtn.getAttribute('data-tab'));
    }

    // Action buttons
    const actionMap = {
        'create-token': createToken,
        'validate-token': validateToken,
        'revoke-token': revokeToken,
        'check-authorization': checkAuthorization,
        'publish-event': publishEvent,
        'subscribe-events': subscribeEvents,
        'view-audit-log': viewAuditLog,
        'generate-report': generateReport
    };
        Object.entries(actionMap).forEach(([key, fn]) => {
            document.querySelectorAll(`[data-action="${key}"]`).forEach(el => {
                el.addEventListener('click', fn);
                bindReport.actions++;
            });
        });
        console.log(`[GAuth Demo] Initial binding complete:`, bindReport);
        if (bindReport.tabs === 0 || bindReport.actions === 0) {
            console.warn('[GAuth Demo] Warning: No bindings detected (tabs or actions). Scheduling retry...');
            setTimeout(rebindDemoHandlers, 500);
        }
    } catch (e) {
        console.error('[GAuth Demo] Initialization error:', e);
        setTimeout(rebindDemoHandlers, 750);
    }
});

// Fallback rebind function (in case of deferred HTML insertion or race conditions)
function rebindDemoHandlers(attempt = 1) {
    const MAX_ATTEMPTS = 5;
    const report = { tabs: 0, actions: 0, attempt };
    document.querySelectorAll('.tab-button[data-tab]').forEach(btn => {
        if (!btn.__gauthBound) {
            btn.addEventListener('click', (e) => {
                const tabId = btn.getAttribute('data-tab');
                showTab(e, tabId);
            });
            btn.__gauthBound = true;
            report.tabs++;
        }
    });
    const actionMap = {
        'create-token': createToken,
        'validate-token': validateToken,
        'revoke-token': revokeToken,
        'check-authorization': checkAuthorization,
        'publish-event': publishEvent,
        'subscribe-events': subscribeEvents,
        'view-audit-log': viewAuditLog,
        'generate-report': generateReport
    };
    Object.entries(actionMap).forEach(([key, fn]) => {
        document.querySelectorAll(`[data-action="${key}"]`).forEach(el => {
            if (!el.__gauthBound) {
                el.addEventListener('click', fn);
                el.__gauthBound = true;
                report.actions++;
            }
        });
    });
    console.log('[GAuth Demo] Rebind attempt', report);
    if ((report.tabs === 0 || report.actions === 0) && attempt < MAX_ATTEMPTS) {
        setTimeout(() => rebindDemoHandlers(attempt + 1), 600);
    } else if (attempt >= MAX_ATTEMPTS && (report.tabs === 0 || report.actions === 0)) {
        console.error('[GAuth Demo] Failed to bind some handlers after retries.');
    }
}

// Navigation
function scrollToDemo() {
    document.getElementById('demo').scrollIntoView({ 
        behavior: 'smooth' 
    });
}

// Token Management Demo Functions
async function createToken() {
    addConsoleOutput('token-output', 'Initiating beta token creation...', 'info');
    try {
        const res = await fetch('/api/v1/token/create', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ttl_seconds: 3600 })
        });
        const data = await res.json();
        if (!data.success) throw new Error(data.message || 'Token creation failed');
        currentToken = data.token;
        addConsoleOutput('token-output', `✓ Token created successfully`, 'success');
        addConsoleOutput('token-output', `  ID: ${currentToken.id}`, 'info');
        addConsoleOutput('token-output', `  Expires: ${currentToken.expiresAt}`, 'info');
        addConsoleOutput('token-output', `  ⚠️ Beta demo token - not cryptographically secure`, 'warning');
        demoState.tokenCreated = true;
        demoState.auditEntries.push({
            timestamp: new Date().toISOString(),
            action: 'TOKEN_CREATED',
            tokenId: currentToken.id,
            beta: true
        });
    } catch (error) {
        addConsoleOutput('token-output', `✗ Token creation failed: ${error.message}`, 'error');
    }
}

async function validateToken() {
    if (!currentToken) {
        addConsoleOutput('token-output', '✗ No token available for validation. Create a token first.', 'error');
        return;
    }
    addConsoleOutput('token-output', 'Validating beta token...', 'info');
    try {
        const res = await fetch('/api/v1/token/validate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token_id: currentToken.id })
        });
        const data = await res.json();
        if (!data.success) {
            addConsoleOutput('token-output', `✗ Token validation failed: ${data.status || data.message}`, 'error');
            if (data.status === 'expired' || data.status === 'revoked') {
                currentToken = null;
                demoState.tokenCreated = false;
            }
            return;
        }
        addConsoleOutput('token-output', '✓ Token validation successful', 'success');
        addConsoleOutput('token-output', `  Valid until: ${data.token.expiresAt}`, 'info');
        demoState.auditEntries.push({
            timestamp: new Date().toISOString(),
            action: 'TOKEN_VALIDATED',
            tokenId: currentToken.id,
            valid: true,
            beta: true
        });
    } catch (error) {
        addConsoleOutput('token-output', `✗ Token validation failed: ${error.message}`, 'error');
    }
}

async function revokeToken() {
    if (!currentToken) {
        addConsoleOutput('token-output', '✗ No token available for revocation.', 'error');
        return;
    }
    addConsoleOutput('token-output', `Revoking token ${currentToken.id}...`, 'info');
    try {
        const res = await fetch('/api/v1/token/revoke', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token_id: currentToken.id })
        });
        const data = await res.json();
        if (!data.success) {
            addConsoleOutput('token-output', `✗ Token revocation failed: ${data.status || data.message}`, 'error');
            return;
        }
        addConsoleOutput('token-output', `✓ Token ${currentToken.id} revoked successfully`, 'success');
        demoState.auditEntries.push({
            timestamp: new Date().toISOString(),
            action: 'TOKEN_REVOKED',
            tokenId: currentToken.id,
            beta: true
        });
        currentToken = null;
        demoState.tokenCreated = false;
    } catch (error) {
        addConsoleOutput('token-output', `✗ Token revocation failed: ${error.message}`, 'error');
    }
}

// Token Metrics Panel
async function fetchTokenMetrics() {
    try {
        const res = await fetch('/api/v1/token/metrics');
        const data = await res.json();
        if (!data.success) throw new Error('Failed to fetch token metrics');
        const m = data.metrics;
        const panel = document.getElementById('token-metrics-panel');
        if (panel) {
            panel.innerHTML = `<div class='text-xs text-gray-700'>
                <b>Created:</b> ${m.created} &nbsp; <b>Validated:</b> ${m.validated} &nbsp; <b>Revoked:</b> ${m.revoked} &nbsp; <b>Total:</b> ${m.total}
            </div>`;
        }
    } catch (e) {
        const panel = document.getElementById('token-metrics-panel');
        if (panel) panel.innerHTML = `<span class='text-red-500'>Failed to load token metrics</span>`;
    }
}

// Optionally, add event listeners for a metrics button
// Tab system
function showTab(e, tabId) {
    // Hide all tab contents
    const tabContents = document.querySelectorAll('.tab-content');
    tabContents.forEach(content => {
        content.style.display = 'none';
        content.classList.remove('active');
    });
    
    // Remove active class from all tab buttons
    const tabButtons = document.querySelectorAll('.tab-button');
    tabButtons.forEach(button => button.classList.remove('active'));
    
    // Show selected tab content
    const selectedTab = document.getElementById(tabId);
    if (selectedTab) {
        selectedTab.style.display = 'block';
        selectedTab.classList.add('active');
    }
    
    // Add active class to clicked button
    if (e && e.target) {
        e.target.classList.add('active');
    }
}

// DOM bindings without inline handlers (CSP friendly)
document.addEventListener('DOMContentLoaded', () => {
    try {
        console.log('[GAuth Demo] DOMContentLoaded fired - initializing bindings');
        const bindReport = { tabs: 0, actions: 0 };
    // Tab buttons
        document.querySelectorAll('.tab-button[data-tab]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tabId = btn.getAttribute('data-tab');
                showTab(e, tabId);
            });
            bindReport.tabs++;
        });

        // Samples tab logic
        initSamplesTab();
// --- Samples Tab Logic ---
function initSamplesTab() {
    const runAdvancedSuiteBtn = document.getElementById('run-advanced-suite');
    const samplesList = document.getElementById('samples-list');
    const samplesSearch = document.getElementById('samples-search');
    const samplesOutput = document.getElementById('samples-output');
    let allExamples = [];
    let filteredExamples = [];
    const filterAdvanced = document.getElementById('filter-advanced');
    const filterNegative = document.getElementById('filter-negative');
    const filterBasics = document.getElementById('filter-basics');
    const runAllBasicsBtn = document.getElementById('run-all-basics');
    const poaMetricsPanel = document.getElementById('poa-metrics');
    const metricsElems = {
        req: document.getElementById('m-poa-req'),
        ok: document.getElementById('m-poa-ok'),
        jur: document.getElementById('m-poa-jur'),
        scope: document.getElementById('m-poa-scope'),
        missing: document.getElementById('m-poa-missing'),
        updated: document.getElementById('poa-metrics-updated')
    };

    if (!samplesList || !samplesSearch || !samplesOutput) return;

    // Fetch catalog
    fetch('/api/v1/beta/examples/catalog')
        .then(res => res.json())
        .then(data => {
            allExamples = (data.examples || []).map(e => enrichExample(e));
            filteredExamples = allExamples;
            renderSamplesList(filteredExamples);
            startMetricsPolling();
        })
        .catch(() => {
            samplesList.innerHTML = '<div class="text-red-500 p-4">Failed to load examples.</div>';
        });

    // Search filter
    function applyFilters() {
        const q = samplesSearch.value.toLowerCase();
        filteredExamples = allExamples.filter(ex => {
            if (q && !(ex.id.toLowerCase().includes(q) || (ex.title||'').toLowerCase().includes(q) || (ex.description||'').toLowerCase().includes(q))) return false;
            const isAdv = ex.isAdvanced;
            const hasNeg = ex.hasNegative;
            const isBasic = !isAdv && !hasNeg;
            if (!filterAdvanced.checked && isAdv) return false;
            if (!filterNegative.checked && hasNeg) return false;
            if (!filterBasics.checked && isBasic) return false;
            return true;
        });
        renderSamplesList(filteredExamples);
    }
    samplesSearch.addEventListener('input', applyFilters);
    [filterAdvanced, filterNegative, filterBasics].forEach(cb => cb && cb.addEventListener('change', applyFilters));

    function enrichExample(ex) {
        const title = (ex.title || '').toLowerCase();
        const desc = (ex.description || '').toLowerCase();
        ex.isAdvanced = /advanced/.test(title) || /advanced_poa/.test(ex.id);
        ex.hasNegative = /(negative|invalid|disallowed|missing)/.test(desc);
        return ex;
    }

    if (runAdvancedSuiteBtn) {
        runAdvancedSuiteBtn.addEventListener('click', () => runAdvancedSuite());
    }

    if (runAllBasicsBtn) {
        runAllBasicsBtn.addEventListener('click', () => runAllBasics());
    }

    function renderSamplesList(examples) {
        if (!examples.length) {
            samplesList.innerHTML = '<div class="text-gray-400 p-4">No examples found.</div>';
            return;
        }
        // Sort: featured categories first (gauth_protocol_basics), then alphabetical
        const featuredPrefix = 'gauth_protocol_basics';
        const sorted = [...examples].sort((a,b) => {
            const aFeat = a.id.startsWith(featuredPrefix) ? 0 : 1;
            const bFeat = b.id.startsWith(featuredPrefix) ? 0 : 1;
            if (aFeat !== bFeat) return aFeat - bFeat;
            return (a.title || a.id).localeCompare(b.title || b.id);
        });
        samplesList.innerHTML = sorted.map(ex => {
            const isAdvanced = /advanced/i.test(ex.title || '') || /advanced_poa/i.test(ex.id);
            const hasNegatives = /negative|invalid|disallowed|missing/i.test(ex.description || '');
            const badges = [
                isAdvanced ? '<span class="ml-2 inline-block bg-purple-100 text-purple-700 text-[10px] px-2 py-0.5 rounded">ADV</span>' : '',
                hasNegatives ? '<span class="ml-1 inline-block bg-red-100 text-red-700 text-[10px] px-2 py-0.5 rounded" title="Contains negative validation cases">NEG</span>' : ''
            ].join('');
            return `
            <div class="group border-b border-gray-100 px-4 py-2 hover:bg-gray-50 transition">
                <div class="flex items-start justify-between gap-4">
                    <div class="flex-1 min-w-0">
                        <div class="font-semibold text-gray-800 flex items-center">${escapeHtml(ex.title || ex.id)} ${badges}</div>
                        <div class="text-xs text-gray-500 truncate" title="${escapeHtml(ex.description || ex.id)}">${escapeHtml(ex.description || ex.id)}</div>
                        <div class="text-[10px] text-gray-400 mt-0.5">ID: ${escapeHtml(ex.id)}</div>
                    </div>
                    <div class="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition">
                        <button class="run-sample-btn bg-blue-600 hover:bg-blue-700 text-white font-semibold text-xs py-1 px-3 rounded shadow" data-sample-id="${ex.id}"><i class="fas fa-play mr-1"></i>Run</button>
                    </div>
                </div>
            </div>`;
        }).join('');
        samplesList.querySelectorAll('.run-sample-btn').forEach(btn => {
            btn.addEventListener('click', () => runSample(btn.getAttribute('data-sample-id')));
        });
    }

    function runSample(id) {
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Queued ${id}]</span><br><span class='text-yellow-400'>Submitting job...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
    fetch('/api/v1/beta/examples/run', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id })
        })
        .then(res => res.json())
        .then(data => {
            if (!data.success) {
                samplesOutput.innerHTML = `<span class='text-red-400'>Failed to queue job: ${escapeHtml(data.message || 'unknown error')}</span>`;
                return;
            }
            const jobId = data.job_id;
            samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} queued for ${id}]</span><br><span class='text-yellow-400'>State: ${data.state}</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            attachLogStream(jobId, id);
        })
        .catch(err => {
            samplesOutput.innerHTML = `<span class='text-red-400'>Error queuing job: ${escapeHtml(err.message)}`;
        });
    }

    // Attach SSE log stream for job (with reconnect + idle watchdog)
    function attachLogStream(jobId, exampleId) {
        let finished = false;
        let lastEventTs = Date.now();
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 5;
        let es;
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>Waiting for logs...</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
        let logLines = [];
        let state = 'unknown';
        let output = '';
        let error = '';

        function startStream() {
            const primaryURL = `/api/v1/beta/examples/run/${jobId}/logs`;
            const fallbackURL = `/api/v1/educational/examples/run/${jobId}/logs`;
            function openES(url, triedFallback){
                es = new EventSource(url);
                es.onerror = () => {
                    if(!triedFallback){
                        es.close();
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[primary stream failed – trying deprecated path]</span>`;
                        openES(fallbackURL, true);
                    }
                };
            }
            openES(primaryURL, false);
            es.addEventListener('open', () => {
                lastEventTs = Date.now();
                reconnectAttempts = 0;
            });
            es.addEventListener('status', e => {
                lastEventTs = Date.now();
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                    updateSamplesOutput();
                } catch {}
            });
            es.addEventListener('log', e => {
                lastEventTs = Date.now();
                logLines.push(e.data);
                updateSamplesOutput();
            });
            es.addEventListener('done', e => {
                lastEventTs = Date.now();
                finished = true;
                try {
                    const s = JSON.parse(e.data);
                    state = s.state;
                    output = s.output || '';
                    error = s.error || '';
                } catch {}
                updateSamplesOutput();
                es.close();
            });
            es.onerror = () => {
                if (!finished) {
                    es.close();
                    if (reconnectAttempts < maxReconnectAttempts) {
                        reconnectAttempts++;
                        const delay = Math.min(3000, 500 * reconnectAttempts);
                        samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[reconnecting attempt ${reconnectAttempts} in ${delay}ms]</span>`;
                        setTimeout(startStream, delay);
                    } else {
                        samplesOutput.innerHTML += `<br><span class='text-red-400'>[stream error: max retries]</span>`;
                    }
                }
            };
        }
        startStream();

        const watchdog = setInterval(() => {
            if (finished) { clearInterval(watchdog); return; }
            const idle = Date.now() - lastEventTs;
            if (idle > 15000) {
                samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[idle – waiting for events]</span>`;
                lastEventTs = Date.now();
            }
        }, 5000);

        function updateSamplesOutput() {
            let html = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br>`;
            html += `<span class='text-yellow-400'>State: ${escapeHtml(state)}</span><br>`;
            if (logLines.length) {
                html += `<pre class='text-blue-300 whitespace-pre-wrap mt-2'>${escapeHtml(logLines.join('\n'))}</pre>`;
            }
            if (state === 'done') {
                html += `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'failed') {
                html += `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId})</span><br>`;
                if (error) html += `<span class='text-red-300'>Error: ${escapeHtml(error)}</span><br>`;
                if (output) html += `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            } else if (state === 'timeout') {
                html += `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId})</span><br>`;
                if (output) html += `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(output)}</pre>`;
            }
            html += `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
            samplesOutput.innerHTML = html;
        }
    }
    }

    function pollJob(jobId, exampleId, startTs, chainCtx) {
    fetch(`/api/v1/beta/examples/run/${jobId}/status`)
            .then(r => r.json())
            .then(status => {
                if (!status.success) {
                    samplesOutput.innerHTML = `<span class='text-red-400'>Status error: ${escapeHtml(status.message || 'unknown')}</span>`;
                    return;
                }
                const job = status.job;
                const elapsed = ((Date.now() - startTs) / 1000).toFixed(2);
                if (job.state === 'queued' || job.state === 'running') {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>[Job ${jobId} for ${exampleId}]</span><br><span class='text-yellow-400'>State: ${job.state} (elapsed ${elapsed}s)</span><br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    setTimeout(() => pollJob(jobId, exampleId, startTs, chainCtx), 750);
                    return;
                }
                // Terminal states
                if (job.state === 'done') {
                    samplesOutput.innerHTML = `<span class='text-green-400'>✓ Completed ${exampleId} (job ${jobId}) in ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '<span class="text-gray-400">(no output)</span>') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else if (job.state === 'failed') {
                    samplesOutput.innerHTML = `<span class='text-red-400'>✗ Failed ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.error ? `<span class='text-red-300'>Error: ${escapeHtml(job.error)}</span><br>` : '') +
                        (job.output ? `<pre class='text-red-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed, error: job.error });
                        runNextInChain(chainCtx); // continue even on failure for educational completeness
                    }
                } else if (job.state === 'timeout') {
                    samplesOutput.innerHTML = `<span class='text-orange-400'>⌛ Timeout ${exampleId} (job ${jobId}) after ${elapsed}s</span><br>` +
                        (job.output ? `<pre class='text-yellow-200 whitespace-pre-wrap mt-2'>${escapeHtml(job.output)}</pre>` : '') +
                        `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, output: job.output, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                } else {
                    samplesOutput.innerHTML = `<span class='text-gray-400'>Job ${jobId} ended in state ${job.state}</span>`;
                    if (chainCtx) {
                        chainCtx.results.push({ id: exampleId, state: job.state, elapsed });
                        runNextInChain(chainCtx);
                    }
                }
            })
            .catch(err => {
                samplesOutput.innerHTML = `<span class='text-red-400'>Polling error: ${escapeHtml(err.message)}</span>`;
                if (chainCtx) {
                    chainCtx.results.push({ id: exampleId, state: 'error', error: err.message });
                    runNextInChain(chainCtx);
                }
            });
    }

    function runAllBasics() {
        const chain = ['gauth_protocol_basics:minimal_poa', 'gauth_protocol_basics:delegation', 'gauth_protocol_basics:token'];
        const ctx = { queue: chain, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running basic examples sequentially...</span>`;
        runNextInChain(ctx);
    }

    function runAdvancedSuite() {
        // Use all advanced and negative samples from the loaded catalog
        const advanced = allExamples.filter(ex => ex.isAdvanced);
        const negative = allExamples.filter(ex => ex.hasNegative);
        // Remove duplicates by id
        const ids = Array.from(new Set([...advanced, ...negative].map(ex => ex.id)));
        if (!ids.length) {
            samplesOutput.innerHTML = `<span class='text-yellow-400'>No advanced or negative samples found in catalog.</span>`;
            return;
        }
        const ctx = { queue: ids, results: [] };
        samplesOutput.innerHTML = `<span class='text-gray-400'>[Composite] Running advanced/negative examples sequentially...</span>`;
        runNextInChain(ctx);
    }
    function runNextInChain(ctx) {
        if (!ctx.queue.length) {
            // render summary
            const summary = ctx.results.map(r => `# ${r.id} -> ${r.state} (${r.elapsed||'-'}s)`).join('\n');
            samplesOutput.innerHTML = `<span class='text-green-400'>Composite run complete.</span><br><pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(summary)}</pre>`;
            // Show export buttons
            const exportDiv = document.getElementById('composite-export-buttons');
            if (exportDiv) {
                exportDiv.classList.remove('hidden');
                // Store summary in exportDiv for download handlers
                exportDiv.dataset.summary = JSON.stringify(ctx.results);
                exportDiv.dataset.summaryText = summary;
            }
            return;
        }
        const next = ctx.queue.shift();
    fetch('/api/v1/beta/examples/run', {
            method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: next })
        }).then(r => r.json()).then(data => {
            if (!data.success) {
                ctx.results.push({ id: next, state: 'queue_failed', error: data.message });
                runNextInChain(ctx);
                return;
            }
            pollJob(data.job_id, next, Date.now(), ctx);
        }).catch(err => {
            ctx.results.push({ id: next, state: 'error', error: err.message });
            runNextInChain(ctx);
        });
    }

    // Event and Audit Stream Control
    const startEventBtn = document.getElementById('startEventStream');
    const stopEventBtn = document.getElementById('stopEventStream');
    if (startEventBtn && stopEventBtn) {
        startEventBtn.addEventListener('click', function() {
            if (eventStream) return;
            eventStream = new EventSource('/api/v1/events/stream');
            addConsoleOutput('event-output', 'Started event stream (SSE)', 'info');
            startEventBtn.disabled = true;
            stopEventBtn.disabled = false;
            eventStream.onmessage = function(e) {
                addConsoleOutput('event-output', `[event] ${e.data}`, 'info');
            };
            eventStream.onerror = function() {
                addConsoleOutput('event-output', 'Event stream error or closed', 'error');
                eventStream.close();
                eventStream = null;
                startEventBtn.disabled = false;
                stopEventBtn.disabled = true;
            };
        });
        stopAuditBtn.addEventListener('click', function() {
            if (auditStream) {
                auditStream.close();
                auditStream = null;
                addConsoleOutput('audit-output', 'Stopped audit stream', 'warning');
                startAuditBtn.disabled = false;
                stopAuditBtn.disabled = true;
            }
        });
    }
    // ...existing code...
// Removed stray closing parenthesis and brace that broke the script
        startAuditBtn.addEventListener('click', function() {
            if (auditStream) return;
            auditStream = new EventSource('/api/v1/audit/stream');
            addConsoleOutput('audit-output', 'Started audit stream (SSE)', 'info');
            startAuditBtn.disabled = true;
            stopAuditBtn.disabled = false;
            auditStream.onmessage = function(e) {
                addConsoleOutput('audit-output', `[audit] ${e.data}`, 'info');
            };
            auditStream.onerror = function() {
                addConsoleOutput('audit-output', 'Audit stream error or closed', 'error');
                auditStream.close();
                auditStream = null;
                startAuditBtn.disabled = false;
                stopAuditBtn.disabled = true;
            };
        });
        stopAuditBtn.addEventListener('click', function() {
            if (auditStream) {
                auditStream.close();
                auditStream = null;
                addConsoleOutput('audit-output', 'Stopped audit stream', 'warning');
                startAuditBtn.disabled = false;
                stopAuditBtn.disabled = true;
            }
        });
    }
    // ...existing code...
// ...existing code...

// Authorization Demo Functions
async function checkAuthorization() {
    addConsoleOutput('authz-output', 'Checking POA authorization...', 'info');
    try {
        const clientId = document.getElementById('client-id-input')?.value || 'demo-client-' + generateRandomId();
        const res = await fetch('/api/v1/poa/authorize', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ client_id: clientId })
        });
        const data = await res.json();
        if (!data.success) {
            addConsoleOutput('authz-output', `✗ Authorization failed: ${data.message || 'unknown error'}`, 'error');
            return;
        }
        addConsoleOutput('authz-output', '✓ POA authorization successful', 'success');
        addConsoleOutput('authz-output', `  Client ID: ${clientId}`, 'info');
        demoState.auditEntries.push({
            timestamp: new Date().toISOString(),
            action: 'POA_AUTHORIZED',
            clientId: clientId,
            beta: true
        });
    } catch (error) {
        addConsoleOutput('authz-output', `✗ Authorization failed: ${error.message}`, 'error');
    }
}

// Event Demo Functions
async function publishEvent() {
    addConsoleOutput('event-output', 'Publishing demo event...', 'info');
    try {
        const eventType = document.getElementById('event-type-input')?.value || 'demo_event';
        const eventData = document.getElementById('event-data-input')?.value || { message: 'Demo event from web interface', timestamp: new Date().toISOString() };
        
        let data;
        try {
            data = typeof eventData === 'string' ? JSON.parse(eventData) : eventData;
        } catch {
            data = eventData;
        }
        
        const res = await fetch('/api/v1/events/emit', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ type: eventType, data: data })
        });
        const response = await res.json();
        if (!response.success) {
            addConsoleOutput('event-output', `✗ Event publish failed: ${response.message || 'unknown error'}`, 'error');
            return;
        }
        addConsoleOutput('event-output', '✓ Event published successfully', 'success');
        addConsoleOutput('event-output', `  Type: ${eventType}`, 'info');
        addConsoleOutput('event-output', `  Event ID: ${response.event?.id}`, 'info');
        demoState.auditEntries.push({
            timestamp: new Date().toISOString(),
            action: 'EVENT_PUBLISHED',
            eventType: eventType,
            eventId: response.event?.id,
            beta: true
        });
    } catch (error) {
        addConsoleOutput('event-output', `✗ Event publish failed: ${error.message}`, 'error');
    }
}

async function subscribeEvents() {
    if (demoState.subscriptionsActive) {
        addConsoleOutput('event-output', 'Event subscription already active', 'warning');
        return;
    }
    
    addConsoleOutput('event-output', 'Starting event stream subscription...', 'info');
    try {
        const eventSource = new EventSource('/api/v1/events/stream');
        
        eventSource.onopen = function() {
            addConsoleOutput('event-output', '✓ Event stream connected', 'success');
            demoState.subscriptionsActive = true;
        };
        
        eventSource.onmessage = function(event) {
            try {
                const data = JSON.parse(event.data);
                if (data.ok) {
                    addConsoleOutput('event-output', 'Event stream initialized', 'info');
                } else {
                    addConsoleOutput('event-output', `[${data.type || 'event'}] ${JSON.stringify(data)}`, 'info');
                }
            } catch (e) {
                addConsoleOutput('event-output', `[event] ${event.data}`, 'info');
            }
        };
        
        eventSource.onerror = function() {
            addConsoleOutput('event-output', 'Event stream error - attempting reconnection...', 'error');
            demoState.subscriptionsActive = false;
            // Auto-reconnect after a delay
            setTimeout(() => {
                if (!demoState.subscriptionsActive) {
                    subscribeEvents();
                }
            }, 3000);
        };
        
        // Store reference for cleanup
        window.currentEventSource = eventSource;
        
    } catch (error) {
        addConsoleOutput('event-output', `✗ Failed to start event subscription: ${error.message}`, 'error');
    }
}

// --- Move audit demo functions to top so they are defined before use in actionMap ---
async function viewAuditLog() {
    addConsoleOutput('audit-output', 'Fetching audit logs...', 'info');
    try {
        const limit = document.getElementById('audit-limit-input')?.value || 50;
        const res = await fetch(`/api/v1/audit/logs?limit=${limit}`);
        const data = await res.json();
        if (!data.success) {
            addConsoleOutput('audit-output', `✗ Failed to fetch audit logs: ${data.message || 'unknown error'}`, 'error');
            return;
        }
        addConsoleOutput('audit-output', `✓ Retrieved ${data.count} audit entries`, 'success');
        if (data.entries && data.entries.length > 0) {
            data.entries.forEach(entry => {
                const time = new Date(entry.at).toLocaleTimeString();
                addConsoleOutput('audit-output', `[${time}] ${entry.action} by ${entry.actor} on ${entry.resource} - ${entry.outcome}`, 'info');
            });
        } else {
            addConsoleOutput('audit-output', 'No audit entries found', 'info');
        }
        demoState.auditEntries = data.entries || [];
    } catch (error) {
        addConsoleOutput('audit-output', `✗ Failed to fetch audit logs: ${error.message}`, 'error');
    }
}

async function generateReport() {
    addConsoleOutput('audit-output', 'Generating audit report...', 'info');
    try {
        // First fetch audit logs if we don't have them
        if (!demoState.auditEntries.length) {
            await viewAuditLog();
            // Wait a bit for the fetch to complete
            await new Promise(resolve => setTimeout(resolve, 500));
        }
        
        if (!demoState.auditEntries.length) {
            addConsoleOutput('audit-output', 'No audit data available for report generation', 'warning');
            return;
        }
        
        // Generate summary statistics
        const stats = {
            total: demoState.auditEntries.length,
            byAction: {},
            byOutcome: {},
            timeRange: {
                start: null,
                end: null
            }
        };
        
        demoState.auditEntries.forEach(entry => {
            // Count by action
            stats.byAction[entry.action] = (stats.byAction[entry.action] || 0) + 1;
            
            // Count by outcome
            stats.byOutcome[entry.outcome] = (stats.byOutcome[entry.outcome] || 0) + 1;
            
            // Track time range
            const entryTime = new Date(entry.at);
            if (!stats.timeRange.start || entryTime < stats.timeRange.start) {
                stats.timeRange.start = entryTime;
            }
            if (!stats.timeRange.end || entryTime > stats.timeRange.end) {
                stats.timeRange.end = entryTime;
            }
        });
        
        addConsoleOutput('audit-output', '=== AUDIT REPORT ===', 'success');
        addConsoleOutput('audit-output', `Total Entries: ${stats.total}`, 'info');
        addConsoleOutput('audit-output', `Time Range: ${stats.timeRange.start?.toLocaleString()} - ${stats.timeRange.end?.toLocaleString()}`, 'info');
        
        addConsoleOutput('audit-output', 'Actions:', 'info');
        Object.entries(stats.byAction).forEach(([action, count]) => {
            addConsoleOutput('audit-output', `  ${action}: ${count}`, 'info');
        });
        
        addConsoleOutput('audit-output', 'Outcomes:', 'info');
        Object.entries(stats.byOutcome).forEach(([outcome, count]) => {
            addConsoleOutput('audit-output', `  ${outcome}: ${count}`, 'info');
        });
        
        addConsoleOutput('audit-output', '=== END REPORT ===', 'success');
        
        // Store report in demo state
        demoState.lastReport = {
            generatedAt: new Date().toISOString(),
            stats: stats,
            entries: demoState.auditEntries
        };
        
    } catch (error) {
        addConsoleOutput('audit-output', `✗ Report generation failed: ${error.message}`, 'error');
    }
}