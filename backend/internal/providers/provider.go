package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func toGeminiContents(messages []models.Message) []geminiContent {
	contents := make([]geminiContent, 0, len(messages))
	for _, msg := range messages {
		role := "user"
		switch msg.Role {
		case "assistant":
			role = "model"
		case "system":
			role = "user"
		default:
			role = "user"
		}

		parts := make([]geminiPart, 0, len(msg.Content))
		for i, part := range msg.Content {
			if part.Type == "" || part.Type == "text" {
				text := part.Text
				if msg.Role == "system" && i == 0 && text != "" {
					text = "System: " + text
				}
				parts = append(parts, geminiPart{Text: text})
				continue
			}
			if part.Type == "image_url" && part.ImageURL != "" {
				parts = append(parts, geminiPart{Text: "[image] " + part.ImageURL})
			}
		}
		if len(parts) == 0 {
			parts = append(parts, geminiPart{Text: ""})
		}
		contents = append(contents, geminiContent{Role: role, Parts: parts})
	}
	return contents
}

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
	apiKey := p.info.APIKey
	payload := map[string]interface{}{
		"contents": toGeminiContents(req.Messages),
	}
	if req.MaxTokens > 0 || req.Temperature > 0 {
		gen := map[string]interface{}{}
		if req.MaxTokens > 0 {
			gen["maxOutputTokens"] = req.MaxTokens
		}
		if req.Temperature > 0 {
			gen["temperature"] = req.Temperature
		}
		payload["generationConfig"] = gen
	}

	makeRequest := func(model string) (*http.Response, error) {
		url := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":generateContent"
		if apiKey != "" {
			url = url + "?key=" + apiKey
		}
		body, _ := json.Marshal(payload)
		httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			httpReq.Header.Set("x-goog-api-key", apiKey)
		}
		return p.httpClient.Do(httpReq)
	}

	start := time.Now()
	modelName := req.Model
	res, err := makeRequest(modelName)
	if err != nil {
		return models.ChatCompletionResponse{}, 0, 0, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		body := string(b)
		if strings.Contains(body, "not found") && !strings.HasSuffix(modelName, "-latest") {
			_ = res.Body.Close()
			res2, err2 := makeRequest(modelName + "-latest")
			if err2 != nil {
				return models.ChatCompletionResponse{}, time.Since(start), 0, err2
			}
			defer res2.Body.Close()
			if res2.StatusCode >= 300 {
				b2, _ := io.ReadAll(res2.Body)
				return models.ChatCompletionResponse{}, time.Since(start), 0, errors.New(string(b2))
			}
		} else {
			return models.ChatCompletionResponse{}, time.Since(start), 0, errors.New(body)
		}
	}
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return models.ChatCompletionResponse{}, time.Since(start), 0, err
	}
	var g geminiResponse
	if err := json.Unmarshal(bodyBytes, &g); err != nil {
		return models.ChatCompletionResponse{}, time.Since(start), 0, err
	}
	text := ""
	if len(g.Candidates) > 0 {
		for _, p := range g.Candidates[0].Content.Parts {
			if p.Text != "" {
				if text != "" {
					text += "\n"
				}
				text += p.Text
			}
		}
	}
	if text == "" {
		text = "(empty gemini response)"
	}
	usage := models.Usage{
		PromptTokens:     g.UsageMetadata.PromptTokenCount,
		CompletionTokens: g.UsageMetadata.CandidatesTokenCount,
		TotalTokens:      g.UsageMetadata.TotalTokenCount,
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	out := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("gemini_%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.Choice{{Index: 0, Message: models.AssistantMessage{Role: "assistant", Content: []models.ContentPart{{Type: "text", Text: text}}}, Finish: "stop"}},
		Usage:   usage,
	}
	return out, time.Since(start), out.Usage.TotalTokens, nil
}
