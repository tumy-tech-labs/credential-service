package ratelimit

// Limiter controls request admission.
type Limiter interface {
	Allow(key string) (bool, error)
}

// NoopLimiter always allows requests.
type NoopLimiter struct{}

// Allow always returns true.
func (NoopLimiter) Allow(key string) (bool, error) { return true, nil }
