# Marketing Assets Gallery

This document collects the high-fidelity visual assets generated for the AgentAuth launch campaign.

## Social Media Visuals

````carousel
![OAuth vs AgentAuth Infographic](assets/oauth_vs_agentauth_infographic_v1.png)
<!-- slide -->
![AgentAuth $25 Trillion Stat Card](assets/agentauth_stat_card_25t_v1.png)
````

## Asset Descriptions

### 1. The OAuth Trap Comparison
**Use Case**: Twitter Thread 1, LinkedIn Post 1 (Launch Announcement).
**Purpose**: Visually demonstrate the security gap between standard OAuth (Access only) and AgentAuth (Legal Authority). The neon "Trap" vs "Shield" design emphasizes safety.

### 2. $25 Trillion Economy Stat Card
**Use Case**: Instagram Stat Card, Pitch Decks.
**Purpose**: High-impact financial hook to capture investor and executive attention. Highlights the massive market opportunity and the urgent need for infrastructure upgrades.

### Proof of Rigor: Fuzzing Verification (2026-01-04)

**Executed:** `go test -fuzz=FuzzSignatureVerification -fuzztime=10s -v ./pkg/agentauth_aap_001`
**Result:** 33,902 executions in 12s (No Failures)

```text
fuzz: elapsed: 0s, gathering baseline coverage: 3/3 completed, now fuzzing with 11 workers
fuzz: elapsed: 3s, execs: 22836 (7612/sec), new interesting: 31 (total: 34)
fuzz: elapsed: 6s, execs: 33902 (3688/sec), new interesting: 49 (total: 52)
--- PASS: FuzzSignatureVerification (12.01s)
PASS
ok      github.com/mauriciomferz/AgentAuth/pkg/agentauth_aap_001        18.075s
```
