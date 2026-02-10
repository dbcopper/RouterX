# Repository Guidelines

This document is the contributor guide for the RouterX monorepo. It is designed to be practical and brief, with examples you can run locally.

## Project Structure & Module Organization
- `backend/`: Go 1.22+ API, providers, router, and database/Redis integrations.
- `frontend/`: Next.js 14+ Admin Console (App Router) with Tailwind and shadcn/ui.
- `deploy/`: Docker Compose and observability configs (Prometheus, Grafana, Jaeger/Tempo).
- `scripts/`: Seed data and load test utilities (e.g., `scripts/seed`, `scripts/loadtest`).
- `migrations/`: SQL migrations for PostgreSQL (goose or golang-migrate).
- `docs/` (optional): Architecture notes, ADRs, or diagrams.

## Build, Test, and Development Commands
Use the Makefile as the source of truth:
- `make up`: start the full stack via Docker Compose.
- `make down`: stop all services.
- `make migrate`: run DB migrations.
- `make seed`: seed demo data (fake API keys only).
- `make test`: run backend/frontend tests.
- `make lint`: lint Go and frontend code.
- `make loadtest`: run k6/vegeta scripts.

## Coding Style & Naming Conventions
- Go: `gofmt` enforced; packages use short, clear names.
- TypeScript/React: 2-space indentation, `camelCase` for vars/functions, `PascalCase` for components.
- API routes follow `/v1/*` naming, adapters named `{provider}_adapter`.
- Config uses `.env` locally; never commit secrets. See `.env.example`.

## Testing Guidelines
- Go: `go test ./...` with table-driven tests where practical.
- Frontend: use `npm test`/`pnpm test` if configured.
- Name tests `*_test.go` and UI tests `*.test.ts(x)` where applicable.
- Add tests for routing rules, provider fallbacks, and vision capability checks.

## Commit & Pull Request Guidelines
- Git history is not yet established. Use Conventional Commits (e.g., `feat: add routing rules`).
- PRs should include: summary, linked issue (if any), testing notes, and screenshots for UI changes.
- Avoid committing generated assets unless required.

## Security & Configuration Tips
- Keep `ENABLE_REAL_CALLS=false` for local dev unless testing real providers.
- Never log prompt/response plaintext; use hashes/lengths only.
- Ensure `.env` and secrets are gitignored.
