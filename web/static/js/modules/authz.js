// authz.js - Authorization and Event System logic migrated from legacy app-BK.js
import { demoState, addAuditEntry } from "./state.js";
import { addConsoleOutput } from "./console.js";

// Internal buffers for downloadable logs
window._agentauthEventLogBuffer = window._agentauthEventLogBuffer || [];

function updateEventStatus(status, pulse=false) {
    const badge = document.getElementById('event-stream-status');
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
        void badge.offsetWidth; // reflow
        badge.classList.add('animate-pulse');
        setTimeout(()=>badge.classList.remove('animate-pulse'), 1200);
    }
}

function startEventSSEWithBackoff(startBtn, stopBtn) {
    if (window._eventSSE) return;
    let attempt = 0;
    let closedManually = false;
    updateEventStatus(attempt===0?'connecting':'reconnecting');
    const connect = () => {
        attempt++;
        const es = new EventSource('/api/v1/events/stream');
        window._eventSSE = es;
        es.onopen = () => {
            addConsoleOutput('event-output', '✓ Event SSE connected', 'success');
            updateEventStatus('connected', true);
            if (startBtn) startBtn.disabled = true;
            if (stopBtn) stopBtn.disabled = false;
        };
        es.onmessage = (ev) => {
            try {
                const data = JSON.parse(ev.data);
                window._agentauthEventLogBuffer.push({ ts: Date.now(), raw: ev.data, parsed: data });
                addConsoleOutput('event-output', `[event:${data.type||'msg'}] ${ev.data}`, 'info');
            } catch {
                window._agentauthEventLogBuffer.push({ ts: Date.now(), raw: ev.data });
                addConsoleOutput('event-output', `[event] ${ev.data}`, 'info');
            }
        };
        es.onerror = () => {
            if (closedManually) return; // user stop
            addConsoleOutput('event-output', 'Event SSE error/closed – scheduling reconnect', 'error');
            if (window._eventSSE) { window._eventSSE.close(); delete window._eventSSE; }
            if (startBtn) startBtn.disabled = true;
            if (stopBtn) stopBtn.disabled = true;
            updateEventStatus('reconnecting');
            const backoff = Math.min(1000 * Math.pow(2, attempt), 15000); // exponential cap 15s
            setTimeout(()=>{ if (!closedManually) connect(); }, backoff);
        };
        window._stopEventStreamInternal = () => {
            closedManually = true;
            if (window._eventSSE) { window._eventSSE.close(); delete window._eventSSE; }
            addConsoleOutput('event-output', 'Stopped event SSE stream', 'warning');
            updateEventStatus('disconnected');
            if (startBtn) startBtn.disabled = false;
            if (stopBtn) stopBtn.disabled = true;
        };
    };
    connect();
}

export function downloadEventLog() {
    const data = (window._agentauthEventLogBuffer||[]).map(e=>e);
    const blob = new Blob([JSON.stringify({ exportedAt: new Date().toISOString(), count: data.length, events: data }, null, 2)], { type: 'application/json' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'event_stream_log.json';
    a.click();
    URL.revokeObjectURL(a.href);
}

export async function checkAuthorization() {
    addConsoleOutput('authz-output', 'Checking POA authorization...', 'info');
    try {
        const clientId = document.getElementById('client-id-input')?.value || 'demo-client-' + Math.random().toString(36).slice(2,8);
        // Potential future UI fields (fallback defaults inline)
        const jurisdiction = document.getElementById('jurisdiction-input')?.value || 'US';
        const powerType = document.getElementById('power-type-input')?.value || 'financial_transactions';
        const scope = document.getElementById('scope-input')?.value || 'ai_power_of_attorney,financial_transactions';
        const principalId = document.getElementById('principal-id-input')?.value || 'principal-xyz';
        const aiAgentId = document.getElementById('ai-agent-id-input')?.value || 'agent-123';
        const legalBasis = document.getElementById('legal-basis-input')?.value || 'law2025';
        const redirectURI = document.getElementById('redirect-uri-input')?.value || 'https://cb.example.com';
        const responseType = 'code';
        const state = 'demo';
        const payload = {
            client_id: clientId,
            principal_id: principalId,
            ai_agent_id: aiAgentId,
            power_type: powerType,
            jurisdiction,
            scope,
            legal_basis: legalBasis,
            redirect_uri: redirectURI,
            response_type: responseType,
            state
        };
        const res = await fetch('/api/v1/poa/authorize', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();
        if (!data.success) {
            addConsoleOutput('authz-output', `✗ Authorization failed: ${data.message || 'unknown error'}`, 'error');
            return;
        }
        addConsoleOutput('authz-output', '✓ POA authorization successful', 'success');
        addConsoleOutput('authz-output', `  Client ID: ${clientId}`, 'info');
        addConsoleOutput('authz-output', `  Jurisdiction: ${jurisdiction}`, 'info');
        addConsoleOutput('authz-output', `  Scope: ${scope}`, 'info');
        addAuditEntry({
            timestamp: new Date().toISOString(),
            action: 'POA_AUTHORIZED',
            clientId,
            beta: true,
            jurisdiction,
            scope
        });
    } catch (error) {
        addConsoleOutput('authz-output', `✗ Authorization failed: ${error.message}`, 'error');
    }
}

export async function publishEvent() {
    addConsoleOutput('event-output', 'Publishing demo event...', 'info');
    try {
        const eventType = document.getElementById('event-type-input')?.value || 'demo_event';
        const eventData = document.getElementById('event-data-input')?.value || '{"message":"Demo event from web interface","timestamp":"'+new Date().toISOString()+'"}';
        let data;
        try {
            data = typeof eventData === 'string' ? JSON.parse(eventData) : eventData;
        } catch {
            data = eventData;
        }
        const res = await fetch('/api/v1/events/emit', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ type: eventType, data })
        });
        const response = await res.json();
        if (!response.success) {
            addConsoleOutput('event-output', `✗ Event publish failed: ${response.message || 'unknown error'}`, 'error');
            return;
        }
        addConsoleOutput('event-output', '✓ Event published successfully', 'success');
        addConsoleOutput('event-output', `  Type: ${eventType}`, 'info');
        addConsoleOutput('event-output', `  Event ID: ${response.event?.id}`, 'info');
        addAuditEntry({
            timestamp: new Date().toISOString(),
            action: 'EVENT_PUBLISHED',
            eventType,
            eventId: response.event?.id,
            beta: true
        });
    } catch (error) {
        addConsoleOutput('event-output', `✗ Event publish failed: ${error.message}`, 'error');
    }
}

export async function subscribeEvents() {
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
            setTimeout(() => {
                if (!demoState.subscriptionsActive) {
                    subscribeEvents();
                }
            }, 3000);
        };
        window.currentEventSource = eventSource;
    } catch (error) {
        addConsoleOutput('event-output', `✗ Failed to start event subscription: ${error.message}`, 'error');
    }
}

export function authzInit() {
    document.querySelectorAll('[data-action="check-authorization"]').forEach(el => el.addEventListener('click', checkAuthorization));
    document.querySelectorAll('[data-action="publish-event"]').forEach(el => el.addEventListener('click', publishEvent));
    document.querySelectorAll('[data-action="subscribe-events"]').forEach(el => el.addEventListener('click', subscribeEvents));
    // New explicit start/stop stream buttons (separate from legacy subscribe logic)
    const startBtn = document.getElementById('startEventStream');
    const stopBtn = document.getElementById('stopEventStream');
    if (startBtn && !startBtn.__agentauthBound) {
        startBtn.__agentauthBound = true;
        startBtn.addEventListener('click', () => {
            if (window._eventSSE) {
                addConsoleOutput('event-output', 'Event stream already running', 'warning');
                return;
            }
            addConsoleOutput('event-output', 'Opening event SSE stream...', 'info');
            startEventSSEWithBackoff(startBtn, stopBtn);
        });
    }
    if (stopBtn && !stopBtn.__agentauthBound) {
        stopBtn.__agentauthBound = true;
        stopBtn.addEventListener('click', () => {
            if (!window._eventSSE) {
                addConsoleOutput('event-output', 'No active event stream', 'warning');
                return;
            }
            if (window._stopEventStreamInternal) window._stopEventStreamInternal();
        });
        stopBtn.disabled = true; // initial state
    }
    // Download event log button (optional presence)
    const dlBtn = document.getElementById('download-event-log');
    if (dlBtn && !dlBtn.__agentauthBound) {
        dlBtn.__agentauthBound = true;
        dlBtn.addEventListener('click', downloadEventLog);
    }
}

window.AgentAuth = window.AgentAuth || {}; Object.assign(window.AgentAuth, { checkAuthorization, publishEvent, subscribeEvents, downloadEventLog });
