// joblogs.js - Live Job Log Streaming (SSE) and reusable attachLogStream for samples
import { escapeHtml } from './console.js';

let currentLogES = null;
let currentLogJobId = null;
// Cache last N log lines per job (shared with samples & manual streaming)
const JOB_LOG_CACHE = new Map(); // jobId -> { lines:[], state:'', output:'', error:'' }
const MAX_CACHE_LINES = 500;

export function attachLogStream(jobId, exampleId) {
  const samplesOutput = document.getElementById('samples-output');
  if (!samplesOutput) return;
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

  function startStream() {
    const primaryURL = `/api/v1/beta/examples/run/${jobId}/logs`;
    const fallbackURL = `/api/v1/educational/examples/run/${jobId}/logs`;
    function openES(url, triedFallback) {
      es = new EventSource(url);
      es.onerror = () => {
        if (!triedFallback) {
          es.close();
          samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[primary stream failed – trying deprecated path]</span>`;
          openES(fallbackURL, true);
        }
      };
    }
    openES(primaryURL, false);
    es.addEventListener('open', () => { lastEventTs = Date.now(); reconnectAttempts = 0; });
  es.addEventListener('status', e => { lastEventTs = Date.now(); try { const s = JSON.parse(e.data); state = s.state; output = s.output || ''; error = s.error || ''; persist(); updateSamplesOutput(); } catch {} });
  es.addEventListener('log', e => { lastEventTs = Date.now(); logLines.push(e.data); if (logLines.length > MAX_CACHE_LINES) logLines.shift(); persist(); updateSamplesOutput(); });
  es.addEventListener('done', e => { lastEventTs = Date.now(); finished = true; try { const s = JSON.parse(e.data); state = s.state; output = s.output || ''; error = s.error || ''; } catch {} persist(); updateSamplesOutput(); es.close(); });
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
  function persist(){
    JOB_LOG_CACHE.set(jobId, { lines: [...logLines], state, output, error });
  }
  // Restore cached lines if we had prior data (e.g., re-attaching)
  if (JOB_LOG_CACHE.has(jobId)) {
    const cached = JOB_LOG_CACHE.get(jobId);
    logLines = [...cached.lines]; state = cached.state; output = cached.output; error = cached.error; updateSamplesOutput();
  }
  startStream();
  const watchdog = setInterval(() => { if (finished) { clearInterval(watchdog); return; } const idle = Date.now() - lastEventTs; if (idle > 15000) { samplesOutput.innerHTML += `<br><span class='text-yellow-400'>[idle – waiting for events]</span>`; lastEventTs = Date.now(); } }, 5000);
}

export function cancelSampleStream(){
  // Gracefully close sample SSE (only within samples panel context)
  try { if (currentLogES) { currentLogES.close(); } } catch {}
}

export function restoreCachedJob(jobId){
  const samplesOutput = document.getElementById('samples-output');
  if (!samplesOutput) return;
  if (JOB_LOG_CACHE.has(jobId)) {
    const c = JOB_LOG_CACHE.get(jobId);
    const lines = c.lines || [];
    const state = c.state || 'unknown';
    let html = `<span class='text-gray-400'>[Restored Job ${jobId}]</span><br>`;
    html += `<span class='text-yellow-400'>State: ${escapeHtml(state)}</span><br>`;
    if (lines.length) html += `<pre class='text-blue-300 whitespace-pre-wrap mt-2'>${escapeHtml(lines.join('\n'))}</pre>`;
    if (c.output) html += `<pre class='text-green-300 whitespace-pre-wrap mt-2'>${escapeHtml(c.output)}</pre>`;
    if (c.error) html += `<span class='text-red-300'>Error: ${escapeHtml(c.error)}</span>`;
    html += `<br><span class='text-blue-400'>gauth-samples></span> <span class='blinking-cursor'>_</span>`;
    samplesOutput.innerHTML = html;
  }
}

export function startManualJobStream(jobId) {
  stopManualJobStream();
  const statusEl = document.getElementById('logStatus');
  const outputEl = document.getElementById('logOutput');
  if (!jobId || !statusEl || !outputEl) return;
  statusEl.textContent = 'Connecting to job log stream...';
  outputEl.textContent = '';
  const primaryURL = `/api/v1/beta/examples/run/${jobId}/logs`;
  const fallbackURL = `/api/v1/educational/examples/run/${jobId}/logs`;
  let triedFallback = false;
  const es = new EventSource(primaryURL);
  currentLogES = es; currentLogJobId = jobId;
  es.addEventListener('open', () => { statusEl.textContent = `Connected to job ${jobId}`; });
  es.addEventListener('status', e => { try { const s = JSON.parse(e.data); statusEl.textContent = `State: ${s.state}`; cacheManual(s); } catch {} });
  es.addEventListener('log', e => { appendLine(e.data); });
  es.addEventListener('done', e => { try { const s = JSON.parse(e.data); statusEl.textContent = `Completed with state: ${s.state}`; cacheManual(s); } catch { statusEl.textContent = 'Completed'; } es.close(); });
  es.onerror = () => {
    es.close();
    if (!triedFallback) {
      triedFallback = true;
      statusEl.textContent = 'Primary stream failed, trying fallback...';
      const es2 = new EventSource(fallbackURL);
      currentLogES = es2;
      es2.addEventListener('open', () => { statusEl.textContent = `Connected (fallback) to job ${jobId}`; });
  es2.addEventListener('log', e => { appendLine(e.data); });
  es2.addEventListener('done', () => { statusEl.textContent += ' (done)'; es2.close(); });
  function appendLine(line){ outputEl.textContent += line + '\n'; outputEl.scrollTop = outputEl.scrollHeight; let entry = JOB_LOG_CACHE.get(jobId) || { lines:[], state:'', output:'', error:'' }; entry.lines.push(line); if (entry.lines.length>MAX_CACHE_LINES) entry.lines.shift(); JOB_LOG_CACHE.set(jobId, entry); }
  function cacheManual(s){ let entry = JOB_LOG_CACHE.get(jobId) || { lines:[], state:'', output:'', error:'' }; entry.state = s.state; entry.output = s.output || entry.output; entry.error = s.error || entry.error; JOB_LOG_CACHE.set(jobId, entry); }
  // Restore cached lines if exist
  if (JOB_LOG_CACHE.has(jobId)) { const c = JOB_LOG_CACHE.get(jobId); outputEl.textContent = c.lines.join('\n') + (c.lines.length?'\n':''); }
      es2.onerror = () => { statusEl.textContent = 'Stream error (fallback).'; es2.close(); };
    } else {
      statusEl.textContent = 'Stream error.';
    }
  };
}

export function stopManualJobStream() {
  if (currentLogES) { currentLogES.close(); currentLogES = null; }
  const statusEl = document.getElementById('logStatus');
  if (statusEl && currentLogJobId) statusEl.textContent = `Stopped stream for job ${currentLogJobId}`;
  currentLogJobId = null;
}

export function jobLogsInit() {
  const startBtn = document.getElementById('startLogStream');
  const stopBtn = document.getElementById('stopLogStream');
  if (startBtn) {
    startBtn.addEventListener('click', () => {
      const jobId = document.getElementById('logJobId')?.value?.trim();
      if (!jobId) { const s = document.getElementById('logStatus'); if (s) s.textContent = 'Enter a job ID first.'; return; }
      startManualJobStream(jobId);
      startBtn.disabled = true; if (stopBtn) stopBtn.disabled = false;
    });
  }
  if (stopBtn) {
    stopBtn.addEventListener('click', () => {
      stopManualJobStream();
      stopBtn.disabled = true; if (startBtn) startBtn.disabled = false;
    });
  }
}

window.GAuth = window.GAuth || {}; Object.assign(window.GAuth, { startManualJobStream, stopManualJobStream, attachLogStream, cancelSampleStream, restoreCachedJob });