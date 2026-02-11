CREATE TABLE IF NOT EXISTS model_pricing (
  model TEXT PRIMARY KEY,
  price_per_1k_usd NUMERIC(12,6) NOT NULL
);
