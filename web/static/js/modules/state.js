// Shared demo state and token helpers (initial modularization)
export let currentToken = null;
export const demoState = {
  tokenCreated: false,
  subscriptionsActive: false,
  auditEntries: [],
};

export function setCurrentToken(tok) {
  currentToken = tok;
  demoState.tokenCreated = !!tok;
}

export function addAuditEntry(entry) {
  demoState.auditEntries.push({ ...entry, at: entry.at || new Date().toISOString() });
  if (demoState.auditEntries.length > 1000) demoState.auditEntries.shift();
}
