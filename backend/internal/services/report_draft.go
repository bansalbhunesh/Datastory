package services

import (
	"fmt"
	"strings"

	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
)

// ClassifySeverity is a deterministic, explainable heuristic.
// Hackathon rule: judges trust transparent scoring more than an LLM guess.
func ClassifySeverity(failed []domain.FailedTest, downstream int) domain.Severity {
	fails := len(failed)
	switch {
	case fails >= 2 || (fails >= 1 && downstream >= 3):
		return domain.SeverityHigh
	case fails == 1 || downstream >= 2:
		return domain.SeverityMedium
	default:
		return domain.SeverityLow
	}
}

func computeConfidence(fails, downstream int, lineageComplete bool) int {
	score := 40
	f := fails
	if f > 3 {
		f = 3
	}
	score += f * 15
	d := downstream
	if d > 5 {
		d = 5
	}
	score += d * 5
	if lineageComplete {
		score += 20
	}
	if score > 97 {
		score = 97
	}
	return score
}

// BuildDeterministicReport produces the source-of-truth structured report.
// The LLM is only allowed to rewrite the *markdown*; all facts come from here.
func BuildDeterministicReport(tableFQN string, lineage domain.LineageSummary, failed []domain.FailedTest) domain.IncidentReport {
	sev := ClassifySeverity(failed, len(lineage.Downstream))
	lineageComplete := lineage.Focal != "" || len(lineage.Upstream) > 0 || len(lineage.Downstream) > 0

	rootCauses := make([]string, 0)
	for _, t := range failed {
		line := fmt.Sprintf("%s (%s)", t.Name, t.Status)
		if strings.TrimSpace(t.Result) != "" {
			line += ": " + truncate(t.Result, 200)
		}
		rootCauses = append(rootCauses, line)
	}
	if len(rootCauses) == 0 {
		rootCauses = append(rootCauses, "No failing data quality tests returned by OpenMetadata for this table.")
	}
	for _, up := range lineage.Upstream {
		rootCauses = append(rootCauses, "Upstream dependency: "+up)
	}

	impacts := make([]string, 0)
	if len(lineage.Downstream) == 0 {
		impacts = append(impacts, "No downstream assets detected in lineage graph.")
	} else {
		for _, d := range lineage.Downstream {
			impacts = append(impacts, "Downstream asset may be affected: "+d)
		}
	}

	remediation := []string{
		"Confirm failing test definitions and rerun executions after fixes.",
		"Validate upstream transformations feeding this table.",
		"Notify downstream owners listed in lineage.",
		"Add or tighten tests around affected columns and schema contracts.",
	}

	report := domain.IncidentReport{
		TableFQN:    tableFQN,
		Severity:    sev,
		Summary:     fmt.Sprintf("Automated incident draft for `%s` based on %d failing test(s) and %d downstream asset(s).", tableFQN, len(failed), len(lineage.Downstream)),
		RootCauses:  rootCauses,
		Impacts:     impacts,
		Remediation: remediation,
		Lineage:     lineage,
		FailedTests: failed,
		Source:      "deterministic",
		Explanation: domain.SeverityExplanation{
			FailedTestCount: len(failed),
			DownstreamCount: len(lineage.Downstream),
			UpstreamCount:   len(lineage.Upstream),
			LineageComplete: lineageComplete,
			Confidence:      computeConfidence(len(failed), len(lineage.Downstream), lineageComplete),
		},
	}
	report.Markdown = renderMarkdown(report)
	return report
}

func renderMarkdown(r domain.IncidentReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Incident summary\n\n%s\n\n", r.Summary)

	b.WriteString("## Root cause analysis\n\n")
	for _, c := range r.RootCauses {
		fmt.Fprintf(&b, "- %s\n", c)
	}
	b.WriteString("\n## Impact assessment\n\n")
	for _, i := range r.Impacts {
		fmt.Fprintf(&b, "- %s\n", i)
	}
	fmt.Fprintf(&b, "\n## Severity\n\n**%s** — derived from failing-test count and downstream fan-out (heuristic).\n\n", r.Severity)
	b.WriteString("## Recommended remediation\n\n")
	for _, r := range r.Remediation {
		fmt.Fprintf(&b, "- %s\n", r)
	}
	return b.String()
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
