# AI Bidding Platform

## Coding Standards

See [`claude/claude.local.md`](claude/claude.local.md) for all coding standards, architecture rules, and service conventions. That document is the source of truth — follow it before touching any code.

## Project Planning (GSD)

This project uses the GSD workflow. Planning artifacts live in `.planning/`:

| Artifact | Path | Purpose |
|----------|------|---------|
| Project context | `.planning/PROJECT.md` | What we're building and why |
| Config | `.planning/config.json` | Workflow preferences |
| Requirements | `.planning/REQUIREMENTS.md` | Scoped v1 requirements with REQ-IDs |
| Roadmap | `.planning/ROADMAP.md` | 5-phase execution plan |
| State | `.planning/STATE.md` | Current phase and progress |
| Codebase map | `.planning/codebase/` | Architecture, structure, conventions, concerns |

**Current milestone:** Banking Service (5 phases)

**Current phase:** Phase 1 — Service Foundation

Read `.planning/STATE.md` for active phase context before starting any task.

## GSD Workflow Rules

- Before planning a phase: read `.planning/ROADMAP.md` and the relevant phase's success criteria
- Before implementing: read `.planning/codebase/CONVENTIONS.md` and `STRUCTURE.md`
- After completing a phase: run `/gsd:verify-phase` before opening a PR
- Never skip the plan-check step — it catches gaps before execution

## Quick Context

- **What**: Autonomous AI auction platform — 4 LLM bot agents (Gemini Flash + ADK Go) competing in real-time auctions
- **Services**: auction-service (8081), bid-service (8082), bff (8080), bot-service (8083), **banking-service (8084)** [new]
- **Event bus**: RabbitMQ topic exchange `auction.events`; new queue `banking.q` binds to `auction.ended`
- **This milestone**: Add financial constraints to bots — £1M starting wallet, item inventory, 70% buyout mechanic
