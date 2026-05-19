# Testing
<!-- last_mapped: 2026-05-19 -->

## Frameworks & Libraries

### Go (Backend)
- **testify** (`github.com/stretchr/testify`) — assertions (`assert`) and hard-stop assertions (`require`)
- Standard `testing` package — `t.Run`, `t.Helper`, table-driven tests
- No external test runner; uses `go test ./...`

### Frontend
- **Vitest** (via `vite`) — test runner
- **@testing-library/react** — component testing utilities
- **@testing-library/jest-dom** — custom DOM matchers
- Setup file: `frontend/src/__tests__/setup.ts`

## Test File Locations

### Go
Tests are colocated with production code in the same package:

| Package | Test Files |
|---|---|
| `bid-service/internal/service` | `bid_service_test.go` |
| `bid-service/internal/http` | `handlers_test.go`, `router_test.go`, `middleware_test.go` |
| `bid-service/internal/lock` | `redis_lock_test.go` |
| `bid-service/internal/mq` | `publisher_test.go`, `consumer_test.go` |
| `bff/internal/http` | `handlers_test.go`, `router_test.go` |
| `bff/internal/mq` | `consumer_test.go` |
| `bff/internal/broadcaster` | `broadcaster_test.go` |
| `bff/internal/store` | `state_store_test.go` |
| `auction-service/internal/service` | `auction_service_test.go` |
| `auction-service/internal/http` | `handlers_test.go`, `router_test.go`, `middleware_test.go` |
| `auction-service/internal/mq` | `publisher_test.go` |
| `auction-service/internal/domain` | `auction_test.go` |
| `bot-service/internal/bidclient` | `client_test.go` |
| `bot-service/internal/mq` | `consumer_test.go` |
| `shared/events` | `envelope_test.go` |

### Frontend
- `frontend/src/__tests__/` — all frontend tests
  - `AuctionCard.test.tsx`
  - `BidFeed.test.tsx`
  - `WinnerBanner.test.tsx`
  - `useAuction.test.ts`

## Mocking Strategy

### Go: Interface-based test doubles in `internal/test/`

Each service has `internal/test/mocks.go` with hand-written mocks implementing domain port interfaces. Mocks track call counts and arguments for assertion:

```go
// services/bid-service/internal/test/mocks.go
type MockBidRepository struct {
    mu sync.Mutex
    CreateCalls int
    CreateArgs  []CreateBidArgs
    CreateErr   error
    // ...
}
// Compile-time guard:
var _ domain.BidRepository = (*MockBidRepository)(nil)
```

**Key mocks in bid-service:**
- `MockBidRepository` — tracks Create/GetHighestBid/ListByAuction calls
- `MockAuctionSnapshotRepository` — tracks Upsert/GetByID/UpdateStatus calls
- `MockEventPublisher` — tracks PublishBidPlaced/PublishBidRejected calls
- `MockLockManager` — tracks AcquireLock/ReleaseLock calls

Test helpers use functional options for setup:
```go
// services/bid-service/internal/test/testutil.go
testutil.CreateTestSnapshot(
    testutil.WithSnapshotAuctionID(1),
    testutil.WithSnapshotStartPrice(10.0),
    testutil.WithSnapshotStatus(domain.AuctionStatusActive),
)
```

### No external mock generators (no gomock/mockery) — all mocks are handwritten.

## Test Patterns

### Unit Tests (Service Layer)
Service tests inject all mocks via constructor. Tests verify:
- Correct mock methods called the right number of times
- Arguments passed are correct (using `assert.Same` for pointer identity)
- Error propagation (errors from mocks bubble up correctly)
- Lock always released via defer on every code path

Example test structure from `bid_service_test.go`:
```go
func TestPlaceBid_HappyPath_FirstBid(t *testing.T) {
    ctx := context.Background()
    bidRepo := &testutil.MockBidRepository{GetHighestBidResult: 0}
    snapshotRepo := &testutil.MockAuctionSnapshotRepository{GetByIDResult: snapshot}
    lockMgr := &testutil.MockLockManager{}
    publisher := &testutil.MockEventPublisher{}
    svc := newTestService(bidRepo, snapshotRepo, lockMgr, publisher, t)

    bid, err := svc.PlaceBid(ctx, 1, 100, 20.0)
    require.NoError(t, err)
    assert.Equal(t, 1, bidRepo.CreateCalls)
    assert.Equal(t, 1, publisher.PublishBidPlacedCalls)
}
```

### Table-Driven Tests
Used for testing multiple related cases (e.g., lock-release-on-every-path):
```go
cases := []struct{ name string; ... }{...}
for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) { ... })
}
```

### HTTP Handler Tests
Handlers tested with `httptest.NewRecorder()` and `httptest.NewRequest()`. Router tested for registered routes.

### Test Naming Convention
`Test<Type>_<Scenario>_<Condition>` — e.g.:
- `TestPlaceBid_HappyPath_FirstBid`
- `TestPlaceBid_LockAcquisitionFailed`
- `TestPlaceBid_BidTooLow_NoExistingBids`
- `TestPlaceBid_LockReleasedOnEveryPath`

## Coverage & Gaps

### Well-tested
- `bid-service/internal/service` — comprehensive service-layer unit tests (13 test cases)
- Lock release on every path explicitly tested
- Publish error non-failure tested (best-effort publish)

### Not tested / gaps
- `bot-service` — only `bidclient` and `mq/consumer` have tests; agent logic, tool implementations, spending cap logic appear untested
- `auction-service` — repo layer (DB queries) untested; only service/handler/mq tested
- No integration tests against real DB/Redis/RabbitMQ observed
- `bff` — broadcaster and state store unit-tested but WebSocket handler end-to-end not covered
- Frontend — 4 component tests; hooks tested via `useAuction.test.ts`; no E2E tests

## Running Tests

```bash
# All Go tests
make test
# or
go test ./...

# Single service
cd services/bid-service && go test ./...

# Frontend tests
cd frontend && npm test
```

## Standards (from claude.local.md)

- Tests must be **deterministic** — no `time.Sleep` unless unavoidable
- Use `require` for setup assertions (hard stop), `assert` for verification
- Test all three layers: domain rules, service logic, event contracts (happy + failure + idempotency)
- Integration tests for DB repos (currently aspirational — not yet implemented)
