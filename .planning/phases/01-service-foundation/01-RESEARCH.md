# Phase 1: Service Foundation - Research

**Researched:** 2026-05-19
**Domain:** Go module scaffolding, Ports & Adapters, bun ORM domain types, go.work workspace
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Wallet embeds `bun.BaseModel` with `CreatedAt time.Time` and `UpdatedAt time.Time`.
- **D-02:** Wallet PK is `BotID int64 bun:",pk"` — no separate auto-increment. One wallet per bot (IDs 1–4).
- **D-03:** Wallet table name: `wallets`.
- **D-04:** Item has `ID int64 bun:",pk,autoincrement"`.
- **D-05:** Item has `CreatedAt time.Time` only — no `UpdatedAt` (items are immutable: insert on win, delete on buyout).
- **D-06:** Item table name: `items`.
- **D-07:** Service layer assembles wallet + items via two separate repo calls (no bun has-many relationship loading).
- **Locked (STATE.md):** `withTx(tx bun.Tx)` pattern on repos for cross-repo transactions.
- **Locked (STATE.md):** Module path: `github.com/Osireg17/AI-Bidding-Platform/services/banking-service`.
- **Locked (STATE.md):** Port: 8084.
- **Locked (STATE.md):** `float64` for balance (not NUMERIC — consistent with existing bid amounts).
- **Locked (STATE.md):** Bot IDs are constants 1–4; wallet seed is hardcoded to these IDs.
- **Locked (STATE.md):** All DB operations in `RecordWin` and `Buyout` must be single transactions (`bun.RunInTx`).

### Claude's Discretion

- **withTx pattern design:** Phase 1 must define port interfaces that support the `withTx(tx bun.Tx)` pattern — no existing service demonstrates this. Planner must design the interface shape (method on the repo interface, or a separate `Transactional` interface).
- **main.go scope in Phase 1:** Phase 1 success criteria only requires `go build ./...` to exit 0. Whether Phase 1 includes a minimal `main.go` stub or defers entirely to Phase 4 is left to the planner (Phase 4 success criteria confirm `/health` lives in Phase 4).

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INF-01 | Banking service compiles and boots as a standalone Go microservice (port 8084, `GET /health`) | Phase 1 scope is compile-only; `/health` endpoint lives in Phase 4. A minimal `main.go` stub may be required to satisfy `go build ./...`. Config package provides port 8084. |
| INF-04 | `go.work` workspace updated to include the new `banking-service` module | go.work syntax verified from repo root file. Add `./services/banking-service` to the `use` block. Go toolchain version is `1.25.6`. |
</phase_requirements>

---

## Summary

Phase 1 is a pure scaffolding phase: no runtime connections, no HTTP server running, no database calls. The goal is a compilable Go module with correct domain types, port interfaces, sentinel errors, and config — enough for all downstream phases (2–4) to depend on these contracts.

The codebase already has four Go services following an identical layout (`cmd/<svc>/main.go` + `internal/{config,domain,repo,service,http,mq}`). The banking-service must replicate this structure exactly. All patterns needed — bun struct tags, sentinel errors, ports interface, config env-var parsing — are directly extractable from `services/bid-service/` and `services/bot-service/` without deviation.

The one genuinely new pattern in this codebase is the `withTx(tx bun.Tx)` cross-repo transaction pattern. No existing service has it, so the planner must design the interface shape. The rest of the work is faithful replication of established patterns applied to new domain types.

**Primary recommendation:** Mirror `services/bid-service/` structure exactly. Every file in this phase has a direct analogue in an existing service — use those as templates, not documentation.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Domain types (Wallet, Item) | Domain layer (`internal/domain/`) | — | Types are pure data structs with no business logic in Phase 1; bun struct tags live here |
| Port interfaces (WalletRepository, ItemRepository, BankingService) | Domain layer (`internal/domain/ports.go`) | — | Ports & Adapters: domain owns the interface contracts; repo and service are adapters |
| Sentinel errors | Domain layer (`internal/domain/errors.go`) | — | All services in this codebase follow this pattern; errors are domain facts |
| Config env-var parsing | Config layer (`internal/config/config.go`) | — | Env-driven config with sensible defaults; identical pattern to auction-service |
| Module declaration | `go.mod` at service root | `go.work` at repo root | go.mod declares module identity; go.work registers it in the workspace |
| main.go stub | `cmd/banking-service/main.go` | — | Needed to satisfy `go build ./...`; must exist even as a minimal stub |

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/uptrace/bun` | `v1.2.18` | ORM — struct tags on domain types, `bun.BaseModel`, `bun.Tx` for transactions | Already in every Go service in this repo [VERIFIED: bid-service/go.mod] |
| `github.com/uptrace/bun/dialect/pgdialect` | `v1.2.18` | Postgres dialect for bun | Paired with bun in all existing services [VERIFIED: bid-service/go.mod] |
| `github.com/uptrace/bun/driver/pgdriver` | `v1.2.18` | Postgres driver for bun | Paired with bun in all existing services [VERIFIED: bid-service/go.mod] |
| `go.uber.org/zap` | `v1.27.1` | Structured logging | Mandated by claude.local.md §7; already in all services [VERIFIED: bid-service/go.mod] |
| `github.com/gin-gonic/gin` | `v1.12.0` | HTTP router — needed even in Phase 1 for `go build ./...` to resolve if referenced in stub | Already in bid-service and auction-service [VERIFIED: bid-service/go.mod] |
| `github.com/stretchr/testify` | `v1.11.1` | Test assertions for sentinel error tests | Mandated by claude.local.md §8; in all services [VERIFIED: bid-service/go.mod] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/rabbitmq/amqp091-go` | `v1.10.0` | RabbitMQ client — needed for MQ port interface type if defined in Phase 1 | Include if `EventConsumer` port is stubbed in Phase 1; defer import if not |

**Version verification:** All versions above are taken directly from `services/bid-service/go.mod` which is the authoritative version reference for this workspace. [VERIFIED: services/bid-service/go.mod]

**Installation (go.mod for banking-service):**
```bash
# Run from services/banking-service/ after creating go.mod
go mod tidy
```

The go.mod direct dependencies for Phase 1 are:
```
github.com/uptrace/bun v1.2.18
github.com/uptrace/bun/dialect/pgdialect v1.2.18
github.com/uptrace/bun/driver/pgdriver v1.2.18
go.uber.org/zap v1.27.1
github.com/gin-gonic/gin v1.12.0
github.com/stretchr/testify v1.11.1
```

---

## Package Legitimacy Audit

Phase 1 installs no new packages — all libraries are already present in the workspace via existing services. The versions above are pulled directly from committed `go.mod` files in the repo. No external package discovery was required.

| Package | Registry | Existing in Repo | Disposition |
|---------|----------|-----------------|-------------|
| `github.com/uptrace/bun` | pkg.go.dev | Yes — bid-service, bot-service, auction-service | Approved (in-workspace) |
| `github.com/gin-gonic/gin` | pkg.go.dev | Yes — bid-service, auction-service, bff | Approved (in-workspace) |
| `go.uber.org/zap` | pkg.go.dev | Yes — bid-service, bot-service | Approved (in-workspace) |
| `github.com/stretchr/testify` | pkg.go.dev | Yes — all services | Approved (in-workspace) |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Architecture Patterns

### System Architecture Diagram

```
go.work (repo root)
    └── use ./services/banking-service  ← INF-04

services/banking-service/
    ├── go.mod                          ← module identity + deps
    ├── cmd/banking-service/
    │   └── main.go                     ← minimal stub (satisfies go build ./...)
    └── internal/
        ├── config/
        │   └── config.go               ← env-var parsing (PORT, DATABASE_URL, etc.)
        └── domain/
            ├── wallet.go               ← Wallet struct + bun tags (D-01, D-02, D-03)
            ├── item.go                 ← Item struct + bun tags (D-04, D-05, D-06)
            ├── errors.go               ← 4 sentinel errors (success criterion 5)
            └── ports.go                ← WalletRepository, ItemRepository interfaces
                                           (withTx pattern must be expressible here)
```

Data flow for downstream phases:
```
Phase 2 (repo) ──implements──► WalletRepository, ItemRepository (ports.go)
Phase 3 (service) ──calls──► repos via port interfaces
Phase 3 (service) ──calls RunInTx──► withTx variants on repos (defined here)
Phase 4 (main.go) ──wires──► config → db → repos → service → http + mq
```

### Recommended Project Structure

```
services/banking-service/
├── cmd/
│   └── banking-service/
│       └── main.go         # minimal stub — package main, func main() {}
├── internal/
│   ├── config/
│   │   └── config.go       # Config struct + Load() func
│   ├── domain/
│   │   ├── wallet.go       # Wallet struct
│   │   ├── item.go         # Item struct
│   │   ├── errors.go       # sentinel errors
│   │   └── ports.go        # repository + service interfaces
│   ├── repo/               # empty dir — Phase 2
│   ├── service/            # empty dir — Phase 3
│   ├── http/               # empty dir — Phase 3
│   └── mq/                 # empty dir — Phase 4
└── go.mod
```

Empty directories are fine — they confirm the intended structure to Phase 2–4 planners. Alternatively, omit empty dirs and add them in the phase that uses them. Either is valid; consistency with existing services (which do not pre-create empty dirs) suggests adding them only when needed.

### Pattern 1: Sentinel Errors

**What:** Package-level `var` declarations using `errors.New`. Handlers use `errors.Is()` to map to HTTP status codes.
**When to use:** Any business rule violation that needs distinct HTTP mapping.

```go
// Source: services/bid-service/internal/domain/errors.go [VERIFIED: codebase]
package domain

import "errors"

var (
    ErrWalletNotFound       = errors.New("wallet not found")
    ErrItemNotFound         = errors.New("item not found")
    ErrInsufficientBalance  = errors.New("insufficient balance")
    ErrItemNotOwnedByBot    = errors.New("item not owned by this bot")
)
```

### Pattern 2: bun Domain Types with BaseModel

**What:** Domain struct embeds `bun.BaseModel` for the table name tag. PK and column tags are explicit.
**When to use:** Any struct that maps to a DB table.

```go
// Source: services/bot-service/internal/domain/bot.go + CONTEXT.md decisions D-01 to D-06
// [VERIFIED: codebase]
package domain

import (
    "time"
    "github.com/uptrace/bun"
)

type Wallet struct {
    bun.BaseModel `bun:"table:wallets"`
    BotID         int64     `bun:",pk"`
    Balance       float64   `bun:",notnull"`
    CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
    UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

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

### Pattern 3: Ports Interface

**What:** Interfaces defined in the domain package. Concrete implementations live in `repo/` and `mq/`.
**When to use:** Every external dependency (DB, MQ, external service).

```go
// Source: services/bid-service/internal/domain/ports.go [VERIFIED: codebase]
// Pattern — adapt for banking domain
package domain

import "context"

type WalletRepository interface {
    GetWallet(ctx context.Context, botID int64) (*Wallet, error)
    UpdateBalance(ctx context.Context, botID int64, newBalance float64) error
    // withTx variant — see withTx pattern below
    WithTx(tx bun.IDB) WalletRepository
}

type ItemRepository interface {
    GetItemsByBot(ctx context.Context, botID int64) ([]*Item, error)
    InsertItem(ctx context.Context, item *Item) error
    DeleteItem(ctx context.Context, itemID int64) error
    GetItem(ctx context.Context, itemID int64) (*Item, error)
    WithTx(tx bun.IDB) ItemRepository
}
```

### Pattern 4: withTx (NEW to this codebase — planner discretion)

**What:** The `withTx` pattern lets the service layer pass a transaction handle to repos without exposing transaction mechanics to the service.

**Two viable interface shapes — planner must choose one:**

**Option A — `WithTx(tx bun.IDB) WalletRepository`** (method on the repo interface that returns a transaction-scoped copy):
```go
// Usage in service layer:
err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
    txWalletRepo := walletRepo.WithTx(tx)
    txItemRepo   := itemRepo.WithTx(tx)
    return txWalletRepo.UpdateBalance(ctx, botID, newBalance) &&
           txItemRepo.InsertItem(ctx, item)
})
```

**Option B — Separate `Transactional` interface** wrapping a callback:
```go
type Transactional interface {
    RunInTx(ctx context.Context, fn func(ctx context.Context, tx bun.Tx) error) error
}
```

Option A is more idiomatic to this codebase's DI style (constructors + interface-scoped). Option B is more explicit. Both are valid — the interface shape defined in Phase 1 must be consistent with whatever the Phase 2 repo implementations satisfy.

**Note:** `bun.IDB` is the interface satisfied by both `*bun.DB` and `bun.Tx`, making Option A type-safe. [ASSUMED — based on bun library design; planner should verify `bun.IDB` exists in v1.2.18]

### Pattern 5: Config

**What:** `Config` struct populated by `Load()` reading env vars with defaults.
**When to use:** All runtime configuration. Never hardcode ports or URLs.

```go
// Source: services/auction-service/internal/config/config.go [VERIFIED: codebase]
// Banking-service adaptation:
type Config struct {
    Port        int
    DatabaseURL string
    RabbitMQURL string
    LogLevel    string
}

func Load() (*Config, error) {
    port := getEnvInt("PORT", 0)
    if port == 0 {
        port = getEnvInt("BANKING_SERVICE_PORT", 8084)
    }
    dbURL := getEnv("DATABASE_URL", "")
    if dbURL == "" {
        return nil, fmt.Errorf("DATABASE_URL is required")
    }
    // ...
}
```

### Pattern 6: go.mod + go.work

**What:** Each service has its own `go.mod`. The repo root `go.work` lists all modules.
**When to use:** Adding a new service.

Existing go.work (verified from repo root): [VERIFIED: go.work]
```
go 1.25.6

use (
    ./services/auction-service
    ./services/bff
    ./services/bid-service
    ./services/bot-service
    ./shared
)
```

Required change — add one line:
```
    ./services/banking-service
```

New `services/banking-service/go.mod` must declare:
```
module github.com/Osireg17/AI-Bidding-Platform/services/banking-service

go 1.25.6
```

Toolchain version `1.25.6` is confirmed from all existing go.mod files and go.work. [VERIFIED: go.work, bid-service/go.mod, bot-service/go.mod]

### Anti-Patterns to Avoid

- **bun has-many relationship loading:** D-07 explicitly prohibits this. All item lookups use a separate `GetItemsByBot` call at the service layer — not a bun `.Relation()` call.
- **Business logic in domain types:** Wallet and Item are pure data structs. Validation and rules live in the service layer (Phase 3).
- **Global singletons or `init()` side effects:** All deps via constructors. No `init()` with side effects. [VERIFIED: claude.local.md §4]
- **Defining bot ID constants by importing bot-service:** `internal/` packages cannot be imported cross-module. Replicate the 4 bot ID constants locally in banking-service. [VERIFIED: CONTEXT.md code_context]
- **Skipping `go vet`:** Success criterion 2 requires `go vet ./...` to pass. Avoid unused imports, unreachable code, and incorrect struct tags during scaffolding.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bun struct tags for table name | Custom table registry | `bun.BaseModel` embedding + `bun:"table:xxx"` tag | Already the codebase standard; bun uses this for all ORM operations |
| Sentinel error type checking | Custom error types with `Error()` method matching | `errors.New` + `errors.Is()` | All existing handlers use `errors.Is()`; custom types would break the established pattern |
| Transaction scoping | Passing `*bun.DB` to repos and opening transactions inside repo methods | `bun.RunInTx` with withTx pattern | Leaking tx management into repos violates single responsibility |
| Env-var parsing | `os.Getenv` scattered throughout codebase | `config.Load()` with explicit defaults | All existing services centralise config this way |

**Key insight:** Phase 1 is entirely replication work. Every component has a direct analogue in the existing codebase. The only genuinely new design decision is the withTx interface shape.

---

## Common Pitfalls

### Pitfall 1: go.mod requires block missing during `go build ./...`

**What goes wrong:** `go build ./...` fails with "cannot find module providing package" because the go.mod `require` block is absent or incomplete.
**Why it happens:** Writing a minimal go.mod without `go mod tidy` after adding imports.
**How to avoid:** After creating all `.go` files with their imports, run `go mod tidy` from `services/banking-service/`. This populates the `require` block with correct versions aligned to the workspace.
**Warning signs:** Build error mentioning a missing module path before any logic error.

### Pitfall 2: go.work not updated before running `go build`

**What goes wrong:** The workspace fails to resolve the banking-service module even after go.mod is created.
**Why it happens:** `go.work` must list `./services/banking-service` in the `use` block. Forgetting this step means the workspace ignores the new module.
**How to avoid:** Update go.work as the first task in the plan, before any other file is created. Success criterion 3 explicitly tests this with `grep "banking-service" go.work`.
**Warning signs:** `go: no modules were loaded from the main module's requirements`

### Pitfall 3: Bot ID constants duplicated via cross-module import

**What goes wrong:** Code attempts `import "github.com/Osireg17/AI-Bidding-Platform/services/bot-service/internal/domain"` to reuse the Bot constants — this fails because `internal/` cannot be imported from outside the module.
**Why it happens:** Developers see constants in bot-service and assume they're reusable.
**How to avoid:** Define local bot ID constants in `banking-service/internal/domain/` (e.g., a `bots.go` file with `const Alice int64 = 1` etc., or just use the numeric literals in the seed — the constants are used only in Phase 2 seed logic).
**Warning signs:** Compiler error: `use of internal package ... not allowed`

### Pitfall 4: `bun.IDB` assumption for withTx

**What goes wrong:** Port interface uses `bun.IDB` as the transaction parameter type, but the actual bun v1.2.18 API uses a different interface name or a concrete `bun.Tx`.
**Why it happens:** `bun.IDB` is a training-data assumption — not verified against v1.2.18 source.
**How to avoid:** Before writing ports.go, check what type `bun.RunInTx` passes to its callback — the callback signature is the authoritative answer for what tx type to accept in `WithTx`. Run: `go doc github.com/uptrace/bun RunInTx` after `go mod tidy`.
**Warning signs:** Compiler error on ports.go related to bun type mismatch.

### Pitfall 5: Phase 1 main.go scope creep

**What goes wrong:** Planner includes real Gin router, DB connection, or RabbitMQ setup in Phase 1 main.go — these require env vars and running infrastructure, making `go build ./...` succeed but the binary crash at startup during CI.
**Why it happens:** INF-01 mentions "boots as a standalone Go microservice" — but Phase 1 only needs compile success; boot is Phase 4.
**How to avoid:** Phase 1 main.go is a minimal stub: `package main; func main() {}`. Nothing more. Phase 4 replaces it entirely.

---

## Code Examples

### go.mod (minimal for Phase 1)

```go
// Source: modelled on services/bid-service/go.mod [VERIFIED: codebase]
module github.com/Osireg17/AI-Bidding-Platform/services/banking-service

go 1.25.6

require (
    github.com/gin-gonic/gin v1.12.0
    github.com/stretchr/testify v1.11.1
    github.com/uptrace/bun v1.2.18
    github.com/uptrace/bun/dialect/pgdialect v1.2.18
    github.com/uptrace/bun/driver/pgdriver v1.2.18
    go.uber.org/zap v1.27.1
)
```

`go mod tidy` will populate indirect deps automatically from the workspace cache.

### Sentinel errors (errors.go)

```go
// Source: mirrors services/bid-service/internal/domain/errors.go [VERIFIED: codebase]
package domain

import "errors"

var (
    ErrWalletNotFound      = errors.New("wallet not found")
    ErrItemNotFound        = errors.New("item not found")
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrItemNotOwnedByBot   = errors.New("item not owned by this bot")
)
```

These are the exact four errors named in Phase 1 success criterion 5.

### Config (config.go)

```go
// Source: mirrors services/auction-service/internal/config/config.go [VERIFIED: codebase]
package config

import (
    "fmt"
    "os"
    "strconv"
)

type Config struct {
    Port        int
    DatabaseURL string
    RabbitMQURL string
    LogLevel    string
}

func Load() (*Config, error) {
    port := getEnvInt("PORT", 0)
    if port == 0 {
        port = getEnvInt("BANKING_SERVICE_PORT", 8084)
    }
    dbURL := getEnv("DATABASE_URL", "")
    if dbURL == "" {
        return nil, fmt.Errorf("DATABASE_URL is required")
    }
    rabbitURL := getEnv("RABBITMQ_URL", "")
    if rabbitURL == "" {
        return nil, fmt.Errorf("RABBITMQ_URL is required")
    }
    return &Config{
        Port:        port,
        DatabaseURL: dbURL,
        RabbitMQURL: rabbitURL,
        LogLevel:    getEnv("LOG_LEVEL", "info"),
    }, nil
}

func getEnv(key, fallback string) string {
    if val, ok := os.LookupEnv(key); ok {
        return val
    }
    return fallback
}

func getEnvInt(key string, fallback int) int {
    val, ok := os.LookupEnv(key)
    if !ok {
        return fallback
    }
    parsed, err := strconv.Atoi(val)
    if err != nil {
        return fallback
    }
    return parsed
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Global DB variable in each package | DI via constructors | Established in this codebase from the start | All repos receive `*bun.DB` at construction time — no globals |
| Direct `database/sql` | bun ORM with struct tags | Established in this codebase from the start | Use bun throughout; no raw `database/sql` except when bun requires it for error checking (`sql.ErrNoRows`) |

**Deprecated/outdated in this codebase:**
- `database/sql` direct usage: only acceptable as an error sentinel source (`sql.ErrNoRows` in repo methods). Never use `sql.Open` — always use `pgdriver.Open` + `bun.NewDB`.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `bun.IDB` exists as an interface in bun v1.2.18, satisfied by both `*bun.DB` and `bun.Tx` | Architecture Patterns — withTx | Port interface would need to use `bun.Tx` directly or a different approach; discoverable immediately with `go doc` after `go mod tidy` |

**All other claims were verified directly from the codebase.**

---

## Open Questions

1. **withTx interface shape (planner discretion)**
   - What we know: The `withTx(tx bun.Tx)` pattern is locked in STATE.md. No existing service demonstrates it.
   - What's unclear: Whether to express this as `WithTx(tx bun.IDB) WalletRepository` (Option A) or a separate `Transactional` interface (Option B). The actual bun type for the tx parameter needs verification.
   - Recommendation: Planner should add a task "verify `bun.IDB` exists via `go doc`" as the first domain task, then choose Option A (returns a scoped copy of the repo) as it most closely fits the constructor-based DI style of this codebase.

2. **main.go stub vs. empty entry point**
   - What we know: `go build ./...` requires a valid `main` package if `cmd/banking-service/main.go` exists.
   - What's unclear: Whether the planner should create a minimal `package main; func main() {}` stub or defer the `cmd/` directory entirely to Phase 4.
   - Recommendation: Create the stub. It costs one file and prevents a confusing gap in the directory structure. Phase 4 overwrites it completely.

---

## Environment Availability

Phase 1 is code/config only — no running services, no network connections, no database calls. The only external dependency is the Go toolchain.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | `go build ./...`, `go vet ./...` | Yes | go1.26.1 (local), workspace targets 1.25.6 | — |
| go.work workspace | INF-04 | Yes | Already exists at repo root | — |

**Missing dependencies with no fallback:** None.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | testify v1.11.1 |
| Config file | none — go test standard |
| Quick run command | `cd services/banking-service && go test ./internal/domain/...` |
| Full suite command | `cd services/banking-service && go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INF-01 | Module compiles clean | build check | `go build ./...` | ❌ Wave 0 — no test file needed, build IS the test |
| INF-04 | go.work updated | shell check | `grep "banking-service" go.work` | ❌ Wave 0 — verified via grep, not a Go test |
| SC-5 | 4 sentinel errors referenceable | unit | `go test ./internal/domain/... -run TestSentinelErrors` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go build ./... && go vet ./...` from `services/banking-service/`
- **Per wave merge:** `go test ./internal/domain/...`
- **Phase gate:** All 5 success criteria green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `services/banking-service/internal/domain/errors_test.go` — verifies all 4 sentinel errors are referenceable and distinct (`errors.Is` does not cross-match them)

*(No existing test infrastructure exists for this new service — Wave 0 must create the test file alongside the implementation.)*

---

## Security Domain

Banking-service in Phase 1 contains no HTTP handlers, no authentication, no input validation, and no DB calls. The domain types and port interfaces carry no security surface. ASVS categories are not applicable to this phase.

Per REQUIREMENTS.md Out of Scope: "Authentication on banking API — internal service behind Railway private networking, consistent with other internal APIs." No auth requirements apply to this service at any phase.

---

## Project Constraints (from CLAUDE.md)

The following directives from `claude/claude.local.md` directly constrain this phase:

| Directive | Impact on Phase 1 |
|-----------|------------------|
| `gofmt` on all Go files (§4) | All generated files must be `gofmt`-formatted before commit |
| No `init()` side effects (§4) | config.go and domain files must not use `init()` |
| No circular dependencies (§4) | `domain` package must not import `config`, `repo`, or any other internal package |
| Dependency injection via constructors (§5) | No global `db` or `logger` vars — Phase 1 does not wire anything, but stubs must not introduce globals |
| Tests added for new behaviour (§17 DoD) | `errors_test.go` must exist before phase is marked done |
| Only `cmd` contains `main()` (§3) | No `main()` function outside `cmd/banking-service/main.go` |

---

## Sources

### Primary (HIGH confidence)

- `services/bid-service/go.mod` — library versions for all direct dependencies
- `services/bid-service/internal/domain/errors.go` — sentinel error pattern (verbatim)
- `services/bid-service/internal/domain/ports.go` — port interface pattern (verbatim)
- `services/bid-service/internal/repo/bid_repo.go` — repo constructor pattern (verbatim)
- `services/auction-service/internal/config/config.go` — config Load() pattern (verbatim)
- `services/bot-service/internal/domain/bot.go` — bun struct tags, bot ID constants
- `go.work` — toolchain version (1.25.6), workspace use block format
- `.planning/phases/01-service-foundation/01-CONTEXT.md` — all locked decisions (D-01 through D-07)
- `.planning/STATE.md` — withTx pattern, module path, port 8084, float64 balance
- `.planning/ROADMAP.md` — Phase 1 vs Phase 4 boundary (/health lives in Phase 4)
- `claude/claude.local.md` — coding standards, package layout rules, DoD

### Secondary (MEDIUM confidence)

- None needed — all findings verified from codebase.

### Tertiary (LOW confidence)

- `bun.IDB` interface existence [A1] — training knowledge, needs `go doc` verification after `go mod tidy`.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified from committed go.mod files in the workspace
- Architecture: HIGH — all patterns verified from existing services; only withTx is new (documented as assumption)
- Pitfalls: HIGH — derived from direct code inspection and locked decisions in CONTEXT.md

**Research date:** 2026-05-19
**Valid until:** 2026-06-19 (stable Go/bun ecosystem; re-verify if bun is upgraded)
