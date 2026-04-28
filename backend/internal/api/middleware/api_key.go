package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const apiKeyHeader = "X-API-Key"

// APIKey rejects requests without a valid API key when configured.
// Comparison is constant-time to avoid leaking key length / prefix via timing.
func APIKey(expected string) gin.HandlerFunc {
	expected = strings.TrimSpace(expected)
	expectedBytes := []byte(expected)
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if expected == "" {
			c.Next()
			return
		}
		got := []byte(c.GetHeader(apiKeyHeader))
		if subtle.ConstantTimeEq(int32(len(got)), int32(len(expectedBytes))) != 1 ||
			subtle.ConstantTimeCompare(got, expectedBytes) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid api key"})
			return
		}
		c.Next()
	}
}
