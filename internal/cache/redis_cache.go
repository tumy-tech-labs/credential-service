package cache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisDecisionCache implements DecisionCache backed by Redis.
type RedisDecisionCache struct {
	client *redis.Client
}

// NewRedisDecisionCacheFromEnv initializes a RedisDecisionCache using standard env vars.
func NewRedisDecisionCacheFromEnv() (*RedisDecisionCache, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil, errors.New("REDIS_ADDR is required")
	}

	db := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		parsed, err := strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("parse REDIS_DB: %w", err)
		}
		db = parsed
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &RedisDecisionCache{client: client}, nil
}

// Get retrieves a cached value.
func (r *RedisDecisionCache) Get(key string) ([]byte, bool, error) {
	if r == nil || r.client == nil {
		return nil, false, errors.New("redis client not configured")
	}
	res, err := r.client.Get(context.Background(), key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return res, true, nil
}

// Set stores a value with TTL.
func (r *RedisDecisionCache) Set(key string, value []byte, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return errors.New("redis client not configured")
	}
	return r.client.Set(context.Background(), key, value, ttl).Err()
}
