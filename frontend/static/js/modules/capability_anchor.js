// capability_anchor.js - renders capability anchor status panel
// Periodically fetches /api/v1/beta/capabilities/anchor/status and updates DOM.
// Uses backoffFetchJSON from refresh.js. Refresh interval: 20s.

import { backoffFetchJSON } from './refresh.js';

const REFRESH_MS = 20000;

function formatAge(sec) {
  if (typeof sec !== 'number' || isNaN(sec) return '—';
  if (sec < 60) return sec.toFixed(0) + 's';
  if (sec < 3600) return (sec/60).toFixed(1) + 'm';
  return (sec/3600).toFixed(1) + 'h';
}

export function initCapabilityAnchor() {
  const panel = document.getElementById('capability-anchor-panel');
  if (!panel) return;
  const statusEl = document.getElementById('cap-anchor-status');
  const hashEl = document.getElementById('cap-anchor-registry-hash');
  const lastWriteEl = document.getElementById('cap-anchor-last-write');
  const ageEl = document.getElementById('cap-anchor-age');
  const staleBadge = document.getElementById('cap-anchor-stale-badge');
  const emittedEl = document.getElementById('cap-anchor-emitted');
  const skippedEl = document.getElementById('cap-anchor-skipped');
  const changedEl = document.getElementById('cap-anchor-hash-changed');
  const notarizedAtEl = document.getElementById('cap-anchor-notarized-at');
  const notarizedAgeEl = document.getElementById('cap-anchor-notarized-age');
  const providerEl = document.getElementById('cap-anchor-notary-provider');

  async function refresh() {
    statusEl.textContent = 'Refreshing…';
    try {
      const data = await backoffFetchJSON('/api/v1/beta/capabilities/anchor/status');
      if (!data || !data.success) {
        statusEl.textContent = 'Status unavailable';
        return;
      }
      statusEl.textContent = data.configured ? 'Configured' : 'Not configured';
      hashEl.textContent = data.registry_hash || '—';
      lastWriteEl.textContent = data.last_write || '—';
      ageEl.textContent = formatAge(data.age_seconds);
      if (data.stale) {
        staleBadge.classList.remove('hidden');
        staleBadge.textContent = 'Stale';
        staleBadge.className = 'px-2 py-0.5 rounded text-xs bg-red-600 text-white';
      } else {
        staleBadge.classList.remove('hidden');
        staleBadge.textContent = 'Fresh';
        staleBadge.className = 'px-2 py-0.5 rounded text-xs bg-green-600 text-white';
      }
      if (typeof data.emitted_total === 'number') emittedEl.textContent = data.emitted_total;
      if (typeof data.skipped_total === 'number') skippedEl.textContent = data.skipped_total;
      if (typeof data.hash_changed_total === 'number') changedEl.textContent = data.hash_changed_total;
      // Notarization fields (prototype)
      notarizedAtEl.textContent = data.last_notarized_at || '—';
      notarizedAgeEl.textContent = formatAge(data.notarized_age_seconds);
      providerEl.textContent = (data.notarization_receipt && data.notarization_receipt.provider) || data.notarization_provider || '—';
    } catch (e) {
      statusEl.textContent = 'Error fetching status';
      console.warn('[capability-anchor] fetch error', e);
    }
  }

  refresh();
  setInterval(refresh, REFRESH_MS);
}
