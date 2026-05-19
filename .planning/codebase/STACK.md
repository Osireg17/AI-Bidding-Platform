# Technology Stack

**Analysis Date:** 2026-05-19

## Languages

**Primary:**
- Go 1.25.6 — all four backend microservices and the shared module
- TypeScript 5.6.3 — React frontend

**Secondary:**
- SQL — PostgreSQL migrations in `infra/migrations/`

## Runtime

**Environment:**
- Go toolchain 1.25.6 (Go workspace via `go.work`)
- Node.js 22 (frontend build via `frontend/Dockerfile`)

**Package Manager:**
- Go modules (per-service `go.mod` + workspace `go.work`)
- npm (frontend) — lockfile: `frontend/package-lock.json` present

## Frameworks

**Core (backend):**
- Gin v1.12.0 — HTTP router used in `auction-service`, `bid-service`, and `bff`; not used in `bot-service` (consumer-only)

**Core (frontend):**
- React 18.3.1 — UI library
- Vite 5.4.11 — dev server and bundler
- Tailwind CSS 4.2.2 — utility-first CSS (integrated via `@tailwindcss/vite` plugin)
- shadcn/ui 4.1.0 — component library layered on Radix UI

**Testing:**
- Backend: `github.com/stretchr/testify v1.11.1` (all services)
- Frontend: Vitest 2.1.8 with `@testing-library/react 16.0.1`

**Build/Dev:**
- Make — `Makefile` at repo root coordinates infra, service, and frontend commands
- Docker Compose — `infra/compose/docker-compose.yml` for local infra (Postgres, Redis, RabbitMQ)
- Caddy 2 (alpine) — static file server for the frontend in production (`frontend/Caddyfile`)

## Key Dependencies

**Critical:**
- `github.com/uptrace/bun v1.2.18` — ORM and query builder used in `auction-service`, `bid-service`, `bot-service`; PostgreSQL dialect via `bun/dialect/pgdialect` and driver via `bun/driver/pgdriver`
- `github.com/rabbitmq/amqp091-go v1.10.0` — RabbitMQ AMQP client, present in all four services and the shared module
- `github.com/redis/go-redis/v9 v9.17.3` — Redis client used exclusively in `bid-service` for distributed bid locking
- `google.golang.org/adk v0.5.0` — Google Agent Development Kit, used in `auction-service` and `bot-service` for LLM agent orchestration
- `google.golang.org/genai v1.48.0` — Google Generative AI SDK, used alongside ADK for Gemini model access

**Infrastructure:**
- `go.uber.org/zap v1.27.1` — structured JSON logger in all services and the shared module
- `github.com/google/uuid v1.6.0` — UUID generation across services
- `github.com/go-playground/validator/v10 v10.30.1` — request validation in Gin-based services
- Radix UI primitives (`@radix-ui/react-scroll-area`, `@radix-ui/react-slot`) — headless component primitives for shadcn/ui
- `class-variance-authority v0.7.1`, `clsx v2.1.1`, `tailwind-merge v2.5.4` — Tailwind variant and class utilities

## Configuration

**Environment:**
- All backend services read configuration exclusively from environment variables via `os.LookupEnv` in each service's `internal/config/config.go`
- Local development: `.env` file at repo root, loaded by the Makefile via `env $(shell grep -v '^#' .env | xargs)`
- Production: variables injected by Railway at deploy time
- Template: `.env.example` at repo root documents all required variables

**Required variables per service:**

| Variable | Services |
|---|---|
| `DATABASE_URL` | auction-service, bid-service, bot-service |
| `RABBITMQ_URL` | all four services |
| `REDIS_URL` | bid-service |
| `BID_SERVICE_URL` | auction-service, bff, bot-service |
| `AUCTION_SERVICE_URL` | bff |
| `GEMINI_API_KEY` | auction-service (optional — disables agent if absent) |
| `GOOGLE_API_KEY` | bot-service (required) |
| `ALLOWED_ORIGIN` | bff (defaults to `*`) |
| `LOG_LEVEL` | all services (defaults to `info`) |
| `PORT` | all services (Railway injection; falls back to per-service port var) |
| `VITE_BFF_URL` | frontend build (Docker `ARG`) |

**Build:**
- `frontend/vite.config.ts` — Vite config; defines `@` path alias to `./src`, dev proxy for `/api` to `VITE_BFF_URL`
- `frontend/tsconfig.app.json`, `frontend/tsconfig.node.json` — TypeScript compiler settings
- `go.work` — Go workspace linking all four services and `shared`

## Platform Requirements

**Development:**
- Go 1.25.6+
- Node.js 22+, npm
- Docker and Docker Compose (for local infra)
- Make

**Production:**
- Deployment target: Railway (all five deployables — four Go services + frontend — have `railway.json` config files)
- Go services built with Railpack builder, output binary `out`, started with `./out`
- Frontend built as a static site, served by Caddy 2 via `frontend/Caddyfile`
- Health check endpoints: `/health` on all Go services; `/` on the frontend

---

*Stack analysis: 2026-05-19*
