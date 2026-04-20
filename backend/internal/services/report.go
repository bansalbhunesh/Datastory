package services

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
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
	om        *openmetadata.Client
	llm       llm.Client
	log       *slog.Logger
	cache     *ttlCache
	sf        syncx.SingleFlight
	incidents IncidentStore
}

func NewReportService(om *openmetadata.Client, l llm.Client, log *slog.Logger, cacheTTL time.Duration, incidents IncidentStore) *ReportService {
	return &ReportService{
		om:        om,
		llm:       l,
		log:       log,
		cache:     newTTLCache(cacheTTL, 1000),
		incidents: incidents,
	}
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
	cacheKey := "report:" + strings.ToLower(strings.TrimSpace(fqn))
	if v, ok := s.cache.get(cacheKey); ok {
		if cached, ok := v.(domain.IncidentReport); ok {
			r := cached
			return &r, nil
		}
	}
	v, err, _ := s.sf.Do(cacheKey, func() (any, error) {
		return s.generateUncached(ctx, fqn, log)
	})
	if err != nil {
		return nil, err
	}
	report := v.(domain.IncidentReport)
	s.cache.set(cacheKey, report)
	return &report, nil
}

func (s *ReportService) generateUncached(ctx context.Context, fqn string, log *slog.Logger) (domain.IncidentReport, error) {
	var zero domain.IncidentReport

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
		return zero, err
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
	if s.incidents != nil {
		_ = s.incidents.Append(domain.IncidentLogEntry{
			ID:        shortID(fqn + report.Markdown),
			CreatedAt: time.Now().Unix(),
			TableFQN:  report.TableFQN,
			Severity:  report.Severity,
			Source:    report.Source,
		})
	}
	return report, nil
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
	if len(hits) > 1 && hits[0].Score == hits[1].Score {
		return "", errs.BadRequest("ambiguous query; provide full tableFQN")
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

func (s *ReportService) Lineage(ctx context.Context, in GenerateInput) (*domain.LineageSummary, string, error) {
	fqn, err := s.resolveFQN(ctx, in)
	if err != nil {
		return nil, "", err
	}
	raw, err := s.lineage(ctx, fqn)
	if err != nil {
		return nil, "", errs.Upstream("lineage fetch", err)
	}
	sum, err := SummarizeLineage(raw)
	if err != nil {
		return nil, "", errs.Upstream("lineage parse", err)
	}
	return &sum, fqn, nil
}

func (s *ReportService) ListIncidents(tableFQN string, limit int) ([]domain.IncidentLogEntry, error) {
	if s.incidents == nil {
		return []domain.IncidentLogEntry{}, nil
	}
	return s.incidents.ListByTable(strings.TrimSpace(tableFQN), limit)
}

// rewriteMarkdown asks the LLM to rewrite the deterministic markdown in incident-report style.
// We include facts explicitly and instruct the model not to invent anything.
func (s *ReportService) rewriteMarkdown(ctx context.Context, r domain.IncidentReport) (string, error) {
	system := "You are a senior data engineer writing an internal data incident report. Use ONLY the facts provided. If something is unknown, say so explicitly. Output valid Markdown. Do NOT invent tables, owners, or causes."

	var b strings.Builder
	b.WriteString("Facts (authoritative):\n")
	fmt.Fprintf(&b, "- Table: %s\n", r.TableFQN)
	fmt.Fprintf(&b, "- Computed severity: %s\n", r.Severity)
	up := capSlice(r.Lineage.Upstream, 5)
	down := capSlice(r.Lineage.Downstream, 10)
	fmt.Fprintf(&b, "- Upstream: %s\n", join(up, "none"))
	fmt.Fprintf(&b, "- Downstream: %s\n", join(down, "none"))
	fmt.Fprintf(&b, "- Lineage edge counts: upstream=%d downstream=%d\n", r.Lineage.UpstreamRaw, r.Lineage.DownstreamRaw)
	failed := capFailedTests(r.FailedTests, 5)
	if len(failed) == 0 {
		b.WriteString("- Failing tests: none returned by OpenMetadata\n")
	} else {
		b.WriteString("- Failing tests:\n")
		for _, t := range failed {
			fmt.Fprintf(&b, "  - %s [%s]: %s\n", t.Name, t.Status, truncate(t.Result, 150))
		}
	}
	b.WriteString("\nWrite an incident report with these exact H2 sections, in this order:\n")
	b.WriteString("1. `## Incident summary` (2-4 sentences)\n")
	b.WriteString("2. `## Root cause analysis` (bullets, tie to lineage + tests)\n")
	b.WriteString("3. `## Impact assessment` (bullets; downstream consumers)\n")
	b.WriteString("4. `## Severity` (MUST match the computed severity: " + string(r.Severity) + ")\n")
	b.WriteString("5. `## Recommended remediation` (bullets, concrete steps)\n")
	b.WriteString("Return markdown only, no preamble or code fences.\n")

	md, err := s.llm.Rewrite(ctx, system, b.String(), 900)
	if err != nil {
		return "", err
	}
	if !looksLikeValidReport(md, r.Severity) {
		return "", errs.Upstream("llm output failed schema check", nil)
	}
	return md, nil
}

func looksLikeValidReport(md string, sev domain.Severity) bool {
	headers := extractH2(md)
	required := []string{
		"incident summary",
		"root cause analysis",
		"impact assessment",
		"severity",
		"recommended remediation",
	}
	for _, r := range required {
		if !slices.Contains(headers, r) {
			return false
		}
	}
	sevBody := sectionBody(md, "severity")
	if sevBody == "" {
		return false
	}
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(string(sev)) + `\b`)
	return re.FindStringIndex(strings.ToUpper(sevBody)) != nil
}

func join(ss []string, empty string) string {
	if len(ss) == 0 {
		return empty
	}
	return strings.Join(ss, ", ")
}

func extractH2(md string) []string {
	out := []string{}
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") {
			out = append(out, strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "## "))))
		}
	}
	return out
}

func sectionBody(md, name string) string {
	lines := strings.Split(md, "\n")
	target := "## " + strings.ToLower(strings.TrimSpace(name))
	start := -1
	for i, line := range lines {
		if strings.ToLower(strings.TrimSpace(line)) == target {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return ""
	}
	var b strings.Builder
	for i := start; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trim, "## ") {
			break
		}
		b.WriteString(lines[i])
		b.WriteString("\n")
	}
	return b.String()
}

func shortID(seed string) string {
	sum := sha1.Sum([]byte(seed + time.Now().UTC().Format(time.RFC3339Nano)))
	return fmt.Sprintf("%x", sum[:8])
}

func capSlice(in []string, max int) []string {
	if len(in) <= max {
		return in
	}
	return in[:max]
}

func capFailedTests(in []domain.FailedTest, max int) []domain.FailedTest {
	if len(in) <= max {
		return in
	}
	return in[:max]
}
