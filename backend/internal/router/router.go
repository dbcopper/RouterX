package router

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"routerx/internal/models"
	"routerx/internal/providers"
	"routerx/internal/store"
)

type CircuitState struct {
	Mu          sync.Mutex
	Samples     []bool
	OpenUntil   time.Time
	WindowSize  int
	Threshold   float64
	Cooldown    time.Duration
}

func (c *CircuitState) Allow() bool {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if time.Now().Before(c.OpenUntil) {
		return false
	}
	return true
}

func (c *CircuitState) Record(ok bool) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Samples = append(c.Samples, ok)
	if len(c.Samples) > c.WindowSize {
		c.Samples = c.Samples[len(c.Samples)-c.WindowSize:]
	}
	if len(c.Samples) >= 10 {
		fail := 0
		for _, s := range c.Samples {
			if !s { fail++ }
		}
		rate := float64(fail) / float64(len(c.Samples))
		if rate >= c.Threshold {
			c.OpenUntil = time.Now().Add(c.Cooldown)
		}
	}
}

type Router struct {
	Store        *store.Store
	EnableReal   bool
	Redis        *redis.Client
	Circuits     map[string]*CircuitState
	Mu           sync.Mutex
}

func New(store *store.Store, enableReal bool, redisClient *redis.Client) *Router {
	return &Router{Store: store, EnableReal: enableReal, Redis: redisClient, Circuits: map[string]*CircuitState{}}
}

func (r *Router) circuitFor(providerID string) *CircuitState {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if c, ok := r.Circuits[providerID]; ok {
		return c
	}
	c := &CircuitState{WindowSize: 20, Threshold: 0.5, Cooldown: 30 * time.Second}
	r.Circuits[providerID] = c
	return c
}

// Route implements OpenRouter-like auto-routing:
// 1. Look up model in model_catalog -> provider_type
// 2. Pick enabled providers of that type (try all for fallback)
// 3. If model not in catalog, fall back to routing_rules (optional override)
// 4. If nothing works, return clear error
func (r *Router) Route(ctx context.Context, tenantID string, req models.ChatCompletionRequest, stream bool, send providers.StreamSender) (models.ChatCompletionResponse, string, bool, time.Duration, int, error) {
	capability := "text"
	if requestHasImage(req) {
		capability = "vision"
	}

	// Step 1: Check if tenant has a routing rule override for this capability
	rule, ruleErr := r.Store.GetRoutingRule(ctx, tenantID, capability)

	// Step 2: If model not specified, try to get default from rule or use "default"
	if req.Model == "" {
		if rule != nil && rule.Model != "" {
			req.Model = rule.Model
		} else {
			req.Model = "default"
		}
	}

	var errs []string

	// Step 3: Try auto-routing via model_catalog
	providerType, catalogOK, catalogErr := r.Store.GetModelProvider(ctx, req.Model)
	if catalogOK && providerType != "" {
		resp, providerName, fallback, ttft, tokens, err := r.tryProvidersByType(ctx, providerType, capability, req, stream, send)
		if err == nil {
			return resp, providerName, fallback, ttft, tokens, nil
		}
		errs = append(errs, fmt.Sprintf("auto-route(%s): %v", providerType, err))
	} else if catalogErr != nil {
		errs = append(errs, fmt.Sprintf("catalog lookup: %v", catalogErr))
	} else {
		errs = append(errs, "model not in catalog")
	}

	// Step 4: Fall back to routing_rules if available
	if ruleErr == nil && rule != nil {
		primary, err := r.Store.GetProviderByID(ctx, rule.PrimaryProviderID)
		if err == nil {
			resp, providerName, _, ttft, tokens, err := r.tryProvider(ctx, primary, req, stream, send)
			if err == nil {
				return resp, providerName, false, ttft, tokens, nil
			}
			errs = append(errs, fmt.Sprintf("rule-primary(%s): %v", primary.Name, err))
			// Try secondary
			if rule.SecondaryProviderID != "" {
				secondary, err2 := r.Store.GetProviderByID(ctx, rule.SecondaryProviderID)
				if err2 == nil {
					resp2, provider2, _, ttft2, tokens2, err2 := r.tryProvider(ctx, secondary, req, stream, send)
					if err2 == nil {
						return resp2, provider2, true, ttft2, tokens2, nil
					}
					errs = append(errs, fmt.Sprintf("rule-secondary(%s): %v", secondary.Name, err2))
				}
			}
		}
	}

	// Step 5: No routing succeeded — show full error chain
	if len(errs) > 0 {
		return models.ChatCompletionResponse{}, "", false, 0, 0, fmt.Errorf("routing failed for model %s: %s", req.Model, strings.Join(errs, "; "))
	}
	return models.ChatCompletionResponse{}, "", false, 0, 0, fmt.Errorf("no provider available for model %s (not in model_catalog, no routing rules for tenant %s)", req.Model, tenantID)
}

// tryProvidersByType tries all enabled providers of the given type, with fallback
func (r *Router) tryProvidersByType(ctx context.Context, providerType, capability string, req models.ChatCompletionRequest, stream bool, send providers.StreamSender) (models.ChatCompletionResponse, string, bool, time.Duration, int, error) {
	providersList, err := r.Store.GetEnabledProvidersByType(ctx, providerType)
	if err != nil || len(providersList) == 0 {
		return models.ChatCompletionResponse{}, "", false, 0, 0, errors.New("no enabled provider for type: " + providerType)
	}

	// Filter by capability
	var candidates []store.Provider
	for _, p := range providersList {
		if capability == "vision" && !p.SupportsVision {
			continue
		}
		if capability == "text" && !p.SupportsText {
			continue
		}
		candidates = append(candidates, p)
	}
	if len(candidates) == 0 {
		return models.ChatCompletionResponse{}, "", false, 0, 0, errors.New("no provider supports " + capability + " for type: " + providerType)
	}

	var lastErr error
	for i, p := range candidates {
		pCopy := p
		resp, providerName, _, ttft, tokens, err := r.tryProvider(ctx, &pCopy, req, stream, send)
		if err == nil {
			return resp, providerName, i > 0, ttft, tokens, nil
		}
		lastErr = err
	}
	return models.ChatCompletionResponse{}, "", false, 0, 0, lastErr
}

func (r *Router) tryProvider(ctx context.Context, p *store.Provider, req models.ChatCompletionRequest, stream bool, send providers.StreamSender) (models.ChatCompletionResponse, string, bool, time.Duration, int, error) {
	if !p.Enabled {
		return models.ChatCompletionResponse{}, p.Name, false, 0, 0, errors.New("provider disabled")
	}
	if requestHasImage(req) && !p.SupportsVision {
		return models.ChatCompletionResponse{}, p.Name, false, 0, 0, errors.New("provider lacks vision")
	}
	if !requestHasImage(req) && !p.SupportsText {
		return models.ChatCompletionResponse{}, p.Name, false, 0, 0, errors.New("provider lacks text")
	}
	circuit := r.circuitFor(p.ID)
	if !circuit.Allow() {
		return models.ChatCompletionResponse{}, p.Name, false, 0, 0, errors.New("circuit open")
	}
	provider := providers.NewProvider(*p, r.EnableReal)
	resp, ttft, tokens, err := provider.Chat(ctx, req, stream, send)
	circuit.Record(err == nil)
	if r.Redis != nil {
		status := "ok"
		if err != nil {
			status = "fail"
		}
		_ = r.Redis.Set(ctx, "provider_health:"+p.ID, status, 30*time.Second).Err()
	}
	return resp, p.Name, false, ttft, tokens, err
}

func requestHasImage(req models.ChatCompletionRequest) bool {
	for _, msg := range req.Messages {
		if models.ContentHasImage(msg.Content) {
			return true
		}
	}
	return false
}

func (r *Router) GetCircuitStates() map[string]bool {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	states := map[string]bool{}
	for id, c := range r.Circuits {
		states[id] = !c.Allow()
	}
	return states
}
