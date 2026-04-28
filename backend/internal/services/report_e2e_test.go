package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bansalbhunesh/Datastory/backend/internal/clients/openmetadata"
	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
	"github.com/bansalbhunesh/Datastory/backend/internal/logging"
)

type stubLLM struct {
	enabled  bool
	response string
	err      error
	calls    int32
}

func (s *stubLLM) Enabled() bool { return s.enabled }
func (s *stubLLM) Rewrite(ctx context.Context, system, user string, _ int) (string, error) {
	atomic.AddInt32(&s.calls, 1)
	if s.err != nil {
		return "", s.err
	}
	return s.response, nil
}

// fakeOMServer responds with a fixed FQN, lineage, and quality test set
// keyed off the search query.
func fakeOMServer(t *testing.T, fqn string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/search/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":{"hits":[
			{"_score":50.0,"_source":{"id":"1","name":"x","fullyQualifiedName":"` + fqn + `"}}
		]}}`))
	})
	mux.HandleFunc("/api/v1/lineage/table/name/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"entity": {"fullyQualifiedName":"` + fqn + `"},
			"upstreamEdges":[{"fromEntity":{"fullyQualifiedName":"u"},"toEntity":{"fullyQualifiedName":"` + fqn + `"}}],
			"downstreamEdges":[
				{"fromEntity":{"fullyQualifiedName":"` + fqn + `"},"toEntity":{"fullyQualifiedName":"d1"}},
				{"fromEntity":{"fullyQualifiedName":"` + fqn + `"},"toEntity":{"fullyQualifiedName":"d2"}}
			]
		}`))
	})
	mux.HandleFunc("/api/v1/dataQuality/testCases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"name":"not_null","testCaseResult":{"testCaseStatus":"Failed","result":"5 null"}}
		]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// Generate returns a deterministic report when the LLM is disabled, with all
// expected sections, severity, and warnings about the disabled LLM.
func TestReportService_GenerateDeterministic(t *testing.T) {
	srv := fakeOMServer(t, "svc.db.s.orders")
	om := openmetadata.New(openmetadata.Options{BaseURL: srv.URL, Token: "tok", Timeout: 3 * time.Second})

	store, err := NewSQLiteIncidentStore(filepath.Join(t.TempDir(), "i.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	llm := &stubLLM{enabled: false}
	svc := NewReportService(om, llm, logging.New("error"), 5*time.Second, store)

	rep, err := svc.Generate(context.Background(), GenerateInput{Query: "orders"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if rep.TableFQN != "svc.db.s.orders" {
		t.Fatalf("FQN mismatch: %s", rep.TableFQN)
	}
	if rep.Source != "deterministic" {
		t.Fatalf("expected deterministic source, got %s", rep.Source)
	}
	// One failing test + 2 downstream → MEDIUM per ClassifySeverity rules.
	if rep.Severity != domain.SeverityMedium {
		t.Fatalf("expected MEDIUM, got %s", rep.Severity)
	}
	for _, h := range []string{"## Incident summary", "## Severity"} {
		if !strings.Contains(rep.Markdown, h) {
			t.Fatalf("missing header %s", h)
		}
	}
	if atomic.LoadInt32(&llm.calls) != 0 {
		t.Fatal("LLM should not be called when disabled")
	}
}

// When the LLM is enabled and returns valid markdown, source flips to "claude".
func TestReportService_GenerateWithLLM(t *testing.T) {
	srv := fakeOMServer(t, "svc.db.s.orders")
	om := openmetadata.New(openmetadata.Options{BaseURL: srv.URL, Token: "tok", Timeout: 3 * time.Second})

	store, _ := NewSQLiteIncidentStore(filepath.Join(t.TempDir(), "i.db"))
	llm := &stubLLM{
		enabled: true,
		response: `## Incident summary
A high-severity incident occurred.

## Root cause analysis
- bullet

## Impact assessment
- bullet

## Severity
MEDIUM — this is bad.

## Recommended remediation
- fix
`,
	}
	svc := NewReportService(om, llm, logging.New("error"), 5*time.Second, store)
	rep, err := svc.Generate(context.Background(), GenerateInput{Query: "orders"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if rep.Source != "claude" {
		t.Fatalf("expected claude source, got %s", rep.Source)
	}
	if !strings.Contains(rep.Markdown, "A high-severity incident") {
		t.Fatalf("expected LLM markdown, got %q", rep.Markdown)
	}
}

// LLM error → fall back to deterministic + record warning.
func TestReportService_LLMFailureFallsBack(t *testing.T) {
	srv := fakeOMServer(t, "svc.db.s.orders")
	om := openmetadata.New(openmetadata.Options{BaseURL: srv.URL, Token: "tok", Timeout: 3 * time.Second})

	store, _ := NewSQLiteIncidentStore(filepath.Join(t.TempDir(), "i.db"))
	llm := &stubLLM{enabled: true, err: context.DeadlineExceeded}
	svc := NewReportService(om, llm, logging.New("error"), 5*time.Second, store)
	rep, err := svc.Generate(context.Background(), GenerateInput{Query: "orders"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if rep.Source != "deterministic" {
		t.Fatalf("expected deterministic fallback, got %s", rep.Source)
	}
	found := false
	for _, w := range rep.Warnings {
		if strings.Contains(w, "AI enhancement unavailable") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected warning, got %v", rep.Warnings)
	}
}

// Two concurrent requests for the same FQN must coalesce via SingleFlight,
// hitting the LLM at most once.
func TestReportService_SingleFlightCoalesces(t *testing.T) {
	srv := fakeOMServer(t, "svc.db.s.orders")
	om := openmetadata.New(openmetadata.Options{BaseURL: srv.URL, Token: "tok", Timeout: 3 * time.Second})

	store, _ := NewSQLiteIncidentStore(filepath.Join(t.TempDir(), "i.db"))
	llm := &stubLLM{
		enabled: true,
		response: `## Incident summary
ok
## Root cause analysis
- ok
## Impact assessment
- ok
## Severity
MEDIUM — ok
## Recommended remediation
- ok
`,
	}
	svc := NewReportService(om, llm, logging.New("error"), 5*time.Second, store)

	done := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func() {
			_, err := svc.Generate(context.Background(), GenerateInput{TableFQN: "svc.db.s.orders"})
			done <- err
		}()
	}
	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Fatalf("call err: %v", err)
		}
	}
	if got := atomic.LoadInt32(&llm.calls); got > 1 {
		t.Fatalf("expected at most 1 LLM call, got %d", got)
	}
}
