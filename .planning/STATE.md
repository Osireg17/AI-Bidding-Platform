# Project State: AI Bidding Platform — Banking Service Milestone

**Last updated:** 2026-05-19
**Updated by:** roadmapper (initial creation)

---

## Project Reference

**Core value:** Bots with real financial constraints and inventory behave more interestingly than bots with unlimited funds — the balance gate, buyout mechanic, and item ownership make each auction outcome meaningful.

**Current focus:** Banking service milestone — adding persistent wallets, item inventory, balance-gated bidding, and atomic win recording to the existing four-service platform.

---

## Current Position

**Phase:** 1 — Service Foundation
**Plan:** None started
**Status:** Not started
**Progress:** ░░░░░░░░░░ 0% (0/5 phases complete)

---

## Milestone Summary

| Phase | Name | Status |
|-------|------|--------|
| 1 | Service Foundation | Not started |
| 2 | Repository Layer | Not started |
| 3 | Service + API Layer | Not started |
| 4 | Event Consumer + Entrypoint | Not started |
| 5 | Bot Integration | Not started |

---

## Performance Metrics

**Plans executed:** 0
**Plans succeeded first try:** 0
**Phases complete:** 0 / 5
**Requirements complete:** 0 / 16

---

## Accumulated Context

### Key Decisions (locked)

| Decision | Rationale |
|----------|-----------|
| 70% buyout rate (fixed) | Simple mechanic; no negotiation complexity needed |
| ON CONFLICT DO NOTHING for wallet seed | Protects live balances from accidental reset on restart |
| Graceful degradation when banking service is down | Bots must never be fully blocked by infrastructure failure |
| Float64 for balance (not NUMERIC) | Consistent with existing bid amounts (known tech debt, not fixed here) |
| `withTx(tx)` pattern on repos | Enables atomic multi-repo operations without leaking transaction handling into service layer |
| EVT-01 (AuctionEndedPayload.Title) shipped in PR #34 | Title field already available in shared event schema — no schema change needed in this milestone |

### Active Constraints

- Banking service port: 8084 (next sequential after 8080–8083)
- No LLM/ADK dependency in banking-service — pure data service
- Go module path: `github.com/Osireg17/AI-Bidding-Platform/services/banking-service`
- Shared event schema changes must be additive only (no breaking changes to existing consumers)
- All DB operations in `RecordWin` and `Buyout` must be single transactions (bun `RunInTx`)
- Bot IDs are pre-known constants 1–4; wallet seed is hardcoded to these IDs

### Implementation Conventions (from codebase)

- Internal package layout: `cmd/<svc>/main.go` + `internal/{config,domain,service,repo,http,mq,observability,test}`
- Ports defined in `domain/ports.go`; concrete implementations in `repo/` and `mq/`
- Handler pattern: parse input → call service → map error → respond (never touch DB directly in handlers)
- Typed sentinel errors in `domain/errors.go`; handlers use `errors.Is()` switch to map to HTTP codes
- Dependency injection via constructors only; no global singletons; no `init()` side effects
- Structured logging with zap; every log line includes relevant domain IDs
- RabbitMQ: all events wrapped in `shared/events.Envelope`; consumers must be idempotent

### Todos

- [ ] Verify exact go version in go.work before writing banking-service go.mod (match workspace toolchain)
- [ ] Confirm `railway.json` schema by copying from `services/bid-service/railway.json` as pattern
- [ ] Check bot-service `NewBotAgent` signature before adding `bankingClient` param (affects all 4 agent init calls in main.go)

### Blockers

None currently.

---

## Session Continuity

### Last Action

Roadmap created. No implementation started.

### Next Action

Start Phase 1: create `services/banking-service/` directory structure, write `go.mod`, domain types (`wallet.go`, `item.go`, `errors.go`, `ports.go`), config, and update `go.work`.

### Context for Next Session

- EVT-01 is already complete — `AuctionEndedPayload.Title` exists in `shared/events/auction_events.go`
- Tickets BANK-1 and BANK-2 from `docs/plans/banking-service.md` are already done (PR #34)
- Start implementation from BANK-3 (`go.mod`) through BANK-7 (`ports.go`) + BANK-12 (`config.go`)
- Reference existing service for patterns: `services/bid-service/` for repo patterns, `services/auction-service/` for HTTP patterns, `services/bot-service/` for domain/config/consumer patterns

---
*State initialized: 2026-05-19*
