// AgentAuth Metrics & Monitoring - Real-time metrics visualization and monitoring
class AgentAuthMetricsMonitor {
    constructor(apiClient) {
        this.api = apiClient;
        this.updateIntervals = new Map();
        this.charts = new Map();
        this.eventStreams = new Map();
        this.metricsCache = new Map();
        
        this.initialize();
    }

    initialize() {
        this.setupMetricsUpdates();
        this.setupEventStreams();
        this.setupExportHandlers();
    }

    setupMetricsUpdates() {
        // Update violation metrics every 3 seconds
        this.updateIntervals.set('violations', setInterval(() => {
            this.updateViolationMetrics();
        }, 3000));

        // Update semantic metrics every 5 seconds
        this.updateIntervals.set('semantics', setInterval(() => {
            this.updateSemanticMetrics();
        }, 5000));

        // Update capability anchor status every 10 seconds
        this.updateIntervals.set('capability', setInterval(() => {
            this.updateCapabilityAnchorStatus();
        }, 10000));

        // Update POA metrics every 5 seconds
        this.updateIntervals.set('poa', setInterval(() => {
            this.updatePOAMetrics();
        }, 5000));

        // Update authorization metrics every 2 seconds
        this.updateIntervals.set('authz', setInterval(() => {
            this.updateAuthzMetrics();
        }, 2000));

        // Update policy metrics every 5 seconds
        this.updateIntervals.set('policy', setInterval(() => {
            this.updatePolicyMetrics();
        }, 5000));

        // Update decision metrics every 3 seconds
        this.updateIntervals.set('decisions', setInterval(() => {
            this.updateDecisionMetrics();
        }, 3000));

        // Update audit preview every 10 seconds
        this.updateIntervals.set('audit', setInterval(() => {
            this.updateAuditPreview();
        }, 10000));

        // Update lifecycle timeline every 5 seconds
        this.updateIntervals.set('lifecycle', setInterval(() => {
            this.updateLifecycleTimeline();
        }, 5000));

        // Update capability registry every 15 seconds
        this.updateIntervals.set('registry', setInterval(() => {
            this.updateCapabilityRegistry();
        }, 15000));

        // Start initial updates
        setTimeout(() => this.runInitialUpdates(), 1000);
    }

    async runInitialUpdates() {
        try {
            await Promise.allSettled([
                this.updatePOAMetrics(),
                this.updateCapabilityRegistry(),
                this.updateHealthStatus()
            ]);
        } catch (error) {
            console.warn('Some initial metrics updates failed:', error);
        }
    }

    async updateHealthStatus() {
        try {
            const health = await this.api.getHealth();
            this.renderHealthStatus(health);
        } catch (error) {
            this.handleMetricsError('health', error);
        }
    }

    renderHealthStatus(data) {
        // Update various status indicators with health data
        const statusElements = document.querySelectorAll('.status-indicator');
        statusElements.forEach(el => {
            if (data.success) {
                el.textContent = 'Active';
                el.classList.add('text-green-600');
                el.classList.remove('text-red-600');
            } else {
                el.textContent = 'Error';
                el.classList.add('text-red-600');
                el.classList.remove('text-green-600');
            }
        });

        // Update uptime if available
        if (data.data && data.data.uptime) {
            const uptimeEl = document.getElementById('system-uptime');
            if (uptimeEl) {
                uptimeEl.textContent = data.data.uptime;
            }
        }
    }

    async updateViolationMetrics() {
        try {
            const metrics = await this.api.getViolationMetrics();
            this.renderViolationMetrics(metrics);
        } catch (error) {
            this.handleMetricsError('violations', error);
        }
    }

    renderViolationMetrics(data) {
        const statusEl = document.getElementById('violation-metrics-status');
        const totalEl = document.getElementById('violation-total');
        const rate60El = document.getElementById('violation-rate-60');
        const rate300El = document.getElementById('violation-rate-300');
        const tbodyEl = document.getElementById('violation-counters-tbody');
        const surgeEl = document.getElementById('violation-surge-flag');

        if (statusEl) {
            statusEl.textContent = data.success ? 'Active' : 'Error';
            statusEl.setAttribute('aria-live', 'polite');
        }

        if (data.success && data.counters) {
            const counters = data.counters;
            const total = Object.values(counters).reduce((sum, val) => sum + val, 0);

            if (totalEl) totalEl.textContent = total;
            if (rate60El) rate60El.textContent = (data.rate_60s || 0).toFixed(2);
            if (rate300El) rate300El.textContent = (data.rate_300s || 0).toFixed(2);

            // Update surge flag
            if (surgeEl) {
                const isNormal = (data.rate_60s || 0) < 10; // Arbitrary threshold
                surgeEl.textContent = isNormal ? 'Normal' : 'High';
                surgeEl.className = `px-2 py-0.5 rounded ${isNormal ? 'bg-green-600' : 'bg-red-600'} text-white text-xs`;
            }

            // Update counters table
            if (tbodyEl) {
                const rows = Object.entries(counters).map(([category, count]) => 
                    `<tr><td class="text-left px-2 py-1">${category}</td><td class="text-right px-2 py-1">${count}</td></tr>`
                ).join('');
                tbodyEl.innerHTML = rows || '<tr><td colspan="2" class="text-center text-gray-400 py-2">No violations</td></tr>';
            }
        }
    }

    async updateSemanticMetrics() {
        try {
            const metrics = await this.api.getSemanticMetrics();
            this.renderSemanticMetrics(metrics);
        } catch (error) {
            this.handleMetricsError('semantic', error);
        }
    }

    renderSemanticMetrics(data) {
        const statusEl = document.getElementById('semantic-metrics-status');
        const tbodyEl = document.getElementById('semantic-counters-tbody');
        const rateBadgeEl = document.getElementById('semantic-rate-badge');

        if (statusEl) {
            statusEl.textContent = data.success ? 'Active' : 'Error';
        }

        if (data.success && data.counters) {
            const counters = data.counters;

            // Update rate badge
            if (rateBadgeEl) {
                const hasRates = data.rate_per_minute_60s && Object.keys(data.rate_per_minute_60s).length > 0;
                rateBadgeEl.textContent = hasRates ? 'Rates available' : 'No rates';
            }

            // Update counters table
            if (tbodyEl) {
                const rows = Object.entries(counters).map(([counter, value]) => 
                    `<tr><td class="text-left px-2 py-1">${counter}</td><td class="text-right px-2 py-1">${value}</td></tr>`
                ).join('');
                tbodyEl.innerHTML = rows || '<tr><td colspan="2" class="text-center text-gray-400 py-2">No counters</td></tr>';
            }
        }
    }

    async updateCapabilityAnchorStatus() {
        try {
            const status = await this.api.getCapabilityAnchorStatus();
            this.renderCapabilityAnchorStatus(status);
        } catch (error) {
            this.handleMetricsError('capability-anchor', error);
        }
    }

    renderCapabilityAnchorStatus(data) {
        const statusEl = document.getElementById('cap-anchor-status');
        const registryHashEl = document.getElementById('cap-anchor-registry-hash');
        const lastWriteEl = document.getElementById('cap-anchor-last-write');
        const ageEl = document.getElementById('cap-anchor-age');
        const emittedEl = document.getElementById('cap-anchor-emitted');
        const skippedEl = document.getElementById('cap-anchor-skipped');
        const hashChangedEl = document.getElementById('cap-anchor-hash-changed');

        if (statusEl) {
            statusEl.textContent = data.success ? 'Active' : 'Error';
        }

        if (data.success && data.status) {
            const status = data.status;
            
            if (registryHashEl) registryHashEl.textContent = status.registry_hash || '—';
            if (lastWriteEl) lastWriteEl.textContent = status.last_write || '—';
            if (ageEl) ageEl.textContent = status.age || '—';
            if (emittedEl) emittedEl.textContent = status.emitted_total || 0;
            if (skippedEl) skippedEl.textContent = status.skipped_total || 0;
            if (hashChangedEl) hashChangedEl.textContent = status.hash_changed_total || 0;
        }
    }

    async updatePOAMetrics() {
        try {
            const metrics = await this.api.getPOAMetrics();
            this.renderPOAMetrics(metrics);
        } catch (error) {
            this.handleMetricsError('poa', error);
        }
    }

    renderPOAMetrics(data) {
        const poaPanel = document.getElementById('poa-metrics');
        const reqEl = document.getElementById('m-poa-req');
        const okEl = document.getElementById('m-poa-ok');
        const jurEl = document.getElementById('m-poa-jur');
        const scopeEl = document.getElementById('m-poa-scope');
        const missingEl = document.getElementById('m-poa-missing');
        const updatedEl = document.getElementById('poa-metrics-updated');

        if (data.success && poaPanel) {
            poaPanel.classList.remove('hidden');
            
            const metrics = data.metrics || {};
            if (reqEl) reqEl.textContent = metrics.total_requests || 0;
            if (okEl) okEl.textContent = metrics.success || 0;
            if (jurEl) jurEl.textContent = metrics.jurisdiction_errors || 0;
            if (scopeEl) scopeEl.textContent = metrics.scope_errors || 0;
            if (missingEl) missingEl.textContent = metrics.missing_delegation || 0;
            
            if (updatedEl) updatedEl.textContent = new Date().toLocaleTimeString();
        }
    }

    async updateAuthzMetrics() {
        try {
            const metrics = await this.api.getAuthzMetrics();
            this.renderAuthzMetrics(metrics);
        } catch (error) {
            this.handleMetricsError('authz', error);
        }
    }

    renderAuthzMetrics(data) {
        const panelEl = document.getElementById('authz-metrics-panel');
        const decisionsEl = document.getElementById('m-authz-decisions');
        const cacheHitEl = document.getElementById('m-authz-cache-hit');
        const conflictsEl = document.getElementById('m-authz-conflicts');
        const latAvgEl = document.getElementById('m-authz-lat-avg');
        const latP99El = document.getElementById('m-authz-lat-p99');
        const regexSizeEl = document.getElementById('m-authz-regex-size');
        const regexEvictEl = document.getElementById('m-authz-regex-evict');
        const regexMatchesEl = document.getElementById('m-authz-regex-matches');
        const updatedEl = document.getElementById('authz-latency-updated');

        if (panelEl) {
            panelEl.textContent = data.success ? 'Connected' : 'Disconnected';
        }

        if (data.success && data.metrics) {
            const m = data.metrics;
            if (decisionsEl) decisionsEl.textContent = m.total_decisions || 0;
            if (cacheHitEl) cacheHitEl.textContent = m.cache_hit_rate ? `${(m.cache_hit_rate * 100).toFixed(1)}%` : '0%';
            if (conflictsEl) conflictsEl.textContent = m.conflicts || 0;
            if (latAvgEl) latAvgEl.textContent = m.avg_latency_us || 0;
            if (latP99El) latP99El.textContent = m.p99_latency_us || 0;
            if (regexSizeEl) regexSizeEl.textContent = m.regex_cache_size || 0;
            if (regexEvictEl) regexEvictEl.textContent = m.regex_evictions || 0;
            if (regexMatchesEl) regexMatchesEl.textContent = m.regex_matches || 0;
            
            if (updatedEl) updatedEl.textContent = new Date().toLocaleTimeString();

            // Update histogram if data available
            this.updateLatencyHistogram('authz-latency-histogram', m.latency_histogram);
        }
    }

    async updatePolicyMetrics() {
        try {
            const metrics = await this.api.getPolicyMetrics();
            this.renderPolicyMetrics(metrics);
        } catch (error) {
            this.handleMetricsError('policy', error);
        }
    }

    renderPolicyMetrics(data) {
        const totalEl = document.getElementById('m-policy-total');
        const allowEl = document.getElementById('m-policy-allow');
        const denyEl = document.getElementById('m-policy-deny');
        const p99El = document.getElementById('m-policy-p99');
        const reasonEl = document.getElementById('m-policy-last-reason');
        const updatedEl = document.getElementById('policy-latency-updated');

        if (data.success && data.metrics) {
            const m = data.metrics;
            if (totalEl) totalEl.textContent = m.total_evaluations || 0;
            if (allowEl) allowEl.textContent = m.allow_decisions || 0;
            if (denyEl) denyEl.textContent = m.deny_decisions || 0;
            if (p99El) p99El.textContent = m.p99_latency_us || 0;
            if (reasonEl) reasonEl.textContent = m.last_reason || '—';
            
            if (updatedEl) updatedEl.textContent = new Date().toLocaleTimeString();

            // Update histogram if data available
            this.updateLatencyHistogram('policy-latency-histogram', m.latency_histogram);
        }
    }

    async updateDecisionMetrics() {
        try {
            const metrics = await this.api.getDecisionMetrics();
            this.renderDecisionMetrics(metrics);
        } catch (error) {
            this.handleMetricsError('decision', error);
        }
    }

    renderDecisionMetrics(data) {
        const statusEl = document.getElementById('decision-metrics-status');
        const countsEl = document.getElementById('decision-counts-tbody');
        const reasonsEl = document.getElementById('decision-reasons-tbody');

        if (statusEl) {
            statusEl.textContent = data.success ? 'Active' : 'Error';
        }

        if (data.success) {
            // Render decision counts
            if (countsEl && data.counts) {
                const rows = data.counts.map(item => 
                    `<tr>
                        <td class="px-2 py-1">${item.action}</td>
                        <td class="px-2 py-1">${item.resource}</td>
                        <td class="px-2 py-1">
                            <span class="px-2 py-0.5 rounded text-xs font-semibold ${item.outcome === 'allow' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
                                ${item.outcome.toUpperCase()}
                            </span>
                        </td>
                        <td class="px-2 py-1">${item.count}</td>
                    </tr>`
                ).join('');
                countsEl.innerHTML = rows || '<tr><td colspan="4" class="text-center text-gray-400 py-4">No data</td></tr>';
            }

            // Render decision reasons
            if (reasonsEl && data.reasons) {
                const rows = data.reasons.map(item => 
                    `<tr>
                        <td class="px-2 py-1">${item.action}</td>
                        <td class="px-2 py-1">${item.resource}</td>
                        <td class="px-2 py-1">
                            <span class="px-2 py-0.5 rounded text-xs font-semibold ${item.outcome === 'allow' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
                                ${item.outcome.toUpperCase()}
                            </span>
                        </td>
                        <td class="px-2 py-1">
                            <span class="px-1 py-0.5 rounded text-xs ${this.getReasonBadgeClass(item.reason)}">
                                ${item.reason}
                            </span>
                        </td>
                        <td class="px-2 py-1">${item.count}</td>
                    </tr>`
                ).join('');
                reasonsEl.innerHTML = rows || '<tr><td colspan="5" class="text-center text-gray-400 py-4">No data</td></tr>';
            }
        }
    }

    getReasonBadgeClass(reason) {
        const classes = {
            'init': 'bg-blue-100 text-blue-800',
            'status_change': 'bg-yellow-100 text-yellow-800',
            'noop': 'bg-gray-100 text-gray-800',
            'maintenance': 'bg-purple-100 text-purple-800',
            'rate_limited': 'bg-orange-100 text-orange-800',
            'policy_violation': 'bg-red-100 text-red-800'
        };
        return classes[reason] || 'bg-gray-100 text-gray-800';
    }

    async updateAuditPreview() {
        try {
            const audit = await this.api.getAuditPreview(10);
            this.renderAuditPreview(audit);
        } catch (error) {
            this.handleMetricsError('audit', error);
        }
    }

    renderAuditPreview(data) {
        const statusEl = document.getElementById('audit-preview-status');
        const tbodyEl = document.getElementById('audit-preview-tbody');

        if (statusEl) {
            statusEl.textContent = data.success ? 'Active' : 'Error';
        }

        if (data.success && data.entries && tbodyEl) {
            const rows = data.entries.map(entry => 
                `<tr>
                    <td class="px-2 py-1 font-mono text-xs">${entry.id || '—'}</td>
                    <td class="px-2 py-1 text-xs">${entry.action || '—'}</td>
                    <td class="px-2 py-1 text-xs">${entry.resource || '—'}</td>
                    <td class="px-2 py-1">
                        <span class="px-2 py-0.5 rounded text-xs font-semibold ${entry.outcome === 'success' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
                            ${entry.outcome || 'unknown'}
                        </span>
                    </td>
                    <td class="px-2 py-1 text-xs">${entry.reason || '—'}</td>
                    <td class="px-2 py-1 text-xs">${entry.at ? new Date(entry.at).toLocaleTimeString() : '—'}</td>
                </tr>`
            ).join('');
            tbodyEl.innerHTML = rows;
        } else if (tbodyEl) {
            tbodyEl.innerHTML = '<tr><td colspan="6" class="text-center text-gray-400 py-4">No audit entries</td></tr>';
        }
    }

    async updateLifecycleTimeline() {
        try {
            const timeline = await this.api.getLifecycleTimeline();
            this.renderLifecycleTimeline(timeline);
        } catch (error) {
            this.handleMetricsError('lifecycle', error);
        }
    }

    renderLifecycleTimeline(data) {
        const statusEl = document.getElementById('lifecycle-timeline-status');
        const tbodyEl = document.getElementById('lifecycle-timeline-tbody');

        if (statusEl) {
            statusEl.textContent = data.success ? 'Active' : 'Error';
        }

        if (data.success && data.events && tbodyEl) {
            const rows = data.events.slice(0, 10).map(event => 
                `<tr>
                    <td class="px-2 py-1 text-xs">${event.entity_type || '—'}</td>
                    <td class="px-2 py-1 font-mono text-xs">${event.entity_id || '—'}</td>
                    <td class="px-2 py-1 text-xs">${event.old_status || '—'}</td>
                    <td class="px-2 py-1 text-xs">${event.new_status || '—'}</td>
                    <td class="px-2 py-1">
                        <span class="px-2 py-0.5 rounded text-xs font-semibold ${event.outcome === 'success' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
                            ${event.outcome || 'unknown'}
                        </span>
                    </td>
                    <td class="px-2 py-1 text-xs">
                        <span class="px-1 py-0.5 rounded text-xs ${this.getReasonBadgeClass(event.reason)}">
                            ${event.reason || '—'}
                        </span>
                    </td>
                    <td class="px-2 py-1 font-mono text-xs">${event.latency_ns || 0}</td>
                    <td class="px-2 py-1 text-xs">${event.at ? new Date(event.at).toLocaleTimeString() : '—'}</td>
                </tr>`
            ).join('');
            tbodyEl.innerHTML = rows;
        } else if (tbodyEl) {
            tbodyEl.innerHTML = '<tr><td colspan="8" class="text-center text-gray-400 py-4">No timeline events</td></tr>';
        }
    }

    async updateCapabilityRegistry() {
        try {
            const registry = await this.api.getCapabilities();
            this.renderCapabilityRegistry(registry);
        } catch (error) {
            this.handleMetricsError('registry', error);
        }
    }

    renderCapabilityRegistry(data) {
        const hashEl = document.getElementById('cap-registry-hash');
        const prevHashEl = document.getElementById('cap-registry-prev-hash');
        const changedAtEl = document.getElementById('cap-registry-changed-at');
        const listEl = document.getElementById('cap-registry-list');

        if (data.success && data.capabilities) {
            // Use current timestamp since no hash is provided
            if (hashEl) hashEl.textContent = 'N/A (Live)';
            if (prevHashEl) prevHashEl.textContent = '—';
            if (changedAtEl) changedAtEl.textContent = new Date().toLocaleString();
            
            if (listEl && data.capabilities) {
                const items = data.capabilities.map(cap => 
                    `<li class="text-xs">
                        <span class="font-mono">${cap.id}</span>
                        <span class="text-gray-500 ml-2">v${cap.version}</span>
                        <span class="text-green-500 ml-1">${cap.stable ? '✓' : '⚠'}</span>
                    </li>`
                ).join('');
                listEl.innerHTML = items;
            }
        }
    }

    updateLatencyHistogram(elementId, histogramData) {
        const svg = document.getElementById(elementId);
        if (!svg || !histogramData) return;

        // Clear existing content
        svg.innerHTML = '';

        // Simple histogram visualization
        const buckets = histogramData.buckets || [];
        if (buckets.length === 0) return;

        const maxCount = Math.max(...buckets.map(b => b.count), 1);
        const barWidth = 260 / buckets.length;

        buckets.forEach((bucket, i) => {
            const height = (bucket.count / maxCount) * 50;
            const x = i * barWidth;
            const y = 60 - height;

            const rect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
            rect.setAttribute('x', x);
            rect.setAttribute('y', y);
            rect.setAttribute('width', barWidth - 1);
            rect.setAttribute('height', height);
            rect.setAttribute('fill', '#3B82F6');
            rect.setAttribute('opacity', '0.7');

            svg.appendChild(rect);
        });
    }

    setupEventStreams() {
        // Setup event streams for real-time updates if available
        try {
            // Example event stream setup (if endpoint exists)
            /*
            const eventStream = this.api.createEventStream(
                '/api/v1/beta/metrics/stream',
                (data) => this.handleMetricsEvent(data),
                (error) => console.warn('Metrics stream error:', error)
            );
            this.eventStreams.set('metrics', eventStream);
            */
        } catch (error) {
            console.warn('Event streams not available:', error);
        }
    }

    setupExportHandlers() {
        // Setup export button handlers
        const exportButtons = {
            'export-decisions-json': () => this.exportDecisionMetrics('json'),
            'export-decisions-csv': () => this.exportDecisionMetrics('csv'),
            'export-audit-json': () => this.exportAuditLog('json'),
            'export-audit-csv': () => this.exportAuditLog('csv'),
            'export-lifecycle-json': () => this.exportLifecycleTimeline('json'),
            'export-lifecycle-csv': () => this.exportLifecycleTimeline('csv')
        };

        Object.entries(exportButtons).forEach(([buttonId, handler]) => {
            const button = document.getElementById(buttonId);
            if (button) {
                button.addEventListener('click', handler);
            }
        });
    }

    async exportDecisionMetrics(format) {
        try {
            const data = await this.api.getDecisionMetrics();
            if (data.success) {
                const filename = `decision-metrics-${new Date().toISOString().split('T')[0]}.${format}`;
                await this.api.exportData(data, filename, format);
            }
        } catch (error) {
            console.error('Failed to export decision metrics:', error);
        }
    }

    async exportAuditLog(format) {
        try {
            const data = await this.api.exportAuditLog(format);
            const filename = `audit-log-${new Date().toISOString().split('T')[0]}.${format}`;
            await this.api.exportData(data, filename, format);
        } catch (error) {
            console.error('Failed to export audit log:', error);
        }
    }

    async exportLifecycleTimeline(format) {
        try {
            const data = await this.api.getLifecycleTimeline();
            if (data.success) {
                const filename = `lifecycle-timeline-${new Date().toISOString().split('T')[0]}.${format}`;
                await this.api.exportData(data.events, filename, format);
            }
        } catch (error) {
            console.error('Failed to export lifecycle timeline:', error);
        }
    }

    handleMetricsError(type, error) {
        console.warn(`${type} metrics update failed:`, error);
        
        // Update status elements to show error
        const statusElements = {
            'violation': 'violation-metrics-status',
            'semantic': 'semantic-metrics-status',
            'capability-anchor': 'cap-anchor-status',
            'poa': 'poa-metrics-updated',
            'authz': 'authz-metrics-panel',
            'policy': 'policy-latency-updated',
            'decision': 'decision-metrics-status',
            'audit': 'audit-preview-status',
            'lifecycle': 'lifecycle-timeline-status',
            'registry': 'cap-registry-hash'
        };

        const elementId = statusElements[type];
        if (elementId) {
            const element = document.getElementById(elementId);
            if (element) {
                element.textContent = 'Error';
                element.className += ' text-red-600';
            }
        }
    }

    // Cleanup method
    destroy() {
        // Clear all intervals
        this.updateIntervals.forEach(interval => clearInterval(interval));
        this.updateIntervals.clear();

        // Close all event streams
        this.eventStreams.forEach(stream => {
            if (stream && typeof stream.close === 'function') {
                stream.close();
            }
        });
        this.eventStreams.clear();

        // Clear charts
        this.charts.clear();
        
        // Clear cache
        this.metricsCache.clear();
    }
}

// Initialize metrics monitor when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    if (window.AgentAuthAPI) {
        window.gAuthMetrics = new AgentAuthMetricsMonitor(window.AgentAuthAPI);
        console.log('AgentAuth Metrics Monitor initialized');
    } else {
        console.error('AgentAuth API Client not available for metrics');
    }
});

console.log('AgentAuth Metrics Monitor loaded');