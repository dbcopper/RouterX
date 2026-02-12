CREATE TABLE IF NOT EXISTS webhooks (
  id SERIAL PRIMARY KEY,
  url TEXT NOT NULL,
  events TEXT[] NOT NULL DEFAULT '{request.completed}',
  secret TEXT NOT NULL DEFAULT '',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
