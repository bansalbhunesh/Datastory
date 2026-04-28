package openmetadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// 5xx must be retried up to maxAttempts; eventual 200 should succeed.
func TestDoJSON_RetriesOn5xx(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":{"hits":[]}}`))
	}))
	t.Cleanup(srv.Close)

	c := New(Options{BaseURL: srv.URL, Token: "tok", Timeout: 5 * time.Second})
	if _, err := c.SearchTables(context.Background(), "x", 1); err != nil {
		t.Fatalf("expected success after retries: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

// 401 + creds → drop token, log in, retry once.
func TestDoJSON_Reauthenticates(t *testing.T) {
	var loginHits int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users/login", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&loginHits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"new-token"}`))
	})
	mux.HandleFunc("/api/v1/search/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer new-token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"hits":{"hits":[]}}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := New(Options{BaseURL: srv.URL, Email: "a@b", Password: "p", Timeout: 5 * time.Second})
	c.setToken("stale")
	if _, err := c.SearchTables(context.Background(), "x", 1); err != nil {
		t.Fatalf("expected success after re-auth: %v", err)
	}
	if got := atomic.LoadInt32(&loginHits); got != 1 {
		t.Fatalf("expected exactly one login, got %d", got)
	}
}

// 4xx that isn't 401 should NOT retry.
func TestDoJSON_NoRetryOn4xx(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	c := New(Options{BaseURL: srv.URL, Token: "tok", Timeout: 5 * time.Second})
	if _, err := c.SearchTables(context.Background(), "x", 1); err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 1 attempt, got %d", got)
	}
}

// SearchTables success path: hits parsed into TableHit list with score.
func TestSearchTables_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "table_search_index") {
			t.Errorf("missing index in query: %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":{"hits":[
			{"_score":12.5,"_source":{"id":"1","name":"orders","fullyQualifiedName":"svc.db.s.orders"}},
			{"_score":10.0,"_source":{"id":"2","name":"o2","fullyQualifiedName":""}}
		]}}`))
	}))
	t.Cleanup(srv.Close)

	c := New(Options{BaseURL: srv.URL, Token: "tok"})
	hits, err := c.SearchTables(context.Background(), "ord", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected entries with FQN only, got %d", len(hits))
	}
	if hits[0].Score != 12.5 {
		t.Fatalf("score not propagated: %v", hits[0].Score)
	}
}

// ListFailedTests filters out non-failing statuses and parses both object/array result forms.
func TestListFailedTests_FiltersAndParses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"name":"good","testCaseResult":{"testCaseStatus":"Success"}},
			{"name":"bad1","testCaseResult":{"testCaseStatus":"Failed","result":"r1"}},
			{"name":"bad2","testCaseResult":[
				{"testCaseStatus":"Success","result":"old"},
				{"testCaseStatus":"Aborted","result":"latest"}
			]}
		]}`))
	}))
	t.Cleanup(srv.Close)

	c := New(Options{BaseURL: srv.URL, Token: "tok"})
	out, err := c.ListFailedTests(context.Background(), "svc.db.s.t", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 failing tests, got %d (%+v)", len(out), out)
	}
}

// Ping treats 401/403 as "reachable" since the server clearly answered.
func TestPing_ReachableEvenIfUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	c := New(Options{BaseURL: srv.URL})
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("expected nil from 401 ping, got %v", err)
	}
}
