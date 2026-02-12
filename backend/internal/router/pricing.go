package router

var ModelPricingUSDPer1K = map[string]float64{
	// OpenAI
	"gpt-4o":        0.005,
	"gpt-4o-mini":   0.0015,
	"gpt-4.1":       0.008,
	"gpt-4.1-mini":  0.002,
	"gpt-4.1-nano":  0.001,
	"gpt-3.5-turbo": 0.001,
	"o1":            0.015,
	"o1-mini":       0.003,
	"o1-pro":        0.060,
	"o3":            0.020,
	"o3-mini":       0.004,
	"o4-mini":       0.004,
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
}

func EstimateCostUSD(model string, tokens int) float64 {
	price, ok := ModelPricingUSDPer1K[model]
	if !ok {
		price = 0.002
	}
	return price * float64(tokens) / 1000.0
}
