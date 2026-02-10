# RouterX

RouterX is a provider-agnostic LLM/VLM gateway and router with a web-based Admin Console. It exposes a single Chat Completions-style API and routes requests across multiple providers with capability-aware fallbacks.

## 5-minute Run
1. Copy env and start services:
   ```bash
   cp .env.example .env
   make up
   ```
2. Run migrations and seed demo data:
   ```bash
   make migrate
   make seed
   ```
3. Open the UI: `http://localhost:3000`
4. Grafana: `http://localhost:3001` (default admin/admin)
5. Jaeger: `http://localhost:16686`

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
- Routing rules select primary/secondary providers per capability (`text` vs `vision`).
- Circuit breaker uses a sliding window error rate and cooldown.
- If primary fails, RouterX falls back to secondary (logged in request metadata).

## Admin Console
- Login via `http://localhost:3000/login`.
- Default admin: `admin` / `admin123` (local only; change in seed or DB).
- Providers, routing rules, tenants, and request logs are visible in the UI.

## Configuration
Key environment variables:
- `ENABLE_REAL_CALLS=false` (default) returns deterministic mock responses.
- `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`.

## Security Notes
- No real API keys are stored or shipped. Add keys via DB or environment.
- Request logs store **metadata only**, never prompt/response text.
- `.env` is gitignored. See `.env.example` for safe defaults.

## Load Test
```bash
make loadtest
```

## Project Structure
- `backend/`: Go API (chi), providers, router, migrations.
- `frontend/`: Next.js Admin Console.
- `deploy/`: Docker Compose + observability.
- `scripts/`: seed and load test scripts.
# RouterX
