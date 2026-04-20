package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const apiKeyHeader = "X-API-Key"

// APIKey rejects requests without a valid API key when configured.
func APIKey(expected string) gin.HandlerFunc {
	expected = strings.TrimSpace(expected)
	return func(c *gin.Context) {
		if expected == "" {
			c.Next()
			return
		}
		if c.GetHeader(apiKeyHeader) != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid api key"})
			return
		}
		c.Next()
	}
}
