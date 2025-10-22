// main.js - unified initialization registry for revamped beta UI
// Provides ordered lifecycle: early DOM init (theme/nav), metrics panels, then heavier streams.

import { initDecisionMetricsPanel } from './decision_metrics.js';
import { initGovernancePanels } from './governance.js';
import { initRotationSummaryPanel } from './rotation_summary.js';
import { initViolationMetricsPanel } from './violation_metrics.js';
import { initSemanticMetricsPanel } from './semantic_metrics.js';
import { initCapabilityAnchor } from './capability_anchor.js';
import { initRevocationTransparency } from './revocation_transparency.js';
import { initMultiSigPanel } from './multisig_panel.js';

function initThemeToggle(){
  const btn = document.getElementById('themeToggle');
  if(!btn) return;
  btn.addEventListener('click', ()=>{
    const body = document.body;
    const dark = body.classList.toggle('theme-dark');
  try { initCapabilityAnchor(); } catch (e) { console.error('capability anchor init failed', e); }
  try { initRevocationTransparency(); } catch (e) { console.error('revocation transparency init failed', e); }
    btn.innerHTML = dark ? '<i class="fas fa-moon mr-2" aria-hidden="true"></i>Dark' : '<i class="fas fa-sun mr-2" aria-hidden="true"></i>Light';
  });
}

function initMobileNav(){
  const button = document.getElementById('mobileNavButton');
  const menu = document.getElementById('mobileNavMenu');
  if(!button || !menu) return;
  button.addEventListener('click', ()=>{
    const expanded = button.getAttribute('aria-expanded') === 'true';
    button.setAttribute('aria-expanded', (!expanded).toString());
    menu.classList.toggle('hidden', expanded);
  });
}

function initTabSystem(){
  const buttons = document.querySelectorAll('.tab-button');
  buttons.forEach(btn => {
    btn.addEventListener('click', ()=>{
      const target = btn.getAttribute('data-tab');
      if(!target) return;
      buttons.forEach(b=>{ b.classList.remove('active'); b.setAttribute('aria-selected','false'); });
      btn.classList.add('active');
      btn.setAttribute('aria-selected','true');
      document.querySelectorAll('.tab-content').forEach(c=>{
        if(c.id === target){ c.classList.remove('hidden'); } else { c.classList.add('hidden'); }
      });
    });
  });
}

function initPanels(){
  initDecisionMetricsPanel();
  initGovernancePanels();
  initRotationSummaryPanel();
  initViolationMetricsPanel();
  initSemanticMetricsPanel();
  initCapabilityAnchor();
  initRevocationTransparency();
  initMultiSigPanel();
}

// Public API (attach to window for backward compatibility testing)
export function initAll(){
  initThemeToggle();
  initMobileNav();
  initTabSystem();
  initPanels();
}

// Auto-run when DOM ready
document.addEventListener('DOMContentLoaded', ()=>{ initAll(); });
// Entry module bridging legacy app.js to new modular structure.
// Progressive migration: feature code will move from app.js into dedicated modules imported here.
import { currentToken, demoState, setCurrentToken, addAuditEntry } from "./state.js";
import { addConsoleOutput, clearConsoleOutput, escapeHtml } from "./console.js";
import { initTokens } from "./tokens.js";
import { metricsInit } from "./metrics.js";
import { auditInit } from "./audit.js";
import { samplesInit } from "./samples.js";
import { authzInit } from "./authz.js";
import { jobLogsInit } from "./joblogs.js";
import { policyInit } from "./policy.js";

// Re-export for other modules / potential future bundler
export { currentToken, demoState, setCurrentToken, addAuditEntry, addConsoleOutput, clearConsoleOutput, escapeHtml };

// Hook for future initialization (e.g., attaching event listeners once DOM ready)
export function initModules() {
  console.log("[GAuth] Initializing modules...");
  try { initTokens(); console.log("[GAuth] tokens ok"); } catch (e) { console.warn("Token module init failed", e); }
  try { metricsInit(); console.log("[GAuth] metrics ok"); } catch (e) { console.warn("Metrics module init failed", e); }
  try { auditInit(); console.log("[GAuth] audit ok"); } catch (e) { console.warn("Audit module init failed", e); }
  try { samplesInit(); console.log("[GAuth] samples ok"); } catch (e) { console.warn("Samples module init failed", e); }
  try { authzInit(); console.log("[GAuth] authz ok"); } catch (e) { console.warn("Authz module init failed", e); }
  try { jobLogsInit(); console.log("[GAuth] joblogs ok"); } catch (e) { console.warn("JobLogs module init failed", e); }
  try { policyInit(); console.log("[GAuth] policy ok"); } catch (e) { console.warn("Policy module init failed", e); }
  return true;
}

// For immediate backward compatibility, attach a minimal namespace (can be removed later)
window.GAuth = window.GAuth || { state: demoState, setCurrentToken, addAuditEntry, addConsoleOutput };
// Provide legacy compatibility alias
window.GAuth.it = initModules;

// Auto-init when DOM ready (safe idempotent)
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => {
    if (!window.__gauthInitialized) {
      window.__gauthInitialized = true;
      initModules();
    }
  });
} else {
  if (!window.__gauthInitialized) {
    window.__gauthInitialized = true;
    initModules();
  }
}
