package services

import (
	"fmt"
	"strings"

	"github.com/bansalbhunesh/Datastory/backend/models"
)

// DraftIncidentReportMarkdown builds a structured incident report without an LLM.
func DraftIncidentReportMarkdown(tableFQN string, lineage models.LineageSummary, failed []models.FailedTestSummary) string {
	var b strings.Builder
	b.WriteString("## Incident summary\n\n")
	b.WriteString(fmt.Sprintf("Automated incident draft for table **`%s`** based on OpenMetadata lineage and data quality signals.\n\n", tableFQN))

	b.WriteString("## Root cause analysis\n\n")
	if len(failed) > 0 {
		b.WriteString("The following data quality executions are in a failing state (best-effort parsing from OpenMetadata):\n\n")
		for _, t := range failed {
			line := fmt.Sprintf("- **%s** — _%s_", t.Name, t.Status)
			if strings.TrimSpace(t.Result) != "" {
				line += fmt.Sprintf(": `%s`", truncateDraft(t.Result, 240))
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	} else {
		b.WriteString("- No failing tests were returned for this table (tests may be missing, passing, or not permitted by token scopes).\n\n")
	}

	if len(lineage.Upstream) > 0 {
		b.WriteString("Upstream context (lineage):\n\n")
		for _, u := range lineage.Upstream {
			b.WriteString(fmt.Sprintf("- `%s`\n", u))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Impact assessment\n\n")
	if len(lineage.Downstream) == 0 {
		b.WriteString("- Downstream impact is unknown from the current lineage graph (no downstream nodes returned).\n\n")
	} else {
		b.WriteString("Downstream assets that may be impacted:\n\n")
		for _, d := range lineage.Downstream {
			b.WriteString(fmt.Sprintf("- `%s`\n", d))
		}
		b.WriteString("\n")
	}

	sev := draftSeverity(len(failed), len(lineage.Downstream))
	b.WriteString("## Severity\n\n")
	b.WriteString(fmt.Sprintf("**%s** — derived from failing tests and downstream fan-out (heuristic).\n\n", sev))

	b.WriteString("## Recommended remediation\n\n")
	b.WriteString("- Confirm failing test definitions and rerun executions after fixes.\n")
	b.WriteString("- Validate upstream transformations feeding this table.\n")
	b.WriteString("- Notify downstream owners listed in lineage.\n")
	b.WriteString("- Add or tighten tests around the failing columns and schema contracts.\n")

	return b.String()
}

func draftSeverity(fails int, downstream int) string {
	if fails >= 2 || (fails >= 1 && downstream >= 3) {
		return "HIGH"
	}
	if fails == 1 || downstream >= 2 {
		return "MEDIUM"
	}
	return "LOW"
}

func truncateDraft(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
