CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Clear all history data for fresh start
TRUNCATE request_logs RESTART IDENTITY CASCADE;
TRUNCATE usage_daily;
DELETE FROM balance_transactions;
DELETE FROM routing_rules;

INSERT INTO tenants (id, name, balance_usd) VALUES ('demo', 'Demo Tenant', 10.00) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, balance_usd=EXCLUDED.balance_usd;
INSERT INTO api_keys (key, tenant_id, name, allowed_models) VALUES ('demo_key_fake_123456', 'demo', 'Default', ARRAY[]::text[]) ON CONFLICT (key) DO UPDATE SET allowed_models=EXCLUDED.allowed_models;

INSERT INTO providers (id, name, type, base_url, api_key, default_model, supports_text, supports_vision, enabled)
VALUES
  ('openai', 'OpenAI', 'openai', NULL, NULL, 'gpt-4.1-mini', true, true, true),
  ('anthropic', 'Anthropic', 'anthropic', NULL, NULL, 'claude-sonnet-4-5', true, true, true),
  ('gemini', 'Gemini', 'gemini', NULL, NULL, 'gemini-2.5-flash', true, true, true),
  ('deepseek', 'DeepSeek', 'deepseek', 'https://api.deepseek.com', NULL, 'deepseek-chat', true, false, true),
  ('mistral', 'Mistral', 'mistral', 'https://api.mistral.ai', NULL, 'mistral-large-latest', true, true, true),
  ('generic1', 'Generic OpenAI Compatible', 'generic-openai', 'http://localhost:9001', NULL, 'local-model', true, false, false)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, type=EXCLUDED.type, base_url=EXCLUDED.base_url, api_key=EXCLUDED.api_key, default_model=EXCLUDED.default_model, supports_text=EXCLUDED.supports_text, supports_vision=EXCLUDED.supports_vision, enabled=EXCLUDED.enabled;

INSERT INTO admin_users (id, username, password_hash)
VALUES ('admin', 'admin', crypt('admin123', gen_salt('bf')))
ON CONFLICT (id) DO UPDATE SET username=EXCLUDED.username, password_hash=EXCLUDED.password_hash;

INSERT INTO tenant_users (id, tenant_id, username, password_hash)
VALUES ('demo-user', 'demo', 'demo', crypt('demo123', gen_salt('bf')))
ON CONFLICT (id) DO UPDATE SET tenant_id=EXCLUDED.tenant_id, username=EXCLUDED.username, password_hash=EXCLUDED.password_hash;

-- Comprehensive model pricing (50+ models)
INSERT INTO model_pricing (model, price_per_1k_usd) VALUES
  -- OpenAI
  ('gpt-4o', 0.005000),
  ('gpt-4o-mini', 0.001500),
  ('gpt-4.1', 0.008000),
  ('gpt-4.1-mini', 0.002000),
  ('gpt-4.1-nano', 0.001000),
  ('gpt-3.5-turbo', 0.001000),
  ('o1', 0.015000),
  ('o1-mini', 0.003000),
  ('o1-pro', 0.060000),
  ('o3', 0.020000),
  ('o3-mini', 0.004000),
  ('o4-mini', 0.004000),
  ('text-embedding-3-small', 0.000020),
  ('text-embedding-3-large', 0.000130),
  ('text-embedding-ada-002', 0.000100),
  -- Anthropic
  ('claude-sonnet-4-5', 0.006000),
  ('claude-opus-4', 0.015000),
  ('claude-3-5-sonnet', 0.006000),
  ('claude-3-5-haiku', 0.001000),
  ('claude-3-opus', 0.015000),
  ('claude-3-haiku', 0.000500),
  -- Gemini
  ('gemini-2.5-pro', 0.005000),
  ('gemini-2.5-flash', 0.001500),
  ('gemini-2.0-flash', 0.001000),
  ('gemini-1.5-pro', 0.003500),
  ('gemini-1.5-flash', 0.001000),
  -- DeepSeek
  ('deepseek-chat', 0.000270),
  ('deepseek-reasoner', 0.000550),
  ('deepseek-coder', 0.000270),
  -- Mistral
  ('mistral-large-latest', 0.004000),
  ('mistral-medium-latest', 0.002700),
  ('mistral-small-latest', 0.001000),
  ('codestral-latest', 0.001000),
  ('mistral-embed', 0.000100),
  ('open-mistral-nemo', 0.000300),
  ('open-mixtral-8x22b', 0.002000),
  -- Meta Llama (via compatible providers)
  ('meta-llama/llama-3.3-70b-instruct', 0.000600),
  ('meta-llama/llama-3.1-405b-instruct', 0.003000),
  ('meta-llama/llama-3.1-70b-instruct', 0.000600),
  ('meta-llama/llama-3.1-8b-instruct', 0.000060),
  -- Qwen (via compatible providers)
  ('qwen/qwen-2.5-72b-instruct', 0.000400),
  ('qwen/qwen-2.5-coder-32b-instruct', 0.000200)
ON CONFLICT (model) DO UPDATE SET price_per_1k_usd=EXCLUDED.price_per_1k_usd;

-- Comprehensive model catalog (50+ models, model -> provider_type mapping for auto-routing)
INSERT INTO model_catalog (model, provider_type) VALUES
  -- OpenAI
  ('gpt-4o', 'openai'),
  ('gpt-4o-mini', 'openai'),
  ('gpt-4.1', 'openai'),
  ('gpt-4.1-mini', 'openai'),
  ('gpt-4.1-nano', 'openai'),
  ('gpt-3.5-turbo', 'openai'),
  ('o1', 'openai'),
  ('o1-mini', 'openai'),
  ('o1-pro', 'openai'),
  ('o3', 'openai'),
  ('o3-mini', 'openai'),
  ('o4-mini', 'openai'),
  ('text-embedding-3-small', 'openai'),
  ('text-embedding-3-large', 'openai'),
  ('text-embedding-ada-002', 'openai'),
  -- Anthropic
  ('claude-sonnet-4-5', 'anthropic'),
  ('claude-opus-4', 'anthropic'),
  ('claude-3-5-sonnet', 'anthropic'),
  ('claude-3-5-haiku', 'anthropic'),
  ('claude-3-opus', 'anthropic'),
  ('claude-3-haiku', 'anthropic'),
  -- Gemini
  ('gemini-2.5-pro', 'gemini'),
  ('gemini-2.5-flash', 'gemini'),
  ('gemini-2.0-flash', 'gemini'),
  ('gemini-1.5-pro', 'gemini'),
  ('gemini-1.5-flash', 'gemini'),
  -- DeepSeek
  ('deepseek-chat', 'deepseek'),
  ('deepseek-reasoner', 'deepseek'),
  ('deepseek-coder', 'deepseek'),
  -- Mistral
  ('mistral-large-latest', 'mistral'),
  ('mistral-medium-latest', 'mistral'),
  ('mistral-small-latest', 'mistral'),
  ('codestral-latest', 'mistral'),
  ('mistral-embed', 'mistral'),
  ('open-mistral-nemo', 'mistral'),
  ('open-mixtral-8x22b', 'mistral'),
  -- Meta Llama (routed through generic-openai providers)
  ('meta-llama/llama-3.3-70b-instruct', 'generic-openai'),
  ('meta-llama/llama-3.1-405b-instruct', 'generic-openai'),
  ('meta-llama/llama-3.1-70b-instruct', 'generic-openai'),
  ('meta-llama/llama-3.1-8b-instruct', 'generic-openai'),
  -- Qwen (routed through generic-openai providers)
  ('qwen/qwen-2.5-72b-instruct', 'generic-openai'),
  ('qwen/qwen-2.5-coder-32b-instruct', 'generic-openai')
ON CONFLICT (model) DO UPDATE SET provider_type=EXCLUDED.provider_type;
