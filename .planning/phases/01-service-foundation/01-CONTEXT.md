# Phase 1: Service Foundation - Context

**Gathered:** 2026-05-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Scaffold a compilable banking-service Go module: domain types (Wallet, Item), port interfaces, sentinel errors, config, and go.work registration. No runtime connections or HTTP server required — the phase goal is a clean `go build ./...` and `go vet ./...`. All subsequent phases (repo, service, entrypoint) depend on these contracts being defined here.

</domain>

<decisions>
## Implementation Decisions

### Domain Types — Wallet

- **D-01:** Wallet struct embeds `bun.BaseModel` marker and includes explicit timestamp fields: `CreatedAt time.Time` and `UpdatedAt time.Time` (populated by bun on insert and update respectively).
- **D-02:** Wallet primary key is `BotID int64 bun:",pk"` — no separate auto-increment ID. Bot ID IS the wallet identity; one wallet per bot, bots are pre-known constants 1–4.
- **D-03:** Wallet table name: `wallets`.

```go
type Wallet struct {
    bun.BaseModel `bun:"table:wallets"`
    BotID     int64     `bun:",pk"`
    Balance   float64   `bun:",notnull"`
    CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
    UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}
```

### Domain Types — Item

- **D-04:** Item has auto-increment `ID int64 bun:",pk,autoincrement"` — required so the buyout endpoint `POST /api/banking/buyout/:item_id` can identify individual items.
- **D-05:** Item includes `CreatedAt time.Time` (moment the bot won the auction). No `UpdatedAt` — items are immutable once created; they are inserted on win and deleted on buyout.
- **D-06:** Item table name: `items`.

```go
type Item struct {
    bun.BaseModel `bun:"table:items"`
    ID            int64     `bun:",pk,autoincrement"`
    BotID         int64     `bun:",notnull"`
    AuctionID     int64     `bun:",notnull"`
    Title         string    `bun:",notnull"`
    PurchasePrice float64   `bun:",notnull"`
    CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}
```

### Items Relationship (GET /api/wallets/:bot_id response)

- **D-07:** Service layer assembles wallet + items via two separate repo calls. `WalletRepository.GetWallet()` returns only wallet fields. A separate `ItemRepository.GetItemsByBot()` returns the bot's items. The service combines them before returning. This matches the pattern in every existing service (bid-service, auction-service — all use separate repo queries, none use bun has-many relationships).

### Undiscussed Areas — Left to Planner

- **Transaction API (withTx pattern):** Locked in STATE.md Key Decisions as `withTx(tx bun.Tx) pattern on repos`. Planner must read STATE.md and design the port interfaces accordingly. The existing services do not have a cross-repo transaction pattern — this is new to this codebase.
- **main.go scope in Phase 1:** Phase 1 success criteria only requires `go build ./...` to exit 0. Whether Phase 1 includes a working Gin /health endpoint or a minimal stub is left to the planner to determine based on the Phase 4 entrypoint boundary.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning Artifacts
- `.planning/ROADMAP.md` — Phase 1 success criteria (5 items); Phase 4 success criteria (confirms /health lives in Phase 4, not Phase 1)
- `.planning/REQUIREMENTS.md` — INF-01 (compile + /health scaffold), INF-04 (go.work update)
- `.planning/STATE.md` — Locked decisions: withTx pattern, module path, port 8084, float64 balance, bot ID constants 1–4

### Coding Standards
- `claude/claude.local.md` — Full project coding standards (Go conventions, package layout, error handling, logging)

### Existing Service Patterns (read for analogy)
- `services/bot-service/internal/domain/bot.go` — Bot domain type with bun struct tags; bot ID constants (1–4 already defined here as `Alice`, `Steve`, `Victor`, `Charlie`)
- `services/bid-service/internal/repo/bid_repo.go` — Standard repo pattern: `PostgresBidRepo` with `db *bun.DB`, constructor `NewPostgresBidRepo`, context on all methods
- `services/bid-service/internal/domain/errors.go` — Sentinel error pattern to replicate in banking-service
- `services/auction-service/internal/config/config.go` — Config env-var parsing pattern to replicate

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `services/bot-service/internal/domain/bot.go`: Bot ID constants (1–4) are already defined. The banking-service seed can reference these same IDs without redefining them — or replicate the constants locally (banking-service cannot import bot-service's internal package, so local replication is required).

### Established Patterns
- **Repo pattern:** `PostgresXxxRepo` struct with `db *bun.DB`; constructor `NewPostgresXxxRepo(db *bun.DB)`; all methods accept `ctx context.Context` as first param. See `services/bid-service/internal/repo/bid_repo.go`.
- **Sentinel errors:** `var ErrXxx = errors.New("descriptive message")` in `domain/errors.go`. Handlers switch on `errors.Is(err, domain.ErrXxx)`. See `services/bid-service/internal/domain/`.
- **Dependency injection:** All deps via constructors; no global singletons; no `init()` side effects.
- **bun struct tags:** `bun:",pk"` for natural PK, `bun:",pk,autoincrement"` for auto-increment, `bun:"table:xxx"` on embedded `bun.BaseModel`.

### Integration Points
- `go.work` (repo root) — must add `./services/banking-service` to the `use` block
- `shared/events/auction_events.go` — `AuctionEndedPayload` is the event the banking-service consumer will handle in Phase 4; Phase 1 does not consume it but the port interface for the consumer should be defined here

</code_context>

<specifics>
## Specific Ideas

- The user confirmed: bun.BaseModel embedding for Wallet (timestamps) and `created_at` on Item aligns with portfolio-quality code — easy to query "when did this bot win this item?"
- Items assembled at service layer (not bun has-many) is consistent with all existing services — do not introduce bun relationship loading in this codebase.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 1-Service Foundation*
*Context gathered: 2026-05-19*
