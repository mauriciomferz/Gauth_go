// tokens.js - token creation/validation/revocation logic (extracted from legacy app.js)
import { currentToken, demoState, setCurrentToken, addAuditEntry, tokenMetrics, incrementTokenMetric } from "./state.js";
import { addConsoleOutput } from "./console.js";

export async function createToken() {
	addConsoleOutput("token-output", "Initiating beta token creation...", "info");
	try {
		const res = await fetch("/api/v1/token/create", {
			method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ttl_seconds: 3600 })
		});
		const data = await res.json();
		if (!data.success) throw new Error(data.message || "Token creation failed");
		setCurrentToken(data.token);
		incrementTokenMetric('created');
		addConsoleOutput("token-output", "✓ Token created successfully", "success");
		addConsoleOutput("token-output", `  ID: ${currentToken.id}`, "info");
		addConsoleOutput("token-output", `  Expires: ${currentToken.expiresAt}`, "info");
		addConsoleOutput("token-output", "  ⚠️ Beta demo token - not cryptographically secure", "warning");
		addAuditEntry({ action: "TOKEN_CREATED", tokenId: currentToken.id, beta: true });
	} catch (error) {
		addConsoleOutput("token-output", `✗ Token creation failed: ${error.message}`, "error");
	}
}

export async function validateToken() {
	if (!currentToken) { addConsoleOutput("token-output", "✗ No token available for validation. Create a token first.", "error"); return; }
	addConsoleOutput("token-output", "Validating beta token...", "info");
	try {
		const res = await fetch("/api/v1/token/validate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ token_id: currentToken.id }) });
		const data = await res.json();
		if (!data.success) {
			addConsoleOutput("token-output", `✗ Token validation failed: ${data.status || data.message}`, "error");
			if (data.status === "expired" || data.status === "revoked") {
				setCurrentToken(null);
			}
			return;
		}
		incrementTokenMetric('validated');
		addConsoleOutput("token-output", "✓ Token validation successful", "success");
		addConsoleOutput("token-output", `  Valid until: ${data.token.expiresAt}`, "info");
		addAuditEntry({ action: "TOKEN_VALIDATED", tokenId: currentToken.id, valid: true, beta: true });
	} catch (error) {
		addConsoleOutput("token-output", `✗ Token validation failed: ${error.message}`, "error");
	}
}

export async function revokeToken() {
	if (!currentToken) { addConsoleOutput("token-output", "✗ No token available for revocation.", "error"); return; }
	addConsoleOutput("token-output", `Revoking token ${currentToken.id}...`, "info");
	try {
		const res = await fetch("/api/v1/token/revoke", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ token_id: currentToken.id }) });
		const data = await res.json();
		if (!data.success) { addConsoleOutput("token-output", `✗ Token revocation failed: ${data.status || data.message}`, "error"); return; }
		incrementTokenMetric('revoked');
		addConsoleOutput("token-output", `✓ Token ${currentToken.id} revoked successfully`, "success");
		addAuditEntry({ action: "TOKEN_REVOKED", tokenId: currentToken.id, beta: true });
		setCurrentToken(null);
	} catch (error) {
		addConsoleOutput("token-output", `✗ Token revocation failed: ${error.message}`, "error");
	}
}

export function showTokenMetrics() {
	const panel = document.getElementById('token-metrics-panel');
	if (!panel) return;
	
	panel.innerHTML = `
		<div class="bg-gray-100 rounded p-3 space-y-2">
			<div class="font-semibold text-gray-700">Token Operation Metrics:</div>
			<div class="grid grid-cols-2 gap-2">
				<div class="bg-white p-2 rounded">
					<div class="text-xs text-gray-500">Created</div>
					<div class="text-lg font-bold text-green-600">${tokenMetrics.created}</div>
				</div>
				<div class="bg-white p-2 rounded">
					<div class="text-xs text-gray-500">Validated</div>
					<div class="text-lg font-bold text-blue-600">${tokenMetrics.validated}</div>
				</div>
				<div class="bg-white p-2 rounded">
					<div class="text-xs text-gray-500">Revoked</div>
					<div class="text-lg font-bold text-red-600">${tokenMetrics.revoked}</div>
				</div>
				<div class="bg-white p-2 rounded">
					<div class="text-xs text-gray-500">Total Operations</div>
					<div class="text-lg font-bold text-purple-600">${tokenMetrics.total}</div>
				</div>
			</div>
			<div class="text-xs text-gray-500 mt-2">
				Metrics track token operations during this session
			</div>
		</div>
	`;
}

export function initTokens() {
	// Wire existing buttons by data-action
	const map = { "create-token": createToken, "validate-token": validateToken, "revoke-token": revokeToken };
	Object.entries(map).forEach(([a, fn]) => {
		document.querySelectorAll(`[data-action="${a}"]`).forEach(el => el.addEventListener("click", fn));
	});
	
	// Wire Show Token Metrics button
	const showMetricsBtn = document.getElementById('show-token-metrics');
	if (showMetricsBtn) {
		showMetricsBtn.addEventListener('click', showTokenMetrics);
	}
}

// Legacy global bridge (will be removed later)
window.GAuth = window.GAuth || {}; Object.assign(window.GAuth, { createToken, validateToken, revokeToken });
