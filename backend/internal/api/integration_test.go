package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bansalbhunesh/Datastory/backend/internal/api"
	"github.com/bansalbhunesh/Datastory/backend/internal/clients/openmetadata"
	"github.com/bansalbhunesh/Datastory/backend/internal/logging"
	"github.com/bansalbhunesh/Datastory/backend/internal/services"
)

type stubLLM struct{ enabled bool }

func (stubLLM) Rewrite(ctx context.Context, system, user string, _ int) (string, error) {
	return "", nil
}
func (s stubLLM) Enabled() bool { return s.enabled }

func fakeOM(t *testing.T, fqn string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"test"}`))
	})
	mux.HandleFunc("/api/v1/search/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":{"hits":[
			{"_score":50,"_source":{"id":"1","name":"orders","fullyQualifiedName":"` + fqn + `"}}
		]}}`))
	})
	mux.HandleFunc("/api/v1/lineage/table/name/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"entity":{"fullyQualifiedName":"` + fqn + `"},
			"upstreamEdges":[{"fromEntity":{"fullyQualifiedName":"u"},"toEntity":{"fullyQualifiedName":"` + fqn + `"}}],
			"downstreamEdges":[{"fromEntity":{"fullyQualifiedName":"` + fqn + `"},"toEntity":{"fullyQualifiedName":"d1"}}]
		}`))
	})
	mux.HandleFunc("/api/v1/dataQuality/testCases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// Full HTTP-stack integration test: assembles the real router with real
// middleware and exercises all public API endpoints against a fake OM.
func TestEndToEnd_HTTPStack(t *testing.T) {
	omSrv := fakeOM(t, "svc.db.public.orders")

	om := openmetadata.New(openmetadata.Options{BaseURL: omSrv.URL, Token: "tok", Timeout: 5 * time.Second})
	store, err := services.NewSQLiteIncidentStore(filepath.Join(t.TempDir(), "i.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	llm := stubLLM{enabled: false}
	log := logging.New("error")

	reportSvc := services.NewReportService(om, llm, log, 5*time.Second, store)
	handlers := api.NewHandlers(reportSvc, om, llm, log, api.HandlersConfig{GenerateTimeout: 10 * time.Second})
	router := api.NewRouter(handlers, log, api.RouterConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
		MaxBodyBytes:   1 << 16,
		APIKey:         "test-key",
		RateLimitRPS:   100,
		RateLimitBurst: 100,
	})

	authHeader := func(req *http.Request) {
		req.Header.Set("X-API-Key", "test-key")
	}

	send := func(method, path string, body string) *httptest.ResponseRecorder {
		var r io.Reader
		if body != "" {
			r = bytes.NewBufferString(body)
		}
		req := httptest.NewRequest(method, path, r)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		authHeader(req)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("/healthz needs no auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d", rec.Code)
		}
	})

	t.Run("/api/health is public", func(t *testing.T) {
		// Health and /healthz are intentionally outside the API-key gate so
		// load balancers / liveness probes can hit them unauthenticated.
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 unauthenticated, got %d", rec.Code)
		}
	})

	t.Run("/api/ready requires API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("missing-key: expected 401, got %d", rec.Code)
		}
	})

	t.Run("/api/ready returns shape", func(t *testing.T) {
		rec := send(http.MethodGet, "/api/ready", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if _, ok := body["openmetadata"]; !ok {
			t.Fatalf("missing openmetadata key: %v", body)
		}
	})

	t.Run("/api/search/tables returns hits", func(t *testing.T) {
		rec := send(http.MethodGet, "/api/search/tables?q=orders", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("/api/generate-report end-to-end", func(t *testing.T) {
		rec := send(http.MethodPost, "/api/generate-report", `{"query":"orders"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body=%s", rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got, _ := body["tableFQN"].(string); got != "svc.db.public.orders" {
			t.Fatalf("tableFQN: %v", body["tableFQN"])
		}
		if got, _ := body["source"].(string); got != "deterministic" {
			t.Fatalf("source: %v", body["source"])
		}
		md, _ := body["markdown"].(string)
		if !strings.Contains(md, "## Incident summary") {
			t.Fatalf("markdown missing section: %q", md)
		}
		if rid := rec.Header().Get("X-Request-ID"); rid == "" {
			t.Fatal("missing X-Request-ID")
		}
	})

	t.Run("incidents history grew after generate", func(t *testing.T) {
		// Persist is async; allow up to 1s.
		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) {
			rec := send(http.MethodGet, "/api/incidents?tableFQN=svc.db.public.orders&limit=10", "")
			if rec.Code != http.StatusOK {
				t.Fatalf("got %d", rec.Code)
			}
			var body struct {
				Incidents []map[string]any `json:"incidents"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			if len(body.Incidents) > 0 {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatal("expected at least one incident persisted within 1s")
	})

	t.Run("oversized body rejected", func(t *testing.T) {
		big := `{"query":"` + strings.Repeat("x", 100*1024) + `"}`
		rec := send(http.MethodPost, "/api/generate-report", big)
		if rec.Code == http.StatusOK {
			t.Fatalf("expected non-200 for oversized body, got 200")
		}
	})

	t.Run("invalid JSON returns sanitized 400", func(t *testing.T) {
		rec := send(http.MethodPost, "/api/generate-report", `{not-json`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got %d", rec.Code)
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		msg, _ := body["error"].(string)
		if strings.Contains(msg, "looking for beginning") || strings.Contains(msg, "unmarshal") {
			t.Fatalf("error leaks parser internals: %q", msg)
		}
	})
}
