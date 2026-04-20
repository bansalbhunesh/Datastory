package services

import (
	"strings"
	"testing"

	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
)

func TestClassifySeverity(t *testing.T) {
	cases := []struct {
		name       string
		failed     int
		downstream int
		want       domain.Severity
	}{
		{"no fails, no downstream", 0, 0, domain.SeverityLow},
		{"one fail", 1, 0, domain.SeverityMedium},
		{"two fails", 2, 0, domain.SeverityHigh},
		{"one fail + 3 downstream", 1, 3, domain.SeverityHigh},
		{"zero fails + 2 downstream", 0, 2, domain.SeverityMedium},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failed := make([]domain.FailedTest, tc.failed)
			got := ClassifySeverity(failed, tc.downstream)
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestBuildDeterministicReport_MarkdownHasAllSections(t *testing.T) {
	r := BuildDeterministicReport("svc.db.public.orders",
		domain.LineageSummary{
			Focal:      "svc.db.public.orders",
			Upstream:   []string{"svc.db.public.raw_orders"},
			Downstream: []string{"svc.db.public.orders_daily"},
		},
		[]domain.FailedTest{{Name: "not_null_id", Status: "Failed", Result: "5 null ids"}},
	)
	for _, h := range []string{"## Incident summary", "## Root cause analysis", "## Impact assessment", "## Severity", "## Recommended remediation"} {
		if !strings.Contains(r.Markdown, h) {
			t.Errorf("missing section %q in markdown:\n%s", h, r.Markdown)
		}
	}
	if r.Source != "deterministic" {
		t.Errorf("source = %s, want deterministic", r.Source)
	}
}

func TestLooksLikeValidReport(t *testing.T) {
	good := `## Incident summary
text
## Root cause analysis
- x
## Impact assessment
- y
## Severity
HIGH — justification
## Recommended remediation
- z`
	if !looksLikeValidReport(good, domain.SeverityHigh) {
		t.Errorf("expected valid")
	}
	// Severity mismatch: deterministic says HIGH but LLM wrote LOW — must be rejected.
	bad := strings.Replace(good, "HIGH", "LOW", 1)
	if looksLikeValidReport(bad, domain.SeverityHigh) {
		t.Errorf("expected invalid on severity mismatch")
	}
	// Missing section
	missing := strings.Replace(good, "## Impact assessment", "## Impacts", 1)
	if looksLikeValidReport(missing, domain.SeverityHigh) {
		t.Errorf("expected invalid on missing section")
	}
}
