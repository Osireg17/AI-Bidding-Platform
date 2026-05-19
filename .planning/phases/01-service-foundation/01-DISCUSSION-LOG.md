# Phase 1: Service Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-19
**Phase:** 1-Service Foundation
**Areas discussed:** Domain type fields

---

## Domain type fields

### Q1 — Wallet timestamps

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, embed bun.BaseModel on Wallet | Gets created_at and updated_at for free via bun — matches bot.go pattern | ✓ |
| No, just bot_id and balance | Wallet is a simple ledger — no audit timestamps needed at this stage | |
| You decide | Let Claude pick the simpler option | |

**User's choice:** Embed bun.BaseModel on Wallet (yes to timestamps)
**Notes:** User wants timestamps for auditability.

---

### Q2 — Item created_at

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, add created_at to Item | Useful for ordering items by purchase date and debugging | ✓ |
| No, just ID + bot_id + auction_id + title + purchase_price | Minimal — auction ID already ties item to a point in time | |
| You decide | Let Claude pick | |

**User's choice:** Yes, add created_at to Item
**Notes:** No updated_at — items are immutable once created (inserted on win, deleted on buyout).

---

### Q3 — Wallet updated_at

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, add updated_at to Wallet | Makes it easy to see when a bot's balance last changed | ✓ |
| No, just created_at is enough | Balance changes traceable via items table | |
| You decide | Let Claude pick | |

**User's choice:** Yes, add updated_at to Wallet
**Notes:** updated_at should be refreshed on every balance change (debit on win, credit on buyout).

---

### Q4 — Items relationship (Wallet → Items)

| Option | Description | Selected |
|--------|-------------|----------|
| Service layer assembles it (separate queries) | Two repo calls combined in service; cleaner ports, easier to test | ✓ |
| Wallet domain type has Items []Item field | bun has-many relationship; single query | |
| You decide | Let Claude pick | |

**User's choice:** "How is it done within the other services" (freeform)
**Notes:** Confirmed via codebase scout that no existing service uses bun has-many. All use separate repo queries assembled at service layer. User accepted "service layer assembles it" as the consistent approach.

---

## Claude's Discretion

- **Transaction API (withTx pattern):** Not discussed. Left to planner based on STATE.md locked decision `withTx(tx bun.Tx) pattern on repos`.
- **main.go scope in Phase 1:** Not discussed. Left to planner to resolve against Phase 1 vs Phase 4 success criteria boundary.

## Deferred Ideas

None.
