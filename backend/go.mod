module routerx

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.10
	github.com/go-chi/cors v1.2.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.5.5
	github.com/jackc/pgx/v5/pgxpool v5.5.5
	github.com/redis/go-redis/v9 v9.5.1
	github.com/rs/xid v1.5.0
	github.com/segmentio/ksuid v1.0.4
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.19.0
	github.com/prometheus/client_golang v1.19.0
)
