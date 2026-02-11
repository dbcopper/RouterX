package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	"routerx/internal/api"
	"routerx/internal/config"
	"routerx/internal/limiter"
	"routerx/internal/metrics"
	"routerx/internal/middleware"
	"routerx/internal/observability"
	"routerx/internal/router"
	"routerx/internal/store"
)

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	cfg := config.Load()

	switch cmd {
	case "migrate":
		runMigrations(cfg)
		return
	case "seed":
		runSeed(cfg)
		return
	default:
		// serve
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx := context.Background()
	shutdown, err := observability.InitTracer(ctx, cfg.OtelEndpoint, cfg.OtelServiceName)
	if err == nil {
		defer shutdown(ctx)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: parseRedisAddr(cfg.RedisURL)})

	st := store.New(pool)
	r := router.New(st, cfg.EnableRealCalls, redisClient)
	metrics.Register()
	lim := limiter.New(redisClient, 10, 5)

	srv := &api.Server{Store: st, Router: r, Limiter: lim, Logger: logger, JWTSecret: cfg.JWTSecret}

	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"}, AllowedHeaders: []string{"*"}}))
	router.Use(func(next http.Handler) http.Handler { return otelhttp.NewHandler(next, "http") })

	router.Get("/health", srv.Health)
	router.Handle("/metrics", promhttp.Handler())

	router.Route("/v1", func(r chi.Router) {
		r.Use(middleware.WithAPIKey(st))
		r.Post("/chat/completions", srv.ChatCompletions)
	})

	router.Route("/admin", func(r chi.Router) {
		r.Post("/login", srv.AdminLogin)
		r.Group(func(r chi.Router) {
			r.Use(middleware.AdminAuth(cfg.JWTSecret))
			r.Get("/providers", srv.AdminProviders)
			r.Post("/providers", srv.AdminCreateProvider)
			r.Put("/providers/{id}", srv.AdminUpdateProvider)
			r.Get("/tenants", srv.AdminTenants)
			r.Put("/tenants/{id}/balance", srv.AdminUpdateTenantBalance)
			r.Get("/requests", srv.AdminRequests)
			r.Get("/model-pricing", srv.AdminListModelPricing)
			r.Post("/model-pricing", srv.AdminUpsertModelPricing)
		})
	})

	router.Route("/auth", func(r chi.Router) {
		r.Post("/login", srv.AuthLogin)
		r.Post("/register", srv.AuthRegister)
	})

	router.Route("/user", func(r chi.Router) {
		r.Post("/login", srv.TenantLogin)
		r.Group(func(r chi.Router) {
			r.Use(middleware.TenantUserAuth(cfg.JWTSecret))
			r.Get("/profile", srv.TenantProfile)
			r.Get("/usage", srv.TenantUsage)
			r.Get("/api-keys", srv.TenantAPIKeys)
			r.Post("/api-keys", srv.TenantCreateAPIKey)
			r.Delete("/api-keys/{key}", srv.TenantDeleteAPIKey)
		})
	})

	addr := ":" + cfg.Port
	logger.Info("server starting", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}

func parseRedisAddr(url string) string {
	// minimal parse for redis://host:port/db
	trimmed := url
	if len(trimmed) > 8 && trimmed[:8] == "redis://" {
		trimmed = trimmed[8:]
	}
	for i, ch := range trimmed {
		if ch == '/' {
			return trimmed[:i]
		}
	}
	return trimmed
}

func runMigrations(cfg config.Config) {
	flag.CommandLine.Parse(os.Args[2:])
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Println("db connect failed:", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := migrateDir(ctx, pool, resolvePath("migrations")); err != nil {
		fmt.Println("migrate failed:", err)
		os.Exit(1)
	}
	fmt.Println("migrations applied")
}

func runSeed(cfg config.Config) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Println("db connect failed:", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := seedData(ctx, pool); err != nil {
		fmt.Println("seed failed:", err)
		os.Exit(1)
	}
	fmt.Println("seed completed")
}

func migrateDir(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY, applied_at TIMESTAMP NOT NULL)`); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() { continue }
		name := e.Name()
		var exists bool
		row := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE filename=$1)`, name)
		if err := row.Scan(&exists); err != nil { return err }
		if exists { continue }
		b, err := os.ReadFile(dir + "/" + name)
		if err != nil { return err }
		if _, err := pool.Exec(ctx, string(b)); err != nil { return err }
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (filename, applied_at) VALUES ($1,$2)`, name, time.Now().UTC()); err != nil { return err }
	}
	return nil
}

func seedData(ctx context.Context, pool *pgxpool.Pool) error {
	b, err := os.ReadFile(resolvePath("scripts/seed.sql"))
	if err != nil { return err }
	_, err = pool.Exec(ctx, string(b))
	return err
}

func resolvePath(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if _, err := os.Stat("../" + path); err == nil {
		return "../" + path
	}
	return path
}
