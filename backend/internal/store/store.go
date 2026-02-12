package store

import (
	"context"
	"errors"
	"fmt"
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
	HasAPIKey      bool   `json:"has_api_key"`
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
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	BalanceUSD    float64    `json:"balance_usd"`
	CreatedAt     time.Time  `json:"created_at"`
	LastActive    *time.Time `json:"last_active"`
	Suspended     bool       `json:"suspended"`
	TotalTopupUSD float64    `json:"total_topup_usd"`
	TotalSpentUSD float64    `json:"total_spent_usd"`
	RateLimitRPM  int        `json:"rate_limit_rpm"`
	SpendLimitUSD float64    `json:"spend_limit_usd"`
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

type TenantRequestSummary struct {
	TotalRequests int              `json:"total_requests"`
	TotalTokens   int              `json:"total_tokens"`
	TotalCostUSD  float64          `json:"total_cost_usd"`
	Daily         []TenantDayUsage `json:"daily"`
	Recent        []TenantDayUsage `json:"recent"`
	RecentModels  []TenantRecentModelUsage `json:"recent_models"`
}

type TenantDayUsage struct {
	Day      time.Time `json:"day"`
	Requests int       `json:"requests"`
	Tokens   int       `json:"tokens"`
	CostUSD  float64   `json:"cost_usd"`
}

type TenantRecentModelUsage struct {
	Model  string    `json:"model"`
	Bucket time.Time `json:"bucket"`
	Tokens int       `json:"tokens"`
}

type ModelUsageSummary struct {
	Model    string  `json:"model"`
	Provider string  `json:"provider"`
	Tokens   int     `json:"tokens"`
	CostUSD  float64 `json:"cost_usd"`
	Requests int     `json:"requests"`
}

type BalanceTransaction struct {
	ID           int       `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Type         string    `json:"type"`
	AmountUSD    float64   `json:"amount_usd"`
	BalanceAfter float64   `json:"balance_after"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

func New(db *pgxpool.Pool) *Store {
	return &Store{DB: db}
}

func (s *Store) GetTenantByAPIKey(ctx context.Context, key string) (*Tenant, error) {
	row := s.DB.QueryRow(ctx, `SELECT t.id, t.name, t.balance_usd, t.created_at, t.last_active, t.suspended, t.total_topup_usd, t.total_spent_usd FROM api_keys k JOIN tenants t ON k.tenant_id=t.id WHERE k.key=$1`, key)
	var t Tenant
	if err := row.Scan(&t.ID, &t.Name, &t.BalanceUSD, &t.CreatedAt, &t.LastActive, &t.Suspended, &t.TotalTopupUSD, &t.TotalSpentUSD); err != nil {
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
		p.HasAPIKey = p.APIKey != ""
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
	p.HasAPIKey = p.APIKey != ""
	return &p, nil
}

func (s *Store) GetEnabledProvidersByType(ctx context.Context, providerType string) ([]Provider, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, name, type, COALESCE(base_url,''), COALESCE(api_key,''), default_model, supports_text, supports_vision, enabled FROM providers WHERE type=$1 AND enabled=true`, providerType)
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
		p.HasAPIKey = p.APIKey != ""
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (s *Store) GetTenantByID(ctx context.Context, id string) (*Tenant, error) {
	row := s.DB.QueryRow(ctx, `SELECT id, name, balance_usd, created_at, last_active, suspended, total_topup_usd, total_spent_usd, rate_limit_rpm, spend_limit_usd FROM tenants WHERE id=$1`, id)
	var t Tenant
	if err := row.Scan(&t.ID, &t.Name, &t.BalanceUSD, &t.CreatedAt, &t.LastActive, &t.Suspended, &t.TotalTopupUSD, &t.TotalSpentUSD, &t.RateLimitRPM, &t.SpendLimitUSD); err != nil {
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
	_, err := s.DB.Exec(ctx, `INSERT INTO request_logs (tenant_id, provider, model, latency_ms, ttft_ms, tokens, cost_usd, prompt_hash, fallback_used, status_code, error_code, user_id, app_title, app_referer, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		log.TenantID, log.Provider, log.Model, log.LatencyMS, log.TTFTMS, log.Tokens, log.CostUSD, log.PromptHash, log.FallbackUsed, log.StatusCode, log.ErrorCode, log.UserID, log.AppTitle, log.AppReferer, log.CreatedAt)
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
	rows, err := s.DB.Query(ctx, `SELECT id, name, balance_usd, created_at, last_active, suspended, total_topup_usd, total_spent_usd, rate_limit_rpm, spend_limit_usd FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.BalanceUSD, &t.CreatedAt, &t.LastActive, &t.Suspended, &t.TotalTopupUSD, &t.TotalSpentUSD, &t.RateLimitRPM, &t.SpendLimitUSD); err != nil {
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
	rows, err := s.DB.Query(ctx, `SELECT id, tenant_id, provider, model, latency_ms, ttft_ms, tokens, cost_usd, prompt_hash, fallback_used, status_code, error_code, created_at FROM request_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.RequestLog
	for rows.Next() {
		var l models.RequestLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.Provider, &l.Model, &l.LatencyMS, &l.TTFTMS, &l.Tokens, &l.CostUSD, &l.PromptHash, &l.FallbackUsed, &l.StatusCode, &l.ErrorCode, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (s *Store) GetRequestLog(ctx context.Context, id int) (*models.RequestLog, error) {
	row := s.DB.QueryRow(ctx, `SELECT id, tenant_id, provider, model, latency_ms, ttft_ms, tokens, cost_usd, prompt_hash, fallback_used, status_code, error_code, user_id, app_title, app_referer, created_at FROM request_logs WHERE id=$1`, id)
	var r models.RequestLog
	if err := row.Scan(&r.ID, &r.TenantID, &r.Provider, &r.Model, &r.LatencyMS, &r.TTFTMS, &r.Tokens, &r.CostUSD, &r.PromptHash, &r.FallbackUsed, &r.StatusCode, &r.ErrorCode, &r.UserID, &r.AppTitle, &r.AppReferer, &r.CreatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) DeleteRequestLog(ctx context.Context, id int) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM request_logs WHERE id=$1`, id)
	return err
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

func (s *Store) UpdateProviderAPIKey(ctx context.Context, id, apiKey string) error {
	_, err := s.DB.Exec(ctx, `UPDATE providers SET api_key=$2 WHERE id=$1`, id, apiKey)
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
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, `UPDATE tenants SET total_spent_usd = total_spent_usd + $2 WHERE id=$1`, tenantID, cost)
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

func (s *Store) ListModelsByProviderType(ctx context.Context, providerType string) ([]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT model FROM model_catalog WHERE provider_type=$1 ORDER BY model`, providerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var models []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

func (s *Store) AddModelCatalog(ctx context.Context, model, providerType string) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO model_catalog (model, provider_type) VALUES ($1,$2) ON CONFLICT (model) DO UPDATE SET provider_type=EXCLUDED.provider_type`, model, providerType)
	return err
}

func (s *Store) DeleteModelCatalog(ctx context.Context, model string) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM model_catalog WHERE model=$1`, model)
	return err
}

type ModelInfo struct {
	Model        string  `json:"id"`
	ProviderType string  `json:"provider_type"`
	PricePer1K   float64 `json:"price_per_1k_usd"`
}

func (s *Store) ListAllModels(ctx context.Context) ([]ModelInfo, error) {
	rows, err := s.DB.Query(ctx, `SELECT mc.model, mc.provider_type, COALESCE(mp.price_per_1k_usd,0) FROM model_catalog mc LEFT JOIN model_pricing mp ON mc.model=mp.model ORDER BY mc.provider_type, mc.model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModelInfo
	for rows.Next() {
		var m ModelInfo
		if err := rows.Scan(&m.Model, &m.ProviderType, &m.PricePer1K); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetTenantRequestSummary(ctx context.Context, tenantID string) (*TenantRequestSummary, error) {
	row := s.DB.QueryRow(ctx, `SELECT COUNT(*), COALESCE(SUM(tokens),0), COALESCE(SUM(cost_usd),0) FROM request_logs WHERE tenant_id=$1 AND status_code=200 AND tokens > 0`, tenantID)
	var totalReq int
	var totalTokens int
	var totalCost float64
	if err := row.Scan(&totalReq, &totalTokens, &totalCost); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(ctx, `SELECT DATE(created_at) as day, COUNT(*), COALESCE(SUM(tokens),0), COALESCE(SUM(cost_usd),0) FROM request_logs WHERE tenant_id=$1 AND status_code=200 AND tokens > 0 GROUP BY day ORDER BY day`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var daily []TenantDayUsage
	for rows.Next() {
		var d TenantDayUsage
		if err := rows.Scan(&d.Day, &d.Requests, &d.Tokens, &d.CostUSD); err != nil {
			return nil, err
		}
		daily = append(daily, d)
	}
	recentRows, err := s.DB.Query(ctx, `
		SELECT to_timestamp(floor(extract(epoch from created_at) / 10800) * 10800) as bucket_start,
		       COUNT(*),
		       COALESCE(SUM(tokens),0),
		       COALESCE(SUM(cost_usd),0)
		FROM request_logs
		WHERE tenant_id=$1 AND status_code=200 AND tokens > 0 AND created_at >= NOW() - interval '24 hours'
		GROUP BY bucket_start
		ORDER BY bucket_start
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer recentRows.Close()
	recentMap := map[int64]TenantDayUsage{}
	for recentRows.Next() {
		var r TenantDayUsage
		if err := recentRows.Scan(&r.Day, &r.Requests, &r.Tokens, &r.CostUSD); err != nil {
			return nil, err
		}
		recentMap[r.Day.Unix()] = r
	}
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)
	bucket := time.Duration(3) * time.Hour
	var recent []TenantDayUsage
	for i := 0; i < 8; i++ {
		ts := start.Add(time.Duration(i) * bucket)
		key := ts.Unix() - (ts.Unix() % int64(bucket.Seconds()))
		if val, ok := recentMap[key]; ok {
			recent = append(recent, val)
		} else {
			recent = append(recent, TenantDayUsage{Day: time.Unix(key, 0).UTC()})
		}
	}

	recentModelRows, err := s.DB.Query(ctx, `
		SELECT model,
		       to_timestamp(floor(extract(epoch from created_at) / 10800) * 10800) as bucket_start,
		       COALESCE(SUM(tokens),0)
		FROM request_logs
		WHERE tenant_id=$1 AND status_code=200 AND tokens > 0 AND created_at >= NOW() - interval '24 hours'
		GROUP BY model, bucket_start
		ORDER BY bucket_start
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer recentModelRows.Close()
	var recentModels []TenantRecentModelUsage
	for recentModelRows.Next() {
		var r TenantRecentModelUsage
		if err := recentModelRows.Scan(&r.Model, &r.Bucket, &r.Tokens); err != nil {
			return nil, err
		}
		recentModels = append(recentModels, r)
	}
	return &TenantRequestSummary{
		TotalRequests: totalReq,
		TotalTokens:   totalTokens,
		TotalCostUSD:  totalCost,
		Daily:         daily,
		Recent:        recent,
		RecentModels:  recentModels,
	}, rows.Err()
}

// ---- Admin Dashboard Stats ----

type HourlyBucket struct {
	Hour     time.Time `json:"hour"`
	Requests int       `json:"requests"`
	Errors   int       `json:"errors"`
}

type AdminDashboardStats struct {
	TotalTenants  int            `json:"total_tenants"`
	ActiveTenants int            `json:"active_tenants"`
	Requests24h   int            `json:"requests_24h"`
	Errors24h     int            `json:"errors_24h"`
	ErrorRate     float64        `json:"error_rate"`
	AvgLatencyMS  float64        `json:"avg_latency_ms"`
	P95LatencyMS  float64        `json:"p95_latency_ms"`
	Cost24h       float64        `json:"cost_24h"`
	Tokens24h     int            `json:"tokens_24h"`
	HourlySeries  []HourlyBucket `json:"hourly_series"`
	// All-time stats
	TotalRequestsAllTime int     `json:"total_requests_all_time"`
	TotalTokensAllTime   int     `json:"total_tokens_all_time"`
	TotalCostAllTime     float64 `json:"total_cost_all_time"`
	TotalRevenueAllTime  float64 `json:"total_revenue_all_time"`
}

func (s *Store) GetAdminDashboardStats(ctx context.Context) (*AdminDashboardStats, error) {
	stats := &AdminDashboardStats{}

	// tenant counts
	row := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`)
	_ = row.Scan(&stats.TotalTenants)

	row = s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE last_active >= NOW() - interval '24 hours'`)
	_ = row.Scan(&stats.ActiveTenants)

	// 24h request stats
	row = s.DB.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE status_code >= 400),
		       COALESCE(AVG(latency_ms), 0),
		       COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms), 0),
		       COALESCE(SUM(cost_usd), 0),
		       COALESCE(SUM(tokens), 0)
		FROM request_logs WHERE created_at >= NOW() - interval '24 hours'
	`)
	_ = row.Scan(&stats.Requests24h, &stats.Errors24h, &stats.AvgLatencyMS, &stats.P95LatencyMS, &stats.Cost24h, &stats.Tokens24h)

	if stats.Requests24h > 0 {
		stats.ErrorRate = float64(stats.Errors24h) / float64(stats.Requests24h) * 100
	}

	// All-time request stats
	row = s.DB.QueryRow(ctx, `
		SELECT COUNT(*),
		       COALESCE(SUM(tokens), 0),
		       COALESCE(SUM(cost_usd), 0)
		FROM request_logs
	`)
	_ = row.Scan(&stats.TotalRequestsAllTime, &stats.TotalTokensAllTime, &stats.TotalCostAllTime)

	// All-time revenue (sum of all topups)
	row = s.DB.QueryRow(ctx, `SELECT COALESCE(SUM(total_topup_usd), 0) FROM tenants`)
	_ = row.Scan(&stats.TotalRevenueAllTime)

	// hourly series
	rows, err := s.DB.Query(ctx, `
		SELECT date_trunc('hour', created_at) AS hour,
		       COUNT(*),
		       COUNT(*) FILTER (WHERE status_code >= 400)
		FROM request_logs
		WHERE created_at >= NOW() - interval '24 hours'
		GROUP BY hour ORDER BY hour
	`)
	if err != nil {
		return stats, nil
	}
	defer rows.Close()
	hourMap := map[int64]HourlyBucket{}
	for rows.Next() {
		var b HourlyBucket
		if err := rows.Scan(&b.Hour, &b.Requests, &b.Errors); err != nil {
			continue
		}
		hourMap[b.Hour.Unix()] = b
	}
	now := time.Now().UTC()
	for i := 23; i >= 0; i-- {
		h := now.Add(-time.Duration(i) * time.Hour).Truncate(time.Hour)
		if b, ok := hourMap[h.Unix()]; ok {
			stats.HourlySeries = append(stats.HourlySeries, b)
		} else {
			stats.HourlySeries = append(stats.HourlySeries, HourlyBucket{Hour: h})
		}
	}

	return stats, nil
}

// ---- Paginated Request Logs ----

type RequestLogFilters struct {
	TenantID   string
	Provider   string
	Model      string
	StatusCode int
	SortBy     string
	SortDir    string
}

type PaginatedRequestLogs struct {
	Data     []models.RequestLog `json:"data"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

func (s *Store) ListRequestLogsPaginated(ctx context.Context, page, pageSize int, f RequestLogFilters) (*PaginatedRequestLogs, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	argN := 1

	if f.TenantID != "" {
		where += fmt.Sprintf(" AND tenant_id=$%d", argN)
		args = append(args, f.TenantID)
		argN++
	}
	if f.Provider != "" {
		where += fmt.Sprintf(" AND provider=$%d", argN)
		args = append(args, f.Provider)
		argN++
	}
	if f.Model != "" {
		where += fmt.Sprintf(" AND model ILIKE $%d", argN)
		args = append(args, "%"+f.Model+"%")
		argN++
	}
	if f.StatusCode > 0 {
		where += fmt.Sprintf(" AND status_code=$%d", argN)
		args = append(args, f.StatusCode)
		argN++
	}

	// count
	var total int
	countQ := "SELECT COUNT(*) FROM request_logs " + where
	if err := s.DB.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, err
	}

	// sort
	sortCol := "created_at"
	switch f.SortBy {
	case "latency_ms", "tokens", "cost_usd", "created_at", "model", "provider":
		sortCol = f.SortBy
	}
	sortDir := "DESC"
	if f.SortDir == "asc" {
		sortDir = "ASC"
	}

	offset := (page - 1) * pageSize
	dataQ := fmt.Sprintf(`SELECT id, tenant_id, provider, model, latency_ms, ttft_ms, tokens, cost_usd, prompt_hash, fallback_used, status_code, error_code, created_at
		FROM request_logs %s ORDER BY %s %s LIMIT $%d OFFSET $%d`, where, sortCol, sortDir, argN, argN+1)
	args = append(args, pageSize, offset)

	rows, err := s.DB.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.RequestLog
	for rows.Next() {
		var l models.RequestLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.Provider, &l.Model, &l.LatencyMS, &l.TTFTMS, &l.Tokens, &l.CostUSD, &l.PromptHash, &l.FallbackUsed, &l.StatusCode, &l.ErrorCode, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return &PaginatedRequestLogs{Data: logs, Total: total, Page: page, PageSize: pageSize}, rows.Err()
}

// ---- Routing Rules ----

func (s *Store) ListRoutingRulesByTenant(ctx context.Context, tenantID string) ([]RoutingRule, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, tenant_id, capability, primary_provider_id, COALESCE(secondary_provider_id,''), model FROM routing_rules WHERE tenant_id=$1 ORDER BY capability`, tenantID)
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

func (s *Store) DeleteRoutingRule(ctx context.Context, id string) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM routing_rules WHERE id=$1`, id)
	return err
}

// ---- Provider Health ----

type ProviderHealthStatus struct {
	ProviderID    string `json:"provider_id"`
	ProviderName  string `json:"provider_name"`
	Type          string `json:"type"`
	Enabled       bool   `json:"enabled"`
	HealthStatus  string `json:"health_status"`
	CircuitOpen   bool   `json:"circuit_open"`
	AvgLatencyMS  int64  `json:"avg_latency_ms"`
}

func (s *Store) ListModelUsage(ctx context.Context) ([]ModelUsageSummary, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT model,
		       provider,
		       COUNT(*) as requests,
		       COALESCE(SUM(tokens),0) as tokens,
		       COALESCE(SUM(cost_usd),0) as cost_usd
		FROM request_logs
		WHERE status_code=200 AND tokens > 0
		GROUP BY model, provider
		ORDER BY cost_usd DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ModelUsageSummary
	for rows.Next() {
		var m ModelUsageSummary
		if err := rows.Scan(&m.Model, &m.Provider, &m.Requests, &m.Tokens, &m.CostUSD); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ---- Balance Transactions ----

func (s *Store) RecordTransaction(ctx context.Context, tenantID, txType string, amount, balanceAfter float64, description string) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO balance_transactions (tenant_id, type, amount_usd, balance_after, description) VALUES ($1,$2,$3,$4,$5)`,
		tenantID, txType, amount, balanceAfter, description)
	return err
}

func (s *Store) ListTransactions(ctx context.Context, tenantID string, limit int) ([]BalanceTransaction, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB.Query(ctx, `SELECT id, tenant_id, type, amount_usd, balance_after, COALESCE(description,''), created_at FROM balance_transactions WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var txs []BalanceTransaction
	for rows.Next() {
		var tx BalanceTransaction
		if err := rows.Scan(&tx.ID, &tx.TenantID, &tx.Type, &tx.AmountUSD, &tx.BalanceAfter, &tx.Description, &tx.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (s *Store) SuspendTenant(ctx context.Context, tenantID string, suspended bool) error {
	_, err := s.DB.Exec(ctx, `UPDATE tenants SET suspended=$2 WHERE id=$1`, tenantID, suspended)
	return err
}

func (s *Store) UpdateTenantLimits(ctx context.Context, tenantID string, rateLimitRPM int, spendLimitUSD float64) error {
	_, err := s.DB.Exec(ctx, `UPDATE tenants SET rate_limit_rpm=$2, spend_limit_usd=$3 WHERE id=$1`, tenantID, rateLimitRPM, spendLimitUSD)
	return err
}

// ---- Webhooks ----

type Webhook struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, url, events, secret, enabled, created_at FROM webhooks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hooks []Webhook
	for rows.Next() {
		var h Webhook
		if err := rows.Scan(&h.ID, &h.URL, &h.Events, &h.Secret, &h.Enabled, &h.CreatedAt); err != nil {
			return nil, err
		}
		hooks = append(hooks, h)
	}
	return hooks, rows.Err()
}

func (s *Store) CreateWebhook(ctx context.Context, url string, events []string, secret string) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO webhooks (url, events, secret) VALUES ($1, $2, $3)`, url, events, secret)
	return err
}

func (s *Store) DeleteWebhook(ctx context.Context, id int) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM webhooks WHERE id=$1`, id)
	return err
}

func (s *Store) GetEnabledWebhooks(ctx context.Context, event string) ([]Webhook, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, url, events, secret, enabled, created_at FROM webhooks WHERE enabled=true AND $1=ANY(events)`, event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hooks []Webhook
	for rows.Next() {
		var h Webhook
		if err := rows.Scan(&h.ID, &h.URL, &h.Events, &h.Secret, &h.Enabled, &h.CreatedAt); err != nil {
			return nil, err
		}
		hooks = append(hooks, h)
	}
	return hooks, rows.Err()
}
