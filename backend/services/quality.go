package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bansalbhunesh/Datastory/backend/models"
)

// TableEntityLink builds the OpenMetadata entityLink token for a table FQN.
// Docs: https://docs.open-metadata.org/v1.12.x/api-reference/data-quality/test-cases
func TableEntityLink(tableFQN string) string {
	tableFQN = strings.TrimSpace(tableFQN)
	return fmt.Sprintf("<#E::table::%s>", tableFQN)
}

type testCasesResponse struct {
	Data []testCase `json:"data"`
}

type testCase struct {
	Name                 string          `json:"name"`
	FullyQualifiedName   string          `json:"fullyQualifiedName"`
	Description          string          `json:"description"`
	TestCaseResult       json.RawMessage `json:"testCaseResult"`
}

// ListFailedTests returns recent failing / aborted executions if present in payload.
func (c *OpenMetadataClient) ListFailedTests(ctx context.Context, tableFQN string, limit int) ([]models.FailedTestSummary, error) {
	tableFQN = strings.TrimSpace(tableFQN)
	if tableFQN == "" {
		return nil, fmt.Errorf("table fqn required")
	}
	if limit <= 0 {
		limit = 50
	}

	entityLink := TableEntityLink(tableFQN)
	vals := url.Values{}
	vals.Set("entityLink", entityLink)
	vals.Set("fields", "testCaseResult,testDefinition,testSuite")
	vals.Set("limit", fmt.Sprintf("%d", limit))

	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/dataQuality/testCases", vals, nil, &raw); err != nil {
		return nil, err
	}

	var out testCasesResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	// Some OM versions return a bare array.
	if len(out.Data) == 0 {
		var asArr []testCase
		if err := json.Unmarshal(raw, &asArr); err == nil && len(asArr) > 0 {
			out.Data = asArr
		}
	}

	failed := make([]models.FailedTestSummary, 0)
	for _, tc := range out.Data {
		status, resultSnippet := extractLatestResult(tc.TestCaseResult)
		if !isBadStatus(status) {
			continue
		}
		failed = append(failed, models.FailedTestSummary{
			Name:        tc.Name,
			FQN:         tc.FullyQualifiedName,
			Status:      status,
			Result:      resultSnippet,
			Description: tc.Description,
		})
	}
	return failed, nil
}

func isBadStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "aborted", "error":
		return true
	default:
		return false
	}
}

// extractLatestResult parses testCaseResult; shape varies by OM version.
func extractLatestResult(raw json.RawMessage) (status string, message string) {
	if len(raw) == 0 {
		return "", ""
	}

	// Common shapes:
	// 1) { "timestamp": 1, "testCaseStatus": "Failed", "result": "..." , ... }
	// 2) [ { ... } ] (history)
	var asObj map[string]any
	if err := json.Unmarshal(raw, &asObj); err == nil && asObj != nil {
		return stringifyStatus(asObj), stringifyResult(asObj)
	}

	var asArr []map[string]any
	if err := json.Unmarshal(raw, &asArr); err == nil && len(asArr) > 0 {
		last := asArr[len(asArr)-1]
		return stringifyStatus(last), stringifyResult(last)
	}

	return "", strings.TrimSpace(string(raw))
}

func stringifyStatus(m map[string]any) string {
	for _, k := range []string{"testCaseStatus", "status", "caseStatus"} {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func stringifyResult(m map[string]any) string {
	for _, k := range []string{"result", "message", "failedRows", "errorStack"} {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if strings.TrimSpace(t) != "" {
					return t
				}
			default:
				b, _ := json.Marshal(t)
				if len(b) > 0 && string(b) != "null" {
					return string(b)
				}
			}
		}
	}
	return ""
}
