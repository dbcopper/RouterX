package api

import (
	"encoding/json"
	"net/http"
	"time"

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
	promptHash := util.HashString(util.NormalizeSpaces(extractText(req)))
	start := time.Now()

	stream := req.Stream
	var ttft time.Duration
	var tokens int
	providerName := ""
	fallbackUsed := false
	var resp models.ChatCompletionResponse
	var routeErr error

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
		resp, providerName, fallbackUsed, ttft, tokens, routeErr = s.Router.Route(r.Context(), tenant.ID, req, true, send)
	} else {
		resp, providerName, fallbackUsed, ttft, tokens, routeErr = s.Router.Route(r.Context(), tenant.ID, req, false, nil)
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

	_ = s.Store.InsertRequestLog(r.Context(), models.RequestLog{
		TenantID:     tenant.ID,
		Provider:     providerName,
		Model:        req.Model,
		LatencyMS:    latency.Milliseconds(),
		TTFTMS:       ttft.Milliseconds(),
		Tokens:       tokens,
		PromptHash:   promptHash,
		FallbackUsed: fallbackUsed,
		StatusCode:   status,
		ErrorCode:    errCode(routeErr),
		CreatedAt:    time.Now().UTC(),
	})
	_ = s.Store.RecordUsageDaily(r.Context(), tenant.ID, providerName, req.Model, tokens, time.Now().UTC())

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

func (s *Server) AdminProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.Store.ListProviders(r.Context())
	if err != nil {
		http.Error(w, "failed to list providers", http.StatusInternalServerError)
		return
	}
	writeJSON(w, providers)
}

func (s *Server) AdminRoutingRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.Store.ListRoutingRules(r.Context())
	if err != nil {
		http.Error(w, "failed to list routing rules", http.StatusInternalServerError)
		return
	}
	writeJSON(w, rules)
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
		for _, part := range msg.Content {
			if part.Type == "text" {
				buf += part.Text + " "
			}
			if part.Type == "image_url" {
				buf += "[image] "
			}
		}
	}
	return buf
}
