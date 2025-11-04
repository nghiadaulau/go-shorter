package cache

import (
	"sync"
	"time"
)

type entry struct {
	val string
	exp time.Time
}

type TTLCache struct {
	mu   sync.RWMutex
	data map[string]entry
	ttl  time.Duration
}

func NewTTL(ttl time.Duration) *TTLCache {
	return &TTLCache{data: make(map[string]entry), ttl: ttl}
}

func (c *TTLCache) Get(key string) (string, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.exp) {
		return "", false
	}
	return e.val, true
}

func (c *TTLCache) Set(key, val string) {
	c.mu.Lock()
	c.data[key] = entry{val: val, exp: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}
