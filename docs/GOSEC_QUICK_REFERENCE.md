# Gosec Quick Reference Card

**🔍 For Developers: How to Work with Gosec Security Scanning**

---

## TL;DR

```bash
# Test locally before pushing
./scripts/test-gosec.sh  # (if available)

# Or manually:
gosec -no-fail -exclude-dir=examples -exclude-dir=test ./...
```

**CI will NOT fail** if gosec panics - your PRs are safe! ✅

---

## Quick Commands

### Run Gosec Locally (CI-Like)

```bash
go mod download
go mod verify
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec -no-fail -fmt sarif -out gosec-results.sarif \
  -exclude-dir=examples -exclude-dir=test -exclude-dir=tests \
  -exclude-dir=cmd/examples ./...
```

### Check Results

```bash
# If you have jq installed
jq '.runs[0].results | length' gosec-results.sarif

# Manual check
cat gosec-results.sarif | grep -c '"ruleId"'
```

### View Specific Issues

```bash
jq -r '.runs[0].results[] | 
  "[\(.level)] \(.ruleId): \(.message.text) 
   File: \(.locations[0].physicalLocation.artifactLocation.uri)
   Line: \(.locations[0].physicalLocation.region.startLine)"' \
  gosec-results.sarif
```

---

## Understanding Gosec in CI

### What Happens When You Push?

```
1. CI starts → Build passes → Security scan runs
2. Gosec analyzes your code
   ├─ Success? → Upload results to Security tab
   └─ Panic? → Continue anyway (no failure!) ✅
3. Your PR status: ✅ Green (always)
```

### Why Won't It Fail My PR?

We use **triple error handling**:
1. `continue-on-error: true` (GitHub Actions level)
2. `|| true` (shell level)
3. `exit 0` (forced success)

**Result:** Gosec issues are **warnings, not blockers**.

---

## Suppressing False Positives

### Option 1: Inline Comment

```go
// #nosec G404 - Using crypto/rand, not math/rand
token := generateSecureToken()
```

### Option 2: Block Suppression

```go
// #nosec G304 - Path is validated via allowlist
func loadConfig(path string) error {
    // ... safe file loading ...
}
```

### Option 3: Update `.gosec.json`

```json
{
  "exclude": [
    "G104",  // Unhandled errors
    "G404"   // Add your rule here
  ]
}
```

---

## Common Gosec Rules

| Rule | Description | When to Suppress |
|------|-------------|------------------|
| **G101** | Hardcoded credentials | Never - fix it! |
| **G104** | Unhandled errors | When using `defer` cleanup |
| **G201** | SQL injection | Never - use parameterized queries |
| **G204** | Subprocess with variable | When input is validated |
| **G304** | File path from variable | When path is from safe config |
| **G401** | Weak crypto (MD5/SHA1) | Never - use SHA256+ |
| **G404** | Weak random | When using `crypto/rand` |
| **G115** | Integer overflow | Go 1.25+ often false positive |

**Rule of Thumb:**
- 🔴 **Never suppress:** G101, G201, G401, G501
- 🟡 **Carefully suppress:** G104, G204, G304
- 🟢 **Often false positive:** G404, G115 (in Go 1.25+)

---

## Gosec in Different Workflows

### CI Workflow (`.github/workflows/ci.yml`)
- **Format:** SARIF
- **Output:** `results.sarif`
- **Upload:** GitHub Security tab
- **When:** Every push/PR

### Staging Deploy (`.github/workflows/deploy-staging.yml`)
- **Format:** JSON
- **Output:** `gosec-report.json`
- **Upload:** Artifact (90 days retention)
- **When:** Deploy to staging

---

## Configuration Files

### `.gosec.json` (Project Root)

```json
{
  "global": {
    "exclude-generated": true,
    "confidence": "medium",
    "severity": "medium"
  },
  "exclude-dirs": [
    "examples",      // Demo code
    "test",          // Test files
    "tests",         // Test files (alt)
    "vendor",        // Dependencies
    "cmd/examples"   // CLI examples
  ],
  "exclude": [
    "G104",  // Unhandled errors (defer cleanup)
    "G204",  // Subprocess (validated inputs)
    "G304",  // File paths (safe config)
    "G404",  // Weak random (crypto/rand usage)
    "G115"   // Integer overflow (Go 1.25+ FP)
  ]
}
```

**To modify:** Edit this file, commit, push. CI will use new rules automatically.

---

## Checking Security Results

### GitHub UI

1. Go to **Security** tab → **Code scanning**
2. Filter by **Tool: gosec**
3. View alerts by severity

### GitHub CLI

```bash
# List all gosec alerts
gh api repos/{owner}/{repo}/code-scanning/alerts \
  --jq '.[] | select(.tool.name == "gosec")'

# Count by severity
gh api repos/{owner}/{repo}/code-scanning/alerts \
  --jq '[.[] | select(.tool.name == "gosec")] | 
        group_by(.rule.severity) | 
        map({severity: .[0].rule.severity, count: length})'
```

---

## Troubleshooting

### "SARIF file not generated"

**Cause:** Gosec panicked before writing results.

**Solution:**
```bash
# Try JSON format instead
gosec -no-fail -fmt json -out test.json ./...
cat test.json | jq .
```

### "403 Forbidden" on SARIF upload

**Cause:** GitHub Advanced Security not enabled.

**Solution:** Ask admin to enable, or results go to artifacts instead.

### "Too many false positives"

**Solution:** Update `.gosec.json` to exclude more rules.

---

## Best Practices

### ✅ Do's

- ✅ Run gosec locally before pushing
- ✅ Suppress false positives with `#nosec` + reason
- ✅ Check Security tab weekly
- ✅ Fix G101 (credentials) immediately
- ✅ Use `crypto/rand`, not `math/rand`

### ❌ Don'ts

- ❌ Don't suppress G101 (credentials)
- ❌ Don't ignore G201 (SQL injection)
- ❌ Don't suppress G401 (weak crypto)
- ❌ Don't use `#nosec` without a reason comment
- ❌ Don't suppress all G104 (some are real issues)

---

## When to Escalate

**Contact DevSecOps if:**
1. Gosec consistently panics on new package
2. Security tab shows 100+ new issues after your PR
3. You're unsure if a G101 is a real credential
4. Need help understanding a security finding

**Slack:** `#devsecops` or `#security`

---

## Quick Cheat Sheet

```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Scan everything
gosec ./...

# Scan with CI config
gosec -no-fail -exclude-dir=examples -exclude-dir=test ./...

# Scan specific package
gosec ./pkg/oidc

# Output to SARIF
gosec -fmt sarif -out results.sarif ./...

# Exclude specific rules
gosec -exclude=G104,G404 ./...

# Show only high severity
gosec -severity high ./...

# Quiet mode (errors only)
gosec -quiet ./...
```

---

## Resources

- **Full Documentation:** `docs/CI_GOSEC_COMPREHENSIVE_FIX.md`
- **Gosec Rules:** https://github.com/securego/gosec#rules
- **GitHub Security:** https://github.com/{org}/agentauth/security/code-scanning

---

**Last Updated:** 2025-01-19  
**Maintainer:** DevSecOps Team  
**Questions?** See `docs/CI_GOSEC_COMPREHENSIVE_FIX.md` or ask in `#devsecops`
