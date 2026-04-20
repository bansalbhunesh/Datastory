package openmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
)

type testCaseResult struct {
	Timestamp      int64  `json:"timestamp"`
	TestCaseStatus string `json:"testCaseStatus"`
	Result         string `json:"result"`
	FailedRows     any    `json:"failedRows,omitempty"`
}

type testCase struct {
	Name               string          `json:"name"`
	FullyQualifiedName string          `json:"fullyQualifiedName"`
	Description        string          `json:"description"`
	TestCaseResult     json.RawMessage `json:"testCaseResult"`
}

// ListFailedTests returns failing / aborted tests for a table FQN.
func (c *Client) ListFailedTests(ctx context.Context, tableFQN string, limit int) ([]domain.FailedTest, error) {
	tableFQN = strings.TrimSpace(tableFQN)
	if tableFQN == "" {
		return nil, fmt.Errorf("table fqn required")
	}
	if limit <= 0 {
		limit = 50
	}
	vals := url.Values{}
	vals.Set("entityLink", fmt.Sprintf("<#E::table::%s>", tableFQN))
	vals.Set("fields", "testCaseResult,testDefinition,testSuite")
	vals.Set("limit", fmt.Sprintf("%d", limit))

	var wrap struct {
		Data []testCase `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/dataQuality/testCases", vals, nil, &wrap); err != nil {
		return nil, err
	}

	out := make([]domain.FailedTest, 0, len(wrap.Data))
	for _, tc := range wrap.Data {
		status, msg, ts := latestResult(tc.TestCaseResult)
		if !isBadStatus(status) {
			continue
		}
		out = append(out, domain.FailedTest{
			Name:        tc.Name,
			FQN:         tc.FullyQualifiedName,
			Status:      status,
			Result:      msg,
			Description: tc.Description,
			UpdatedAt:   ts,
		})
	}
	return out, nil
}

func isBadStatus(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "failed", "aborted", "error":
		return true
	}
	return false
}

func latestResult(raw json.RawMessage) (status, msg string, ts int64) {
	if len(raw) == 0 {
		return "", "", 0
	}
	// Object form
	var obj testCaseResult
	if err := json.Unmarshal(raw, &obj); err == nil && obj.TestCaseStatus != "" {
		return obj.TestCaseStatus, pickMsg(obj), obj.Timestamp
	}
	// Array (history) form — pick last.
	var arr []testCaseResult
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		last := arr[len(arr)-1]
		return last.TestCaseStatus, pickMsg(last), last.Timestamp
	}
	return "", "", 0
}

func pickMsg(r testCaseResult) string {
	if strings.TrimSpace(r.Result) != "" {
		return r.Result
	}
	if r.FailedRows != nil {
		b, _ := json.Marshal(r.FailedRows)
		if len(b) > 0 && string(b) != "null" {
			return string(b)
		}
	}
	return ""
}
