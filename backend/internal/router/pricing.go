package router

var ModelPricingUSDPer1K = map[string]float64{
	"gpt-4o": 0.005,
	"gpt-4o-mini": 0.0015,
	"gpt-4.1": 0.008,
	"gpt-4.1-mini": 0.002,
	"gpt-3.5-turbo": 0.001,
	"claude-3-5-sonnet": 0.006,
	"claude-3-5-haiku": 0.001,
	"claude-3-opus": 0.015,
	"gemini-1.5-pro": 0.0035,
	"gemini-1.5-flash": 0.001,
	"gemini-1.0-pro": 0.001,
}

func EstimateCostUSD(model string, tokens int) float64 {
	price, ok := ModelPricingUSDPer1K[model]
	if !ok {
		price = 0.002
	}
	return price * float64(tokens) / 1000.0
}
