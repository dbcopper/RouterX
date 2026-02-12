package models

import (
	"encoding/json"
	"time"
)

// ContentPart represents a typed content block (text or image_url).
type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// Message uses json.RawMessage for Content to transparently handle both
// string ("hello") and array ([{"type":"text","text":"hello"}]) formats.
type Message struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	Name       string          `json:"name,omitempty"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// ParseContentParts extracts typed content parts from raw message content.
func ParseContentParts(raw json.RawMessage) []ContentPart {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	// String content: "hello"
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return []ContentPart{{Type: "text", Text: s}}
		}
		return nil
	}
	// Array content: [{"type":"text","text":"hello"}]
	if raw[0] == '[' {
		var parts []ContentPart
		if err := json.Unmarshal(raw, &parts); err == nil {
			return parts
		}
	}
	return nil
}

// ContentText returns the concatenated text from message content.
func ContentText(raw json.RawMessage) string {
	parts := ParseContentParts(raw)
	text := ""
	for _, p := range parts {
		if p.Type == "" || p.Type == "text" {
			text += p.Text
		}
	}
	return text
}

// ContentHasImage checks if message content contains an image.
func ContentHasImage(raw json.RawMessage) bool {
	parts := ParseContentParts(raw)
	for _, p := range parts {
		if p.Type == "image_url" && p.ImageURL != "" {
			return true
		}
	}
	return false
}

// StreamOptions controls streaming behavior.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ChatCompletionRequest supports all OpenAI Chat Completion API parameters.
type ChatCompletionRequest struct {
	Model               string          `json:"model"`
	Messages            []Message       `json:"messages"`
	Stream              bool            `json:"stream,omitempty"`
	StreamOptions       *StreamOptions  `json:"stream_options,omitempty"`
	MaxTokens           int             `json:"max_tokens,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	Temperature         *float64        `json:"temperature,omitempty"`
	TopP                *float64        `json:"top_p,omitempty"`
	N                   int             `json:"n,omitempty"`
	Stop                json.RawMessage `json:"stop,omitempty"`
	FrequencyPenalty    *float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float64        `json:"presence_penalty,omitempty"`
	Seed                *int            `json:"seed,omitempty"`
	Tools               json.RawMessage `json:"tools,omitempty"`
	ToolChoice          json.RawMessage `json:"tool_choice,omitempty"`
	ParallelToolCalls   *bool           `json:"parallel_tool_calls,omitempty"`
	ResponseFormat      json.RawMessage `json:"response_format,omitempty"`
	LogProbs            *bool           `json:"logprobs,omitempty"`
	TopLogProbs         *int            `json:"top_logprobs,omitempty"`
	User                string          `json:"user,omitempty"`
	Store               *bool           `json:"store,omitempty"`
	Metadata            json.RawMessage `json:"metadata,omitempty"`
	ServiceTier         string          `json:"service_tier,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AssistantMessage matches OpenAI response format: content is a string (or null for tool calls).
type AssistantMessage struct {
	Role      string          `json:"role"`
	Content   *string         `json:"content"`
	ToolCalls json.RawMessage `json:"tool_calls,omitempty"`
}

type Choice struct {
	Index   int              `json:"index"`
	Message AssistantMessage `json:"message"`
	Finish  string           `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type RequestLog struct {
	ID           int       `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	LatencyMS    int64     `json:"latency_ms"`
	TTFTMS       int64     `json:"ttft_ms"`
	Tokens       int       `json:"tokens"`
	CostUSD      float64   `json:"cost_usd"`
	PromptHash   string    `json:"prompt_hash"`
	FallbackUsed bool      `json:"fallback_used"`
	StatusCode   int       `json:"status_code"`
	ErrorCode    string    `json:"error_code"`
	UserID       string    `json:"user_id,omitempty"`
	AppTitle     string    `json:"app_title,omitempty"`
	AppReferer   string    `json:"app_referer,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// StringPtr is a helper to create a *string.
func StringPtr(s string) *string { return &s }
