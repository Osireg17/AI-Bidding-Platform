# claude.local.md — AI Bidding Platform Coding Standard (Go + RabbitMQ + Redis + Postgres + Gin + React)

This document defines the coding standards and "how we build" rules for this repo.
When an AI coding agent proposes changes, it must follow these rules unless explicitly instructed otherwise.
This is a living document -- sections marked [TBD] will be filled in as the project evolves.

---

## 0) Canonical documentation links (read-first)

### Go
- Effective Go: https://go.dev/doc/effective_go
- Go Code Review Comments: https://go.dev/doc/code
- Go Wiki tutorial (structure & patterns): https://go.dev/doc/articles/wiki/

### Web/API & real-time
- WebSocket API (MDN): https://developer.mozilla.org/en-US/docs/Web/API/WebSocket

### Backend framework & libraries
- Gin docs: https://gin-gonic.com/en/docs/
- Testify: https://github.com/stretchr/testify
- zap logging: https://github.com/uber-go/zap
- psql (Go driver helper docs): https://pkg.go.dev/github.com/gopsql/psql

### Messaging / Infra
- RabbitMQ tutorial (Go): https://www.rabbitmq.com/tutorials/tutorial-one-go
- RabbitMQ on Railway: https://railway.com/deploy/rabbitmq
- Railway docs: https://docs.railway.com/

### Redis
- Distributed locks patterns: https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/

### Design patterns
- Refactoring Guru catalog: https://refactoring.guru/design-patterns/catalog

### Frontend
- React build from scratch: https://react.dev/learn/build-a-react-app-from-scratch
- TanStack (Query etc.): https://tanstack.com/
- shadcn/ui: https://ui.shadcn.com/

### AI tooling / agents
- Google ADK docs: https://google.github.io/adk-docs/
- Gemini API (Gemini 3): https://ai.google.dev/gemini-api/docs/gemini-3
- Vertex AI Gemini 3 Flash docs: https://docs.cloud.google.com/vertex-ai/generative-ai/docs/models/gemini/3-flash

---

## 1) Project overview

### What this is
An AI Bidding Platform where autonomous AI bot agents with distinct personalities compete in real-time auctions. A React dashboard shows the auctions, bids, bot activity, and optionally the bots' LLM reasoning -- all updating live via WebSockets.

### High-level architecture
- **4 Go backend services** communicating via REST (synchronous) and RabbitMQ (asynchronous events)
- **React + Vite frontend** connected to BFF via WebSocket for real-time updates
- **PostgreSQL** for persistent storage, **Redis** for distributed locking, **RabbitMQ** for event fanout

### Core system flows
1. **Auction creation**: Auction Service (scheduled job) creates auctions -> persists to DB -> publishes `AuctionCreated` event via RabbitMQ fanout
2. **Bid placement**: Bot Service calls Bid Service via REST -> Bid Service acquires Redis lock -> validates bid -> persists -> releases lock -> returns response -> publishes `BidPlaced` event async
3. **Auction end**: Auction Service (scheduled job) finds expired auctions -> determines winner -> updates DB -> publishes `AuctionEnded` event
4. **Dashboard updates**: BFF consumes all RabbitMQ events -> broadcasts to connected browsers via WebSocket

### Bot agents
- Built with Google ADK for Go (LlmAgent instances running as goroutines)
- Each agent has a unique personality via system prompt (e.g., Aggressive Alice, Sniper Steve, Value Victor, Chaos Charlie)
- Agents use tools (PlaceBid, GetAuctionDetails, CheckMyBalance, AnalyzeValue) to interact with the system
- Single Bot Service process, one RabbitMQ consumer dispatches events to all agents internally
- Database-backed state (balances, bid history, strategy parameters)

---

## 2) Repo principles

### Primary goals
- Correctness first (money-like rules: bids, winners, auction close).
- Deterministic state transitions, traceable events, debuggable production behavior.
- "Simple now, extensible later" (MVP-friendly but not sloppy).

### Service boundaries (MVP)

| Service | Responsibility |
|---|---|
| `auction-service` | Auction lifecycle: create, start/end, determine winners, scheduled jobs |
| `bid-service` | Bid validation, persistence, concurrency control via Redis locks |
| `bff` | WebSocket fan-out to dashboard, read-model endpoints, data aggregation |
| `bot-service` | ADK Go agents, event consumption, strategy execution, tool definitions |

Shared:
- `shared/events`: event schemas + routing key constants + versioning policy
- `shared/pkg`: small shared utilities only (must not become a junk drawer)

---

## 3) Repository structure

```
AI-Bidding-Platform/
├── services/
│   ├── auction-service/    # Go module
│   ├── bid-service/        # Go module
│   ├── bff/                # Go module
│   └── bot-service/        # Go module
├── shared/
│   ├── events/             # Event schemas, routing keys, envelope types
│   └── pkg/                # Small shared utilities
├── frontend/               # React + Vite app
├── infra/
│   ├── compose/             # docker-compose.yml for local dev
│   └── migrations/          # Database migration files
├── claude/                 # Claude coding standard (this file)
├── docs/                   # Project documentation
├── go.work                 # Go workspace file
├── Makefile                # Common tasks: run, build, test
└── README.md
```

### Per-service structure

```
services/<service-name>/
├── cmd/<service-name>/     # main.go entry point
├── internal/
│   ├── config/             # Env parsing, defaults
│   ├── http/               # Gin router + handlers
│   ├── domain/             # Types, business rules
│   ├── service/            # Use-cases / application logic
│   ├── repo/               # DB repositories
│   ├── mq/                 # RabbitMQ publisher/consumer
│   ├── observability/      # Logging, metrics hooks
│   └── test/               # Test helpers, fakes
└── go.mod
```

Rules:
- Only `cmd` contains `main()`.
- Everything else stays in `internal/`.
- `shared` is for event contracts + tiny helpers, not business logic.

---

## 4) Go standards (non-negotiables)

### Formatting & style
- Always use `gofmt` output (no exceptions).
- Prefer clarity over cleverness.
- Avoid "framework magic" abstractions. Explicit dependencies > globals.

### Package design
- Keep packages small and purpose-driven.
- No circular dependencies.
- Keep domain logic out of handlers. Handlers parse/validate input and call a service layer.

### Error handling
- Never ignore errors.
- Allow errors to bubble up to the service layer.
- Wrap errors with context (what failed + key identifiers).
- Return typed/semantic errors for business rules (e.g., "bid too low", "auction ended").

### Context usage
- Every request path uses context.
- Pass context into DB calls and external calls.
- Respect cancellation and deadlines.

### Naming
- Use clear names (`AuctionID`, `BidID`, `AuctionService`, `BidRepository`).
- Avoid single-letter vars except short loops.

Reference: Effective Go + Go Code Review Comments (see links above).

Docs: https://go.dev/doc/effective_go

Docs: https://refactoring.guru/design-patterns/go

---

## 5) Architecture patterns to use (and avoid)

### Recommended patterns
- **Dependency Injection** via constructors (no global singletons).
- **Ports & Adapters** (clean separation):
  - Domain/services depend on interfaces (ports)
  - Infrastructure implements them (adapters)
- **Repository pattern** for DB access
- **Idempotency pattern** for consumers (must)
- **Outbox pattern** (recommended when you start caring about guaranteed publish)

### Avoid
- "God packages" shared across services
- Handlers doing DB queries directly
- Cross-service direct DB access
- Leaking RabbitMQ details throughout the domain layer

Design reference: Refactoring Guru patterns catalog (link above).

---

## 6) Gin HTTP API rules

- Handlers should:
  1. Parse & validate input
  2. Call a service method
  3. Map service outputs/errors to HTTP responses
- Use consistent JSON response envelopes (define once per service).
- Never panic for expected errors.
- Keep routing in one place (router setup).

### API contracts [TBD]

Endpoints will be defined per service as they are built. General patterns:

- **Auction Service**: auction CRUD, lifecycle management
- **Bid Service**: bid placement (`POST /internal/bids`), bid queries
- **BFF**: dashboard data aggregation, WebSocket upgrade endpoint
- **Bot Service**: internal only (no external API, driven by events)

Detailed endpoint specs will be added here as each service is implemented.

Docs: https://gin-gonic.com/en/docs/

---

## 7) Logging (zap) and observability

- Use zap for structured logs.
- Every log line includes:
  - service name
  - correlation/request id (if available)
  - key domain identifiers (auction_id, bid_id, bot_id)
- Log levels:
  - **Debug**: internal flow, dev
  - **Info**: key lifecycle events (auction created, bid placed, auction ended)
  - **Warn**: recoverable issues (bid rejected, lock contention)
  - **Error**: failures requiring attention (DB errors, RabbitMQ connection loss)

Docs: https://github.com/uber-go/zap

Docs: https://betterstack.com/community/guides/logging/go/zap/

---

## 8) Testing standards (testify)

- Aim for:
  - Unit tests for domain rules and service layer
  - Integration tests for DB repos
  - Consumer/producer tests around event contracts (happy + failure + idempotency)
- Use testify for asserts/require.
- Tests must be deterministic (no sleeps unless absolutely unavoidable).

Docs: https://github.com/stretchr/testify

---

## 9) Postgres access rules

- DB access is behind repositories.
- All queries accept context.
- No SQL in handlers.
- Prefer explicit transactions for critical updates (bid placement, auction close).
- Use consistent naming and migrations.

### Database schemas [TBD]

Schemas will be defined per service as they are built. Expected databases:

- **Auction DB**: auctions table (id, title, description, start_price, current_price, status, winner_id, start_time, end_time, etc.)
- **Bid DB**: bids table (id, auction_id, bot_id, amount, timestamp, etc.)
- **Bot DB**: bot config, bot state, bid history, auction tracking, strategy parameters

Detailed schemas will be added here as each service is implemented.

Docs: https://pkg.go.dev/github.com/gopsql/psql

---

## 10) Redis usage rules (locks & coordination)

Use Redis for:
- Distributed locks for bid placement concurrency (per-auction)
- Short-lived coordination, not source-of-truth data

Rules:
- Lock keys must be consistent: `auction:{auction_id}:lock`
- Locks must:
  - have TTL
  - be released safely (defer pattern)
  - not be used as the only correctness mechanism (DB should enforce final truth)

Docs: https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/

Docs: https://docs.railway.com/guides/redis

---

## 11) RabbitMQ rules (MVP topology + conventions)

### Topology

- **Exchange type**: Topic (allows flexible routing key matching)
- **Exchange name**: `auction.events`
- **Queues**:
  - `bff.q` — bound to `auction.*`, `bid.*`
  - `bot.q` — bound to `auction.*`, `bid.*`

### Routing keys [TBD — examples below]

These are starting examples. Final event set will be established as services are built:

- `auction.created`
- `auction.ending_soon`
- `auction.ended`
- `bid.placed`
- `bid.rejected`

### Publishing rules
- Never publish raw domain structs directly.
- Publish versioned event envelopes from `shared/events`.
- Every event envelope includes:
  - `event_id` (uuid)
  - `event_type`
  - `event_version`
  - `occurred_at`
  - `correlation_id` (when available)
  - `payload`

### Consumer rules (must-have)
- **Idempotency**: handle duplicates safely (store processed event_id or use a unique constraint).
- **Retry**: safe reprocessing.
- **Poison message strategy**: DLQ when added later.

Docs:
- RabbitMQ Go tutorial: https://www.rabbitmq.com/tutorials/tutorial-one-go
- Railway RabbitMQ: https://railway.com/deploy/rabbitmq

---

## 12) WebSockets & BFF rules

- BFF is the only service that talks WebSockets to the dashboard.
- BFF consumes events from RabbitMQ and broadcasts to WebSocket clients.
- WebSocket messages must be versioned (same envelope approach as backend events).
- BFF maintains a map of connected clients for broadcasting.

Docs: https://developer.mozilla.org/en-US/docs/Web/API/WebSocket

---

## 13) Frontend standards (React + Vite + TanStack + shadcn)

- Use **TanStack Query** for server state; avoid manual fetch state machines.
- Keep WebSocket subscription logic isolated (one module/hook).
- Use **shadcn/ui** components for consistent UI.
- Use **Tailwind CSS** for styling.

### Dashboard features (target)
- Grid of active auctions with real-time bid updates
- Countdown timers per auction
- Bot status panel (balance, win/loss, activity)
- Activity feed with live event stream
- Optional "Bot Brain" panel showing LLM reasoning stream

Docs:
- React: https://react.dev/learn/build-a-react-app-from-scratch
- TanStack: https://tanstack.com/
- shadcn/ui: https://ui.shadcn.com/

---

## 14) AI agent standards (ADK Go + Gemini)

- Bot Service uses **Google ADK for Go** with **Gemini 3.0 Flash** (streaming).
- Single Bot Service process contains multiple `LlmAgent` instances as goroutines.
- Each agent has a unique personality defined by its system prompt.
- Agents share tool definitions (PlaceBid, GetAuctionDetails, CheckMyBalance, AnalyzeValue).
- Keep agent orchestration separate from domain logic.
- Put model calls behind an interface so it can be swapped or disabled for local dev/testing.
- Database-backed state for persistence across restarts.

### Agent personalities (TBD)
- **Aggressive Alice**: bids early, high increments, targets collectibles
- **Sniper Steve**: waits until final seconds, minimum margin wins
- **Value Victor**: only bids on bargains below 70% estimated value
- **Chaos Charlie**: random bidding for stress testing (no LLM reasoning)
- **Learning Lucy** [future]: adapts strategy based on historical performance

Docs:
- ADK: https://google.github.io/adk-docs/get-started/go/
- Gemini API: https://ai.google.dev/gemini-api/docs/gemini-3
- Vertex AI Gemini 3 Flash: https://docs.cloud.google.com/vertex-ai/generative-ai/docs/models/gemini/3-flash

---

## 15) Local development

### Prerequisites
- Go 1.22+
- Node.js 20+
- Docker & Docker Compose

### Default ports

| Service | Port |
|---|---|
| auction-service | 8081 |
| bid-service | 8082 |
| bff | 8080 |
| bot-service | 8083 |
| React dev server | 5173 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| RabbitMQ | 5672 (AMQP), 15672 (management UI) |

### Commands (defined in Makefile)

```bash
# Start infrastructure (Postgres, Redis, RabbitMQ)
make infra-up

# Stop infrastructure
make infra-down

# Stop infrastructure and delete volumes
make infra-reset

# View infrastructure logs
make infra-logs

# Run a specific service
make run-auction
make run-bid
make run-bff
make run-bot

# Run all services
make run-all

# Run tests
make test

# Run frontend
make run-frontend
```

---

## 16) Deployment

- **Backend services**: Railway (each service gets its own deployment, built from Dockerfile)
- **Frontend**: Vercel (points to BFF's public Railway URL)
- **Infrastructure**: Railway-managed Postgres, Redis, RabbitMQ
- Services communicate internally via Railway's private networking

Docs: https://docs.railway.com/

---

## 17) Definition of done for any PR (AI or human)

- Passes formatting (`gofmt`).
- Tests added/updated for new behavior.
- Logs include key identifiers for significant actions.
- Event contract changes are versioned and documented.
- No cross-service DB access.
- No business logic in HTTP handlers.

---

## 18) Agent operating rules (how the AI coding assistant should work)

When making changes, the agent must:
1. Describe the intent and the files it will touch
2. Prefer small, reviewable diffs
3. Keep public API changes explicit and documented
4. Add tests for logic changes
5. Keep event contracts backward compatible when possible
6. Update this document when new schemas, endpoints, or conventions are established
