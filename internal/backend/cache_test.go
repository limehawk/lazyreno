package backend

import (
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := NewCache(1 * time.Minute)
	c.Set("key", "value")
	val, ok := c.Get("key")
	if !ok || val != "value" {
		t.Errorf("expected value, got %v (ok=%v)", val, ok)
	}
}

func TestCacheExpiry(t *testing.T) {
	c := NewCache(1 * time.Millisecond)
	c.Set("key", "value")
	time.Sleep(5 * time.Millisecond)
	_, ok := c.Get("key")
	if ok {
		t.Error("expected cache miss after expiry")
	}
}

func TestCacheInvalidate(t *testing.T) {
	c := NewCache(1 * time.Minute)
	c.Set("key", "value")
	c.Invalidate("key")
	_, ok := c.Get("key")
	if ok {
		t.Error("expected cache miss after invalidate")
	}
}

func TestCacheInvalidateAll(t *testing.T) {
	c := NewCache(1 * time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)
	c.InvalidateAll()
	_, okA := c.Get("a")
	_, okB := c.Get("b")
	if okA || okB {
		t.Error("expected all keys invalidated")
	}
}
