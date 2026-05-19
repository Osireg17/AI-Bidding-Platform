# Requirements: AI Bidding Platform — Banking Service Milestone

**Defined:** 2026-05-19
**Core Value:** Bots with real financial constraints and inventory behave more interestingly than bots with unlimited funds — the balance gate, buyout mechanic, and item ownership make each auction outcome meaningful.

## Validated Requirements

Requirements already shipped and confirmed working.

### Shared Event Contract

- ✓ **EVT-01**: `AuctionEndedPayload` includes `Title string` field populated by auction-service publisher — merged PR #34

## v1 Requirements

Requirements for this milestone. Each maps to a roadmap phase.

### Banking Service — Infrastructure

- [ ] **INF-01**: Banking service compiles and boots as a standalone Go microservice (port 8084, `GET /health`)
- [ ] **INF-02**: Banking service connects to its own PostgreSQL database and creates wallet/item tables at boot (idempotent)
- [ ] **INF-03**: Banking service connects to RabbitMQ and binds `banking.q` to `auction.ended` routing key
- [ ] **INF-04**: `go.work` workspace updated to include the new `banking-service` module
- [ ] **INF-05**: `railway.json` deploy config created for banking service

### Banking Service — Wallet

- [ ] **WALL-01**: Four wallets (bot IDs 1–4) are seeded at £1,000,000 each on startup; seed is idempotent (ON CONFLICT DO NOTHING — never resets live balances)
- [ ] **WALL-02**: `GET /api/wallets/:bot_id` returns current balance and full item list for a bot (404 if wallet not found)
- [ ] **WALL-03**: Wallet balance is debited atomically when a bot wins an auction (wallet update + item insert in one DB transaction)

### Banking Service — Item Inventory

- [ ] **ITEM-01**: Item record captures bot ID, auction ID, item title, and purchase price when a bot wins an auction
- [ ] **ITEM-02**: `POST /api/banking/buyout/:item_id` sells an item to the bank at 70% of purchase price, atomically crediting the wallet and removing the item (404 if item not found)

### Bot Integration

- [ ] **BOT-01**: Bot agent fetches wallet before bidding; if balance ≥ auction start price, proceeds normally
- [ ] **BOT-02**: If balance < start price and bot owns items, bot sells its most expensive item at 70% payout, then re-evaluates affordability
- [ ] **BOT-03**: If balance < start price and bot owns no items, bot skips the auction
- [ ] **BOT-04**: If banking service is unreachable, bot proceeds without a balance check (graceful degradation — never fully blocked by infrastructure failure)
- [ ] **BOT-05**: Current balance is injected into the LLM prompt so Gemini is aware of the bot's financial position when deciding bid amounts

## v2 Requirements

Deferred. Acknowledged but not in current roadmap.

### Observability

- **OBS-01**: Banking service exposes Prometheus metrics (wallet balance by bot, buyout count, items held)
- **OBS-02**: Structured log events include wallet balance at time of bid decision

### Robustness

- **ROB-01**: Dead-letter exchange for banking.q (currently nack + requeue creates hot loop on bad messages)
- **ROB-02**: Outbox pattern for RecordWin to prevent dropped events between DB commit and publish

## Out of Scope

| Feature | Reason |
|---------|--------|
| Human user wallets | Platform is bot-only; no human bidders |
| Bot-to-bot trading | Only bank buyouts at fixed 70% rate — no negotiation complexity |
| NUMERIC/decimal balance type | Consistent with existing float64 bid amounts; known tech debt, not fixed here |
| Balance top-up / external funding | Bots start with £1M; no mechanism to add more funds |
| Authentication on banking API | Internal service behind Railway private networking, consistent with other internal APIs |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| EVT-01 | — | Complete |
| INF-01 | Phase 1 | Pending |
| INF-02 | Phase 2 | Pending |
| INF-03 | Phase 4 | Pending |
| INF-04 | Phase 1 | Pending |
| INF-05 | Phase 4 | Pending |
| WALL-01 | Phase 2 | Pending |
| WALL-02 | Phase 3 | Pending |
| WALL-03 | Phase 3 | Pending |
| ITEM-01 | Phase 3 | Pending |
| ITEM-02 | Phase 3 | Pending |
| BOT-01 | Phase 5 | Pending |
| BOT-02 | Phase 5 | Pending |
| BOT-03 | Phase 5 | Pending |
| BOT-04 | Phase 5 | Pending |
| BOT-05 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 16 total (+ 1 validated)
- Mapped to phases: 16 / 16 ✓
- Unmapped: 0

---
*Requirements defined: 2026-05-19*
*Last updated: 2026-05-19 — traceability updated by roadmapper*
