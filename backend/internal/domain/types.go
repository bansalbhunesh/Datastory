package domain

type Severity string

const (
	SeverityLow    Severity = "LOW"
	SeverityMedium Severity = "MEDIUM"
	SeverityHigh   Severity = "HIGH"
)

type TableHit struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	FullyQualifiedName string  `json:"fullyQualifiedName"`
	Score              float64 `json:"score,omitempty"`
}

type LineageSummary struct {
	Focal         string   `json:"focal"`
	Upstream      []string `json:"upstream"`
	Downstream    []string `json:"downstream"`
	UpstreamRaw   int      `json:"upstreamRaw"`
	DownstreamRaw int      `json:"downstreamRaw"`
}

type FailedTest struct {
	Name        string `json:"name"`
	FQN         string `json:"fqn,omitempty"`
	Status      string `json:"status"`
	Result      string `json:"result,omitempty"`
	Description string `json:"description,omitempty"`
	UpdatedAt   int64  `json:"updatedAt,omitempty"`
}

// IncidentReport is the canonical structured output produced by services.
type IncidentReport struct {
	TableFQN    string         `json:"tableFQN"`
	Severity    Severity       `json:"severity"`
	Summary     string         `json:"summary"`
	RootCauses  []string       `json:"rootCauses"`
	Impacts     []string       `json:"impacts"`
	Remediation []string       `json:"remediation"`
	Markdown    string         `json:"markdown"`
	Lineage     LineageSummary `json:"lineage"`
	FailedTests []FailedTest   `json:"failedTests"`
	Source      string         `json:"source"` // "deterministic" | "llm"
	Warnings    []string       `json:"warnings,omitempty"`
}

type IncidentLogEntry struct {
	ID        string   `json:"id"`
	CreatedAt int64    `json:"createdAt"`
	TableFQN  string   `json:"tableFQN"`
	Severity  Severity `json:"severity"`
	Source    string   `json:"source"`
}
