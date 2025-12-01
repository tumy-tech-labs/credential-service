package cache

import "testing"

func TestNoopDecisionCache(t *testing.T) {
	cache := NoopDecisionCache{}

	if _, ok, err := cache.Get("key"); err != nil || ok {
		t.Fatalf("expected miss without error, got ok=%v err=%v", ok, err)
	}

	if err := cache.Set("key", []byte("value"), 0); err != nil {
		t.Fatalf("expected no error on set: %v", err)
	}
}
