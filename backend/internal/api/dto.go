package api

import "github.com/bansalbhunesh/Datastory/backend/internal/domain"

// GenerateReportRequest mirrors the POST /api/generate-report body.
// Contract is backward-compatible with the previous frontend.
type GenerateReportRequest struct {
	Query    string `json:"query"`
	TableFQN string `json:"tableFQN"`
}

// GenerateReportResponse is the API-level view of an incident report.
// Field set is a superset of the original response; old consumers keep working.
type GenerateReportResponse struct {
	TableFQN    string                `json:"tableFQN"`
	Markdown    string                `json:"markdown"`
	Severity    domain.Severity       `json:"severity"`
	Summary     string                `json:"summary"`
	RootCauses  []string              `json:"rootCauses"`
	Impacts     []string              `json:"impacts"`
	Remediation []string              `json:"remediation"`
	Lineage     domain.LineageSummary `json:"lineage"`
	FailedTests []domain.FailedTest   `json:"failedTests"`
	Source      string                `json:"source"`
	Warnings    []string              `json:"warnings,omitempty"`
}

func toResponse(r *domain.IncidentReport) GenerateReportResponse {
	return GenerateReportResponse{
		TableFQN:    r.TableFQN,
		Markdown:    r.Markdown,
		Severity:    r.Severity,
		Summary:     r.Summary,
		RootCauses:  r.RootCauses,
		Impacts:     r.Impacts,
		Remediation: r.Remediation,
		Lineage:     r.Lineage,
		FailedTests: r.FailedTests,
		Source:      r.Source,
		Warnings:    r.Warnings,
	}
}
