package middleware

import (
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

// RateLimit enforces an in-memory token bucket per API-key or client IP.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
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
	return func(c *gin.Context) {
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
