# Phase 1: Service Foundation - Pattern Map

**Mapped:** 2026-05-19
**Files analyzed:** 7 new files + 1 file modified
**Analogs found:** 7 / 7 (withTx interface shape has no codebase analog — design discretion)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `services/banking-service/go.mod` | config | — | `services/bid-service/go.mod` | exact |
| `go.work` | config | — | `go.work` (existing) | exact |
| `cmd/banking-service/main.go` | entrypoint | — | `services/bid-service/cmd/bid-service/main.go` | role-match (Phase 1 is a minimal stub only) |
| `internal/domain/wallet.go` | model | CRUD | `services/bid-service/internal/domain/bid.go` | role-match |
| `internal/domain/item.go` | model | CRUD | `services/bid-service/internal/domain/bid.go` | role-match |
| `internal/domain/errors.go` | utility | — | `services/bid-service/internal/domain/errors.go` | exact |
| `internal/domain/ports.go` | utility | request-response | `services/bid-service/internal/domain/ports.go` | role-match |
| `internal/config/config.go` | config | — | `services/auction-service/internal/config/config.go` | exact |

---

## Pattern Assignments

### `services/banking-service/go.mod` (config)

**Analog:** `services/bid-service/go.mod`

**Module declaration pattern** (lines 1–15 of bid-service/go.mod):
```go
module github.com/Osireg17/AI-Bidding-Platform/services/bid-service

go 1.25.6

require (
    github.com/gin-gonic/gin v1.12.0
    github.com/rabbitmq/amqp091-go v1.10.0
    github.com/stretchr/testify v1.11.1
    github.com/uptrace/bun v1.2.18
    github.com/uptrace/bun/dialect/pgdialect v1.2.18
    github.com/uptrace/bun/driver/pgdriver v1.2.18
    go.uber.org/zap v1.27.1
)
```

**Banking-service adaptation — direct deps only (indirect block populated by `go mod tidy`):**
```go
module github.com/Osireg17/AI-Bidding-Platform/services/banking-service

go 1.25.6

require (
    github.com/gin-gonic/gin v1.12.0
    github.com/rabbitmq/amqp091-go v1.10.0
    github.com/stretchr/testify v1.11.1
    github.com/uptrace/bun v1.2.18
    github.com/uptrace/bun/dialect/pgdialect v1.2.18
    github.com/uptrace/bun/driver/pgdriver v1.2.18
    go.uber.org/zap v1.27.1
)
```

**Note:** All versions come verbatim from `services/bid-service/go.mod`. Do not invent or change versions. Run `go mod tidy` from `services/banking-service/` after creating all `.go` files — this populates the indirect block from the workspace cache.

---

### `go.work` (config — modification, not new file)

**Analog:** `go.work` (repo root, lines 1–9)

**Existing file contents** (`go.work` lines 1–9):
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

**Required change — add one line to the `use` block:**
```
    ./services/banking-service
```

**Final result:**
```
go 1.25.6

use (
    ./services/auction-service
    ./services/bff
    ./services/bid-service
    ./services/bot-service
    ./shared
    ./services/banking-service
)
```

**Note:** Update go.work first — before creating any .go files — so `go build ./...` resolves the module from the first build attempt.

---

### `cmd/banking-service/main.go` (entrypoint — Phase 1 stub only)

**Analog:** `services/bid-service/cmd/bid-service/main.go` (full wiring — read for reference only)

**Phase 1 scope:** The full main.go in bid-service (lines 1–160) wires config, logger, DB, RabbitMQ, repos, service, handler, router, and graceful shutdown. Phase 1 must NOT replicate this — it would require running infrastructure. Phase 4 replaces this stub entirely.

**Phase 1 stub pattern:**
```go
package main

func main() {}
```

This satisfies `go build ./...` for a valid `cmd/` entry point while leaving zero footprint for Phase 4 to overwrite.

**Why not the full bid-service main.go pattern:** RESEARCH.md Pitfall 5 explicitly prohibits scope creep into Phase 1 main.go. Phase 4 success criteria confirm `/health` and runtime wiring live there.

---

### `internal/domain/wallet.go` (model, CRUD)

**Analog:** `services/bid-service/internal/domain/bid.go`

**Imports pattern** (bid.go lines 1–8):
```go
package domain

import (
    "fmt"
    "time"

    "github.com/uptrace/bun"
)
```

**Banking-service wallet.go imports** (no `fmt` needed — Wallet has no constructor with validation in Phase 1):
```go
package domain

import (
    "time"

    "github.com/uptrace/bun"
)
```

**bun struct tag pattern** (bid.go lines 17–27 — shows BaseModel embedding, pk, notnull, nullzero, default):
```go
type Bid struct {
    bun.BaseModel `bun:"table:bids,alias:b"`

    ID        int64     `bun:",pk,autoincrement"`
    AuctionID int64     `bun:",notnull"`
    BotID     int64     `bun:",notnull"`
    Amount    float64   `bun:",notnull"`
    Status    BidStatus `bun:",notnull"`
    Reason    string    `bun:",notnull,default:''"`
    CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}
```

**Wallet struct — copy tag style exactly from bid.go, apply decisions D-01/D-02/D-03:**
```go
type Wallet struct {
    bun.BaseModel `bun:"table:wallets"`
    BotID         int64     `bun:",pk"`
    Balance       float64   `bun:",notnull"`
    CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
    UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}
```

**Key differences from Bid:**
- Natural PK (`bun:",pk"` — no `autoincrement`) per D-02
- No `alias:` on BaseModel tag per D-03 (no joins needed in Phase 1)
- Two timestamp fields per D-01 (CreatedAt + UpdatedAt)

**Bot ID constants** — bot-service's `bot.go` (lines 18–24) defines them as `*Bot` pointers, not int64 constants. Banking-service cannot import that package (`internal/` restriction). Define local int64 constants in this same file or a separate `bots.go`:
```go
// Bot ID constants — replicated locally; cannot import bot-service/internal
const (
    BotAlice   int64 = 1
    BotSteve   int64 = 2
    BotVictor  int64 = 3
    BotCharlie int64 = 4
)
```

---

### `internal/domain/item.go` (model, CRUD)

**Analog:** `services/bid-service/internal/domain/bid.go`

**Same imports pattern as wallet.go** (time + bun).

**Item struct — apply decisions D-04/D-05/D-06:**
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

**Key differences from Wallet:**
- Auto-increment PK (`bun:",pk,autoincrement"`) per D-04
- No `UpdatedAt` per D-05 (items are immutable: inserted on win, deleted on buyout)

---

### `internal/domain/errors.go` (utility)

**Analog:** `services/bid-service/internal/domain/errors.go` — exact match

**Full analog file** (lines 1–11):
```go
package domain

import "errors"

var (
    ErrBidTooLow             = errors.New("bid amount is too low")
    ErrAuctionNotActive      = errors.New("auction is not active")
    ErrAuctionNotFound       = errors.New("auction not found")
    ErrInvalidBidData        = errors.New("invalid bid data")
    ErrLockAcquisitionFailed = errors.New("failed to acquire lock for auction")
)
```

**Banking-service adaptation — same structure, different sentinel names (Phase 1 success criterion 5):**
```go
package domain

import "errors"

var (
    ErrWalletNotFound      = errors.New("wallet not found")
    ErrItemNotFound        = errors.New("item not found")
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrItemNotOwnedByBot   = errors.New("item not owned by this bot")
)
```

**Testing pattern:** The corresponding test file `errors_test.go` must verify all four errors are distinct (no cross-match via `errors.Is`). Copy the testify assertion style from any existing `*_test.go` in the repo using `assert.ErrorIs` / `assert.NotErrorIs`.

---

### `internal/domain/ports.go` (utility, request-response)

**Analog:** `services/bid-service/internal/domain/ports.go` — role-match (no transaction pattern)

**Full analog file** (lines 1–41 — shows interface style, method signatures, context-first params, grouped by concern):
```go
package domain

import (
    "context"
    "time"
)

type BidRepository interface {
    Create(ctx context.Context, bid *Bid) error
    GetHighestBid(ctx context.Context, auctionID int64) (float64, error)
    GetWinner(ctx context.Context, auctionID int64) (botID int64, amount float64, err error)
    ListByAuction(ctx context.Context, auctionID int64) ([]*Bid, error)
}

type AuctionSnapshotRepository interface {
    Upsert(ctx context.Context, snapshot *AuctionSnapshot) error
    GetByID(ctx context.Context, auctionID int64) (*AuctionSnapshot, error)
    UpdateStatus(ctx context.Context, auctionID int64, status AuctionStatus) error
}

type EventPublisher interface {
    PublishBidPlaced(ctx context.Context, bid *Bid) error
    PublishBidRejected(ctx context.Context, bid *Bid, reason string) error
    Close() error
}

type LockManager interface {
    AcquireLock(ctx context.Context, auctionID int64, ttl time.Duration) error
    ReleaseLock(ctx context.Context, auctionID int64) error
}
```

**Banking-service adaptation** — copies the interface style exactly. The withTx pattern is new (no analog). Use Option A (`WithTx(tx bun.IDB) WalletRepository`) which matches the constructor-based DI style of this codebase. Verify `bun.IDB` exists via `go doc github.com/uptrace/bun IDB` after `go mod tidy` — if absent, fall back to `bun.Tx`:

```go
package domain

import (
    "context"

    "github.com/uptrace/bun"
)

type WalletRepository interface {
    GetWallet(ctx context.Context, botID int64) (*Wallet, error)
    UpdateBalance(ctx context.Context, botID int64, newBalance float64) error
    WithTx(tx bun.IDB) WalletRepository
}

type ItemRepository interface {
    GetItemsByBot(ctx context.Context, botID int64) ([]*Item, error)
    GetItem(ctx context.Context, itemID int64) (*Item, error)
    InsertItem(ctx context.Context, item *Item) error
    DeleteItem(ctx context.Context, itemID int64) error
    WithTx(tx bun.IDB) ItemRepository
}

type BankingService interface {
    GetWallet(ctx context.Context, botID int64) (*Wallet, []*Item, error)
    RecordWin(ctx context.Context, botID, auctionID int64, title string, purchasePrice float64) error
    Buyout(ctx context.Context, botID, itemID int64) error
}

type EventConsumer interface {
    Start(ctx context.Context) error
    Close() error
}
```

**withTx pattern — usage in service layer (Phase 3 reference, not Phase 1 work):**
```go
err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
    txWalletRepo := walletRepo.WithTx(tx)
    txItemRepo   := itemRepo.WithTx(tx)
    if err := txWalletRepo.UpdateBalance(ctx, botID, newBalance); err != nil {
        return err
    }
    return txItemRepo.InsertItem(ctx, item)
})
```

**Rules from analog:**
- `context.Context` is always the first parameter
- Methods grouped by domain entity (wallet methods together, item methods together)
- Return errors directly — no wrapping at the interface boundary
- `Close() error` on the `EventConsumer` interface (matches `EventPublisher.Close()` in bid-service)

---

### `internal/config/config.go` (config)

**Analog:** `services/auction-service/internal/config/config.go` — exact match

**Full analog file** (lines 1–72):
```go
package config

import (
    "fmt"
    "os"
    "strconv"
    "time"
)

type Config struct {
    Port                 int
    DatabaseURL          string
    RabbitMQURL          string
    BidServiceURL        string
    GeminiAPIKey         string
    LogLevel             string
    EndingSoonThreshold  time.Duration
    AuctionCheckInterval time.Duration
}

func Load() (*Config, error) {
    port := getEnvInt("PORT", 0)
    if port == 0 {
        port = getEnvInt("AUCTION_SERVICE_PORT", 8081)
    }
    dbURL := getEnv("DATABASE_URL", "")
    rabbitURL := getEnv("RABBITMQ_URL", "")
    // ...
    if dbURL == "" {
        return nil, fmt.Errorf("DATABASE_URL is required")
    }
    if rabbitURL == "" {
        return nil, fmt.Errorf("RABBITMQ_URL is required")
    }
    return &Config{ /* ... */ }, nil
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

**Banking-service adaptation — strip auction-specific fields, add port 8084:**
```go
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
    // Railway injects PORT; fall back to BANKING_SERVICE_PORT for local dev.
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

// getEnv and getEnvInt — copy verbatim from auction-service/internal/config/config.go lines 55–72
```

**Rules from analog:**
- Railway's `PORT` env var takes precedence; service-specific env var is the local-dev fallback
- `DATABASE_URL` and `RABBITMQ_URL` are required (return error if absent)
- `LOG_LEVEL` defaults to `"info"` (no error if absent)
- No `init()` — `Load()` is an explicit constructor call in `main.go`
- No `time.Duration` fields needed in Phase 1 (no scheduled jobs)

---

## Shared Patterns

### bun Struct Tags
**Source:** `services/bid-service/internal/domain/bid.go` lines 17–27
**Apply to:** `wallet.go`, `item.go`
```go
// Natural PK (no autoincrement):
BotID int64 `bun:",pk"`

// Auto-increment PK:
ID int64 `bun:",pk,autoincrement"`

// Non-null column:
Balance float64 `bun:",notnull"`

// Timestamp with nullzero + DB default:
CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`

// Table name on embedded BaseModel:
bun.BaseModel `bun:"table:wallets"`
```

### Sentinel Error Pattern
**Source:** `services/bid-service/internal/domain/errors.go` lines 1–11
**Apply to:** `errors.go`
```go
package domain

import "errors"

var (
    ErrXxx = errors.New("descriptive lowercase message")
)
```
Handlers use `errors.Is(err, domain.ErrXxx)` — never string-match. Never define custom error types with `Error()` methods.

### Config env-var helpers
**Source:** `services/auction-service/internal/config/config.go` lines 55–72
**Apply to:** `config/config.go`

Copy `getEnv` and `getEnvInt` verbatim — they are identical across all services and have no service-specific logic.

### Package-level import path prefix
**Source:** `services/bid-service/internal/repo/bid_repo.go` line 8
**Apply to:** All files that import internal packages
```go
"github.com/Osireg17/AI-Bidding-Platform/services/banking-service/internal/domain"
```
Module path confirmed in STATE.md and RESEARCH.md.

### No init() rule
**Source:** `claude/claude.local.md` §4
**Apply to:** All files in this phase

No `init()` functions with side effects. All initialisation happens in `Load()` (config) or constructors passed into `main()`.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/domain/ports.go` — `WithTx` method shape | utility | CRUD/transaction | No existing service has cross-repo transaction interfaces. Design discretion: use Option A (`WithTx(tx bun.IDB) WalletRepository`). Verify `bun.IDB` exists in v1.2.18 with `go doc` after `go mod tidy`. |

---

## Metadata

**Analog search scope:** `services/bid-service/`, `services/auction-service/`, `services/bot-service/`, `go.work`
**Files scanned:** 8 source files read in full
**Pattern extraction date:** 2026-05-19
