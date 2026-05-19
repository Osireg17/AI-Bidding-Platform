# Roadmap: AI Bidding Platform — Banking Service Milestone

**Defined:** 2026-05-19
**Granularity:** Standard (5-8 phases)
**Requirements:** 16 v1 pending + 1 validated (EVT-01 — complete)

---

## Phases

- [ ] **Phase 1: Service Foundation** - Go module, domain types, ports, errors, config, go.work registration
- [ ] **Phase 2: Repository Layer** - Wallet repo, item repo, DB migrations
- [ ] **Phase 3: Service + API Layer** - BankingService use-case logic, HTTP handlers, Gin router
- [ ] **Phase 4: Event Consumer + Entrypoint** - MQ consumer, main.go wiring, railway.json deploy config
- [ ] **Phase 5: Bot Integration** - bankingclient, bot-service config, Evaluate() balance gate

---

## Phase Details

### Phase 1: Service Foundation
**Goal**: The banking-service module compiles cleanly, domain contracts are defined, and the workspace knows about the new service
**Depends on**: Nothing (first phase)
**Requirements**: INF-01, INF-04
**Success Criteria** (what must be TRUE):
  1. `cd services/banking-service && go build ./...` exits 0 with no errors
  2. `cd services/banking-service && go vet ./...` exits 0 (no lint issues in domain + config packages)
  3. `grep "banking-service" go.work` returns the module path — workspace resolves the new module
  4. Domain types (`Wallet`, `Item`) and all port interfaces compile: `go build ./internal/domain/...` exits 0
  5. `domain.ErrWalletNotFound`, `domain.ErrItemNotFound`, `domain.ErrInsufficientBalance`, `domain.ErrItemNotOwnedByBot` are defined and referenceable in tests
**Plans**: TBD

### Phase 2: Repository Layer
**Goal**: Wallet and item data can be persisted; tables are created idempotently at boot; four wallets are seeded with £1,000,000 each on first run and never reset
**Depends on**: Phase 1
**Requirements**: INF-02, WALL-01
**Success Criteria** (what must be TRUE):
  1. After first boot: `SELECT COUNT(*) FROM wallets;` returns 4; each row has `balance = 1000000`
  2. After restart: `SELECT COUNT(*) FROM wallets;` still returns 4 with unchanged balances — seed is idempotent (ON CONFLICT DO NOTHING)
  3. `SELECT COUNT(*) FROM items;` returns 0 after first boot — items table exists and is empty
  4. `go build ./internal/repo/...` exits 0; `go test ./internal/repo/...` exits 0
  5. Manually inserting a wallet row and restarting does not duplicate or reset it
**Plans**: TBD

### Phase 3: Service + API Layer
**Goal**: Wallet state is readable via REST and buyout transactions execute atomically; the service layer correctly orchestrates wallet + item operations in single DB transactions
**Depends on**: Phase 2
**Requirements**: WALL-02, WALL-03, ITEM-01, ITEM-02
**Success Criteria** (what must be TRUE):
  1. `curl http://localhost:8084/api/wallets/1` returns HTTP 200 with `{"bot_id":1,"balance":1000000,"items":[]}`
  2. `curl http://localhost:8084/api/wallets/99` returns HTTP 404
  3. After `RecordWin(botID=1, winningBid=50000, title="Vintage Clock", auctionID=42)`: `SELECT balance FROM wallets WHERE bot_id = 1` equals 950000; `SELECT COUNT(*) FROM items WHERE bot_id = 1` equals 1 with `title = 'Vintage Clock'`
  4. `curl -X POST http://localhost:8084/api/banking/buyout/<item_id>` returns HTTP 200 with `{"new_balance": X}` where X = previous balance + (purchase_price * 0.70); item row is gone from DB
  5. `curl -X POST http://localhost:8084/api/banking/buyout/99999` returns HTTP 404
**Plans**: TBD
**UI hint**: no

### Phase 4: Event Consumer + Entrypoint
**Goal**: The banking service runs end-to-end: it boots, connects to Postgres and RabbitMQ, consumes `auction.ended` events, records wins atomically, and is deployable to Railway
**Depends on**: Phase 3
**Requirements**: INF-03, INF-05, ITEM-01
**Success Criteria** (what must be TRUE):
  1. `curl http://localhost:8084/health` returns HTTP 200 — full service is up with all connections live
  2. After an auction closes with a winner, `SELECT balance FROM wallets WHERE bot_id = <winner>` equals `1000000 - winning_bid` and `SELECT * FROM items WHERE bot_id = <winner>` contains 1 row with the correct `title` matching the auction title
  3. Restarting the banking-service after wins: previous wallet balances and item rows are preserved — no data reset
  4. `cat services/banking-service/railway.json` contains a valid `healthcheckPath: "/health"` entry — deploy config is present
  5. Service startup log contains structured zap entries for DB ping, migration success, wallet seed, MQ consumer start — sequential init is observable
**Plans**: TBD

### Phase 5: Bot Integration
**Goal**: Bots check their balance before bidding and degrade gracefully when the banking service is unreachable; balance is visible in LLM prompts
**Depends on**: Phase 4
**Requirements**: BOT-01, BOT-02, BOT-03, BOT-04, BOT-05
**Success Criteria** (what must be TRUE):
  1. `cd services/bot-service && go build ./...` exits 0 — bankingclient and Evaluate() changes compile cleanly
  2. When a bot's balance equals 1000000 and start price is 500, bot proceeds to bid normally (no skip, no buyout attempted)
  3. When a bot's balance is set to 0 in DB and has no items: trigger `auction.created`; bot logs "insufficient balance, no items to sell, skipping" and does NOT place a bid
  4. When a bot's balance is set to 0 but owns one item worth 100000: bot calls buyout, receives 70000, then re-evaluates — if start_price > 70000 bot skips; if start_price <= 70000 bot proceeds to bid
  5. Stop the banking-service process; trigger `auction.created`; all 4 bots still attempt to bid — graceful degradation confirmed by log line "could not fetch wallet, proceeding without balance check"
  6. LLM prompt for a bot with balance 750000 contains the string `Current Balance: £750000.00` (or equivalent formatted float)
**Plans**: TBD

---

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Service Foundation | 0/3 | Not started | - |
| 2. Repository Layer | 0/2 | Not started | - |
| 3. Service + API Layer | 0/3 | Not started | - |
| 4. Event Consumer + Entrypoint | 0/3 | Not started | - |
| 5. Bot Integration | 0/3 | Not started | - |

---

## Coverage

| Requirement | Phase | Notes |
|-------------|-------|-------|
| EVT-01 | — | Complete (PR #34 merged) |
| INF-01 | Phase 1 | Module compiles + /health route scaffold |
| INF-02 | Phase 2 | DB connection + migrations at boot |
| INF-03 | Phase 4 | RabbitMQ bind + consumer wired in main.go |
| INF-04 | Phase 1 | go.work updated with banking-service module |
| INF-05 | Phase 4 | railway.json created |
| WALL-01 | Phase 2 | Wallet seed (idempotent ON CONFLICT DO NOTHING) |
| WALL-02 | Phase 3 | GET /api/wallets/:bot_id handler |
| WALL-03 | Phase 3 | RecordWin atomic transaction (wallet debit + item insert) |
| ITEM-01 | Phase 3 | Item record created on RecordWin |
| ITEM-02 | Phase 3 | POST /api/banking/buyout/:item_id handler |
| BOT-01 | Phase 5 | Balance gate: balance >= start_price → proceed |
| BOT-02 | Phase 5 | Balance gate: balance < start_price, has items → buyout + re-evaluate |
| BOT-03 | Phase 5 | Balance gate: balance < start_price, no items → skip |
| BOT-04 | Phase 5 | Graceful degradation: banking service unreachable → proceed |
| BOT-05 | Phase 5 | Balance injected into LLM prompt |

**v1 coverage: 16/16 requirements mapped ✓**

---
*Roadmap defined: 2026-05-19*
*Last updated: 2026-05-19 after initial creation*
