package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"routerx/internal/models"
	"routerx/internal/store"
)

type StreamSender func(event string) error

type Provider interface {
	Name() string
	SupportsText() bool
	SupportsVision() bool
	Chat(ctx context.Context, req models.ChatCompletionRequest, stream bool, send StreamSender) (models.ChatCompletionResponse, time.Duration, int, error)
}

type baseProvider struct {
	info         store.Provider
	enableReal   bool
	httpClient   *http.Client
	providerType string
}

func NewProvider(p store.Provider, enableReal bool) Provider {
	switch p.Type {
	case "openai":
		return &openAIProvider{baseProvider{info: p, enableReal: enableReal, httpClient: &http.Client{Timeout: 30 * time.Second}, providerType: "openai"}}
	case "anthropic":
		return &anthropicProvider{baseProvider{info: p, enableReal: enableReal, httpClient: &http.Client{Timeout: 30 * time.Second}, providerType: "anthropic"}}
	case "gemini":
		return &geminiProvider{baseProvider{info: p, enableReal: enableReal, httpClient: &http.Client{Timeout: 30 * time.Second}, providerType: "gemini"}}
	case "generic-openai":
		return &genericOpenAIProvider{baseProvider{info: p, enableReal: enableReal, httpClient: &http.Client{Timeout: 30 * time.Second}, providerType: "generic-openai"}}
	default:
		return &genericOpenAIProvider{baseProvider{info: p, enableReal: enableReal, httpClient: &http.Client{Timeout: 30 * time.Second}, providerType: "generic-openai"}}
	}
}

func (b *baseProvider) Name() string { return b.info.Name }
func (b *baseProvider) SupportsText() bool { return b.info.SupportsText }
func (b *baseProvider) SupportsVision() bool { return b.info.SupportsVision }

func dummyResponse(provider string, req models.ChatCompletionRequest) (models.ChatCompletionResponse, int) {
	content := fmt.Sprintf("Dummy response from %s. Model=%s. Messages=%d.", provider, req.Model, len(req.Messages))
	resp := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("dummy_%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.Choice{{
			Index: 0,
			Message: models.AssistantMessage{
				Role: "assistant",
				Content: []models.ContentPart{{Type: "text", Text: content}},
			},
			Finish: "stop",
		}},
		Usage: models.Usage{PromptTokens: 10, CompletionTokens: 15, TotalTokens: 25},
	}
	return resp, resp.Usage.TotalTokens
}

func (b *baseProvider) chatDummy(stream bool, send StreamSender, req models.ChatCompletionRequest) (models.ChatCompletionResponse, time.Duration, int, error) {
	start := time.Now()
	resp, tokens := dummyResponse(b.info.Name, req)
	if stream && send != nil {
		chunks := []string{"This is a dummy ", "streamed response ", "from RouterX."}
		for _, c := range chunks {
			data := fmt.Sprintf("{\"choices\":[{\"delta\":{\"content\":%q}}]}", c)
			if err := send(data); err != nil {
				return resp, time.Since(start), tokens, err
			}
			time.Sleep(50 * time.Millisecond)
		}
		_ = send("[DONE]")
	}
	return resp, time.Since(start), tokens, nil
}

func (b *baseProvider) doOpenAIRequest(ctx context.Context, url string, payload interface{}, apiKey string) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	return b.httpClient.Do(req)
}

func parseOpenAIResponse(resp *http.Response, model string) (models.ChatCompletionResponse, error) {
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return models.ChatCompletionResponse{}, errors.New(string(b))
	}
	var out models.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return models.ChatCompletionResponse{}, err
	}
	if out.Model == "" {
		out.Model = model
	}
	return out, nil
}

type openAIProvider struct{ baseProvider }

type anthropicProvider struct{ baseProvider }

type geminiProvider struct{ baseProvider }

type genericOpenAIProvider struct{ baseProvider }

func (p *openAIProvider) Chat(ctx context.Context, req models.ChatCompletionRequest, stream bool, send StreamSender) (models.ChatCompletionResponse, time.Duration, int, error) {
	if !p.enableReal {
		return p.chatDummy(stream, send, req)
	}
	url := "https://api.openai.com/v1/chat/completions"
	payload := map[string]interface{}{
		"model": req.Model,
		"messages": req.Messages,
		"stream": stream,
	}
	start := time.Now()
	res, err := p.doOpenAIRequest(ctx, url, payload, p.info.APIKey)
	if err != nil {
		return models.ChatCompletionResponse{}, 0, 0, err
	}
	defer res.Body.Close()
	out, err := parseOpenAIResponse(res, req.Model)
	return out, time.Since(start), out.Usage.TotalTokens, err
}

func (p *genericOpenAIProvider) Chat(ctx context.Context, req models.ChatCompletionRequest, stream bool, send StreamSender) (models.ChatCompletionResponse, time.Duration, int, error) {
	if !p.enableReal {
		return p.chatDummy(stream, send, req)
	}
	base := p.info.BaseURL
	if base == "" {
		return models.ChatCompletionResponse{}, 0, 0, errors.New("base_url required")
	}
	url := fmt.Sprintf("%s/v1/chat/completions", base)
	payload := map[string]interface{}{
		"model": req.Model,
		"messages": req.Messages,
		"stream": stream,
	}
	start := time.Now()
	res, err := p.doOpenAIRequest(ctx, url, payload, p.info.APIKey)
	if err != nil {
		return models.ChatCompletionResponse{}, 0, 0, err
	}
	defer res.Body.Close()
	out, err := parseOpenAIResponse(res, req.Model)
	return out, time.Since(start), out.Usage.TotalTokens, err
}

func (p *anthropicProvider) Chat(ctx context.Context, req models.ChatCompletionRequest, stream bool, send StreamSender) (models.ChatCompletionResponse, time.Duration, int, error) {
	if !p.enableReal {
		return p.chatDummy(stream, send, req)
	}
	url := "https://api.anthropic.com/v1/messages"
	payload := map[string]interface{}{
		"model": req.Model,
		"messages": req.Messages,
		"max_tokens": 256,
	}
	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.info.APIKey)
	start := time.Now()
	res, err := p.httpClient.Do(httpReq)
	if err != nil {
		return models.ChatCompletionResponse{}, 0, 0, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return models.ChatCompletionResponse{}, time.Since(start), 0, errors.New(string(b))
	}
	// Best-effort normalization
	out := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("anthropic_%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.Choice{{Index: 0, Message: models.AssistantMessage{Role: "assistant", Content: []models.ContentPart{{Type: "text", Text: "(anthropic response normalized)"}}}, Finish: "stop"}},
	}
	return out, time.Since(start), out.Usage.TotalTokens, nil
}

func (p *geminiProvider) Chat(ctx context.Context, req models.ChatCompletionRequest, stream bool, send StreamSender) (models.ChatCompletionResponse, time.Duration, int, error) {
	if !p.enableReal {
		return p.chatDummy(stream, send, req)
	}
	url := "https://generativelanguage.googleapis.com/v1beta/models/" + req.Model + ":generateContent"
	payload := map[string]interface{}{
		"contents": req.Messages,
	}
	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	start := time.Now()
	res, err := p.httpClient.Do(httpReq)
	if err != nil {
		return models.ChatCompletionResponse{}, 0, 0, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return models.ChatCompletionResponse{}, time.Since(start), 0, errors.New(string(b))
	}
	out := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("gemini_%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.Choice{{Index: 0, Message: models.AssistantMessage{Role: "assistant", Content: []models.ContentPart{{Type: "text", Text: "(gemini response normalized)"}}}, Finish: "stop"}},
	}
	return out, time.Since(start), out.Usage.TotalTokens, nil
}
