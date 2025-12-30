
import { demoState, addAuditEntry } from "./state.js";
import { addConsoleOutput } from "./console.js";

// Internal buffer for audit stream log export
window._gauthAuditLogBuffer = window._gauthAuditLogBuffer || [];

function updateAuditStatus(status, pulse=false) {
	const badge = document.getElementById('audit-stream-status');
	if (!badge) return;
	const base = 'px-2 py-0.5 text-xs rounded font-semibold';
	if (status === 'connected') {
		badge.className = base + ' bg-green-600 text-white';
		badge.textContent = 'Connected';
	} else if (status === 'connecting') {
		badge.className = base + ' bg-yellow-600 text-white';
		badge.textContent = 'Connecting...';
	} else if (status === 'reconnecting') {
		badge.className = base + ' bg-yellow-500 text-white';
		badge.textContent = 'Reconnecting...';
	} else {
		badge.className = base + ' bg-gray-400 text-white';
		badge.textContent = 'Disconnected';
	}
	if (pulse) {
		badge.classList.remove('animate-pulse');
		void badge.offsetWidth; // reflow flush
		badge.classList.add('animate-pulse');
		setTimeout(()=>badge.classList.remove('animate-pulse'),1200);
	}
}

function startAuditSSEWithBackoff(startBtn, stopBtn){
	if (window._auditSSE) return;
	let attempt = 0; let closedManually = false;
	updateAuditStatus('connecting');
	const connect = () => {
		attempt++;
		const es = new EventSource('/api/v1/audit/stream');
		window._auditSSE = es;
		es.onopen = () => {
			addConsoleOutput('audit-output', '✓ Audit SSE connected', 'success');
			updateAuditStatus('connected', true);
			if (startBtn) startBtn.disabled = true;
			if (stopBtn) stopBtn.disabled = false;
		};
		es.onmessage = (ev) => {
			try {
				const data = JSON.parse(ev.data);
				window._gauthAuditLogBuffer.push({ ts: Date.now(), raw: ev.data, parsed: data });
				if (data.entries) {
					addConsoleOutput('audit-output', `[batch ${data.entries.length} entries]`, 'info');
				} else {
					addConsoleOutput('audit-output', `[audit] ${ev.data}`, 'info');
				}
			} catch {
				window._gauthAuditLogBuffer.push({ ts: Date.now(), raw: ev.data });
				addConsoleOutput('audit-output', `[audit] ${ev.data}`, 'info');
			}
		};
		es.onerror = () => {
			if (closedManually) return;
			addConsoleOutput('audit-output', 'Audit SSE error/closed – scheduling reconnect', 'error');
			if (window._auditSSE) { window._auditSSE.close(); delete window._auditSSE; }
			if (startBtn) startBtn.disabled = true;
			if (stopBtn) stopBtn.disabled = true;
			updateAuditStatus('reconnecting');
			const backoff = Math.min(1000 * Math.pow(2, attempt), 15000);
			setTimeout(()=>{ if(!closedManually) connect(); }, backoff);
		};
		window._stopAuditStreamInternal = () => {
			closedManually = true;
			if (window._auditSSE) { window._auditSSE.close(); delete window._auditSSE; }
			addConsoleOutput('audit-output', 'Stopped audit SSE stream', 'warning');
			updateAuditStatus('disconnected');
			if (startBtn) startBtn.disabled = false;
			if (stopBtn) stopBtn.disabled = true;
		};
	};
	connect();
}

export function downloadAuditLog(){
	const data = (window._gauthAuditLogBuffer||[]).map(e=>e);
	const blob = new Blob([JSON.stringify({ exportedAt: new Date().toISOString(), count: data.length, entries: data }, null, 2)], { type: 'application/json' });
	const a = document.createElement('a');
	a.href = URL.createObjectURL(blob);
	a.download = 'audit_stream_log.json';
	a.click();
	URL.revokeObjectURL(a.href);
}

export async function viewAuditLog() {
	addConsoleOutput("audit-output", "Fetching audit logs...", "info");
	try {
		const limit = document.getElementById("audit-limit-input")?.value || 50;
		const res = await fetch(`/api/v1/audit/logs?limit=${limit}`);
		const data = await res.json();
		if (!data.success) {
			addConsoleOutput("audit-output", `✗ Failed to fetch audit logs: ${data.message || "unknown error"}`, "error");
			return;
		}
		addConsoleOutput("audit-output", `✓ Retrieved ${data.count} audit entries`, "success");
		if (data.entries && data.entries.length > 0) {
			data.entries.forEach(entry => {
				const time = new Date(entry.at).toLocaleTimeString();
				addConsoleOutput("audit-output", `[${time}] ${entry.action} by ${entry.actor} on ${entry.resource} - ${entry.outcome}`, "info");
			});
		} else {
			addConsoleOutput("audit-output", "No audit entries found", "info");
		}
		demoState.auditEntries = data.entries || [];
	} catch (error) {
		addConsoleOutput("audit-output", `✗ Failed to fetch audit logs: ${error.message}`, "error");
	}
}

export async function generateReport() {
	addConsoleOutput("audit-output", "Generating audit report...", "info");
	try {
		if (!demoState.auditEntries.length) {
			await viewAuditLog();
			await new Promise(resolve => setTimeout(resolve, 500));
		}
		if (!demoState.auditEntries.length) {
			addConsoleOutput("audit-output", "No audit data available for report generation", "warning");
			return;
		}
		const stats = {
			total: demoState.auditEntries.length,
			byAction: {},
			byOutcome: {},
			timeRange: { start: null, end: null }
		};
		demoState.auditEntries.forEach(entry => {
			stats.byAction[entry.action] = (stats.byAction[entry.action] || 0) + 1;
			stats.byOutcome[entry.outcome] = (stats.byOutcome[entry.outcome] || 0) + 1;
			const entryTime = new Date(entry.at);
			if (!stats.timeRange.start || entryTime < stats.timeRange.start) stats.timeRange.start = entryTime;
			if (!stats.timeRange.end || entryTime > stats.timeRange.end) stats.timeRange.end = entryTime;
		});
		addConsoleOutput("audit-output", "=== AUDIT REPORT ===", "success");
		addConsoleOutput("audit-output", `Total Entries: ${stats.total}`, "info");
		addConsoleOutput("audit-output", `Time Range: ${stats.timeRange.start?.toLocaleString()} - ${stats.timeRange.end?.toLocaleString()}`, "info");
		addConsoleOutput("audit-output", "Actions:", "info");
		Object.entries(stats.byAction).forEach(([action, count]) => {
			addConsoleOutput("audit-output", `  ${action}: ${count}`, "info");
		});
		addConsoleOutput("audit-output", "Outcomes:", "info");
		Object.entries(stats.byOutcome).forEach(([outcome, count]) => {
			addConsoleOutput("audit-output", `  ${outcome}: ${count}`, "info");
		});
		addConsoleOutput("audit-output", "=== END REPORT ===", "success");
		demoState.lastReport = { generatedAt: new Date().toISOString(), stats, entries: demoState.auditEntries };
	} catch (error) {
		addConsoleOutput("audit-output", `✗ Report generation failed: ${error.message}`, "error");
	}
}

export function auditInit() {
	document.querySelectorAll("[data-action=\"view-audit-log\"]").forEach(el => el.addEventListener("click", viewAuditLog));
	document.querySelectorAll("[data-action=\"generate-report\"]").forEach(el => el.addEventListener("click", generateReport));
	// SSE audit stream buttons
	const startBtn = document.getElementById('startAuditStream');
	const stopBtn = document.getElementById('stopAuditStream');
	if (startBtn && !startBtn.__gauthBound) {
		startBtn.__gauthBound = true;
		startBtn.addEventListener('click', () => {
			if (window._auditSSE) {
				addConsoleOutput('audit-output', 'Audit stream already running', 'warning');
				return;
			}
			addConsoleOutput('audit-output', 'Starting audit SSE stream...', 'info');
			startAuditSSEWithBackoff(startBtn, stopBtn);
		});
	}
	if (stopBtn && !stopBtn.__gauthBound) {
		stopBtn.__gauthBound = true;
		stopBtn.addEventListener('click', () => {
			if (!window._auditSSE) {
				addConsoleOutput('audit-output', 'No active audit stream', 'warning');
				return;
			}
			if (window._stopAuditStreamInternal) window._stopAuditStreamInternal();
		});
		stopBtn.disabled = true;
	}
    const dlBtn = document.getElementById('download-audit-log');
    if (dlBtn && !dlBtn.__gauthBound) {
        dlBtn.__gauthBound = true;
        dlBtn.addEventListener('click', downloadAuditLog);
    }
}

// Legacy global bridge (for compatibility)
window.AgentAuth = window.AgentAuth || {}; Object.assign(window.AgentAuth, { viewAuditLog, generateReport, downloadAuditLog });
