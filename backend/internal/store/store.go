package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"routerx/internal/models"
)

type Store struct {
	DB *pgxpool.Pool
}

type Provider struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"-"`
	DefaultModel   string `json:"default_model"`
	SupportsText   bool   `json:"supports_text"`
	SupportsVision bool   `json:"supports_vision"`
	Enabled        bool   `json:"enabled"`
}

type RoutingRule struct {
	ID                  string `json:"id"`
	TenantID            string `json:"tenant_id"`
	Capability          string `json:"capability"`
	PrimaryProviderID   string `json:"primary_provider_id"`
	SecondaryProviderID string `json:"secondary_provider_id"`
	Model               string `json:"model"`
}

type Tenant struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	BalanceUSD float64    `json:"balance_usd"`
	CreatedAt  time.Time  `json:"created_at"`
	LastActive *time.Time `json:"last_active"`
}

type APIKey struct {
	Key           string    `json:"key"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	AllowedModels []string  `json:"allowed_models"`
	CreatedAt     time.Time `json:"created_at"`
}

type AdminUser struct {
	ID           string
	Username     string
	PasswordHash string
}

type TenantUser struct {
	ID           string
	TenantID     string
	Username     string
	PasswordHash string
}

type ModelPricing struct {
	Model        string  `json:"model"`
	PricePer1KUSD float64 `json:"price_per_1k_usd"`
}

type ModelCatalog struct {
	Model        string `json:"model"`
	ProviderType string `json:"provider_type"`
}

func New(db *pgxpool.Pool) *Store {
	return &Store{DB: db}
}

func (s *Store) GetTenantByAPIKey(ctx context.Context, key string) (*Tenant, error) {
	row := s.DB.QueryRow(ctx, `SELECT t.id, t.name, t.balance_usd, t.created_at, t.last_active FROM api_keys k JOIN tenants t ON k.tenant_id=t.id WHERE k.key=$1`, key)
	var t Tenant
	if err := row.Scan(&t.ID, &t.Name, &t.BalanceUSD, &t.CreatedAt, &t.LastActive); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) GetAPIKey(ctx context.Context, key string) (*APIKey, error) {
	row := s.DB.QueryRow(ctx, `SELECT key, tenant_id, COALESCE(name,''), COALESCE(allowed_models, ARRAY[]::text[]), created_at FROM api_keys WHERE key=$1`, key)
	var k APIKey
	if err := row.Scan(&k.Key, &k.TenantID, &k.Name, &k.AllowedModels, &k.CreatedAt); err != nil {
		return nil, err
	}
	return &k, nil
}

func (s *Store) GetProviders(ctx context.Context) ([]Provider, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, name, type, COALESCE(base_url,''), COALESCE(api_key,''), default_model, supports_text, supports_vision, enabled FROM providers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var providers []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.BaseURL, &p.APIKey, &p.DefaultModel, &p.SupportsText, &p.SupportsVision, &p.Enabled); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (s *Store) GetProviderByID(ctx context.Context, id string) (*Provider, error) {
	row := s.DB.QueryRow(ctx, `SELECT id, name, type, COALESCE(base_url,''), COALESCE(api_key,''), default_model, supports_text, supports_vision, enabled FROM providers WHERE id=$1`, id)
	var p Provider
	if err := row.Scan(&p.ID, &p.Name, &p.Type, &p.BaseURL, &p.APIKey, &p.DefaultModel, &p.SupportsText, &p.SupportsVision, &p.Enabled); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) GetTenantByID(ctx context.Context, id string) (*Tenant, error) {
	row := s.DB.QueryRow(ctx, `SELECT id, name, balance_usd, created_at, last_active FROM tenants WHERE id=$1`, id)
	var t Tenant
	if err := row.Scan(&t.ID, &t.Name, &t.BalanceUSD, &t.CreatedAt, &t.LastActive); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) GetRoutingRule(ctx context.Context, tenantID, capability string) (*RoutingRule, error) {
	row := s.DB.QueryRow(ctx, `SELECT id, tenant_id, capability, primary_provider_id, secondary_provider_id, model FROM routing_rules WHERE tenant_id=$1 AND capability=$2 LIMIT 1`, tenantID, capability)
	var r RoutingRule
	if err := row.Scan(&r.ID, &r.TenantID, &r.Capability, &r.PrimaryProviderID, &r.SecondaryProviderID, &r.Model); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) InsertRequestLog(ctx context.Context, log models.RequestLog) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO request_logs (tenant_id, provider, model, latency_ms, ttft_ms, tokens, cost_usd, prompt_hash, fallback_used, status_code, error_code, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)` ,
		log.TenantID, log.Provider, log.Model, log.LatencyMS, log.TTFTMS, log.Tokens, log.CostUSD, log.PromptHash, log.FallbackUsed, log.StatusCode, log.ErrorCode, log.CreatedAt)
	return err
}

func (s *Store) GetAdminByUsername(ctx context.Context, username string) (*AdminUser, error) {
	row := s.DB.QueryRow(ctx, `SELECT id, username, password_hash FROM admin_users WHERE username=$1`, username)
	var u AdminUser
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetTenantUserByUsername(ctx context.Context, username string) (*TenantUser, error) {
	row := s.DB.QueryRow(ctx, `SELECT id, tenant_id, username, password_hash FROM tenant_users WHERE username=$1`, username)
	var u TenantUser
	if err := row.Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) ListProviders(ctx context.Context) ([]Provider, error) {
	return s.GetProviders(ctx)
}

func (s *Store) ListRoutingRules(ctx context.Context) ([]RoutingRule, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, tenant_id, capability, primary_provider_id, secondary_provider_id, model FROM routing_rules`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []RoutingRule
	for rows.Next() {
		var r RoutingRule
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Capability, &r.PrimaryProviderID, &r.SecondaryProviderID, &r.Model); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (s *Store) ListTenants(ctx context.Context) ([]Tenant, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, name, balance_usd, created_at, last_active FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.BalanceUSD, &t.CreatedAt, &t.LastActive); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

func (s *Store) ListAPIKeysByTenant(ctx context.Context, tenantID string) ([]APIKey, error) {
	rows, err := s.DB.Query(ctx, `SELECT key, tenant_id, COALESCE(name,''), COALESCE(allowed_models, ARRAY[]::text[]), created_at FROM api_keys WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.Key, &k.TenantID, &k.Name, &k.AllowedModels, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *Store) ListRequestLogs(ctx context.Context, limit int) ([]models.RequestLog, error) {
	rows, err := s.DB.Query(ctx, `SELECT tenant_id, provider, model, latency_ms, ttft_ms, tokens, cost_usd, prompt_hash, fallback_used, status_code, error_code, created_at FROM request_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.RequestLog
	for rows.Next() {
		var l models.RequestLog
		if err := rows.Scan(&l.TenantID, &l.Provider, &l.Model, &l.LatencyMS, &l.TTFTMS, &l.Tokens, &l.CostUSD, &l.PromptHash, &l.FallbackUsed, &l.StatusCode, &l.ErrorCode, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (s *Store) UpsertProvider(ctx context.Context, p Provider) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO providers (id, name, type, base_url, api_key, default_model, supports_text, supports_vision, enabled)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, type=EXCLUDED.type, base_url=EXCLUDED.base_url, api_key=EXCLUDED.api_key, default_model=EXCLUDED.default_model, supports_text=EXCLUDED.supports_text, supports_vision=EXCLUDED.supports_vision, enabled=EXCLUDED.enabled`,
		p.ID, p.Name, p.Type, p.BaseURL, p.APIKey, p.DefaultModel, p.SupportsText, p.SupportsVision, p.Enabled)
	return err
}

func (s *Store) UpdateProvider(ctx context.Context, p Provider) error {
	_, err := s.DB.Exec(ctx, `UPDATE providers SET base_url=$2, api_key=$3, default_model=$4, supports_text=$5, supports_vision=$6, enabled=$7 WHERE id=$1`,
		p.ID, p.BaseURL, p.APIKey, p.DefaultModel, p.SupportsText, p.SupportsVision, p.Enabled)
	return err
}

func (s *Store) UpsertRoutingRule(ctx context.Context, r RoutingRule) error {
	if r.TenantID == "" {
		return errors.New("tenant_id required")
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO routing_rules (id, tenant_id, capability, primary_provider_id, secondary_provider_id, model)
	VALUES ($1,$2,$3,$4,$5,$6)
	ON CONFLICT (id) DO UPDATE SET tenant_id=EXCLUDED.tenant_id, capability=EXCLUDED.capability, primary_provider_id=EXCLUDED.primary_provider_id, secondary_provider_id=EXCLUDED.secondary_provider_id, model=EXCLUDED.model`,
		r.ID, r.TenantID, r.Capability, r.PrimaryProviderID, r.SecondaryProviderID, r.Model)
	return err
}

func (s *Store) CreateTenant(ctx context.Context, t Tenant) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO tenants (id, name) VALUES ($1,$2) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name`, t.ID, t.Name)
	return err
}

func (s *Store) CreateTenantUser(ctx context.Context, u TenantUser) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO tenant_users (id, tenant_id, username, password_hash) VALUES ($1,$2,$3,$4)`, u.ID, u.TenantID, u.Username, u.PasswordHash)
	return err
}

func (s *Store) CreateAPIKey(ctx context.Context, k APIKey) error {
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now().UTC()
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO api_keys (key, tenant_id, name, allowed_models, created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (key) DO NOTHING`, k.Key, k.TenantID, k.Name, k.AllowedModels, k.CreatedAt)
	return err
}

func (s *Store) DeleteAPIKey(ctx context.Context, tenantID, key string) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM api_keys WHERE key=$1 AND tenant_id=$2`, key, tenantID)
	return err
}

func (s *Store) RecordUsageDaily(ctx context.Context, tenantID, provider, model string, tokens int, day time.Time) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO usage_daily (tenant_id, provider, model, day, tokens, cost_usd) VALUES ($1,$2,$3,$4,$5,$6)
	ON CONFLICT (tenant_id, provider, model, day) DO UPDATE SET tokens = usage_daily.tokens + EXCLUDED.tokens, cost_usd = usage_daily.cost_usd + EXCLUDED.cost_usd`, tenantID, provider, model, day, tokens, 0)
	return err
}

func (s *Store) AddUsageCost(ctx context.Context, tenantID, provider, model string, tokens int, cost float64, day time.Time) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO usage_daily (tenant_id, provider, model, day, tokens, cost_usd) VALUES ($1,$2,$3,$4,$5,$6)
	ON CONFLICT (tenant_id, provider, model, day) DO UPDATE SET tokens = usage_daily.tokens + EXCLUDED.tokens, cost_usd = usage_daily.cost_usd + EXCLUDED.cost_usd`, tenantID, provider, model, day, tokens, cost)
	return err
}

func (s *Store) UpdateTenantBalance(ctx context.Context, tenantID string, balance float64) error {
	_, err := s.DB.Exec(ctx, `UPDATE tenants SET balance_usd=$2 WHERE id=$1`, tenantID, balance)
	return err
}

func (s *Store) UpdateTenantLastActive(ctx context.Context, tenantID string, at time.Time) error {
	_, err := s.DB.Exec(ctx, `UPDATE tenants SET last_active=$2 WHERE id=$1`, tenantID, at)
	return err
}

func (s *Store) ListModelPricing(ctx context.Context) ([]ModelPricing, error) {
	rows, err := s.DB.Query(ctx, `SELECT model, price_per_1k_usd FROM model_pricing ORDER BY model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ModelPricing
	for rows.Next() {
		var m ModelPricing
		if err := rows.Scan(&m.Model, &m.PricePer1KUSD); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *Store) UpsertModelPricing(ctx context.Context, m ModelPricing) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO model_pricing (model, price_per_1k_usd) VALUES ($1,$2) ON CONFLICT (model) DO UPDATE SET price_per_1k_usd=EXCLUDED.price_per_1k_usd`, m.Model, m.PricePer1KUSD)
	return err
}

func (s *Store) GetModelPrice(ctx context.Context, model string) (float64, bool, error) {
	row := s.DB.QueryRow(ctx, `SELECT price_per_1k_usd FROM model_pricing WHERE model=$1`, model)
	var price float64
	if err := row.Scan(&price); err != nil {
		return 0, false, err
	}
	return price, true, nil
}

func (s *Store) GetModelProvider(ctx context.Context, model string) (string, bool, error) {
	row := s.DB.QueryRow(ctx, `SELECT provider_type FROM model_catalog WHERE model=$1`, model)
	var p string
	if err := row.Scan(&p); err != nil {
		return "", false, err
	}
	return p, true, nil
}
