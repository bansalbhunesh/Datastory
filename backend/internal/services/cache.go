package services

import (
	"sync"
	"time"
)

type ttlEntry struct {
	val any
	exp time.Time
}

type ttlCache struct {
	ttl  time.Duration
	mu   sync.RWMutex
	data map[string]ttlEntry
}

func newTTLCache(ttl time.Duration) *ttlCache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &ttlCache{ttl: ttl, data: make(map[string]ttlEntry)}
}

func (c *ttlCache) get(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.exp) {
		return nil, false
	}
	return e.val, true
}

func (c *ttlCache) set(key string, val any) {
	c.mu.Lock()
	c.data[key] = ttlEntry{val: val, exp: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}
