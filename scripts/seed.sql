CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO tenants (id, name, balance_usd) VALUES ('demo', 'Demo Tenant', 10.00) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, balance_usd=EXCLUDED.balance_usd;
INSERT INTO api_keys (key, tenant_id, name, allowed_models) VALUES ('demo_key_fake_123456', 'demo', 'Default', ARRAY['gpt-4.1-mini','gpt-4o-mini','claude-3-5-sonnet','gemini-1.5-pro','gemini-2.5-flash']) ON CONFLICT (key) DO NOTHING;

INSERT INTO providers (id, name, type, base_url, api_key, default_model, supports_text, supports_vision, enabled)
VALUES
  ('openai', 'OpenAI', 'openai', NULL, NULL, 'gpt-4.1-mini', true, true, true),
  ('anthropic', 'Anthropic', 'anthropic', NULL, NULL, 'claude-3-5-sonnet', true, true, true),
  ('gemini', 'Gemini', 'gemini', NULL, NULL, 'gemini-1.5-pro', true, true, true),
  ('generic1', 'Generic OpenAI Compatible', 'generic-openai', 'http://localhost:9001', NULL, 'local-model', true, false, true)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, type=EXCLUDED.type, base_url=EXCLUDED.base_url, api_key=EXCLUDED.api_key, default_model=EXCLUDED.default_model, supports_text=EXCLUDED.supports_text, supports_vision=EXCLUDED.supports_vision, enabled=EXCLUDED.enabled;

INSERT INTO routing_rules (id, tenant_id, capability, primary_provider_id, secondary_provider_id, model)
VALUES
  ('rule-text-demo', 'demo', 'text', 'openai', 'anthropic', 'gpt-4.1-mini'),
  ('rule-vision-demo', 'demo', 'vision', 'openai', 'gemini', 'gpt-4.1-mini')
ON CONFLICT (id) DO UPDATE SET tenant_id=EXCLUDED.tenant_id, capability=EXCLUDED.capability, primary_provider_id=EXCLUDED.primary_provider_id, secondary_provider_id=EXCLUDED.secondary_provider_id, model=EXCLUDED.model;

INSERT INTO admin_users (id, username, password_hash)
VALUES ('admin', 'admin', crypt('admin123', gen_salt('bf')))
ON CONFLICT (id) DO UPDATE SET username=EXCLUDED.username, password_hash=EXCLUDED.password_hash;

INSERT INTO tenant_users (id, tenant_id, username, password_hash)
VALUES ('demo-user', 'demo', 'demo', crypt('demo123', gen_salt('bf')))
ON CONFLICT (id) DO UPDATE SET tenant_id=EXCLUDED.tenant_id, username=EXCLUDED.username, password_hash=EXCLUDED.password_hash;

INSERT INTO model_pricing (model, price_per_1k_usd) VALUES
  ('gpt-4o', 0.005000),
  ('gpt-4o-mini', 0.001500),
  ('gpt-4.1', 0.008000),
  ('gpt-4.1-mini', 0.002000),
  ('gpt-3.5-turbo', 0.001000),
  ('claude-3-5-sonnet', 0.006000),
  ('claude-3-5-haiku', 0.001000),
  ('claude-3-opus', 0.015000),
  ('gemini-1.5-pro', 0.003500),
  ('gemini-1.5-flash', 0.001000),
  ('gemini-1.0-pro', 0.001000)
ON CONFLICT (model) DO UPDATE SET price_per_1k_usd=EXCLUDED.price_per_1k_usd;

INSERT INTO model_catalog (model, provider_type) VALUES
  ('gpt-4o', 'openai'),
  ('gpt-4o-mini', 'openai'),
  ('gpt-4.1', 'openai'),
  ('gpt-4.1-mini', 'openai'),
  ('gpt-3.5-turbo', 'openai'),
  ('claude-3-5-sonnet', 'anthropic'),
  ('claude-3-5-haiku', 'anthropic'),
  ('claude-3-opus', 'anthropic'),
  ('gemini-1.5-pro', 'gemini'),
  ('gemini-1.5-flash', 'gemini'),
  ('gemini-1.0-pro', 'gemini'),
  ('gemini-2.5-flash', 'gemini')
ON CONFLICT (model) DO UPDATE SET provider_type=EXCLUDED.provider_type;
