package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "routerx_requests_total", Help: "Total requests"},
		[]string{"provider", "status"},
	)
	LatencyMS = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "routerx_latency_ms", Help: "Latency in ms", Buckets: prometheus.LinearBuckets(50, 50, 20)},
		[]string{"provider"},
	)
	TTFTMS = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "routerx_ttft_ms", Help: "Time to first token in ms", Buckets: prometheus.LinearBuckets(50, 50, 20)},
		[]string{"provider"},
	)
)

func Register() {
	prometheus.MustRegister(RequestsTotal, LatencyMS, TTFTMS)
}
