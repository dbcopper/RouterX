# RouterX

RouterX is a provider-agnostic LLM/VLM gateway and router with a web-based Admin Console. It exposes a single Chat Completions-style API and routes requests across multiple providers with capability-aware fallbacks. This is AI infrastructure: a control plane for routing, auth, observability, and billing across model providers.

## Quick Start (Docker)
1. Copy env:
   ```bash
   cp .env.example .env
   ```
2. Start services:
   ```bash
   docker compose -f deploy/docker-compose.yml up -d --build
   ```
3. Run migrations and seed demo data:
   ```bash
   docker compose -f deploy/docker-compose.yml exec -T backend /routerx migrate
   docker compose -f deploy/docker-compose.yml exec -T backend /routerx seed
   ```> Note: `deploy.cmd` no longer seeds by default. To seed on deploy, run:
> ```bash
> set ROUTERX_SEED=1
> deploy.cmd
> ```
4. Open the UI: `http://localhost:3000`
5. Grafana: `http://localhost:3001` (default admin/admin)
6. Jaeger: `http://localhost:16686`

> Demo API key is **fake** and for local use only: `demo_key_fake_123456`.

## API Usage
### Non-stream
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer demo_key_fake_123456" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4.1-mini",
    "messages": [
      {"role": "user", "content": [{"type": "text", "text": "Hello"}]}
    ]
  }'
```

### Stream (SSE)
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer demo_key_fake_123456" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4.1-mini",
    "stream": true,
    "messages": [
      {"role": "user", "content": [{"type": "text", "text": "Stream me"}]}
    ]
  }'
```

### Vision example
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer demo_key_fake_123456" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4.1-mini",
    "messages": [
      {"role": "user", "content": [
        {"type": "text", "text": "What is in this image?"},
        {"type": "image_url", "image_url": "https://example.com/cat.jpg"}
      ]}
    ]
  }'
```

## Routing & Fallback
- Model catalog maps model ¡ú provider type (OpenAI/Anthropic/Gemini/Generic).
- Circuit breaker uses a sliding window error rate and cooldown.
- If a provider fails, RouterX can fall back within the same provider type.

## Admin Console
- Login via `http://localhost:3000/login`.
- Default admin: `admin` / `admin123` (local only; change in seed or DB).
- Default user: `demo` / `demo123` (local only; change in seed or DB).
- Providers, pricing, tenants, and request logs are visible in the UI.

## Configuration
Key environment variables:
- `ENABLE_REAL_CALLS=false` (default) returns deterministic mock responses.
- `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`.

## Security Notes
- No real API keys are stored or shipped. Add keys via the Admin UI or DB.
- Request logs store **metadata only**, never prompt/response text.
- `.env` is gitignored. See `.env.example` for safe defaults.

## Local Demo Script
```bash
set ROUTERX_API_KEY=demo_key_fake_123456
python demo.py
```

## Project Structure
- `backend/`: Go API (chi), providers, router, migrations.
- `frontend/`: Next.js Admin Console.
- `deploy/`: Docker Compose + observability.
- `scripts/`: seed and load test scripts.

