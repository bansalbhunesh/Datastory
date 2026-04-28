package services

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestTTLCache_GetSetExpiry(t *testing.T) {
	c := newTTLCache(50*time.Millisecond, 100)
	c.set("a", 1)
	if v, ok := c.get("a"); !ok || v != 1 {
		t.Fatalf("expected hit before TTL: got %v ok=%v", v, ok)
	}
	time.Sleep(60 * time.Millisecond)
	if _, ok := c.get("a"); ok {
		t.Fatalf("expected miss after TTL")
	}
}

func TestTTLCache_EvictionWhenFull(t *testing.T) {
	c := newTTLCache(time.Hour, 3)
	c.set("a", 1)
	c.set("b", 2)
	c.set("c", 3)
	c.set("d", 4) // forces eviction; one of a/b/c should be gone
	count := 0
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := c.get(k); ok {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected exactly one eviction among a,b,c; got %d remaining", count)
	}
	if _, ok := c.get("d"); !ok {
		t.Fatalf("d should still be present")
	}
}

func TestTTLCache_OverwriteDoesNotEvict(t *testing.T) {
	c := newTTLCache(time.Hour, 2)
	c.set("a", 1)
	c.set("b", 2)
	c.set("a", 99) // overwrite, must not evict b
	if v, ok := c.get("b"); !ok || v != 2 {
		t.Fatalf("b unexpectedly evicted")
	}
	if v, _ := c.get("a"); v != 99 {
		t.Fatalf("expected overwrite, got %v", v)
	}
}

// Concurrent get/set: must not race or deadlock under -race.
func TestTTLCache_ConcurrentSafe(t *testing.T) {
	c := newTTLCache(time.Second, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				k := strconv.Itoa((id*200 + j) % 50)
				c.set(k, j)
				_, _ = c.get(k)
			}
		}(i)
	}
	wg.Wait()
}
