package middleware

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	mrand "math/rand/v2"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bansalbhunesh/Datastory/backend/internal/logging"
)

const headerRequestID = "X-Request-ID"

// reasonable upper bound for an incoming X-Request-ID we trust into logs.
const maxIncomingRequestID = 128

// RequestID attaches a request id to ctx + response header.
// Uses incoming X-Request-ID if present and sane, otherwise generates one.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader(headerRequestID))
		if !validRequestID(id) {
			id = newID()
		}
		c.Writer.Header().Set(headerRequestID, id)
		ctx := logging.WithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// validRequestID rejects empty / oversized / control-char IDs to keep logs clean.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxIncomingRequestID {
		return false
	}
	for _, r := range id {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// newID returns 16 hex chars. Falls back to a math/rand source if the OS RNG
// is unavailable so requests never get an all-zero ID that collides in logs.
func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		var seed [32]byte
		binary.BigEndian.PutUint64(seed[:8], uint64(time.Now().UnixNano()))
		r := mrand.NewChaCha8(seed)
		_, _ = r.Read(b)
	}
	return hex.EncodeToString(b)
}
