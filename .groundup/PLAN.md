# Item Appreciation Plan

## Goal
Add appreciation/depreciation to inventory items. When a bot wins an auction, the item is assigned a random ±1–10% appreciation (50/50 direction). The bank buys items at 70% of their current value (purchase price adjusted by appreciation).

## Agreed Design

- `appreciation` stored as float64 percentage fraction (e.g. `0.05` = 5%, `-0.03` = -3%)
- `CurrentValue = purchasePrice * (1 + appreciation)`
- `Buyout payout = currentValue * 0.70`
- Appreciation generated in `RecordWin` (service layer), passed into `NewItem`
- `ItemSummary` exposes `CurrentValue` so bot agent can select the most valuable item to sell
- Tests seed the random source for determinism — no new interface needed

## Key Edge Cases

- Appreciation magnitude is always ≥ 1%, so `currentValue` is always 90–110% of `purchasePrice` — never zero or negative
- `NewItem` already guards `purchasePrice <= 0`, so `currentValue` is always positive
- Existing `Buyout` test uses `item.PurchasePrice * 0.70` — must be updated to `item.PurchasePrice * (1 + item.Appreciation) * 0.70`

---

## Tickets (dependency order)

### TICKET-1: Add Appreciation field to Item domain
**File:** `services/banking-service/internal/domain/item.go`

- Add `Appreciation float64` field to `Item` struct (bun column: `appreciation`)
- Update `NewItem` signature to accept `appreciation float64` as a parameter
- Validate that `appreciation` is within `[-0.10, -0.01]` or `[0.01, 0.10]` — return `ErrInvalidAppreciation` otherwise

**Tests:** unit tests for `NewItem` covering valid appreciation, out-of-range appreciation, zero appreciation

---

### TICKET-2: Add appreciation column to DB migration
**File:** `services/banking-service/internal/repo/migrate.go`

- Add `appreciation FLOAT NOT NULL DEFAULT 0` column to the `items` table migration

---

### TICKET-3: Update banking service logic
**File:** `services/banking-service/internal/service/banking_service.go`

- In `RecordWin`: generate a random appreciation — magnitude `[0.01, 0.10]`, sign random 50/50 — and pass it to `domain.NewItem`
- In `Buyout`: replace `item.PurchasePrice * 0.70` with `item.PurchasePrice * (1 + item.Appreciation) * 0.70`
- In `GetWallet`: populate `CurrentValue = item.PurchasePrice * (1 + item.Appreciation)` on each `ItemSummary`

**Tests:**
- `TestBankingService_RecordWin`: seed rand, assert item is created with expected appreciation
- `TestBankingService_Buyout`: update expected payout to use `currentValue * 0.70`
- `TestBankingService_GetWallet`: assert `CurrentValue` is correctly populated on each item summary

---

### TICKET-4: Add CurrentValue to ItemSummary
**File:** `services/banking-service/internal/domain/ports.go`

- Add `CurrentValue float64` to `ItemSummary`

---

### TICKET-5: Update banking client struct
**File:** `services/bot-service/internal/bankingclient/client.go`

- Add `CurrentValue float64` to `ItemSummary` struct so the bot agent can read it

---

### TICKET-6: Update bot agent item selection
**File:** `services/bot-service/internal/agent/bot_agent.go`

- In `Evaluate`, change the "pick best item to sell" logic from comparing `item.PurchasePrice` to comparing `item.CurrentValue`
