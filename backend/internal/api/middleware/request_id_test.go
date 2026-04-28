package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	got := rec.Header().Get(headerRequestID)
	if got == "" {
		t.Fatal("expected generated request id")
	}
	if len(got) < 8 {
		t.Fatalf("request id too short: %q", got)
	}
}

func TestRequestID_PassesThroughIncoming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(headerRequestID, "trace-abc-123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Header().Get(headerRequestID); got != "trace-abc-123" {
		t.Fatalf("expected pass-through id, got %q", got)
	}
}

func TestRequestID_RejectsInvalidIncoming(t *testing.T) {
	cases := []string{"", strings.Repeat("a", maxIncomingRequestID+1), "bad\x00id", "ctrl\x07char"}
	gin.SetMode(gin.TestMode)
	for _, in := range cases {
		r := gin.New()
		r.Use(RequestID())
		r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		if in != "" {
			req.Header.Set(headerRequestID, in)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		got := rec.Header().Get(headerRequestID)
		if got == in && in != "" {
			t.Fatalf("invalid id %q was passed through verbatim", in)
		}
		if got == "" {
			t.Fatalf("expected generated id when incoming is invalid")
		}
	}
}

func TestNewID_NotAllZeros(t *testing.T) {
	id := newID()
	if id == "0000000000000000" {
		t.Fatal("newID must never return all-zero hex")
	}
	if len(id) != 16 {
		t.Fatalf("expected 16 hex chars, got %d (%q)", len(id), id)
	}
}
