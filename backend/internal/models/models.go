package models

import "time"

type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type Message struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Choice struct {
	Index   int               `json:"index"`
	Message AssistantMessage  `json:"message"`
	Finish  string            `json:"finish_reason"`
}

type AssistantMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
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
	CreatedAt    time.Time `json:"created_at"`
}
