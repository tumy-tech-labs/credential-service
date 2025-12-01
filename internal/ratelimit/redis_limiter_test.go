package ratelimit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisLimiter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping redis integration test in short mode")
	}

	// Setup test Redis connection
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   15, // Use test database
	})
	defer client.Close()

	// Test Redis connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", redisAddr, err)
	}

	// Clean up test keys
	defer func() {
		client.FlushDB(context.Background())
	}()

	limiter := &RedisLimiter{
		client:   client,
		requests: 3,
		window:   time.Second,
	}

	testKey := "test:rate:limit"

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(testKey)
		if err != nil {
			t.Fatalf("Allow() error: %v", err)
		}
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	allowed, err := limiter.Allow(testKey)
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if allowed {
		t.Error("4th request should be denied")
	}

	// Wait for window to slide and try again
	time.Sleep(1100 * time.Millisecond)
	allowed, err = limiter.Allow(testKey)
	if err != nil {
		t.Fatalf("Allow() error: %v", err)
	}
	if !allowed {
		t.Error("Request after window should be allowed")
	}
}

func TestRedisLimiter_SlidingWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping redis integration test in short mode")
	}

	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   15,
	})
	defer client.Close()

	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer client.FlushDB(context.Background())

	limiter := &RedisLimiter{
		client:   client,
		requests: 5,
		window:   2 * time.Second,
	}

	testKey := "test:sliding:window"

	// Make 3 requests immediately
	for i := 0; i < 3; i++ {
		allowed, _ := limiter.Allow(testKey)
		if !allowed {
			t.Errorf("Initial request %d should be allowed", i+1)
		}
	}

	// Wait 1 second (half window)
	time.Sleep(time.Second)

	// Make 2 more requests (should be allowed, total 5)
	for i := 0; i < 2; i++ {
		allowed, _ := limiter.Allow(testKey)
		if !allowed {
			t.Errorf("Mid-window request %d should be allowed", i+1)
		}
	}

	// One more should be denied (would be 6)
	allowed, _ := limiter.Allow(testKey)
	if allowed {
		t.Error("Request exceeding limit should be denied")
	}

	// Wait for first batch to expire (another 1+ seconds)
	time.Sleep(1100 * time.Millisecond)
	// Should be allowed again as first 3 requests expired
	allowed, _ = limiter.Allow(testKey)
	if !allowed {
		t.Error("Request after partial window expiry should be allowed")
	}
}

func TestNewRedisLimiterFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "missing redis addr",
			env: map[string]string{
				"RATELIMIT_REQUESTS": "50",
			},
			wantErr: true,
		},
		{
			name: "invalid requests",
			env: map[string]string{
				"REDIS_ADDR":         "localhost:6379",
				"RATELIMIT_REQUESTS": "not-a-number",
			},
			wantErr: true,
		},
		{
			name: "invalid window",
			env: map[string]string{
				"REDIS_ADDR":       "localhost:6379",
				"RATELIMIT_WINDOW": "not-a-duration",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv("REDIS_ADDR")
			os.Unsetenv("RATELIMIT_REQUESTS")
			os.Unsetenv("RATELIMIT_WINDOW")
			os.Unsetenv("REDIS_PASSWORD")
			os.Unsetenv("REDIS_DB")

			// Set test environment
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			_, err := NewRedisLimiterFromEnv()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRedisLimiterFromEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisLimiter_ErrorHandling(t *testing.T) {
	// Test with nil client
	limiter := &RedisLimiter{
		client:   nil,
		requests: 10,
		window:   time.Minute,
	}

	allowed, err := limiter.Allow("test")
	if err == nil {
		t.Error("Expected error with nil client")
	}
	if allowed {
		t.Error("Should not allow with nil client")
	}
}