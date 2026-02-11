package router

import (
	"context"
	"errors"
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

func (r *Router) Route(ctx context.Context, tenantID string, req models.ChatCompletionRequest, stream bool, send providers.StreamSender) (models.ChatCompletionResponse, string, bool, time.Duration, int, error) {
	capability := "text"
	if requestHasImage(req) {
		capability = "vision"
	}
	rule, err := r.Store.GetRoutingRule(ctx, tenantID, capability)
	if err != nil {
		return models.ChatCompletionResponse{}, "", false, 0, 0, err
	}
	if req.Model == "" {
		req.Model = rule.Model
	}
	primaryID := rule.PrimaryProviderID
	secondaryID := rule.SecondaryProviderID
	if inferredType, ok, _ := r.Store.GetModelProvider(ctx, req.Model); ok && inferredType != "" {
		if prov, ok := r.pickProviderByType(ctx, inferredType, capability); ok {
			primaryID = prov.ID
			// Only allow fallback within the same provider type when a model is explicitly mapped.
			if secondaryID != "" {
				if sec, err := r.Store.GetProviderByID(ctx, secondaryID); err == nil {
					if sec.Type != inferredType {
						secondaryID = ""
					}
				}
			}
		}
	}
	primary, err := r.Store.GetProviderByID(ctx, primaryID)
	if err != nil {
		return models.ChatCompletionResponse{}, "", false, 0, 0, err
	}
	resp, provider, usedFallback, ttft, tokens, err := r.tryProvider(ctx, primary, req, stream, send)
	if err == nil {
		return resp, provider, usedFallback, ttft, tokens, nil
	}
	if secondaryID == "" {
		return models.ChatCompletionResponse{}, primary.Name, false, ttft, tokens, err
	}
	secondary, err2 := r.Store.GetProviderByID(ctx, secondaryID)
	if err2 != nil {
		return models.ChatCompletionResponse{}, primary.Name, false, ttft, tokens, err
	}
	resp2, provider2, _, ttft2, tokens2, err2 := r.tryProvider(ctx, secondary, req, stream, send)
	if err2 != nil {
		return models.ChatCompletionResponse{}, provider2, true, ttft2, tokens2, err2
	}
	return resp2, provider2, true, ttft2, tokens2, nil
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
		for _, part := range msg.Content {
			if part.Type == "image_url" && part.ImageURL != "" {
				return true
			}
		}
	}
	return false
}

func (r *Router) pickProviderByType(ctx context.Context, providerType, capability string) (*store.Provider, bool) {
	providersList, err := r.Store.GetProviders(ctx)
	if err != nil {
		return nil, false
	}
	for _, p := range providersList {
		if !p.Enabled || p.Type != providerType {
			continue
		}
		if capability == "vision" && !p.SupportsVision {
			continue
		}
		if capability == "text" && !p.SupportsText {
			continue
		}
		return &p, true
	}
	return nil, false
}
