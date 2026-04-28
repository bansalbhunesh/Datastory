package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// rewriteTransport runs the HTTP roundtrip through a test transport that
// rewrites the (hardcoded) Anthropic URL to a local httptest server.
type rewriteTransport struct {
	url       string
	wrapped   http.RoundTripper
	gotHeader http.Header
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.gotHeader = req.Header.Clone()
	clone := req.Clone(req.Context())
	target, _ := http.NewRequest(req.Method, r.url, req.Body)
	target.Header = clone.Header
	return r.wrapped.RoundTrip(target)
}

func TestAnthropic_RewriteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-test" {
			t.Errorf("missing api key header: %v", r.Header)
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Errorf("missing anthropic-version header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"hello world"}]}`))
	}))
	t.Cleanup(srv.Close)

	a := NewAnthropic("sk-test", "model-x")
	rt := &rewriteTransport{url: srv.URL, wrapped: srv.Client().Transport}
	a.http = &http.Client{Transport: rt}

	out, err := a.Rewrite(context.Background(), "system", "user", 100)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out != "hello world" {
		t.Fatalf("got %q", out)
	}
}

func TestAnthropic_RewriteEmptyResponseRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[]}`))
	}))
	t.Cleanup(srv.Close)

	a := NewAnthropic("sk-test", "")
	rt := &rewriteTransport{url: srv.URL, wrapped: srv.Client().Transport}
	a.http = &http.Client{Transport: rt}

	if _, err := a.Rewrite(context.Background(), "s", "u", 100); err == nil {
		t.Fatal("expected error on empty content")
	}
}

func TestAnthropic_Rewrite5xxRetriesThenSucceeds(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
	}))
	t.Cleanup(srv.Close)

	a := NewAnthropic("sk-test", "")
	rt := &rewriteTransport{url: srv.URL, wrapped: srv.Client().Transport}
	a.http = &http.Client{Transport: rt}

	out, err := a.Rewrite(context.Background(), "s", "u", 100)
	if err != nil {
		t.Fatalf("expected eventual success: %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Fatalf("unexpected: %q", out)
	}
	if atomic.LoadInt32(&hits) != 2 {
		t.Fatalf("expected 2 hits (one retry), got %d", hits)
	}
}

func TestAnthropic_DisabledWhenNoKey(t *testing.T) {
	a := NewAnthropic("", "")
	if a.Enabled() {
		t.Fatal("expected disabled with empty key")
	}
	if _, err := a.Rewrite(context.Background(), "s", "u", 50); err == nil {
		t.Fatal("expected error when disabled")
	}
}

func TestAnthropic_4xxNotRetried(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad"}}`))
	}))
	t.Cleanup(srv.Close)

	a := NewAnthropic("sk-test", "")
	rt := &rewriteTransport{url: srv.URL, wrapped: srv.Client().Transport}
	a.http = &http.Client{Transport: rt}

	if _, err := a.Rewrite(context.Background(), "s", "u", 50); err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 1 attempt for 4xx, got %d", got)
	}
}
