
// metrics.js - metrics polling and sparkline rendering
import { escapeHtml } from './console.js';
export function startMetricsPolling() {
	const poaMetricsPanel = document.getElementById("poa-metrics");
	if (!poaMetricsPanel) return;
	poaMetricsPanel.classList.remove("hidden");
	const metricsElems = {
		req: document.getElementById("m-poa-req"),
		ok: document.getElementById("m-poa-ok"),
		jur: document.getElementById("m-poa-jur"),
		scope: document.getElementById("m-poa-scope"),
		missing: document.getElementById("m-poa-missing"),
		updated: document.getElementById("poa-metrics-updated")
	};
	const annc = document.getElementById('authz-metrics-annc');
	function fetchMetrics() {
		fetch("/api/v1/poa/metrics").then(r => r.json()).then(data => {
			if (!data.success) return;
			const m = data.metrics || {};
			metricsElems.req.textContent = m.total_requests;
			metricsElems.ok.textContent = m.total_success;
			metricsElems.jur.textContent = m.rejected_jurisdiction;
			metricsElems.scope.textContent = m.rejected_scope;
			metricsElems.missing.textContent = m.rejected_missing_fields;
			const ts = new Date(data.timestamp).toLocaleTimeString();
			metricsElems.updated.textContent = ts;
			if(annc) annc.textContent = `POA metrics updated ${ts}`;
			// Render POA duration sparkline
			const svg = document.getElementById("poa-duration-sparkline");
			const lastVal = document.getElementById("poa-duration-last");
			if (svg && m.recent_durations && Array.isArray(m.recent_durations) && m.recent_durations.length > 0) {
				const values = m.recent_durations;
				const w = 120, h = 24, pad = 2;
				const min = Math.min(...values), max = Math.max(...values);
				const range = max - min || 1;
				const points = values.map((v, i) => {
					const x = pad + i * ((w-2*pad)/(values.length-1||1));
					const y = h - pad - ((v-min)/range)*(h-2*pad);
					return `${x},${y}`;
				}).join(" ");
				svg.innerHTML = `<polyline fill="none" stroke="#2563eb" stroke-width="2" points="${points}" />`;
				lastVal.textContent = values[values.length-1].toFixed(2);
			} else if (svg) {
				svg.innerHTML = "";
				if (lastVal) lastVal.textContent = "";
			}
		}).catch(()=>{});
	}
	fetchMetrics();
	setInterval(fetchMetrics, 4000);
}

export function metricsInit() {
	startMetricsPolling();
	// Wire token metrics button (idempotent) and add lazy first-load
	const btn = document.getElementById("show-token-metrics");
	if (btn && !btn.__gauthBound) {
		btn.__gauthBound = true;
		btn.addEventListener("click", () => {
			console.debug('[metrics] token metrics button clicked');
			fetchTokenMetrics();
		});
		console.debug('[metrics] token metrics button bound');
	}

	// Authorization metrics polling
	startAuthzMetricsPolling();
	// Policy metrics polling (beta)
	startPolicyMetricsPolling();
	bindAuthzEvalForm();
}

// Token metrics fetch and render
export async function fetchTokenMetrics() {
	const panel = document.getElementById("token-metrics-panel");
	if (panel) panel.innerHTML = `<span class='text-gray-400'>Loading token metrics...</span>`;
	try {
		const res = await fetch("/api/v1/token/metrics");
		const data = await res.json();
		if (!data.success) throw new Error("Failed to fetch token metrics");
		const m = data.metrics;
		if (panel) {
			panel.innerHTML = `<div class='text-xs text-gray-700'>
				<b>Created:</b> ${m.created} &nbsp; <b>Validated:</b> ${m.validated} &nbsp; <b>Revoked:</b> ${m.revoked} &nbsp; <b>Total:</b> ${m.total}
			</div>`;
		}
	} catch (e) {
		console.warn('[metrics] failed to load token metrics', e);
		if (panel) panel.innerHTML = `<span class='text-red-500'>Failed to load token metrics</span>`;
	}
}

// Legacy global bridge (for compatibility)
window.AgentAuth = window.AgentAuth || {}; Object.assign(window.AgentAuth, { startMetricsPolling });

// ===== Authorization Metrics Dashboard =====
function startAuthzMetricsPolling() {
	const panel = document.getElementById('authz-metrics-panel');
	if (!panel) return;
	const annc = document.getElementById('authz-metrics-annc');
	function fetchAuthz() {
		fetch('/api/v1/beta/authz/metrics').then(r=>r.json()).then(data => {
			if (!data.success) return;
			panel.innerHTML = `<span class='text-gray-500'>Updated ${new Date(data.timestamp).toLocaleTimeString()}</span>`;
			const m = data.metrics || {};
			const decisions = m.decisions||0;
			const hits = m.cache_hits||0;
			const misses = m.cache_misses||0;
			const hitPct = (hits+misses)>0 ? ((hits/(hits+misses))*100).toFixed(1) : '0.0';
			setText('m-authz-decisions', decisions);
			setText('m-authz-cache-hit', hitPct+'%');
			setText('m-authz-conflicts', m.conflicts||0);
			setText('m-authz-lat-avg', nanosToMicros(m.avg_latency_ns));
			setText('m-authz-lat-p99', nanosToMicros(m.p99_latency_ns));
			setText('m-authz-regex-size', m.regex_cache_size||0);
			setText('m-authz-regex-evict', m.regex_evictions||0);
			setText('m-authz-regex-matches', m.regex_matches||0);
			renderLatencyHistogram(m.latency_histogram||{});
			const upd = document.getElementById('authz-latency-updated');
			const ts = new Date(data.timestamp).toLocaleTimeString();
			if (upd) upd.textContent = ts;
			if(annc) annc.textContent = `Authorization metrics updated ${ts}`;
		}).catch(()=>{});
	}
	fetchAuthz();
	setInterval(fetchAuthz, 5000);
}

function setText(id, v){ const el=document.getElementById(id); if(el) el.textContent=v; }
function nanosToMicros(ns){ return ns? (ns/1000).toFixed(1):'0.0'; }

function renderHistogramGeneric({buckets, svgId, color, labelColor}){
	const svg = document.getElementById(svgId);
	if (!svg) return;
	const entries = Object.entries(buckets||{}).sort((a,b)=>Number(a[0])-Number(b[0]));
	if (entries.length===0){ svg.innerHTML=''; return; }
	const counts = entries.map(e=>e[1]);
	const max = Math.max(...counts);
	const w = 260, h = 60, pad = 4;
	let bars='';
	entries.forEach((e,i)=>{
		const upper = Number(e[0]);
		const cnt = e[1];
		const bw = (w-2*pad)/entries.length - 2;
		const x = pad + i*((w-2*pad)/entries.length);
		const ratio = max>0 ? cnt/max : 0;
		const bh = ratio*(h-14);
		const y = h - pad - bh;
		bars += `<rect x='${x.toFixed(2)}' y='${y.toFixed(2)}' width='${bw.toFixed(2)}' height='${bh.toFixed(2)}' fill='${color}'>`+
			`<title>≤${upper}ns: ${cnt}</title></rect>`;
	});
	const lastUpper = entries[entries.length-1][0];
	svg.innerHTML = `<g>${bars}</g><text x='${w-pad}' y='${h-2}' text-anchor='end' font-size='9' fill='${labelColor}'>≤${lastUpper}ns max bucket</text>`;
}

function renderLatencyHistogram(buckets){
	renderHistogramGeneric({buckets, svgId:'authz-latency-histogram', color:'#6366f1', labelColor:'#374151'});
}

// ===== Policy Metrics (Beta) =====
function startPolicyMetricsPolling(){
	const panel = document.getElementById('policy-metrics-panel');
	if (!panel) return;
	const annc = document.getElementById('policy-metrics-annc');
	function fetchPolicy(){
		fetch('/api/v1/beta/policy/metrics').then(r=>r.json()).then(data => {
			if(!data.success) return;
			const m = data;
			setText('m-policy-total', m.total);
			setText('m-policy-allow', m.allow);
			setText('m-policy-deny', m.deny);
			setText('m-policy-last-reason', (m.last_reason||'').substring(0,120));
			setText('m-policy-p99', nanosToMicros(m.p99_latency_ns)+'µs');
			renderPolicyLatencyHistogram(m.latency_histogram||{});
			const upd = document.getElementById('policy-latency-updated');
			const ts = new Date().toLocaleTimeString();
			if (upd) upd.textContent = ts;
			if(annc) annc.textContent = `Policy metrics updated ${ts}`;
		}).catch(()=>{});
	}
	fetchPolicy();
	setInterval(fetchPolicy, 5000);
}

function renderPolicyLatencyHistogram(buckets){
	renderHistogramGeneric({buckets, svgId:'policy-latency-histogram', color:'#10b981', labelColor:'#065f46'});
}

function bindAuthzEvalForm(){
	const form = document.getElementById('authz-eval-form');
	const out = document.getElementById('authz-eval-result');
	if (!form || !out) return;
	if (form.__gauthBound) return;
	form.__gauthBound = true;
	form.addEventListener('submit', async (e)=>{
		e.preventDefault();
		out.textContent = 'Evaluating...';
		const fd = new FormData(form);
		const subject = fd.get('subject');
		const resource = fd.get('resource');
		const action = fd.get('action');
		const ctx = {
			department: fd.get('department'),
			classification: fd.get('classification'),
			roles: fd.get('roles')
		};
		try {
			const res = await fetch('/api/v1/beta/authz/evaluate', {
				method: 'POST', headers: {'Content-Type':'application/json'},
				body: JSON.stringify({subject, resource, action, context: ctx})
			});
			const data = await res.json();
			if (!data.success){ out.innerHTML = `<span class='text-red-600'>Denied or error: ${data.message||'error'}</span>`; return; }
			const dec = data.decision || {};
			out.innerHTML = `<span class='${dec.allow?'text-green-600':'text-red-600'} font-semibold'>${dec.allow?'ALLOW':'DENY'}</span> <span class='text-gray-600'>${escapeHtml(dec.reason||'')}</span>`;
		} catch(err){ out.innerHTML = `<span class='text-red-600'>Error: ${err.message}</span>`; }
	});
}

// ===== Policy Trace Toggle =====
document.addEventListener('DOMContentLoaded', ()=>{
	const toggle = document.getElementById('policy-trace-toggle');
	const output = document.getElementById('policy-output');
	if(!toggle || !output) return;
	toggle.addEventListener('click', ()=>{
		const hidden = output.classList.toggle('trace-hidden');
		if(hidden){
			// Hide detail lines (keep header + prompt)
			Array.from(output.querySelectorAll('.console-line.success, .console-line.info, .console-line.warning, .console-line.error')).forEach(el=>{
				el.style.display='none';
			});
			toggle.textContent='Show Trace Details';
			toggle.setAttribute('aria-pressed','false');
		}else{
			Array.from(output.querySelectorAll('.console-line')).forEach(el=>{ el.style.display=''; });
			toggle.textContent='Hide Trace Details';
			toggle.setAttribute('aria-pressed','true');
		}
	});
});
