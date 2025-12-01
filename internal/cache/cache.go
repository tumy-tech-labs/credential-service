package cache

import "time"

// DecisionCache captures pluggable storage for authorization decisions.
type DecisionCache interface {
	Get(key string) (value []byte, ok bool, err error)
	Set(key string, value []byte, ttl time.Duration) error
}

// NoopDecisionCache is a disabled cache implementation.
type NoopDecisionCache struct{}

// Get always returns a cache miss.
func (NoopDecisionCache) Get(key string) ([]byte, bool, error) { return nil, false, nil }

// Set is a no-op.
func (NoopDecisionCache) Set(key string, value []byte, ttl time.Duration) error { return nil }
