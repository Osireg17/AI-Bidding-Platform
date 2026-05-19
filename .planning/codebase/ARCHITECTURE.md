# Architecture
<!-- last_mapped: 2026-05-19 -->

## Pattern

**Event-driven microservices** with synchronous REST for command paths and asynchronous RabbitMQ fanout for event distribution. Each service owns its own PostgreSQL database. No cross-service DB access.

## System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│  Browser                                                         │
│  React Dashboard (Vite + TanStack Query)                         │
└────────────────────┬────────────────────────────────────────────┘
                     │ REST / WebSocket / SSE
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│  BFF (port 8080)                                                 │
│  WebSocket fan-out, read-model aggregation                       │
│  StateStore (in-memory auction state)                            │
│  EventBroadcaster (goroutine-safe client map)                    │
└───────────┬─────────────────────────────────────────────────────┘
            │ Consume: auction.*, bid.*
            ▼
┌───────────────────────────────────────────────────────────────┐
│  RabbitMQ  —  Exchange: auction.events (Topic)                 │
│  bff.q (auction.*, bid.*)                                      │
│  bot.q (auction.*, bid.*)                                      │
└──────┬───────────────────────────┬────────────────────────────┘
       │ Publish                    │ Consume
       │                            ▼
       │              ┌─────────────────────────────┐
       │              │  Bot Service (port 8083)     │
       │              │  4 LlmAgent goroutines       │
       │              │  (ADK Go + Gemini Flash)     │
       │              │  fanOut dispatcher           │
       │              │  BotDB (PostgreSQL)          │
       │              └────────────┬────────────────┘
       │                           │ REST POST /internal/bids
       │                           ▼
       │              ┌─────────────────────────────┐
       │              │  Bid Service (port 8082)     │
       │      Publish │  Redis distributed lock      │
       │◄─────────────│  BidDB (PostgreSQL)          │
       │              └─────────────────────────────┘
       │
       │ Publish
┌──────┴──────────────────────────┐
│  Auction Service (port 8081)    │
│  Scheduled auction creation     │
│  Scheduled auction ending       │
│  Winner determination           │
│  AuctionDB (PostgreSQL)         │
└─────────────────────────────────┘
```

## Service Responsibilities

| Service | Port | Responsibility |
|---|---|---|
| `auction-service` | 8081 | Auction lifecycle: create, start/end, winner determination; cron jobs |
| `bid-service` | 8082 | Bid validation, Redis-locked persistence, bid event publishing |
| `bff` | 8080 | WebSocket/SSE gateway; in-memory read model; data aggregation for frontend |
| `bot-service` | 8083 | ADK-based LLM bot agents; event consumption; REST bid placement |

## Per-Service Internal Architecture

Every service uses the same 4-layer pattern:

```
HTTP (Gin router/handlers)
       ↓ calls
Service (use-case / application logic)
       ↓ calls (via domain interfaces/ports)
Domain (types, business rules, port definitions)
       ↑ implemented by
Repo / MQ / Lock (infrastructure adapters)
```

- **Ports** defined in `internal/domain/ports.go` (interfaces only).
- **Adapters** implemented in `internal/repo/`, `internal/mq/`, `internal/lock/`.
- **Domain** contains types and pure business rules (no infrastructure imports).
- **Service** orchestrates: acquires locks, calls repos, publishes events.

## Core Data Flows

### 1. Auction Creation
```
Auction Service (cron job)
  → INSERT into AuctionDB
  → Publish auction.created event (Envelope)
      → RabbitMQ auction.events exchange
          → bff.q consumed by BFF → StateStore upsert → broadcast to WS clients
          → bot.q consumed by Bot Service → dispatch to all agent goroutines
```

### 2. Bid Placement
```
Bot Agent (goroutine, reacting to auction.created / bid.placed)
  → REST POST /internal/bids to Bid Service
      → AcquireLock(auctionID, TTL)        [Redis SET NX EX]
      → GetByID(auctionID) from BidDB snapshot
      → Validate (status active, amount > start_price and > highest_bid)
      → INSERT bid into BidDB
      → ReleaseLock(auctionID)             [defer]
      → Return 201 bid response
      → PublishBidPlaced(bid) async        [best-effort, non-blocking on failure]
          → RabbitMQ bid.placed
              → BFF rebroadcasts to WS clients
              → Bot Service agents react (competitor awareness)
```

**Concurrency control:** Redis `SET NX EX` per auction. Only one bid processed at a time per auction. Lock TTL prevents deadlock on service crash.

### 3. Auction Ending
```
Auction Service (cron job, polls for expired auctions)
  → GetWinner() from BidDB via REST
  → UPDATE auction status = closed, winner_id
  → Publish auction.ended event
      → BFF broadcasts to dashboard
      → Bot agents receive outcome
```

### 4. Dashboard Updates
```
RabbitMQ event
  → BFF MQ Consumer
  → StateStore.Upsert / Apply event
  → Broadcaster.Broadcast(msg) to all registered WS clients
```

## Key Abstractions

| Interface | Package | Purpose |
|---|---|---|
| `BidRepository` | `bid-service/domain` | Bid persistence (Create, GetHighestBid, GetWinner, ListByAuction) |
| `AuctionSnapshotRepository` | `bid-service/domain` | Local auction cache in BidDB (Upsert, GetByID, UpdateStatus) |
| `LockManager` | `bid-service/domain` | Redis-based per-auction distributed lock |
| `EventPublisher` | each service domain | RabbitMQ publish (typed per service) |
| `BotEvaluator` | `bot-service` | LLM agent decision interface (swappable for testing) |
| `StateStore` | `bff` | In-memory read model for connected dashboard |
| `EventBroadcaster` | `bff` | Goroutine-safe WebSocket/SSE client map |
| `shared/events.Envelope` | `shared/events` | Versioned event wrapper for all RabbitMQ messages |

## Shared Infrastructure

- `shared/events/` — event schemas, routing key constants (`auction.created`, `bid.placed`, etc.), `Envelope` type
- `shared/pkg/messaging/` — small RabbitMQ connection helpers
- `go.work` — Go workspace linking all 4 service modules + shared

## Known Architectural Constraints

1. **Self-loop prevention**: Bot Service's `fanOut` dispatcher must skip forwarding `bid.placed` events back to the bot that placed the bid (otherwise infinite bidding loops).
2. **Spending cap short-circuit**: Bot agent checks spending cap before invoking Gemini; caps the LLM call cost per agent turn.
3. **SSE backpressure**: BFF broadcaster drops events on slow/disconnected clients (non-blocking channel send).
4. **AuctionStatus duplication**: String constants for auction status are defined in both `auction-service` and `bid-service` domain packages — a known inconsistency.
5. **Best-effort publish**: `PublishBidPlaced` failure is logged but does not roll back the bid. DB is the source of truth; events are advisory.
6. **No outbox pattern**: Events are published after DB commit in the same goroutine. A crash between commit and publish will cause a dropped event (known risk, outbox recommended for production).
