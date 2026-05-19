# Structure
<!-- last_mapped: 2026-05-19 -->

## Top-Level Layout

```
AI-Bidding-Platform/
├── services/                  # 4 independent Go microservices (one go.mod each)
│   ├── auction-service/       # Auction lifecycle management
│   ├── bid-service/           # Bid processing + Redis locking
│   ├── bff/                   # WebSocket gateway + read model
│   └── bot-service/           # ADK Go AI agents
├── shared/                    # Cross-service contracts + tiny utilities
│   ├── events/                # Event schemas, routing keys, Envelope type
│   └── pkg/
│       └── messaging/         # RabbitMQ connection helpers
├── frontend/                  # React + Vite dashboard
│   └── src/
│       ├── components/        # UI components (AuctionCard, BidFeed, etc.)
│       │   └── ui/            # shadcn/ui primitives (card, badge, etc.)
│       ├── hooks/             # useAuction.ts — WebSocket subscription
│       ├── types/             # auction.ts — shared TypeScript types
│       ├── lib/               # bffUrl.ts, utils.ts
│       └── __tests__/         # Vitest tests
├── infra/
│   ├── compose/               # docker-compose.yml for local dev infra
│   └── migrations/            # SQL schema reference files (see CONCERNS.md)
├── claude/                    # Coding standards (claude.local.md)
├── docs/                      # Design doc PDF + architecture diagrams
├── go.work                    # Go workspace linking all 4 service modules + shared
├── go.work.sum
├── Makefile                   # Common tasks: run-all, test, infra-up/down
├── .env                       # Local secrets (gitignored)
└── .env.example               # Env var reference
```

## Per-Service Internal Structure

All 4 services follow an identical internal package layout:

```
services/<service>/
├── cmd/<service>/
│   └── main.go                # Entry point — wires dependencies, starts server
└── internal/
    ├── config/
    │   └── config.go          # Env parsing via os.Getenv; struct with defaults
    ├── domain/
    │   ├── <entity>.go        # Core types (Bid, Auction, AuctionSnapshot, Bot...)
    │   ├── errors.go          # Typed sentinel errors (ErrBidTooLow, etc.)
    │   └── ports.go           # Interfaces: BidRepository, LockManager, EventPublisher
    ├── service/
    │   └── <name>_service.go  # Use-case orchestration; calls repos + publishers
    ├── repo/
    │   ├── <entity>_repo.go   # PostgreSQL query implementations
    │   └── migrate.go         # Schema creation (bun ORM, IfNotExists)
    ├── http/
    │   ├── router.go          # Gin engine setup, route registration
    │   ├── handlers.go        # HTTP handlers (parse → call service → respond)
    │   └── middleware.go      # Auth/logging middleware
    ├── mq/
    │   ├── publisher.go       # RabbitMQ publish (Envelope wrapping)
    │   └── consumer.go        # RabbitMQ consume + dispatch
    ├── lock/                  # (bid-service only)
    │   └── redis_lock.go      # SET NX EX lock implementation
    ├── observability/
    │   └── logger.go          # zap logger initialisation
    └── test/
        ├── mocks.go           # Hand-written interface mocks with call tracking
        └── testutil.go        # Test helpers, functional option builders
```

**bot-service** has additional packages:
```
bot-service/internal/
├── agent/                     # LlmAgent definitions + personality system prompts
├── bidclient/                 # HTTP client wrapping bid-service REST API
├── tools/                     # ADK tool definitions (PlaceBid, GetAuctionDetails, etc.)
└── mq/consumer.go             # Event fan-out dispatcher to all agent goroutines
```

**bff** has additional packages:
```
bff/internal/
├── broadcaster/               # Goroutine-safe WS client map + broadcast logic
└── store/                     # In-memory StateStore (auction read model)
```

## Key File Locations

| Purpose | Path |
|---|---|
| Bid placement logic | `services/bid-service/internal/service/bid_service.go` |
| Bid handler (HTTP) | `services/bid-service/internal/http/handlers.go` |
| Redis lock impl | `services/bid-service/internal/lock/redis_lock.go` |
| Bid domain ports | `services/bid-service/internal/domain/ports.go` |
| Bid domain errors | `services/bid-service/internal/domain/errors.go` |
| Bid service tests | `services/bid-service/internal/service/bid_service_test.go` |
| Bid test mocks | `services/bid-service/internal/test/mocks.go` |
| Event schemas | `shared/events/auction_events.go` |
| Event envelope | `shared/events/envelope.go` |
| RabbitMQ helpers | `shared/pkg/messaging/` |
| Bot agent definitions | `services/bot-service/internal/agent/` |
| Bot MQ fan-out | `services/bot-service/internal/mq/consumer.go` |
| BFF broadcaster | `services/bff/internal/broadcaster/broadcaster.go` |
| BFF state store | `services/bff/internal/store/state_store.go` |
| WS subscription hook | `frontend/src/hooks/useAuction.ts` |
| Frontend types | `frontend/src/types/auction.ts` |
| Coding standards | `claude/claude.local.md` |
| Local dev commands | `Makefile` |
| Infra compose | `infra/compose/docker-compose.yml` |

## Naming Conventions

| Item | Convention | Examples |
|---|---|---|
| Go files | `snake_case.go` | `bid_service.go`, `redis_lock.go` |
| Go test files | `<file>_test.go` | `bid_service_test.go` |
| Go packages | lowercase, single word | `service`, `domain`, `repo`, `mq` |
| Exported types | PascalCase | `BidService`, `AuctionSnapshot` |
| Interfaces | Noun (no `I` prefix) | `BidRepository`, `LockManager` |
| Constructors | `New<Type>` | `NewBidService`, `NewBidHandler` |
| React components | `PascalCase.tsx` | `AuctionCard.tsx`, `BidFeed.tsx` |
| React hooks | `use<Name>.ts` | `useAuction.ts` |
| React tests | `<Component>.test.tsx` | `AuctionCard.test.tsx` |
| Service ports | sequential from 8080 | BFF:8080, auction:8081, bid:8082, bot:8083 |

## Where to Add New Code

| Need | Location |
|---|---|
| New event type | `shared/events/auction_events.go` + routing key in `shared/events/` |
| New service API endpoint | `internal/http/handlers.go` + register in `router.go` |
| New business rule | `internal/domain/` (type/error) + `internal/service/` (logic) |
| New DB query | `internal/repo/<entity>_repo.go` + add to port interface |
| New bot personality | `bot-service/internal/agent/` + register in consumer fan-out |
| New UI component | `frontend/src/components/<Name>.tsx` |
| New shared utility | `shared/pkg/` (keep small — not a junk drawer) |
| New infra service | `infra/compose/docker-compose.yml` |
