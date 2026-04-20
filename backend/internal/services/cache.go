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
	max  int
	mu   sync.RWMutex
	data map[string]ttlEntry
}

func newTTLCache(ttl time.Duration, maxEntries int) *ttlCache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	if maxEntries <= 0 {
		maxEntries = 1000
	}
	return &ttlCache{ttl: ttl, max: maxEntries, data: make(map[string]ttlEntry)}
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
	if len(c.data) >= c.max {
		now := time.Now()
		removed := false
		for k, v := range c.data {
			if now.After(v.exp) {
				delete(c.data, k)
				removed = true
				break
			}
		}
		if !removed {
			for k := range c.data {
				delete(c.data, k)
				break
			}
		}
	}
	c.data[key] = ttlEntry{val: val, exp: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}
