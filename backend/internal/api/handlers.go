package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/segmentio/ksuid"
	"golang.org/x/crypto/bcrypt"

	"routerx/internal/limiter"
	"routerx/internal/metrics"
	"routerx/internal/middleware"
	"routerx/internal/models"
	"routerx/internal/router"
	"routerx/internal/store"
	"routerx/internal/util"

	"go.uber.org/zap"
)

type Server struct {
	Store     *store.Store
	Router    *router.Router
	Limiter   *limiter.Limiter
	Logger    *zap.Logger
	JWTSecret string
}

func (s *Server) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.TenantFromContext(r.Context())
	if tenant == nil {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}
	// Check if tenant is suspended
	if tenant.Suspended {
		http.Error(w, "account suspended", http.StatusForbidden)
		return
	}
	allowed, err := s.Limiter.Allow(r.Context(), tenant.ID)
	if err != nil || !allowed {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	acq, err := s.Limiter.Acquire(r.Context(), tenant.ID)
	if err != nil || !acq {
		http.Error(w, "too many concurrent requests", http.StatusTooManyRequests)
		return
	}
	defer s.Limiter.Release(r.Context(), tenant.ID)

	var req models.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Model == "" {
		req.Model = "default"
	}
	apiKeyValue := extractAPIKey(r)
	if apiKeyValue != "" {
		if keyRec, err := s.Store.GetAPIKey(r.Context(), apiKeyValue); err == nil {
			if len(keyRec.AllowedModels) > 0 && !contains(keyRec.AllowedModels, req.Model) {
				http.Error(w, "model not allowed for api key", http.StatusForbidden)
				return
			}
		}
	}
	if tenant.BalanceUSD <= 0 {
		http.Error(w, "insufficient balance", http.StatusPaymentRequired)
		return
	}
	promptHash := util.HashString(util.NormalizeSpaces(extractText(req)))
	start := time.Now()

	stream := req.Stream
	var ttft time.Duration
	var tokens int
	providerName := ""
	fallbackUsed := false
	var resp models.ChatCompletionResponse
	var routeErr error

	// Parse provider sort preference from header
	sortMode := router.SortDefault
	if sortHeader := r.Header.Get("X-RouterX-Sort"); sortHeader != "" {
		switch strings.ToLower(sortHeader) {
		case "price":
			sortMode = router.SortPrice
		case "latency":
			sortMode = router.SortLatency
		}
	}

	if stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unsupported", http.StatusInternalServerError)
			return
		}
		send := func(event string) error {
			if event == "[DONE]" {
				_, _ = w.Write([]byte("data: [DONE]\n\n"))
				flusher.Flush()
				return nil
			}
			_, err := w.Write([]byte("data: " + event + "\n\n"))
			flusher.Flush()
			return err
		}
		resp, providerName, fallbackUsed, ttft, tokens, routeErr = s.Router.RouteWithSort(r.Context(), tenant.ID, req, true, send, sortMode)
	} else {
		resp, providerName, fallbackUsed, ttft, tokens, routeErr = s.Router.RouteWithSort(r.Context(), tenant.ID, req, false, nil, sortMode)
	}

	latency := time.Since(start)
	status := http.StatusOK
	if routeErr != nil {
		status = http.StatusBadGateway
		writeError(w, routeErr)
	}

	metrics.RequestsTotal.WithLabelValues(providerName, http.StatusText(status)).Inc()
	metrics.LatencyMS.WithLabelValues(providerName).Observe(float64(latency.Milliseconds()))
	metrics.TTFTMS.WithLabelValues(providerName).Observe(float64(ttft.Milliseconds()))

	cost := 0.0
	if tokens > 0 {
		if price, ok, err := s.Store.GetModelPrice(r.Context(), req.Model); err == nil && ok {
			cost = price * float64(tokens) / 1000.0
		} else {
			cost = router.EstimateCostUSD(req.Model, tokens)
		}
	}
	_ = s.Store.InsertRequestLog(r.Context(), models.RequestLog{
		TenantID:     tenant.ID,
		Provider:     providerName,
		Model:        req.Model,
		LatencyMS:    latency.Milliseconds(),
		TTFTMS:       ttft.Milliseconds(),
		Tokens:       tokens,
		CostUSD:      cost,
		PromptHash:   promptHash,
		FallbackUsed: fallbackUsed,
		StatusCode:   status,
		ErrorCode:    errCode(routeErr),
		CreatedAt:    time.Now().UTC(),
	})
	if status == http.StatusOK && tokens > 0 && cost > 0 {
		_ = s.Store.AddUsageCost(r.Context(), tenant.ID, providerName, req.Model, tokens, cost, time.Now().UTC())
		newBalance := tenant.BalanceUSD - cost
		_ = s.Store.UpdateTenantBalance(r.Context(), tenant.ID, newBalance)
		_ = s.Store.RecordTransaction(r.Context(), tenant.ID, "charge", -cost, newBalance, fmt.Sprintf("%s / %s / %d tokens", providerName, req.Model, tokens))
	}

	s.Logger.Info("request completed",
		zap.String("tenant_id", tenant.ID),
		zap.String("provider", providerName),
		zap.String("model", req.Model),
		zap.Int64("latency_ms", latency.Milliseconds()),
		zap.Int("tokens", tokens),
		zap.String("prompt_hash", promptHash),
		zap.Bool("fallback", fallbackUsed),
	)

	if !stream && routeErr == nil {
		writeJSON(w, resp)
	}
}

func (s *Server) AdminLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	user, err := s.Store.GetAdminByUsername(r.Context(), payload.Username)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token, err := middleware.NewAdminToken(s.JWTSecret, user.Username, 8*time.Hour)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"token": token})
}

func (s *Server) AuthLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	// try admin first
	if admin, err := s.Store.GetAdminByUsername(r.Context(), payload.Username); err == nil {
		if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(payload.Password)); err == nil {
			token, err := middleware.NewAdminToken(s.JWTSecret, admin.Username, 8*time.Hour)
			if err != nil {
				http.Error(w, "failed to issue token", http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]string{"token": token, "role": "admin"})
			return
		}
	}
	user, err := s.Store.GetTenantUserByUsername(r.Context(), payload.Username)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token, err := middleware.NewTenantToken(s.JWTSecret, user.Username, user.TenantID, 8*time.Hour)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"token": token, "role": "tenant"})
}

func (s *Server) AuthRegister(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Tenant   string `json:"tenant_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Username == "" || payload.Password == "" {
		http.Error(w, "missing username or password", http.StatusBadRequest)
		return
	}
	tenantID := ksuid.New().String()
	userID := ksuid.New().String()
	if payload.Tenant == "" {
		payload.Tenant = payload.Username + " Workspace"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to register", http.StatusInternalServerError)
		return
	}
	if err := s.Store.CreateTenant(r.Context(), store.Tenant{ID: tenantID, Name: payload.Tenant}); err != nil {
		http.Error(w, "failed to create tenant", http.StatusInternalServerError)
		return
	}
	_ = s.Store.UpdateTenantBalance(r.Context(), tenantID, 0)
	if err := s.Store.CreateTenantUser(r.Context(), store.TenantUser{ID: userID, TenantID: tenantID, Username: payload.Username, PasswordHash: string(hash)}); err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) TenantLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	user, err := s.Store.GetTenantUserByUsername(r.Context(), payload.Username)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token, err := middleware.NewTenantToken(s.JWTSecret, user.Username, user.TenantID, 8*time.Hour)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"token": token})
}

func (s *Server) AdminProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.Store.ListProviders(r.Context())
	if err != nil {
		http.Error(w, "failed to list providers", http.StatusInternalServerError)
		return
	}
	writeJSON(w, providers)
}

func (s *Server) AdminUpdateProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing provider id", http.StatusBadRequest)
		return
	}
	var payload struct {
		BaseURL        string `json:"base_url"`
		APIKey         string `json:"api_key"`
		DefaultModel   string `json:"default_model"`
		SupportsText   bool   `json:"supports_text"`
		SupportsVision bool   `json:"supports_vision"`
		Enabled        bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	apiKey := payload.APIKey
	if apiKey == "" {
		if existing, err := s.Store.GetProviderByID(r.Context(), id); err == nil {
			apiKey = existing.APIKey
		}
	}
	err := s.Store.UpdateProvider(r.Context(), store.Provider{
		ID:             id,
		BaseURL:        payload.BaseURL,
		APIKey:         apiKey,
		DefaultModel:   payload.DefaultModel,
		SupportsText:   payload.SupportsText,
		SupportsVision: payload.SupportsVision,
		Enabled:        payload.Enabled,
	})
	if err != nil {
		http.Error(w, "failed to update provider", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) AdminClearProviderKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing provider id", http.StatusBadRequest)
		return
	}
	if err := s.Store.UpdateProviderAPIKey(r.Context(), id, ""); err != nil {
		http.Error(w, "failed to clear api key", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) AdminCreateProvider(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name           string `json:"name"`
		Type           string `json:"type"`
		BaseURL        string `json:"base_url"`
		APIKey         string `json:"api_key"`
		DefaultModel   string `json:"default_model"`
		SupportsText   bool   `json:"supports_text"`
		SupportsVision bool   `json:"supports_vision"`
		Enabled        bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Type == "" {
		payload.Type = "generic-openai"
	}
	id := ksuid.New().String()
	provider := store.Provider{
		ID:             id,
		Name:           payload.Name,
		Type:           payload.Type,
		BaseURL:        payload.BaseURL,
		APIKey:         payload.APIKey,
		DefaultModel:   payload.DefaultModel,
		SupportsText:   payload.SupportsText,
		SupportsVision: payload.SupportsVision,
		Enabled:        payload.Enabled,
	}
	if err := s.Store.UpsertProvider(r.Context(), provider); err != nil {
		http.Error(w, "failed to create provider", http.StatusInternalServerError)
		return
	}
	writeJSON(w, provider)
}

func (s *Server) AdminTenants(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListTenants(r.Context())
	if err != nil {
		http.Error(w, "failed to list tenants", http.StatusInternalServerError)
		return
	}
	writeJSON(w, items)
}

func (s *Server) AdminRequests(w http.ResponseWriter, r *http.Request) {
	logs, err := s.Store.ListRequestLogs(r.Context(), 100)
	if err != nil {
		http.Error(w, "failed to list requests", http.StatusInternalServerError)
		return
	}
	writeJSON(w, logs)
}

func (s *Server) AdminModelUsage(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListModelUsage(r.Context())
	if err != nil {
		http.Error(w, "failed to list model usage", http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (s *Server) AdminDeleteRequest(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "missing request id", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid request id", http.StatusBadRequest)
		return
	}
	if err := s.Store.DeleteRequestLog(r.Context(), id); err != nil {
		http.Error(w, "failed to delete request", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) AdminListModelPricing(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListModelPricing(r.Context())
	if err != nil {
		http.Error(w, "failed to list pricing", http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (s *Server) AdminUpsertModelPricing(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Model        string  `json:"model"`
		PricePer1KUSD float64 `json:"price_per_1k_usd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Model == "" {
		http.Error(w, "model required", http.StatusBadRequest)
		return
	}
	if err := s.Store.UpsertModelPricing(r.Context(), store.ModelPricing{Model: payload.Model, PricePer1KUSD: payload.PricePer1KUSD}); err != nil {
		http.Error(w, "failed to upsert pricing", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) AdminListModels(w http.ResponseWriter, r *http.Request) {
	providerType := r.URL.Query().Get("provider_type")
	if providerType == "" {
		http.Error(w, "provider_type required", http.StatusBadRequest)
		return
	}
	list, err := s.Store.ListModelsByProviderType(r.Context(), providerType)
	if err != nil {
		http.Error(w, "failed to list models", http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (s *Server) AdminAddModel(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Model        string `json:"model"`
		ProviderType string `json:"provider_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Model == "" || payload.ProviderType == "" {
		http.Error(w, "model and provider_type required", http.StatusBadRequest)
		return
	}
	if err := s.Store.AddModelCatalog(r.Context(), payload.Model, payload.ProviderType); err != nil {
		http.Error(w, "failed to add model", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) AdminDeleteModel(w http.ResponseWriter, r *http.Request) {
	model := chi.URLParam(r, "model")
	if model == "" {
		http.Error(w, "missing model", http.StatusBadRequest)
		return
	}
	if err := s.Store.DeleteModelCatalog(r.Context(), model); err != nil {
		http.Error(w, "failed to delete model", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) TenantUsage(w http.ResponseWriter, r *http.Request) {
	user := middleware.TenantUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}
	rows, err := s.Store.DB.Query(r.Context(), `SELECT provider, model, day, tokens, cost_usd FROM usage_daily WHERE tenant_id=$1 AND (tokens > 0 OR cost_usd > 0) ORDER BY day DESC LIMIT 30`, user.TenantID)
	if err != nil {
		http.Error(w, "failed to list usage", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type usageRow struct {
		Provider string    `json:"provider"`
		Model    string    `json:"model"`
		Day      time.Time `json:"day"`
		Tokens   int       `json:"tokens"`
		CostUSD  float64   `json:"cost_usd"`
	}
	var out []usageRow
	for rows.Next() {
		var u usageRow
		if err := rows.Scan(&u.Provider, &u.Model, &u.Day, &u.Tokens, &u.CostUSD); err != nil {
			http.Error(w, "failed to list usage", http.StatusInternalServerError)
			return
		}
		out = append(out, u)
	}
	writeJSON(w, out)
}

func (s *Server) TenantSummary(w http.ResponseWriter, r *http.Request) {
	user := middleware.TenantUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}
	summary, err := s.Store.GetTenantRequestSummary(r.Context(), user.TenantID)
	if err != nil {
		http.Error(w, "failed to load summary", http.StatusInternalServerError)
		return
	}
	writeJSON(w, summary)
}

func (s *Server) TenantAPIKeys(w http.ResponseWriter, r *http.Request) {
	user := middleware.TenantUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}
	keys, err := s.Store.ListAPIKeysByTenant(r.Context(), user.TenantID)
	if err != nil {
		http.Error(w, "failed to list api keys", http.StatusInternalServerError)
		return
	}
	writeJSON(w, keys)
}

func (s *Server) TenantCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user := middleware.TenantUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}
	var payload struct {
		Key           string   `json:"key"`
		Name          string   `json:"name"`
		AllowedModels []string `json:"allowed_models"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Key == "" {
		payload.Key = "user_key_" + ksuid.New().String()
	}
	createdAt := time.Now().UTC()
	if err := s.Store.CreateAPIKey(r.Context(), store.APIKey{Key: payload.Key, TenantID: user.TenantID, Name: payload.Name, AllowedModels: payload.AllowedModels, CreatedAt: createdAt}); err != nil {
		http.Error(w, "failed to create api key", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"key": payload.Key, "created_at": createdAt})
}

func (s *Server) TenantDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	user := middleware.TenantUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}
	key := chi.URLParam(r, "key")
	if key == "" {
		http.Error(w, "missing api key", http.StatusBadRequest)
		return
	}
	if err := s.Store.DeleteAPIKey(r.Context(), user.TenantID, key); err != nil {
		http.Error(w, "failed to delete api key", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) TenantProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.TenantUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}
	tenant, err := s.Store.GetTenantByID(r.Context(), user.TenantID)
	if err != nil {
		http.Error(w, "failed to load tenant", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{
		"tenant_id":      tenant.ID,
		"name":           tenant.Name,
		"username":       user.Username,
		"balance_usd":    tenant.BalanceUSD,
		"suspended":      tenant.Suspended,
		"total_topup_usd": tenant.TotalTopupUSD,
		"total_spent_usd": tenant.TotalSpentUSD,
	})
}

func (s *Server) TenantTopup(w http.ResponseWriter, r *http.Request) {
	user := middleware.TenantUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}
	var payload struct {
		Amount float64 `json:"amount_usd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Amount <= 0 {
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}
	tenant, err := s.Store.GetTenantByID(r.Context(), user.TenantID)
	if err != nil {
		http.Error(w, "failed to load tenant", http.StatusInternalServerError)
		return
	}
	newBalance := tenant.BalanceUSD + payload.Amount
	if err := s.Store.UpdateTenantBalance(r.Context(), user.TenantID, newBalance); err != nil {
		http.Error(w, "failed to update balance", http.StatusInternalServerError)
		return
	}
	// Update total_topup_usd and record transaction
	_, _ = s.Store.DB.Exec(r.Context(), `UPDATE tenants SET total_topup_usd = total_topup_usd + $2 WHERE id=$1`, user.TenantID, payload.Amount)
	_ = s.Store.RecordTransaction(r.Context(), user.TenantID, "topup", payload.Amount, newBalance, fmt.Sprintf("Self-service topup $%.2f", payload.Amount))
	writeJSON(w, map[string]interface{}{"balance_usd": newBalance})
}

// ---- Admin Dashboard Stats ----

func (s *Server) AdminDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.Store.GetAdminDashboardStats(r.Context())
	if err != nil {
		http.Error(w, "failed to load stats", http.StatusInternalServerError)
		return
	}
	writeJSON(w, stats)
}

// ---- Paginated Requests ----

func (s *Server) AdminRequestsPaginated(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = 50
	}
	statusCode, _ := strconv.Atoi(r.URL.Query().Get("status_code"))
	filters := store.RequestLogFilters{
		TenantID:   r.URL.Query().Get("tenant_id"),
		Provider:   r.URL.Query().Get("provider"),
		Model:      r.URL.Query().Get("model"),
		StatusCode: statusCode,
		SortBy:     r.URL.Query().Get("sort_by"),
		SortDir:    r.URL.Query().Get("sort_dir"),
	}
	result, err := s.Store.ListRequestLogsPaginated(r.Context(), page, pageSize, filters)
	if err != nil {
		http.Error(w, "failed to list requests", http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

// ---- Routing Rules CRUD ----

func (s *Server) AdminRoutingRules(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID != "" {
		rules, err := s.Store.ListRoutingRulesByTenant(r.Context(), tenantID)
		if err != nil {
			http.Error(w, "failed to list rules", http.StatusInternalServerError)
			return
		}
		writeJSON(w, rules)
		return
	}
	rules, err := s.Store.ListRoutingRules(r.Context())
	if err != nil {
		http.Error(w, "failed to list rules", http.StatusInternalServerError)
		return
	}
	writeJSON(w, rules)
}

func (s *Server) AdminCreateRoutingRule(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		TenantID            string `json:"tenant_id"`
		Capability          string `json:"capability"`
		PrimaryProviderID   string `json:"primary_provider_id"`
		SecondaryProviderID string `json:"secondary_provider_id"`
		Model               string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.TenantID == "" || payload.Capability == "" || payload.PrimaryProviderID == "" || payload.Model == "" {
		http.Error(w, "tenant_id, capability, primary_provider_id, model required", http.StatusBadRequest)
		return
	}
	rule := store.RoutingRule{
		ID:                  ksuid.New().String(),
		TenantID:            payload.TenantID,
		Capability:          payload.Capability,
		PrimaryProviderID:   payload.PrimaryProviderID,
		SecondaryProviderID: payload.SecondaryProviderID,
		Model:               payload.Model,
	}
	if err := s.Store.UpsertRoutingRule(r.Context(), rule); err != nil {
		http.Error(w, "failed to create rule", http.StatusInternalServerError)
		return
	}
	writeJSON(w, rule)
}

func (s *Server) AdminUpdateRoutingRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing rule id", http.StatusBadRequest)
		return
	}
	var payload struct {
		TenantID            string `json:"tenant_id"`
		Capability          string `json:"capability"`
		PrimaryProviderID   string `json:"primary_provider_id"`
		SecondaryProviderID string `json:"secondary_provider_id"`
		Model               string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rule := store.RoutingRule{
		ID:                  id,
		TenantID:            payload.TenantID,
		Capability:          payload.Capability,
		PrimaryProviderID:   payload.PrimaryProviderID,
		SecondaryProviderID: payload.SecondaryProviderID,
		Model:               payload.Model,
	}
	if err := s.Store.UpsertRoutingRule(r.Context(), rule); err != nil {
		http.Error(w, "failed to update rule", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) AdminDeleteRoutingRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing rule id", http.StatusBadRequest)
		return
	}
	if err := s.Store.DeleteRoutingRule(r.Context(), id); err != nil {
		http.Error(w, "failed to delete rule", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// ---- Admin Balance Adjustment ----

func (s *Server) AdminAdjustBalance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing tenant id", http.StatusBadRequest)
		return
	}
	var payload struct {
		BalanceUSD  float64 `json:"balance_usd"`
		Description string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	tenant, err := s.Store.GetTenantByID(r.Context(), id)
	if err != nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}
	diff := payload.BalanceUSD - tenant.BalanceUSD
	if err := s.Store.UpdateTenantBalance(r.Context(), id, payload.BalanceUSD); err != nil {
		http.Error(w, "failed to update balance", http.StatusInternalServerError)
		return
	}
	desc := payload.Description
	if desc == "" {
		desc = fmt.Sprintf("Admin adjustment: $%.2f -> $%.2f", tenant.BalanceUSD, payload.BalanceUSD)
	}
	txType := "adjustment"
	if diff > 0 {
		// Positive adjustment counts as topup
		_, _ = s.Store.DB.Exec(r.Context(), `UPDATE tenants SET total_topup_usd = total_topup_usd + $2 WHERE id=$1`, id, diff)
	}
	_ = s.Store.RecordTransaction(r.Context(), id, txType, diff, payload.BalanceUSD, desc)
	writeJSON(w, map[string]interface{}{"status": "ok", "balance_usd": payload.BalanceUSD})
}

// ---- Provider Health ----

func (s *Server) AdminProviderHealth(w http.ResponseWriter, r *http.Request) {
	providers, err := s.Store.ListProviders(r.Context())
	if err != nil {
		http.Error(w, "failed to list providers", http.StatusInternalServerError)
		return
	}
	circuitStates := s.Router.GetCircuitStates()
	latencies := s.Router.GetProviderLatencies()
	var result []store.ProviderHealthStatus
	for _, p := range providers {
		health := "unknown"
		if s.Router.Redis != nil {
			val, err := s.Router.Redis.Get(r.Context(), "provider_health:"+p.ID).Result()
			if err == nil {
				health = val
			}
		}
		circuitOpen := false
		if open, ok := circuitStates[p.ID]; ok {
			circuitOpen = open
		}
		avgLatency := int64(0)
		if l, ok := latencies[p.ID]; ok {
			avgLatency = l
		}
		result = append(result, store.ProviderHealthStatus{
			ProviderID:   p.ID,
			ProviderName: p.Name,
			Type:         p.Type,
			Enabled:      p.Enabled,
			HealthStatus: health,
			CircuitOpen:  circuitOpen,
			AvgLatencyMS: avgLatency,
		})
	}
	writeJSON(w, result)
}

// ---- Tenant Suspend/Unsuspend ----

func (s *Server) AdminSuspendTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing tenant id", http.StatusBadRequest)
		return
	}
	if err := s.Store.SuspendTenant(r.Context(), id, true); err != nil {
		http.Error(w, "failed to suspend tenant", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) AdminUnsuspendTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing tenant id", http.StatusBadRequest)
		return
	}
	if err := s.Store.SuspendTenant(r.Context(), id, false); err != nil {
		http.Error(w, "failed to unsuspend tenant", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// ---- Tenant Detail ----

func (s *Server) AdminTenantDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing tenant id", http.StatusBadRequest)
		return
	}
	tenant, err := s.Store.GetTenantByID(r.Context(), id)
	if err != nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}
	writeJSON(w, tenant)
}

// ---- Tenant Transactions ----

func (s *Server) AdminTenantTransactions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing tenant id", http.StatusBadRequest)
		return
	}
	limitStr := r.URL.Query().Get("limit")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 100
	}
	txs, err := s.Store.ListTransactions(r.Context(), id, limit)
	if err != nil {
		http.Error(w, "failed to list transactions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, txs)
}

// Embeddings proxies embedding requests to the appropriate provider.
func (s *Server) Embeddings(w http.ResponseWriter, r *http.Request) {
	tenant := middleware.TenantFromContext(r.Context())
	if tenant == nil {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}
	if tenant.Suspended {
		http.Error(w, "account suspended", http.StatusForbidden)
		return
	}
	if tenant.BalanceUSD <= 0 {
		http.Error(w, "insufficient balance", http.StatusPaymentRequired)
		return
	}

	// Read raw body and forward to an OpenAI-compatible provider
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Parse model from request
	var parsed struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Find provider for this model
	providerType, ok, _ := s.Store.GetModelProvider(r.Context(), parsed.Model)
	if !ok || providerType == "" {
		providerType = "openai" // default to openai for embeddings
	}

	providers, err := s.Store.GetEnabledProvidersByType(r.Context(), providerType)
	if err != nil || len(providers) == 0 {
		http.Error(w, "no provider available for embeddings", http.StatusBadGateway)
		return
	}

	var lastErr error
	for _, p := range providers {
		if p.APIKey == "" {
			continue
		}
		url := "https://api.openai.com/v1/embeddings"
		if providerType == "generic-openai" && p.BaseURL != "" {
			url = strings.TrimRight(p.BaseURL, "/") + "/v1/embeddings"
		}

		req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.APIKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			b, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("%s", string(b))
			continue
		}

		// Forward the response directly
		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, resp.Body)
		return
	}

	if lastErr != nil {
		writeError(w, fmt.Errorf("embeddings failed: %w", lastErr))
		return
	}
	http.Error(w, "no provider with API key for embeddings", http.StatusBadGateway)
}

// ListModels returns OpenAI-compatible /v1/models response.
func (s *Server) ListModels(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListAllModels(r.Context())
	if err != nil {
		http.Error(w, "failed to list models", http.StatusInternalServerError)
		return
	}
	type modelObj struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}
	data := make([]modelObj, 0, len(items))
	for _, m := range items {
		data = append(data, modelObj{
			ID:      m.Model,
			Object:  "model",
			Created: 1700000000,
			OwnedBy: m.ProviderType,
		})
	}
	writeJSON(w, map[string]interface{}{
		"object": "list",
		"data":   data,
	})
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse{Error: models.ErrorDetail{Message: err.Error(), Type: "upstream_error", Code: "upstream_failed"}})
}

func errCode(err error) string {
	if err == nil {
		return ""
	}
	return "upstream_failed"
}

func extractText(req models.ChatCompletionRequest) string {
	buf := ""
	for _, msg := range req.Messages {
		text := models.ContentText(msg.Content)
		if text != "" {
			buf += text + " "
		}
		if models.ContentHasImage(msg.Content) {
			buf += "[image] "
		}
	}
	return buf
}

func extractAPIKey(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
