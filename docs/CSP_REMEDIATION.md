---
title: Csp Remediation
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# CSP Remediation and Best Practices

> Last Updated: 2025-10-17
> Status: Active

## Summary of Changes

1. **Externalized All Inline Scripts**
   - All `<script>` blocks previously embedded in `index.html` were moved to dedicated static JS files (`log_stream_panel.js`, `job_panel.js`, and updates to `app.js`).
   - The HTML now references these files using `<script src=...>` tags, ensuring no inline JavaScript is executed.

2. **Removed Inline Event Handlers and Styles**
   - All `onclick` attributes were replaced with `data-action` and `data-example` attributes.
   - Event listeners are now attached in external JS (`app.js`) using `addEventListener`, fully decoupling behavior from markup.
   - All inline `style` attributes related to dynamic behavior were removed or replaced with CSS classes.

3. **CSP Compliance Hardening**
    - Introduced a per-request nonce for scripts (added server-side in `web/server_clean.go`).
    - Removed `unsafe-inline` entirely from both `script-src` and `style-src` directives.
    - Final effective CSP (representative example – nonce value changes per request):
       ```
       Content-Security-Policy: \
          default-src 'self' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; \
          script-src 'self' 'nonce-<RANDOM_NONCE>' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; \
          style-src 'self' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; \
          font-src 'self' https://cdnjs.cloudflare.com data:; \
          img-src 'self' data:; \
          connect-src 'self'; \
          frame-ancestors 'none'; \
          base-uri 'self'
       ```
    - Legacy `web/server.go` (excluded by build tags) was sanitized to remove lingering `unsafe-inline` so future resurrected builds stay compliant.
    - A verification script (`scripts/verify_csp.sh`) plus a Makefile target (`make verify-csp`) and CI workflow enforce policy.

## Best Practices for CSP Compliance

- **Never use inline `<script>` or `onclick`/`onload`/etc. attributes.**
- **Bind all event handlers in external JS files using `addEventListener`.**
- **Reference all scripts and stylesheets with `src` and `href` attributes.**
- **Avoid inline `style` attributes for dynamic styling; use CSS classes instead.**
- **Use a per-request CSP nonce only for allowed inline script placeholders you intentionally keep (currently none).**
- **Test with the browser console open to catch any CSP violations.**

## Key Files
- `web/server_clean.go`: Defines nonce-based CSP header (hardened, no `unsafe-inline`).
- `scripts/verify_csp.sh`: Automated CSP verification (retries, fallback paths, detection of dangerous patterns).
- `Makefile`: Added `build-web`, `run-web`, and `verify-csp` targets for clarity.
- `static/js/*.js`: Externalized behaviors (no inline scripts or event handlers).
- `web/server.go`: Legacy, excluded, sanitized for consistency.

## Verification Steps
1. Build & run the web server:
   ```bash
   make run-web
   ```
2. In another terminal, run the CSP check:
   ```bash
   make verify-csp
   ```
3. Expect output: `PASS: CSP verification passed ...` with no `unsafe-inline` / `unsafe-eval` findings.
4. Open browser dev tools (Network + Console) and confirm no CSP violations.
5. Trigger SSE log streaming – ensure connection succeeds (allowed by `connect-src 'self'`).

---

*Remediation performed on October 11, 2025 by GitHub Copilot. Final hardening (removal of `unsafe-inline` in style-src) completed October 11, 2025.*

## Post-Hardening Audit (Unsafe-Eval & String Execution)

Date: October 11, 2025

An automated repository scan was executed to ensure no JavaScript patterns require `unsafe-eval`:

Patterns Searched:
- `eval(`
- `new Function`
- `setTimeout("` / `setTimeout('` (string-based execution)
- `setInterval("` / `setInterval('`

Findings:
- No occurrences of `eval(` or `new Function` in any `*.js` file.
- All `setTimeout` / `setInterval` usages employ function callbacks (lambdas) — none use string arguments.
- No dynamically constructed script execution primitives found.

Conclusion: The current CSP correctly omits `unsafe-eval` and no application code depends on it. Adding `unsafe-eval` would only weaken security with no functional benefit.

### Ongoing Monitoring

To keep this guarantee, a helper script will be added (`scripts/scan_unsafe_js.sh`). Integrate it into CI or a pre-commit hook to fail builds if unsafe patterns creep in.

Example invocation:
```bash
scripts/scan_unsafe_js.sh
```

Expected output (clean state):
```
Scan complete: no unsafe JavaScript execution patterns detected.
```

If any findings appear, refactor to remove them rather than relaxing the CSP.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
