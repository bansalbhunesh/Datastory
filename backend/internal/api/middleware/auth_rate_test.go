package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIKey_RejectsMissingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(APIKey("secret"))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRateLimit_BlocksBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(1, 1))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req1 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec1 := httptest.NewRecorder()
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	r.ServeHTTP(rec2, req2)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected first request to pass, got %d", rec1.Code)
	}
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request limited, got %d", rec2.Code)
	}
}
