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

function initThemeToggle(){
  const btn = document.getElementById('themeToggle');
  if(!btn) return;
  btn.addEventListener('click', ()=>{
    const body = document.body;
    const dark = body.classList.toggle('theme-dark');
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
  console.log('[AgentAuth] Initializing tab system...');
  const buttons = document.querySelectorAll('.tab-button');
  console.log('[AgentAuth] Found', buttons.length, 'tab buttons');
  buttons.forEach(btn => {
    btn.addEventListener('click', ()=>{
      const target = btn.getAttribute('data-tab');
      console.log('[AgentAuth] Tab clicked:', target);
      if(!target) return;
      buttons.forEach(b=>{ b.classList.remove('active'); b.setAttribute('aria-selected','false'); });
      btn.classList.add('active');
      btn.setAttribute('aria-selected','true');
      document.querySelectorAll('.tab-content').forEach(c=>{
        if(c.id === target){ 
          c.classList.remove('hidden'); 
          console.log('[AgentAuth] Showing tab:', target);
        } else { 
          c.classList.add('hidden'); 
        }
      });
    });
  });
}

function initPanels(){
  try { initDecisionMetricsPanel(); } catch (e) { console.warn('Decision metrics init failed', e); }
  try { initGovernancePanels(); } catch (e) { console.warn('Governance init failed', e); }
  try { initRotationSummaryPanel(); } catch (e) { console.warn('Rotation summary init failed', e); }
  try { initViolationMetricsPanel(); } catch (e) { console.warn('Violation metrics init failed', e); }
  try { initSemanticMetricsPanel(); } catch (e) { console.warn('Semantic metrics init failed', e); }
  try { initCapabilityAnchor(); } catch (e) { console.warn('Capability anchor init failed', e); }
  try { initRevocationTransparency(); } catch (e) { console.warn('Revocation transparency init failed', e); }
  try { initMultiSigPanel(); } catch (e) { console.warn('MultiSig panel init failed', e); }
}

export function initModules() {
  console.log("[AgentAuth] Initializing feature modules...");
  try { initTokens(); console.log("[AgentAuth] ✓ tokens"); } catch (e) { console.error("❌ Token module init failed:", e); }
  try { metricsInit(); console.log("[AgentAuth] ✓ metrics"); } catch (e) { console.error("❌ Metrics module init failed:", e); }
  try { auditInit(); console.log("[AgentAuth] ✓ audit"); } catch (e) { console.error("❌ Audit module init failed:", e); }
  try { samplesInit(); console.log("[AgentAuth] ✓ samples"); } catch (e) { console.error("❌ Samples module init failed:", e); }
  try { authzInit(); console.log("[AgentAuth] ✓ authz"); } catch (e) { console.error("❌ Authz module init failed:", e); }
  try { jobLogsInit(); console.log("[AgentAuth] ✓ joblogs"); } catch (e) { console.error("❌ JobLogs module init failed:", e); }
  try { policyInit(); console.log("[AgentAuth] ✓ policy"); } catch (e) { console.error("❌ Policy module init failed:", e); }
  console.log("[AgentAuth] ✅ Module initialization complete");
  return true;
}

// Public API (attach to window for backward compatibility testing)
export function initAll(){
  console.log('[AgentAuth] ========================================');
  console.log('[AgentAuth] Starting initialization...');
  console.log('[AgentAuth] ========================================');
  initThemeToggle();
  initMobileNav();
  initTabSystem();
  initPanels();
  initModules();
  console.log('[AgentAuth] ========================================');
  console.log('[AgentAuth] ✅ Initialization complete!');
  console.log('[AgentAuth] ========================================');
}

// For immediate backward compatibility, attach a minimal namespace
window.AgentAuth = window.AgentAuth || { state: demoState, setCurrentToken, addAuditEntry, addConsoleOutput };
window.AgentAuth.initModules = initModules;

// Auto-init when DOM ready (safe idempotent)
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => {
    if (!window.__gauthInitialized) {
      window.__gauthInitialized = true;
      initAll();
    }
  });
} else {
  if (!window.__gauthInitialized) {
    window.__gauthInitialized = true;
    initAll();
  }
}
