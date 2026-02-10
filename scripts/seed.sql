CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO tenants (id, name) VALUES ('demo', 'Demo Tenant') ON CONFLICT (id) DO NOTHING;
INSERT INTO api_keys (key, tenant_id) VALUES ('demo_key_fake_123456', 'demo') ON CONFLICT (key) DO NOTHING;

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
