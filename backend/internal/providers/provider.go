package providers

import (
	"bufio"
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
	client := &http.Client{Timeout: 120 * time.Second}
	switch p.Type {
	case "openai":
		return &openAIProvider{baseProvider{info: p, enableReal: enableReal, httpClient: client, providerType: "openai"}}
	case "anthropic":
		return &anthropicProvider{baseProvider{info: p, enableReal: enableReal, httpClient: client, providerType: "anthropic"}}
	case "gemini":
		return &geminiProvider{baseProvider{info: p, enableReal: enableReal, httpClient: client, providerType: "gemini"}}
	case "generic-openai":
		return &genericOpenAIProvider{baseProvider{info: p, enableReal: enableReal, httpClient: client, providerType: "generic-openai"}}
	default:
		return &genericOpenAIProvider{baseProvider{info: p, enableReal: enableReal, httpClient: client, providerType: "generic-openai"}}
	}
}

func (b *baseProvider) Name() string           { return b.info.Name }
func (b *baseProvider) SupportsText() bool      { return b.info.SupportsText }
func (b *baseProvider) SupportsVision() bool    { return b.info.SupportsVision }

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
				Role:    "assistant",
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
	var raw struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			Finish string `json:"finish_reason"`
		} `json:"choices"`
		Usage models.Usage `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return models.ChatCompletionResponse{}, err
	}
	out := models.ChatCompletionResponse{
		ID:      raw.ID,
		Object:  raw.Object,
		Created: raw.Created,
		Model:   raw.Model,
		Usage:   raw.Usage,
	}
	for _, c := range raw.Choices {
		msg := models.AssistantMessage{Role: c.Message.Role}
		if len(c.Message.Content) == 0 || string(c.Message.Content) == "null" {
			msg.Content = []models.ContentPart{}
		} else if c.Message.Content[0] == '"' {
			var s string
			if err := json.Unmarshal(c.Message.Content, &s); err != nil {
				return models.ChatCompletionResponse{}, err
			}
			msg.Content = []models.ContentPart{{Type: "text", Text: s}}
		} else {
			var parts []models.ContentPart
			if err := json.Unmarshal(c.Message.Content, &parts); err != nil {
				return models.ChatCompletionResponse{}, err
			}
			msg.Content = parts
		}
		out.Choices = append(out.Choices, models.Choice{
			Index:   c.Index,
			Message: msg,
			Finish:  c.Finish,
		})
	}
	if out.Model == "" {
		out.Model = model
	}
	return out, nil
}

// handleOpenAIStream reads SSE lines from an OpenAI-compatible stream response,
// forwards each chunk to the client via send(), and returns accumulated tokens.
func handleOpenAIStream(resp *http.Response, model string, send StreamSender) (models.ChatCompletionResponse, int, error) {
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return models.ChatCompletionResponse{}, 0, errors.New(string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	var fullText strings.Builder
	var totalTokens int
	var respID string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			_ = send("[DONE]")
			break
		}
		// Forward the raw chunk to the client
		if send != nil {
			if err := send(data); err != nil {
				return models.ChatCompletionResponse{}, totalTokens, err
			}
		}
		// Parse to extract content for the aggregate response
		var chunk struct {
			ID      string `json:"id"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				TotalTokens int `json:"total_tokens"`
			} `json:"usage,omitempty"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err == nil {
			if chunk.ID != "" {
				respID = chunk.ID
			}
			for _, c := range chunk.Choices {
				fullText.WriteString(c.Delta.Content)
			}
			if chunk.Usage != nil && chunk.Usage.TotalTokens > 0 {
				totalTokens = chunk.Usage.TotalTokens
			}
		}
	}

	// Estimate tokens if the stream didn't provide usage
	if totalTokens == 0 {
		// Rough estimate: ~4 chars per token
		totalTokens = len(fullText.String()) / 4
		if totalTokens < 1 {
			totalTokens = 1
		}
	}

	out := models.ChatCompletionResponse{
		ID:      respID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.Choice{{
			Index:   0,
			Message: models.AssistantMessage{Role: "assistant", Content: []models.ContentPart{{Type: "text", Text: fullText.String()}}},
			Finish:  "stop",
		}},
		Usage: models.Usage{TotalTokens: totalTokens},
	}
	return out, totalTokens, nil
}

// handleAnthropicStream reads SSE from Anthropic's streaming API
func handleAnthropicStream(resp *http.Response, model string, send StreamSender) (models.ChatCompletionResponse, int, error) {
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return models.ChatCompletionResponse{}, 0, errors.New(string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	var fullText strings.Builder
	var totalTokens int

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta.Text != "" {
				fullText.WriteString(event.Delta.Text)
				// Convert to OpenAI-compatible chunk format for client
				chunk := fmt.Sprintf(`{"choices":[{"delta":{"content":%s}}]}`, jsonString(event.Delta.Text))
				if send != nil {
					if err := send(chunk); err != nil {
						return models.ChatCompletionResponse{}, totalTokens, err
					}
				}
			}
		case "message_delta":
			if event.Usage.OutputTokens > 0 {
				totalTokens = event.Usage.InputTokens + event.Usage.OutputTokens
			}
		case "message_stop":
			if send != nil {
				_ = send("[DONE]")
			}
		}
	}

	if totalTokens == 0 {
		totalTokens = len(fullText.String()) / 4
		if totalTokens < 1 {
			totalTokens = 1
		}
	}

	out := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("anthropic_%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.Choice{{
			Index:   0,
			Message: models.AssistantMessage{Role: "assistant", Content: []models.ContentPart{{Type: "text", Text: fullText.String()}}},
			Finish:  "stop",
		}},
		Usage: models.Usage{TotalTokens: totalTokens},
	}
	return out, totalTokens, nil
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
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
	if p.info.APIKey == "" {
		return models.ChatCompletionResponse{}, 0, 0, fmt.Errorf("no API key configured for provider %s (openai)", p.info.Name)
	}
	url := "https://api.openai.com/v1/chat/completions"
	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   stream,
	}
	if stream {
		payload["stream_options"] = map[string]interface{}{"include_usage": true}
	}
	start := time.Now()
	res, err := p.doOpenAIRequest(ctx, url, payload, p.info.APIKey)
	if err != nil {
		return models.ChatCompletionResponse{}, 0, 0, err
	}
	defer res.Body.Close()

	if stream && send != nil {
		out, tokens, err := handleOpenAIStream(res, req.Model, send)
		return out, time.Since(start), tokens, err
	}

	out, err := parseOpenAIResponse(res, req.Model)
	return out, time.Since(start), out.Usage.TotalTokens, err
}

func (p *genericOpenAIProvider) Chat(ctx context.Context, req models.ChatCompletionRequest, stream bool, send StreamSender) (models.ChatCompletionResponse, time.Duration, int, error) {
	if !p.enableReal {
		return p.chatDummy(stream, send, req)
	}
	if p.info.APIKey == "" {
		return models.ChatCompletionResponse{}, 0, 0, fmt.Errorf("no API key configured for provider %s (generic-openai)", p.info.Name)
	}
	base := p.info.BaseURL
	if base == "" {
		return models.ChatCompletionResponse{}, 0, 0, errors.New("base_url required")
	}
	url := fmt.Sprintf("%s/v1/chat/completions", strings.TrimRight(base, "/"))
	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   stream,
	}
	start := time.Now()
	res, err := p.doOpenAIRequest(ctx, url, payload, p.info.APIKey)
	if err != nil {
		return models.ChatCompletionResponse{}, 0, 0, err
	}
	defer res.Body.Close()

	if stream && send != nil {
		out, tokens, err := handleOpenAIStream(res, req.Model, send)
		return out, time.Since(start), tokens, err
	}

	out, err := parseOpenAIResponse(res, req.Model)
	return out, time.Since(start), out.Usage.TotalTokens, err
}

func (p *anthropicProvider) Chat(ctx context.Context, req models.ChatCompletionRequest, stream bool, send StreamSender) (models.ChatCompletionResponse, time.Duration, int, error) {
	if !p.enableReal {
		return p.chatDummy(stream, send, req)
	}
	if p.info.APIKey == "" {
		return models.ChatCompletionResponse{}, 0, 0, fmt.Errorf("no API key configured for provider %s (anthropic)", p.info.Name)
	}
	url := "https://api.anthropic.com/v1/messages"

	// Convert messages to Anthropic format
	var system string
	var anthropicMsgs []map[string]interface{}
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			for _, p := range msg.Content {
				if p.Text != "" {
					system += p.Text + "\n"
				}
			}
			continue
		}
		content := ""
		for _, p := range msg.Content {
			if p.Text != "" {
				content += p.Text
			}
		}
		anthropicMsgs = append(anthropicMsgs, map[string]interface{}{
			"role":    msg.Role,
			"content": content,
		})
	}

	payload := map[string]interface{}{
		"model":      req.Model,
		"messages":   anthropicMsgs,
		"max_tokens": 4096,
	}
	if system != "" {
		payload["system"] = strings.TrimSpace(system)
	}
	if stream {
		payload["stream"] = true
	}

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.info.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	start := time.Now()
	res, err := p.httpClient.Do(httpReq)
	if err != nil {
		return models.ChatCompletionResponse{}, 0, 0, err
	}
	defer res.Body.Close()

	if stream && send != nil {
		out, tokens, err := handleAnthropicStream(res, req.Model, send)
		return out, time.Since(start), tokens, err
	}

	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return models.ChatCompletionResponse{}, time.Since(start), 0, errors.New(string(b))
	}

	var anthropicResp struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Model string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(res.Body).Decode(&anthropicResp); err != nil {
		return models.ChatCompletionResponse{}, time.Since(start), 0, err
	}

	var text string
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}

	totalTokens := anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens
	out := models.ChatCompletionResponse{
		ID:      anthropicResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   anthropicResp.Model,
		Choices: []models.Choice{{
			Index:   0,
			Message: models.AssistantMessage{Role: "assistant", Content: []models.ContentPart{{Type: "text", Text: text}}},
			Finish:  "stop",
		}},
		Usage: models.Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      totalTokens,
		},
	}
	return out, time.Since(start), totalTokens, nil
}

func (p *geminiProvider) Chat(ctx context.Context, req models.ChatCompletionRequest, stream bool, send StreamSender) (models.ChatCompletionResponse, time.Duration, int, error) {
	if !p.enableReal {
		return p.chatDummy(stream, send, req)
	}
	if p.info.APIKey == "" {
		return models.ChatCompletionResponse{}, 0, 0, fmt.Errorf("no API key configured for provider %s (gemini)", p.info.Name)
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

	method := "generateContent"
	if stream {
		method = "streamGenerateContent?alt=sse"
	}

	makeRequest := func(model string) (*http.Response, error) {
		url := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":" + method
		if apiKey != "" && !strings.Contains(url, "key=") {
			if strings.Contains(url, "?") {
				url = url + "&key=" + apiKey
			} else {
				url = url + "?key=" + apiKey
			}
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
			// Use the retry response
			res = res2
		} else {
			return models.ChatCompletionResponse{}, time.Since(start), 0, errors.New(body)
		}
	}

	// Handle Gemini streaming (SSE format with alt=sse)
	if stream && send != nil {
		return handleGeminiStream(res, req.Model, send, start)
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

func handleGeminiStream(resp *http.Response, model string, send StreamSender, start time.Time) (models.ChatCompletionResponse, time.Duration, int, error) {
	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer for potentially large chunks
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var fullText strings.Builder
	var totalTokens int

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var g geminiResponse
		if err := json.Unmarshal([]byte(data), &g); err != nil {
			continue
		}

		for _, cand := range g.Candidates {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					fullText.WriteString(part.Text)
					// Convert to OpenAI-compatible chunk format
					chunk := fmt.Sprintf(`{"choices":[{"delta":{"content":%s}}]}`, jsonString(part.Text))
					if err := send(chunk); err != nil {
						return models.ChatCompletionResponse{}, time.Since(start), totalTokens, err
					}
				}
			}
		}

		if g.UsageMetadata.TotalTokenCount > 0 {
			totalTokens = g.UsageMetadata.TotalTokenCount
		}
	}

	_ = send("[DONE]")

	if totalTokens == 0 {
		totalTokens = len(fullText.String()) / 4
		if totalTokens < 1 {
			totalTokens = 1
		}
	}

	out := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("gemini_%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.Choice{{
			Index:   0,
			Message: models.AssistantMessage{Role: "assistant", Content: []models.ContentPart{{Type: "text", Text: fullText.String()}}},
			Finish:  "stop",
		}},
		Usage: models.Usage{TotalTokens: totalTokens},
	}
	return out, time.Since(start), totalTokens, nil
}
