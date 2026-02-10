package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"routerx/internal/store"
)

type contextKey string

const (
	ctxTenant contextKey = "tenant"
)

func TenantFromContext(ctx context.Context) *store.Tenant {
	val := ctx.Value(ctxTenant)
	if val == nil {
		return nil
	}
	tenant, _ := val.(*store.Tenant)
	return tenant
}

func WithAPIKey(store *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "missing api key", http.StatusUnauthorized)
				return
			}
			key := strings.TrimPrefix(auth, "Bearer ")
			tenant, err := store.GetTenantByAPIKey(r.Context(), key)
			if err != nil {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxTenant, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func AdminAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil || !token.Valid || claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), contextKey("admin"), claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NewAdminToken(secret, username string, ttl time.Duration) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

var ErrUnauthorized = errors.New("unauthorized")
