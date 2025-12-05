// Shared demo state and token helpers (initial modularization)
export let currentToken = null;
export const demoState = {
  tokenCreated: false,
  subscriptionsActive: false,
  auditEntries: [],
};

// Token metrics tracking
export const tokenMetrics = {
  created: 0,
  validated: 0,
  revoked: 0,
  get total() {
    return this.created + this.validated + this.revoked;
  }
};

export function incrementTokenMetric(metric) {
  if (metric in tokenMetrics && metric !== 'total') {
    tokenMetrics[metric]++;
  }
}

export function setCurrentToken(tok) {
  currentToken = tok;
  demoState.tokenCreated = !!tok;
}

export function addAuditEntry(entry) {
  demoState.auditEntries.push({ ...entry, at: entry.at || new Date().toISOString() });
  if (demoState.auditEntries.length > 1000) demoState.auditEntries.shift();
}
