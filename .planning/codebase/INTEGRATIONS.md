# External Integrations

**Analysis Date:** 2026-05-19

## APIs & External Services

**Google Gemini / Generative AI:**
- Service: Google Gemini LLM (models `gemini-3.5-flash`, `gemini-3.1-flash-lite`)
- SDK/Client: `google.golang.org/adk v0.5.0` (Agent Development Kit) + `google.golang.org/genai v1.48.0`
- Auth (auction-service): env var `GEMINI_API_KEY` — optional; if unset, the auction agent is disabled at startup
- Auth (bot-service): env var `GOOGLE_API_KEY` — required; service fails to start without it
- Used in:
  - `services/auction-service/internal/agent/auction_agent.go` — generates new auction items after an auction ends
  - `services/bot-service/internal/agent/bot_agent.go` — AI bidding bot that decides when and how much to bid
- Model fallback chain: `gemini-3.5-flash` → `gemini-3.1-flash-lite` on transient 429 rate limits
- Spending cap handling: `RESOURCE_EXHAUSTED` (429) errors are treated as a hard stop (`ErrSpendingCapExhausted`), not retried with fallback models, since all models share the same project quota

## Data Storage

**Databases:**
- Type/Provider: PostgreSQL 16 (Alpine image in Docker Compose; managed instance on Railway in production)
- Container: `bidding-postgres` (`infra/compose/docker-compose.yml`)
- Connection: env var `DATABASE_URL` (standard PostgreSQL DSN)
- Client: `github.com/uptrace/bun v1.2.18` with `pgdialect` and `pgdriver`
- Used in: `auction-service`, `bid-service`, `bot-service`
- Schema managed by: SQL migration files in `infra/migrations/` (run manually; no migration runner embedded in code)
  - `001_create_auctions.sql`
  - `002_create_bids.sql`
  - `003_create_auction_snapshots.sql`
  - `004_create_bot_bids.sql`

**File Storage:**
- Not detected — no cloud storage SDK imports found

**Caching:**
- Provider: Redis 7 (Alpine image in Docker Compose)
- Container: `bidding-redis` (`infra/compose/docker-compose.yml`)
- Connection: env var `REDIS_URL` (e.g. `redis://localhost:6379`)
- Client: `github.com/redis/go-redis/v9 v9.17.3`
- Used in: `bid-service` only (`services/bid-service/internal/`)
- Purpose: distributed bid locking — `BID_LOCK_TTL_SEC` env var controls lock TTL (default 5 seconds)

## Message Queue

**Provider:** RabbitMQ 3 with management plugin (Alpine image in Docker Compose)
- Container: `bidding-rabbitmq` (`infra/compose/docker-compose.yml`)
- Management UI exposed on port 15672
- Connection: env var `RABBITMQ_URL` (AMQP DSN, e.g. `amqp://guest:guest@localhost:5672/`)
- Client: `github.com/rabbitmq/amqp091-go v1.10.0` — present in all four services and the `shared` module
- Auth: `RABBITMQ_DEFAULT_USER` / `RABBITMQ_DEFAULT_PASS` env vars

**Message flow:**
- `auction-service` publishes auction lifecycle events (`services/auction-service/internal/mq/publisher.go`)
- `bid-service` publishes bid-placed events (`services/bid-service/internal/mq/publisher.go`)
- `bid-service` consumes auction events (`services/bid-service/internal/mq/consumer.go`)
- `bff` consumes bid and auction events and fans them out to SSE clients (`services/bff/internal/mq/consumer.go`)
- `bot-service` consumes auction events to trigger bidding decisions (`services/bot-service/internal/mq/consumer.go`)
- Shared event type definitions live in `shared/events/`

## Authentication & Identity

**Auth Provider:** None — no authentication or identity system is detected. All API endpoints appear to be unauthenticated. There is no JWT, session, OAuth, or API key middleware in any service.

## Real-Time Updates

**Protocol:** Server-Sent Events (SSE)
- Implementation: `services/bff/internal/broadcaster/broadcaster.go` — in-memory fan-out broadcaster
- Handler: `services/bff/internal/http/handlers.go` (`HandleStream`)
- BFF subscribes to RabbitMQ, then re-broadcasts events to connected browser clients over SSE
- SSE logging is skipped in middleware (`services/bff/internal/http/middleware.go`) because the connection is long-lived

## Monitoring & Observability

**Error Tracking:** Not detected — no Sentry, Datadog, or equivalent SDK imports found.

**Logs:**
- Provider: `go.uber.org/zap v1.27.1` — structured JSON production logger in all services
- Logger initialised in each service's `internal/observability/` package with configurable level via `LOG_LEVEL` env var
- Log level defaults to `info`

**Tracing/Metrics:** OpenTelemetry packages appear as transitive indirect dependencies (pulled in by the Google ADK and genai SDKs) but no direct OTel instrumentation is configured in application code.

## CI/CD & Deployment

**Hosting:** Railway
- Each deployable unit has a `railway.json` at its root:
  - `services/auction-service/railway.json` — Railpack builder, binary `out`, healthcheck `/health`
  - `services/bid-service/railway.json` — Railpack builder, binary `out`, healthcheck `/health`
  - `services/bff/railway.json` — Railpack builder, binary `out`, healthcheck `/health`
  - `services/bot-service/railway.json` — Railpack builder, binary `out`, no healthcheck configured
  - `frontend/railway.json` — no builder specified (Railway auto-detects Docker), healthcheck `/`
- Frontend built as Docker image: Node 22 Alpine builder → Caddy 2 Alpine static server

**CI Pipeline:** Not detected — no GitHub Actions, CircleCI, or equivalent config files found.

## Environment Configuration

**Required env vars (all environments):**

| Variable | Required by | Notes |
|---|---|---|
| `DATABASE_URL` | auction-service, bid-service, bot-service | PostgreSQL DSN |
| `RABBITMQ_URL` | all four services | AMQP DSN |
| `REDIS_URL` | bid-service | Redis DSN |
| `BID_SERVICE_URL` | auction-service, bff, bot-service | Internal HTTP base URL |
| `AUCTION_SERVICE_URL` | bff | Internal HTTP base URL |
| `GOOGLE_API_KEY` | bot-service | Gemini API key (hard required) |
| `GEMINI_API_KEY` | auction-service | Gemini API key (soft — agent disabled if absent) |
| `POSTGRES_USER` | Docker Compose only | DB container setup |
| `POSTGRES_PASSWORD` | Docker Compose only | DB container setup |
| `POSTGRES_DB` | Docker Compose only | DB container setup |
| `RABBITMQ_DEFAULT_USER` | Docker Compose only | MQ container setup |
| `RABBITMQ_DEFAULT_PASS` | Docker Compose only | MQ container setup |
| `VITE_BFF_URL` | frontend build | Injected as Docker `ARG` |
| `PORT` | all services | Railway runtime injection |
| `ALLOWED_ORIGIN` | bff | CORS origin; defaults to `*` |
| `LOG_LEVEL` | all services | Defaults to `info` |

**Secrets location:** `.env` file at repo root (local dev); Railway environment variables (production). Template at `.env.example`.

## Webhooks & Callbacks

**Incoming:** None detected.

**Outgoing:** None detected — the bot-service calls the `bid-service` HTTP API directly rather than via webhooks.

---

*Integration audit: 2026-05-19*
