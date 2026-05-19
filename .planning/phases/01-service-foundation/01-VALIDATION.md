---
phase: 1
slug: service-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-19
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + testify v1.11.1 |
| **Config file** | none — go test standard |
| **Quick run command** | `cd services/banking-service && go build ./... && go vet ./...` |
| **Full suite command** | `cd services/banking-service && go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd services/banking-service && go build ./... && go vet ./...`
- **After every plan wave:** Run `cd services/banking-service && go test ./internal/domain/...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | INF-04 | — | N/A | shell check | `grep "banking-service" go.work` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | INF-01 | — | N/A | build check | `cd services/banking-service && go build ./...` | ❌ W0 | ⬜ pending |
| 01-01-03 | 01 | 1 | INF-01 | — | N/A | unit | `go test ./internal/domain/... -run TestSentinelErrors` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `services/banking-service/internal/domain/errors_test.go` — verifies all 4 sentinel errors (`ErrWalletNotFound`, `ErrItemNotFound`, `ErrInsufficientBalance`, `ErrItemNotOwnedByBot`) are referenceable and distinct (errors.Is does not cross-match them)

*(No existing test infrastructure for this new service — Wave 0 must create the test file alongside the implementation.)*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| go.vet passes | INF-01 | Not a Go test — it's a build tool check | `cd services/banking-service && go vet ./...` |
| workspace resolves module | INF-04 | Not a Go test | `grep "banking-service" go.work` in repo root |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
