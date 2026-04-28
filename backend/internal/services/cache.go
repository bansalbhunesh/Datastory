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
	mu   sync.Mutex
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
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(e.exp) {
		delete(c.data, key)
		return nil, false
	}
	return e.val, true
}

func (c *ttlCache) set(key string, val any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.data[key]; !ok && len(c.data) >= c.max {
		c.evictOneLocked()
	}
	c.data[key] = ttlEntry{val: val, exp: time.Now().Add(c.ttl)}
}

// evictOneLocked drops one expired entry, or the entry closest to expiry if
// nothing has expired yet. Must be called with c.mu held.
func (c *ttlCache) evictOneLocked() {
	now := time.Now()
	var oldestKey string
	oldestExp := time.Time{}
	first := true
	for k, v := range c.data {
		if now.After(v.exp) {
			delete(c.data, k)
			return
		}
		if first || v.exp.Before(oldestExp) {
			oldestKey = k
			oldestExp = v.exp
			first = false
		}
	}
	if oldestKey != "" {
		delete(c.data, oldestKey)
	}
}
