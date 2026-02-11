CREATE TABLE IF NOT EXISTS tenants (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  last_active TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_keys (
  key TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants(id)
);

CREATE TABLE IF NOT EXISTS providers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  base_url TEXT,
  api_key TEXT,
  default_model TEXT,
  supports_text BOOLEAN NOT NULL DEFAULT true,
  supports_vision BOOLEAN NOT NULL DEFAULT false,
  enabled BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS routing_rules (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL REFERENCES tenants(id),
  capability TEXT NOT NULL,
  primary_provider_id TEXT NOT NULL REFERENCES providers(id),
  secondary_provider_id TEXT REFERENCES providers(id),
  model TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS request_logs (
  id SERIAL PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  latency_ms BIGINT NOT NULL,
  ttft_ms BIGINT NOT NULL,
  tokens INT NOT NULL,
  prompt_hash TEXT NOT NULL,
  fallback_used BOOLEAN NOT NULL,
  status_code INT NOT NULL,
  error_code TEXT,
  created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS usage_daily (
  tenant_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  day DATE NOT NULL,
  tokens INT NOT NULL,
  PRIMARY KEY (tenant_id, provider, model, day)
);

CREATE TABLE IF NOT EXISTS admin_users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL
);

