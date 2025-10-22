// tokens.js - token creation/validation/revocation logic (extracted from legacy app.js)
import { currentToken, demoState, setCurrentToken, addAuditEntry } from "./state.js";
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
		addConsoleOutput("token-output", `✓ Token ${currentToken.id} revoked successfully`, "success");
		addAuditEntry({ action: "TOKEN_REVOKED", tokenId: currentToken.id, beta: true });
		setCurrentToken(null);
	} catch (error) {
		addConsoleOutput("token-output", `✗ Token revocation failed: ${error.message}`, "error");
	}
}

export function initTokens() {
	// Wire existing buttons by data-action
	const map = { "create-token": createToken, "validate-token": validateToken, "revoke-token": revokeToken };
	Object.entries(map).forEach(([a, fn]) => {
		document.querySelectorAll(`[data-action="${a}"]`).forEach(el => el.addEventListener("click", fn));
	});
}

// Legacy global bridge (will be removed later)
window.GAuth = window.GAuth || {}; Object.assign(window.GAuth, { createToken, validateToken, revokeToken });
