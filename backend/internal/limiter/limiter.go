package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	Redis *redis.Client
	QPS   int
	Conc  int
}

func New(client *redis.Client, qps, conc int) *Limiter {
	return &Limiter{Redis: client, QPS: qps, Conc: conc}
}

func (l *Limiter) Allow(ctx context.Context, tenantID string) (bool, error) {
	key := "qps:" + tenantID + ":" + time.Now().UTC().Format("20060102150405")
	pipe := l.Redis.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 2*time.Second)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}
	if int(incr.Val()) > l.QPS {
		return false, nil
	}
	return true, nil
}

func (l *Limiter) Acquire(ctx context.Context, tenantID string) (bool, error) {
	key := "conc:" + tenantID
	val, err := l.Redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if val == 1 {
		l.Redis.Expire(ctx, key, 60*time.Second)
	}
	if int(val) > l.Conc {
		l.Redis.Decr(ctx, key)
		return false, nil
	}
	return true, nil
}

func (l *Limiter) Release(ctx context.Context, tenantID string) {
	key := "conc:" + tenantID
	_ = l.Redis.Decr(ctx, key).Err()
}
