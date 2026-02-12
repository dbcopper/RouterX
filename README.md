# RouterX

**Production-grade LLM gateway and router** — a single OpenAI-compatible API that routes across multiple model providers with automatic fallback, billing, observability, and a full admin console.

Think of it as a self-hosted OpenRouter: drop-in replacement for any OpenAI SDK, with multi-provider routing, per-tenant billing, and real-time analytics.

## Features

### Core Routing
- **OpenAI-compatible API** — `POST /v1/chat/completions`, `POST /v1/embeddings`, `GET /v1/models`
- **Auto-routing** — model name maps to provider type via model catalog, no configuration needed
- **Multi-provider fallback** — if one provider fails, automatically tries the next healthy one
- **Circuit breaker** — sliding window error rate detection with 30s cooldown per provider
- **Latency-aware sorting** — routes to fastest healthy provider by default
- **50+ models** — OpenAI, Anthropic, Gemini, DeepSeek, Mistral, Meta Llama, Qwen

### Streaming & Passthrough
- **Full SSE streaming** — all providers (OpenAI, Anthropic, Gemini, DeepSeek, Mistral)
- **100% parameter passthrough** — tools, tool_choice, response_format, top_p, frequency_penalty, seed, etc.
- **Vision support** — auto-detects image content and routes to vision-capable providers

### Billing & Tenants
- **Per-tenant billing** — balance tracking, automatic per-request charges, transaction ledger
- **Spending limits** — configurable `spend_limit_usd` per tenant, auto-blocks when exceeded
- **Rate limiting** — configurable RPM per tenant + global concurrency limits via Redis
- **Balance transactions** — full audit trail of topups, charges, and adjustments
- **Suspend/unsuspend** — admin can freeze tenant access instantly
- **`:free` suffix** — append `:free` to any model name to skip billing (for demos/testing)

### BYOK & Provider Control
- **Bring Your Own Key** — `X-RouterX-API-Key` header overrides the system provider key
- **Provider preferences** — `X-RouterX-Provider-Only`, `X-RouterX-Provider-Ignore`, `X-RouterX-Provider-Order`
- **Fallback control** — `X-RouterX-Allow-Fallbacks: false` to disable automatic fallback
- **Sort modes** — `X-RouterX-Sort: latency` or `price` to control provider selection

### Observability
- **Request logs** — every request logged with provider, model, latency, TTFT, tokens, cost, status
- **Response headers** — `X-RouterX-Provider`, `X-RouterX-Latency-Ms`, `X-RouterX-Cost-USD`, `X-RouterX-Fallback`
- **Generation API** — `GET /admin/generation/{id}` for after-the-fact metadata lookup
- **Prompt caching** — `X-RouterX-Cache: true` for Redis-backed response caching (5min TTL)
- **User tracking** — `X-RouterX-User`, `X-Title`, `HTTP-Referer` stored per request
- **Webhooks** — `request.completed` events with HMAC-SHA256 signatures to any URL
- **Prometheus metrics** — request count, latency histogram, TTFT by provider
- **OpenTelemetry tracing** — distributed traces via Jaeger
- **CSV export** — export filtered request logs as CSV

### Admin Console
- **Dashboard** — all-time + 24h KPIs, provider health, model usage breakdown
- **Providers** — add/edit/disable providers, API key management
- **Tenants** — detail view with balance, limits, suspend, transaction history
- **Request logs** — filterable, sortable, paginated with inline delete
- **Model pricing** — per-model pricing overrides (input/output per 1K tokens)
- **Webhooks** — register/delete webhook endpoints with signature verification
- **Advanced routing** — optional per-tenant routing rule overrides

### Tenant User Portal
- **Self-service dashboard** — usage stats, model breakdown, daily charts
- **API key management** — create/delete keys with optional model restrictions
- **Balance topup** — self-service balance addition

## Quick Start (Docker)

```bash
# 1. Copy env
cp .env.example .env

# 2. Start services
docker compose -f deploy/docker-compose.yml up -d --build

# 3. Run migrations and seed demo data
docker compose -f deploy/docker-compose.yml exec -T backend /routerx migrate
docker compose -f deploy/docker-compose.yml exec -T backend /routerx seed

# 4. Open the UI
# Admin console: http://localhost:3000
# Grafana:       http://localhost:3001 (admin/admin)
# Jaeger:        http://localhost:16686
```

> To seed on deploy: `set ROUTERX_SEED=1 && deploy.cmd`

**Default credentials (local only):**
- Admin: `admin` / `admin123`
- Tenant user: `demo` / `demo123`
- Demo API key: `demo_key_fake_123456`

## API Usage

### Chat Completion
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer demo_key_fake_123456" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### Streaming
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer demo_key_fake_123456" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5",
    "stream": true,
    "messages": [{"role": "user", "content": "Tell me a joke"}]
  }'
```

### With BYOK + Provider Control
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer demo_key_fake_123456" \
  -H "X-RouterX-API-Key: sk-your-own-key" \
  -H "X-RouterX-Provider-Only: openai" \
  -H "X-RouterX-Sort: latency" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### Free Mode (No Billing)
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer demo_key_fake_123456" \
  -d '{"model": "gpt-4o:free", "messages": [{"role": "user", "content": "Hi"}]}'
```

### Embeddings
```bash
curl http://localhost:8080/v1/embeddings \
  -H "Authorization: Bearer demo_key_fake_123456" \
  -H "Content-Type: application/json" \
  -d '{"model": "text-embedding-3-small", "input": "Hello world"}'
```

### List Models
```bash
curl http://localhost:8080/v1/models
```

## Custom Headers Reference

| Header | Description |
|--------|-------------|
| `X-RouterX-API-Key` | BYOK: override the provider API key |
| `X-RouterX-Sort` | `latency` or `price` — controls provider selection |
| `X-RouterX-Provider-Only` | Comma-separated list of providers to use exclusively |
| `X-RouterX-Provider-Ignore` | Comma-separated list of providers to exclude |
| `X-RouterX-Provider-Order` | Comma-separated preferred provider order |
| `X-RouterX-Allow-Fallbacks` | `false` to disable automatic fallback |
| `X-RouterX-Cache` | `true` to enable Redis prompt caching |
| `X-RouterX-User` | End-user ID for tracking |
| `X-Title` | App name for attribution |
| `HTTP-Referer` | App referer URL for attribution |

**Response headers** (non-streaming):

| Header | Description |
|--------|-------------|
| `X-RouterX-Provider` | Which provider handled the request |
| `X-RouterX-Latency-Ms` | Total request latency |
| `X-RouterX-Cost-USD` | Estimated cost for this request |
| `X-RouterX-Fallback` | `true` if a fallback provider was used |
| `X-RouterX-Cache-Hit` | `true` if served from cache |

## Supported Providers

| Provider | Type | Models |
|----------|------|--------|
| OpenAI | `openai` | GPT-4o, GPT-4.1, o1/o3/o4, embeddings |
| Anthropic | `anthropic` | Claude Sonnet 4.5, Opus 4, 3.5 family |
| Google | `gemini` | Gemini 2.5/2.0/1.5 Pro & Flash |
| DeepSeek | `deepseek` | DeepSeek Chat, Reasoner, Coder |
| Mistral | `mistral` | Mistral Large/Medium/Small, Codestral |
| Any OpenAI-compatible | `generic-openai` | Custom base URL + API key |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_REAL_CALLS` | `false` | `false` returns mock responses; `true` calls real APIs |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `REDIS_URL` | — | Redis connection string |
| `JWT_SECRET` | — | Secret for admin/tenant JWT tokens |
| `OTEL_ENDPOINT` | — | OpenTelemetry collector endpoint |
| `PORT` | `8080` | Backend server port |

## Project Structure

```
backend/
  cmd/server/       — entrypoint, routing, migrations
  internal/
    api/            — HTTP handlers
    config/         — environment config
    limiter/        — Redis rate limiter
    metrics/        — Prometheus metrics
    middleware/     — auth, API key validation
    models/         — request/response types
    observability/  — OpenTelemetry setup
    providers/      — provider implementations (OpenAI, Anthropic, Gemini, etc.)
    router/         — routing engine, circuit breaker, latency tracker, pricing
    store/          — PostgreSQL data layer
    webhook/        — webhook dispatcher
frontend/
  app/              — Next.js App Router pages
  components/       — shared UI components
  lib/              — API client utilities
deploy/             — Docker Compose + Grafana + Jaeger
migrations/         — SQL migrations (001-012)
scripts/            — seed data, load testing
```

## Security Notes
- No real API keys are stored or shipped. Add keys via the Admin UI or DB.
- Request logs store **metadata only**, never prompt/response text.
- `.env` is gitignored. See `.env.example` for safe defaults.
- Webhook signatures use HMAC-SHA256 for payload verification.
