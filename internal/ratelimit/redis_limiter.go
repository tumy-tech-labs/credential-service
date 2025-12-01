package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter implements rate limiting using Redis with sliding window.
type RedisLimiter struct {
	client   *redis.Client
	requests int           // Max requests per window
	window   time.Duration // Time window
}

// NewRedisLimiterFromEnv creates a Redis rate limiter from environment variables.
func NewRedisLimiterFromEnv() (*RedisLimiter, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil, errors.New("REDIS_ADDR is required for rate limiting")
	}

	// Parse rate limit configuration
	requests, err := strconv.Atoi(getEnvOrDefault("RATELIMIT_REQUESTS", "100"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATELIMIT_REQUESTS: %w", err)
	}

	windowStr := getEnvOrDefault("RATELIMIT_WINDOW", "1m")
	window, err := time.ParseDuration(windowStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RATELIMIT_WINDOW: %w", err)
	}

	db := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisLimiter{
		client:   client,
		requests: requests,
		window:   window,
	}, nil
}

// Allow checks if the request should be allowed based on sliding window rate limiting.
func (r *RedisLimiter) Allow(key string) (bool, error) {
	if r.client == nil {
		return false, errors.New("redis client not configured")
	}

	now := time.Now()
	windowStart := now.Add(-r.window)

	// Use Lua script for atomic sliding window rate limiting
	script := `
		local key = KEYS[1]
		local window_start = ARGV[1]
		local now = ARGV[2]
		local limit = tonumber(ARGV[3])
		
		-- Remove entries outside the window
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)
		
		-- Count current requests in window
		local current = redis.call('ZCARD', key)
		
		if current < limit then
			-- Add current request
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, 3600) -- Expire key after 1 hour of inactivity
			return 1
		else
			return 0
		end
	`

	result, err := r.client.Eval(context.Background(), script, []string{key}, 
		windowStart.UnixNano(), now.UnixNano(), r.requests).Result()
	if err != nil {
		return false, fmt.Errorf("rate limit eval error: %w", err)
	}

	allowed, ok := result.(int64)
	if !ok {
		return false, errors.New("unexpected rate limit response type")
	}

	return allowed == 1, nil
}

// Close closes the Redis connection.
func (r *RedisLimiter) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}