# AI Bidding Platform

## What This Is

An autonomous AI auction system where bot agents with distinct personalities compete against each other in real-time auctions. Four LLM-powered bots (backed by Gemini Flash via Google ADK) bid on sequentially created auctions; a React dashboard shows live auction state and bid activity via WebSocket. The platform is a portfolio/demo system exploring emergent behaviour in constrained AI agents.

## Core Value

Bots with real financial constraints and inventory behave more interestingly than bots with unlimited funds — the balance gate, buyout mechanic, and item ownership make each auction outcome meaningful.

## Requirements

### Validated

- ✓ Auction lifecycle management (create, start, end, winner determination) — existing
- ✓ Bid placement with Redis distributed locking (one bid at a time per auction) — existing
- ✓ Four LLM bot agents with distinct personalities competing via RabbitMQ fanout — existing
- ✓ BFF WebSocket gateway delivering live auction state and bid events to the dashboard — existing
- ✓ React dashboard showing active auctions, bid feed, and winner banners — existing
- ✓ Event-driven architecture: RabbitMQ topic exchange with bff.q and bot.q consumers — existing
- ✓ Per-service PostgreSQL databases with bun ORM schema creation at boot — existing
- ✓ Deployed on Railway (services) + Vercel (frontend) — existing

### Active

- [ ] Banking service with £1,000,000 starting balance per bot
- [ ] Persistent wallet tracking (balance debited on auction win)
- [ ] Item inventory: bots accumulate won items with purchase price and title
- [ ] Buyout mechanic: bots sell most expensive item to bank at 70% when balance < auction start price
- [ ] Balance gate in bot agent: check wallet before bidding; degrade gracefully if banking service is down
- [ ] `auction.ended` event consumer in banking service records wins atomically (debit wallet + insert item)
- [ ] `AuctionEndedPayload` extended with `Title` field so banking service can name items
- [ ] REST API: `GET /api/wallets/:bot_id`, `POST /api/banking/buyout/:item_id`
- [ ] Wallet seed idempotent on restart (ON CONFLICT DO NOTHING — never resets live balances)

### Out of Scope

- Human user accounts or authentication — bots only, no human bidders in this system
- Bot-to-bot trading — only bank buyouts at fixed 70% rate
- Multi-currency or fractional balances — GBP float64 for now (see CONCERNS.md float precision issue)
- DLQ / dead-letter exchange — known gap, not addressed in this milestone

## Context

The codebase map (`.planning/codebase/`) provides full architectural detail. Key context for this milestone:

- Bot IDs are pre-known constants (1–4); banking service seeds wallets for these IDs at startup
- `bot-service` currently has an `isSpendingCapExhausted` check that gates Gemini API spend — the balance gate is a separate, domain-level concept that gates bidding entirely
- `AuctionEndedPayload` in `shared/events/auction_events.go` is the only shared event type that needs extending
- The `go.work` workspace file must be updated to include the new `banking-service` module
- Banking service port: 8084 (sequential after existing 8080–8083)
- Graceful degradation is explicit: if banking service is unreachable, bots continue bidding without balance check

## Constraints

- **Tech stack**: Go (match existing services — Gin, bun, zap, RabbitMQ amqp091-go). No new frameworks
- **Compatibility**: Must not break existing services; shared event schema changes are additive only
- **Pattern**: Follow per-service internal layout exactly (domain → ports → service → repo → http → mq)
- **No LLM dependency**: Banking service has no Gemini/ADK dependency — pure data service
- **Atomicity**: `RecordWin` and `Buyout` must be DB transactions (wallet update + item insert/delete as one unit)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 70% buyout rate (fixed) | Simple mechanic; no negotiation complexity needed | — Pending |
| ON CONFLICT DO NOTHING for wallet seed | Protects live balances from accidental reset on restart | — Pending |
| Graceful degradation on banking service down | Bots should never be completely blocked by infrastructure failure | — Pending |
| Float64 for balance (not NUMERIC) | Consistent with existing bid amounts (known tech debt, not fixed here) | — Pending |
| `withTx(tx)` pattern on repos | Enables atomic multi-repo operations without leaking transaction handling into service layer | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-05-19 after initialization*
