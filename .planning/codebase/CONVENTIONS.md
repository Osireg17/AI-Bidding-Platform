# Conventions
<!-- last_mapped: 2026-05-19 -->

## Language & Formatting

- **Go**: `gofmt` enforced (non-negotiable). Prefer clarity over cleverness; no magic framework abstractions.
- **TypeScript/React**: Standard tsconfig strict mode; no explicit linting config observed (ESLint not configured in package.json).
- File naming: snake_case for Go files (`bid_service.go`, `redis_lock.go`), PascalCase for React components (`AuctionCard.tsx`).

## Package Design

Each Go service follows a strict internal package layout:

```
services/<svc>/
├── cmd/<svc>/main.go          # Only file with main()
└── internal/
    ├── config/                # Env parsing, defaults
    ├── domain/                # Types, ports (interfaces), business rules
    ├── service/               # Use-case / application logic
    ├── repo/                  # DB repository implementations
    ├── http/                  # Gin router, handlers, middleware
    ├── mq/                    # RabbitMQ publisher/consumer
    ├── observability/         # Structured logging (zap)
    └── test/                  # Mocks, test helpers (internal only)
```

Rules:
- No circular imports between packages.
- `shared/` holds only event contracts and tiny utilities — not business logic.
- Handlers never touch the DB directly; they parse input and call a service method.

## Naming Conventions

| Thing | Convention | Example |
|---|---|---|
| Exported types | PascalCase | `BidService`, `AuctionSnapshot` |
| Interfaces (ports) | Noun or Nounverb | `BidRepository`, `LockManager`, `EventPublisher` |
| Constructors | `New<Type>` | `NewBidHandler`, `NewBidService` |
| Error variables | `Err<Description>` | `ErrBidTooLow`, `ErrAuctionNotFound` |
| HTTP request structs | `<verb>Request` | `createBidRequest` (unexported) |
| HTTP response structs | `<thing>Response` | `bidResponse`, `highestBidResponse` |
| Test helpers | Func options with `With<Field>` | `WithSnapshotAuctionID`, `WithSnapshotStatus` |
| Domain IDs | `int64` typed | `AuctionID int64`, `BotID int64` |

## Error Handling

- **Never ignore errors** — every `err` must be checked.
- **Typed sentinel errors** for business rules defined in `domain/errors.go`:
  - `ErrBidTooLow`, `ErrAuctionNotActive`, `ErrAuctionNotFound`, `ErrInvalidBidData`, `ErrLockAcquisitionFailed`
- Handlers use `errors.Is()` switch to map domain errors → HTTP status codes:
  ```go
  case errors.Is(err, domain.ErrBidTooLow), errors.Is(err, domain.ErrAuctionNotActive):
      c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
  ```
- Service-layer errors propagate upward; handlers only do mapping.
- Publish errors are logged but do not fail the primary operation (best-effort publish).

## Dependency Injection

- All dependencies passed via constructors (no global singletons, no `init()` side effects).
- Interfaces defined in `domain/ports.go` per service; concrete implementations in `repo/`, `mq/`, `lock/`.
- Compile-time interface guards used in test mocks:
  ```go
  var _ domain.BidRepository = (*MockBidRepository)(nil)
  ```

## HTTP Handler Pattern (Gin)

Handlers follow a strict 3-step pattern:
1. Parse & validate input (`ShouldBindJSON`, query params)
2. Call service method
3. Map result/error to HTTP response

Example from `services/bid-service/internal/http/handlers.go`:
```go
func (h *BidHandler) HandlePlaceBid(c *gin.Context) {
    var req createBidRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    bid, err := h.svc.PlaceBid(c.Request.Context(), req.AuctionID, req.BotID, req.Amount)
    // ... error mapping then c.JSON(http.StatusCreated, ...)
}
```

## Context Usage

- Every request path receives and passes `context.Context`.
- DB queries, RabbitMQ calls, and external HTTP calls all accept context.
- Bot agent goroutines use context for cancellation/deadline propagation.

## Logging (zap)

- Structured logging via `go.uber.org/zap` in all services.
- Logger initialized in `observability/logger.go`, injected via constructors.
- Every significant log line includes domain identifiers:
  ```go
  h.logger.Error("failed to place bid",
      zap.Int64("auction_id", req.AuctionID),
      zap.Int64("bot_id", req.BotID),
      zap.Error(err),
  )
  ```
- Log levels: Debug (flow), Info (lifecycle), Warn (recoverable), Error (attention needed).

## RabbitMQ Conventions

- Exchange: `auction.events` (Topic type)
- All events published as versioned `shared/events.Envelope` (never raw domain structs)
- Envelope fields: `event_id` (uuid), `event_type`, `event_version`, `occurred_at`, `correlation_id`, `payload`
- Consumers must be idempotent (handle duplicate delivery safely)
- Lock key pattern: `auction:{auction_id}:lock`

## Redis Lock Pattern

```go
// Acquire with TTL, then always defer release
if err := lockMgr.AcquireLock(ctx, auctionID, ttl); err != nil {
    return nil, err
}
defer lockMgr.ReleaseLock(ctx, auctionID)
```

The defer ensures lock release on every code path, including early returns and panics.

## Frontend Conventions

- **TanStack Query** for all server state; no manual `useState`/`useEffect` fetch machines.
- WebSocket subscription isolated in `hooks/useAuction.ts`.
- **shadcn/ui** components in `components/ui/`, project components in `components/`.
- Tailwind CSS for all styling; no external CSS files per component.
- Types in `types/auction.ts` shared across components.
