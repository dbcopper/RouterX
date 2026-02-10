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
	APIKey         string `json:"api_key"`
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
	ID   string `json:"id"`
	Name string `json:"name"`
}

type APIKey struct {
	Key      string
	TenantID string
}

type AdminUser struct {
	ID           string
	Username     string
	PasswordHash string
}

func New(db *pgxpool.Pool) *Store {
	return &Store{DB: db}
}

func (s *Store) GetTenantByAPIKey(ctx context.Context, key string) (*Tenant, error) {
	row := s.DB.QueryRow(ctx, `SELECT t.id, t.name FROM api_keys k JOIN tenants t ON k.tenant_id=t.id WHERE k.key=$1`, key)
	var t Tenant
	if err := row.Scan(&t.ID, &t.Name); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) GetProviders(ctx context.Context) ([]Provider, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, name, type, base_url, api_key, default_model, supports_text, supports_vision, enabled FROM providers`)
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
	row := s.DB.QueryRow(ctx, `SELECT id, name, type, base_url, api_key, default_model, supports_text, supports_vision, enabled FROM providers WHERE id=$1`, id)
	var p Provider
	if err := row.Scan(&p.ID, &p.Name, &p.Type, &p.BaseURL, &p.APIKey, &p.DefaultModel, &p.SupportsText, &p.SupportsVision, &p.Enabled); err != nil {
		return nil, err
	}
	return &p, nil
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
	_, err := s.DB.Exec(ctx, `INSERT INTO request_logs (tenant_id, provider, model, latency_ms, ttft_ms, tokens, prompt_hash, fallback_used, status_code, error_code, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)` ,
		log.TenantID, log.Provider, log.Model, log.LatencyMS, log.TTFTMS, log.Tokens, log.PromptHash, log.FallbackUsed, log.StatusCode, log.ErrorCode, log.CreatedAt)
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
	rows, err := s.DB.Query(ctx, `SELECT id, name FROM tenants`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

func (s *Store) ListRequestLogs(ctx context.Context, limit int) ([]models.RequestLog, error) {
	rows, err := s.DB.Query(ctx, `SELECT tenant_id, provider, model, latency_ms, ttft_ms, tokens, prompt_hash, fallback_used, status_code, error_code, created_at FROM request_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.RequestLog
	for rows.Next() {
		var l models.RequestLog
		if err := rows.Scan(&l.TenantID, &l.Provider, &l.Model, &l.LatencyMS, &l.TTFTMS, &l.Tokens, &l.PromptHash, &l.FallbackUsed, &l.StatusCode, &l.ErrorCode, &l.CreatedAt); err != nil {
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

func (s *Store) CreateAPIKey(ctx context.Context, k APIKey) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO api_keys (key, tenant_id) VALUES ($1,$2) ON CONFLICT (key) DO NOTHING`, k.Key, k.TenantID)
	return err
}

func (s *Store) RecordUsageDaily(ctx context.Context, tenantID, provider, model string, tokens int, day time.Time) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO usage_daily (tenant_id, provider, model, day, tokens) VALUES ($1,$2,$3,$4,$5)
	ON CONFLICT (tenant_id, provider, model, day) DO UPDATE SET tokens = usage_daily.tokens + EXCLUDED.tokens`, tenantID, provider, model, day, tokens)
	return err
}
