package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type limiterState struct {
	tokens float64
	last   time.Time
}

const (
	rateLimitCleanupInterval = 60 * time.Second
	rateLimitStaleAfter      = 5 * time.Minute
)

// RateLimit enforces an in-memory token bucket per API-key or client IP.
// The variant with context cancels the background cleanup goroutine when
// the context is done; use it from places with a clean shutdown lifecycle.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	return RateLimitWithContext(context.Background(), rps, burst)
}

// RateLimitWithContext is the lifecycle-aware variant. The cleanup goroutine
// exits when ctx is done, preventing goroutine leaks in tests / shutdowns.
func RateLimitWithContext(ctx context.Context, rps float64, burst int) gin.HandlerFunc {
	if rps <= 0 {
		rps = 5
	}
	if burst <= 0 {
		burst = 20
	}
	var (
		mu     sync.Mutex
		states = map[string]limiterState{}
	)
	go func() {
		ticker := time.NewTicker(rateLimitCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				mu.Lock()
				for k, s := range states {
					if now.Sub(s.last) > rateLimitStaleAfter {
						delete(states, k)
					}
				}
				mu.Unlock()
			}
		}
	}()
	return func(c *gin.Context) {
		// CORS preflight is browser-issued and usually unauthenticated; never
		// rate-limit it or callers see opaque CORS failures.
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		key := strings.TrimSpace(c.GetHeader(apiKeyHeader))
		if key == "" {
			key = c.ClientIP()
		}
		now := time.Now()
		mu.Lock()
		state := states[key]
		if state.last.IsZero() {
			state.last = now
			state.tokens = float64(burst)
		}
		elapsed := now.Sub(state.last).Seconds()
		state.tokens += elapsed * rps
		if state.tokens > float64(burst) {
			state.tokens = float64(burst)
		}
		state.last = now
		if state.tokens < 1 {
			states[key] = state
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		state.tokens -= 1
		states[key] = state
		mu.Unlock()
		c.Next()
	}
}
