package router

var ModelPricingUSDPer1K = map[string]float64{
	// OpenAI
	"gpt-4o":                  0.005,
	"gpt-4o-mini":             0.0015,
	"gpt-4.1":                 0.008,
	"gpt-4.1-mini":            0.002,
	"gpt-4.1-nano":            0.001,
	"gpt-3.5-turbo":           0.001,
	"o1":                      0.015,
	"o1-mini":                 0.003,
	"o1-pro":                  0.060,
	"o3":                      0.020,
	"o3-mini":                 0.004,
	"o4-mini":                 0.004,
	"text-embedding-3-small":  0.00002,
	"text-embedding-3-large":  0.00013,
	"text-embedding-ada-002":  0.0001,
	// Anthropic
	"claude-sonnet-4-5": 0.006,
	"claude-opus-4":     0.015,
	"claude-3-5-sonnet": 0.006,
	"claude-3-5-haiku":  0.001,
	"claude-3-opus":     0.015,
	"claude-3-haiku":    0.0005,
	// Gemini
	"gemini-2.5-pro":   0.005,
	"gemini-2.5-flash": 0.0015,
	"gemini-2.0-flash": 0.001,
	"gemini-1.5-pro":   0.0035,
	"gemini-1.5-flash": 0.001,
	// DeepSeek
	"deepseek-chat":     0.00027,
	"deepseek-reasoner": 0.00055,
	"deepseek-coder":    0.00027,
	// Mistral
	"mistral-large-latest":  0.004,
	"mistral-medium-latest": 0.0027,
	"mistral-small-latest":  0.001,
	"codestral-latest":      0.001,
	"mistral-embed":         0.0001,
	"open-mistral-nemo":     0.0003,
	"open-mixtral-8x22b":    0.002,
	// Meta Llama
	"meta-llama/llama-3.3-70b-instruct":  0.0006,
	"meta-llama/llama-3.1-405b-instruct": 0.003,
	"meta-llama/llama-3.1-70b-instruct":  0.0006,
	"meta-llama/llama-3.1-8b-instruct":   0.00006,
	// Qwen
	"qwen/qwen-2.5-72b-instruct":        0.0004,
	"qwen/qwen-2.5-coder-32b-instruct":  0.0002,
}

func EstimateCostUSD(model string, tokens int) float64 {
	price, ok := ModelPricingUSDPer1K[model]
	if !ok {
		price = 0.002
	}
	return price * float64(tokens) / 1000.0
}
