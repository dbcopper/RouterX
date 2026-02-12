CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_request_logs_tenant_id ON request_logs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_status_code ON request_logs (status_code);
CREATE INDEX IF NOT EXISTS idx_routing_rules_tenant ON routing_rules (tenant_id, capability);
