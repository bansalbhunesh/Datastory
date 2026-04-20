package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bansalbhunesh/Datastory/backend/internal/clients/llm"
	"github.com/bansalbhunesh/Datastory/backend/internal/clients/openmetadata"
	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
	"github.com/bansalbhunesh/Datastory/backend/internal/errs"
	"github.com/bansalbhunesh/Datastory/backend/internal/logging"
	"github.com/bansalbhunesh/Datastory/backend/internal/syncx"
)

type ReportService struct {
	om    *openmetadata.Client
	llm   llm.Client
	log   *slog.Logger
	cache *ttlCache
}

func NewReportService(om *openmetadata.Client, l llm.Client, log *slog.Logger, cacheTTL time.Duration) *ReportService {
	return &ReportService{om: om, llm: l, log: log, cache: newTTLCache(cacheTTL)}
}

type GenerateInput struct {
	Query    string
	TableFQN string
}

// Generate produces an IncidentReport for a table.
// Order: resolve FQN → parallel(lineage, quality) → deterministic draft → optional LLM rewrite.
func (s *ReportService) Generate(ctx context.Context, in GenerateInput) (*domain.IncidentReport, error) {
	log := logging.FromCtx(ctx, s.log)

	fqn, err := s.resolveFQN(ctx, in)
	if err != nil {
		return nil, err
	}

	var (
		lineage  domain.LineageSummary
		failed   []domain.FailedTest
		warnings = []string{}
		mu       sync.Mutex
	)

	g, gctx := syncx.WithContext(ctx)
	g.Go(func() error {
		raw, err := s.lineage(gctx, fqn)
		if err != nil {
			return errs.Upstream("lineage fetch", err)
		}
		sum, err := SummarizeLineage(raw)
		if err != nil {
			return errs.Upstream("lineage parse", err)
		}
		mu.Lock()
		lineage = sum
		mu.Unlock()
		return nil
	})
	g.Go(func() error {
		tests, err := s.om.ListFailedTests(gctx, fqn, 50)
		if err != nil {
			// Non-fatal: tests are optional.
			log.Warn("list failed tests skipped", "error", err.Error())
			mu.Lock()
			warnings = append(warnings, "data quality tests unavailable: "+err.Error())
			mu.Unlock()
			return nil
		}
		mu.Lock()
		failed = tests
		mu.Unlock()
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	report := BuildDeterministicReport(fqn, lineage, failed)
	report.Warnings = warnings

	if s.llm != nil && s.llm.Enabled() {
		md, err := s.rewriteMarkdown(ctx, report)
		if err != nil {
			log.Warn("llm rewrite failed, using deterministic draft", "error", err.Error())
			report.Warnings = append(report.Warnings, "LLM rewrite failed: "+err.Error())
		} else {
			report.Markdown = md
			report.Source = "claude"
		}
	} else {
		report.Warnings = append(report.Warnings, "CLAUDE_API_KEY not set; returning deterministic draft (facts are from OpenMetadata).")
	}
	return &report, nil
}

func (s *ReportService) resolveFQN(ctx context.Context, in GenerateInput) (string, error) {
	if fqn := strings.TrimSpace(in.TableFQN); fqn != "" {
		return fqn, nil
	}
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return "", errs.BadRequest("missing tableFQN and query")
	}
	if v, ok := s.cache.get("fqn:" + q); ok {
		return v.(string), nil
	}
	hits, err := s.om.SearchTables(ctx, q, 5)
	if err != nil {
		return "", err
	}
	if len(hits) == 0 {
		return "", errs.BadRequest(fmt.Sprintf("no table hits for %q", q))
	}
	fqn := hits[0].FullyQualifiedName
	s.cache.set("fqn:"+q, fqn)
	return fqn, nil
}

func (s *ReportService) lineage(ctx context.Context, fqn string) (json.RawMessage, error) {
	key := "lineage:" + fqn
	if v, ok := s.cache.get(key); ok {
		return v.(json.RawMessage), nil
	}
	raw, err := s.om.LineageJSON(ctx, fqn, 3, 3)
	if err != nil {
		return nil, err
	}
	s.cache.set(key, raw)
	return raw, nil
}

func (s *ReportService) SearchTables(ctx context.Context, q string) ([]domain.TableHit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []domain.TableHit{}, nil
	}
	return s.om.SearchTables(ctx, q, 15)
}

// rewriteMarkdown asks the LLM to rewrite the deterministic markdown in incident-report style.
// We include facts explicitly and instruct the model not to invent anything.
func (s *ReportService) rewriteMarkdown(ctx context.Context, r domain.IncidentReport) (string, error) {
	system := "You are a senior data engineer writing an internal data incident report. Use ONLY the facts provided. If something is unknown, say so explicitly. Output valid Markdown. Do NOT invent tables, owners, or causes."

	var b strings.Builder
	b.WriteString("Facts (authoritative):\n")
	fmt.Fprintf(&b, "- Table: %s\n", r.TableFQN)
	fmt.Fprintf(&b, "- Computed severity: %s\n", r.Severity)
	fmt.Fprintf(&b, "- Upstream: %s\n", join(r.Lineage.Upstream, "none"))
	fmt.Fprintf(&b, "- Downstream: %s\n", join(r.Lineage.Downstream, "none"))
	fmt.Fprintf(&b, "- Lineage edge counts: upstream=%d downstream=%d\n", r.Lineage.UpstreamRaw, r.Lineage.DownstreamRaw)
	if len(r.FailedTests) == 0 {
		b.WriteString("- Failing tests: none returned by OpenMetadata\n")
	} else {
		b.WriteString("- Failing tests:\n")
		for _, t := range r.FailedTests {
			fmt.Fprintf(&b, "  - %s [%s]: %s\n", t.Name, t.Status, truncate(t.Result, 300))
		}
	}
	b.WriteString("\nWrite an incident report with these exact H2 sections, in this order:\n")
	b.WriteString("1. `## Incident summary` (2-4 sentences)\n")
	b.WriteString("2. `## Root cause analysis` (bullets, tie to lineage + tests)\n")
	b.WriteString("3. `## Impact assessment` (bullets; downstream consumers)\n")
	b.WriteString("4. `## Severity` (MUST match the computed severity: " + string(r.Severity) + ")\n")
	b.WriteString("5. `## Recommended remediation` (bullets, concrete steps)\n")

	md, err := s.llm.Rewrite(ctx, system, b.String(), 2048)
	if err != nil {
		return "", err
	}
	if !looksLikeValidReport(md, r.Severity) {
		return "", errs.Upstream("llm output failed schema check", nil)
	}
	return md, nil
}

func looksLikeValidReport(md string, sev domain.Severity) bool {
	lower := strings.ToLower(md)
	required := []string{
		"## incident summary",
		"## root cause analysis",
		"## impact assessment",
		"## severity",
		"## recommended remediation",
	}
	for _, s := range required {
		if !strings.Contains(lower, s) {
			return false
		}
	}
	// Severity must be present literally (guards against LLM hallucinating a different level).
	return strings.Contains(md, string(sev))
}

func join(ss []string, empty string) string {
	if len(ss) == 0 {
		return empty
	}
	return strings.Join(ss, ", ")
}
