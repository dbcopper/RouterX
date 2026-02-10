package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	DatabaseURL        string
	RedisURL           string
	JWTSecret          string
	EnableRealCalls    bool
	DefaultTenantID    string
	OtelEndpoint       string
	OtelServiceName    string
}

func Load() Config {
	return Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://routerx:routerx@localhost:5432/routerx?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:       getEnv("JWT_SECRET", "change_me"),
		EnableRealCalls: getEnvBool("ENABLE_REAL_CALLS", false),
		DefaultTenantID: getEnv("DEFAULT_TENANT_ID", "demo"),
		OtelEndpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		OtelServiceName: getEnv("OTEL_SERVICE_NAME", "routerx-backend"),
	}
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return parsed
}
