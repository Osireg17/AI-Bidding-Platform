# Concerns
<!-- last_mapped: 2026-05-19 -->

## Tech Debt

### 1. Dual Gemini API key env var names
`auction-service` reads `GEMINI_API_KEY`; `bot-service` reads `GOOGLE_API_KEY`. Both refer to the same Gemini credential but use different env var names. `.env.example` documents neither. Causes silent failures when one is set and the other is not.

### 2. Migrations are documentation-only
`infra/migrations/` SQL files are never executed. Schema is created at runtime via `bun.NewCreateTable().IfNotExists()` in each service's `repo/migrate.go`. This means the SQL files can drift from actual schema without any CI gate catching it. Migration tooling (golang-migrate or goose) is not set up.

### 3. Hardcoded bot IDs in consumer dispatch
`bot-service/internal/mq/consumer.go:353` contains a hardcoded list `{1, 3, 4}` instead of referencing `domain.Alice.ID` etc. Adding or removing a bot requires changing both the domain constants and this magic number list.

### 4. `reUnmarshalPayload` copy-pasted across consumers
A double-serialisation helper (`reUnmarshalPayload`) is copied verbatim into three separate consumer files. Any bug fix or behavioural change must be replicated in all three places.

### 5. README model name mismatch
README states "Gemini 3.0 Flash" but code uses `gemini-3.5-flash` / `gemini-3.1-flash-lite`. Model names in documentation will mislead anyone trying to reproduce costs or capabilities.

---

## Known Bugs

### 1. Redis lock ownership race (high risk)
`lock/redis_lock.go` `ReleaseLock` issues a plain `DEL` without verifying the lock UUID. If the 5-second TTL expires mid-operation (e.g. slow DB query), the lock is released by another holder, and `DEL` then deletes the new holder's lock. Correct implementation requires a Lua script: `if GET key == uuid then DEL key`.

### 2. `srv.Close()` instead of `srv.Shutdown()`
`auction-service` and `bff` use `srv.Close()` on SIGTERM, which immediately drops all in-flight connections. `http.Server.Shutdown(ctx)` with a deadline should be used for graceful shutdown.

### 3. BFF state store loses bid history on restart
BFF `hydrate()` in `main.go` replays only the single highest bid per auction on startup, discarding full bid history. After a restart the dashboard shows incomplete data until new events arrive.

### 4. `ProcessExpiredAuctions` blocks scheduler goroutine
`auction-service` uses a synchronous `time.After(30 * time.Minute)` inside the scheduler goroutine when an auction is found closed. This blocks all subsequent auction processing for 30 minutes per occurrence. Should use a non-blocking ticker or move the sleep to a per-auction goroutine.

---

## Security

### 1. No authentication on internal service APIs
`POST /api/bids` (bid-service) and `POST /api/auctions` (auction-service) have no authentication. Any process that can reach these ports can place bids or create auctions. Internal Railway private networking provides network-level isolation, but there are no application-level auth checks.

### 2. BFF CORS defaults to wildcard
`ALLOWED_ORIGIN` env var defaults to `"*"` if unset. In production this allows any origin to make credentialed requests to the BFF.

### 3. No upper-bound validation on bid amounts
`createBidRequest` validates `Amount > 0` but no maximum. A bot (or malicious caller) can place arbitrarily large bids. No domain ceiling enforced.

### 4. API keys absent from `.env.example`
`GEMINI_API_KEY` and `GOOGLE_API_KEY` are not documented in `.env.example`. New developers will not know these are required until runtime failures occur.

---

## Performance

### 1. `ListAuctions` is an unbounded table scan
`auction-service` repo `ListAuctions` fetches all rows with no `LIMIT` or cursor pagination. Will degrade as the auctions table grows.

### 2. HTTP clients have no timeout
All four services construct `&http.Client{}` with no `Timeout` field. A hung downstream (e.g. Gemini API) will hold a goroutine indefinitely.

### 3. HTTP servers have no read/write timeouts
No `ReadTimeout`, `WriteTimeout`, or `IdleTimeout` set on any `http.Server`. Slow clients can hold connections open indefinitely.

### 4. Monetary values stored as `DOUBLE PRECISION` (float64)
All bid amounts use `float64` in Go and `DOUBLE PRECISION` in PostgreSQL. Float arithmetic is imprecise for currency. Should use `NUMERIC(18,4)` in DB and `decimal` library in Go for exact comparisons. The "bid too low" check (`amount > highestBid`) is currently a float comparison.

---

## Fragile Areas

### 1. BFF in-memory state store
`bff/internal/store/state_store.go` holds all auction state in memory. A BFF restart causes complete state loss; the `hydrate()` function only partially recovers it. If multiple BFF replicas were deployed, they would have divergent state.

### 2. RabbitMQ nack with requeue and no DLQ
Consumers nack failed messages with `requeue=true`. Under sustained failure (e.g. a consistently unparseable message) this creates a hot-loop: the broker immediately redelivers and the consumer immediately nacks, burning CPU and preventing other messages from being processed. A dead-letter exchange is not configured.

### 3. Bot selection by magic number list
`bot-service/internal/mq/consumer.go` selects which bots receive `auction.created` via a hardcoded ID list `{1, 3, 4}`. Adding a new bot personality requires updating this list. Not driven by registered bot instances.

### 4. `isRateLimit` divergence between agents
`bot-service` bot-agent has an `isSpendingCapExhausted` check; `auction-agent` uses a simpler `isRateLimit` that does not handle the spending cap case. The two agents have diverged in their error handling for Gemini rate/quota responses.

---

## Test Coverage Gaps

- No end-to-end integration test covering auction creation → bot bid → auction end → dashboard update.
- `isRateLimit` / `isSpendingCapExhausted` helpers have no unit tests.
- Redis lock ownership race (Lua script correctness) is not tested.
- BFF `hydrate()` in `main.go` has zero test coverage.
- `ProcessExpiredAuctions` 30-minute sleep path is excluded from the test suite.
- Bot agent tool implementations (`PlaceBid`, `GetAuctionDetails`, `CheckMyBalance`, `AnalyzeValue`) appear to have no unit tests.
- No integration tests against real PostgreSQL, Redis, or RabbitMQ.
