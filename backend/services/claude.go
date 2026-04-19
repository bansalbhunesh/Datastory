package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bansalbhunesh/Datastory/backend/models"
)

type ClaudeClient struct {
	APIKey string
	Model  string
	Client *http.Client
}

func NewClaudeClient(apiKey, model string) *ClaudeClient {
	return &ClaudeClient{
		APIKey: strings.TrimSpace(apiKey),
		Model:  strings.TrimSpace(model),
		Client: &http.Client{Timeout: 120 * time.Second},
	}
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// GenerateIncidentReportMarkdown asks Claude to turn structured facts into an incident-style report.
func (c *ClaudeClient) GenerateIncidentReportMarkdown(ctx context.Context, tableFQN string, lineage models.LineageSummary, failed []models.FailedTestSummary) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("missing CLAUDE_API_KEY")
	}
	if c.Model == "" {
		c.Model = "claude-sonnet-4-20250514"
	}

	prompt := buildPrompt(tableFQN, lineage, failed)

	body, err := json.Marshal(claudeRequest{
		Model:     c.Model,
		MaxTokens: 2048,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("claude http %s: %s", resp.Status, string(b))
	}

	var cr claudeResponse
	if err := json.Unmarshal(b, &cr); err != nil {
		return "", fmt.Errorf("decode claude: %w", err)
	}
	if cr.Error != nil && strings.TrimSpace(cr.Error.Message) != "" {
		return "", fmt.Errorf("claude error: %s", cr.Error.Message)
	}
	if len(cr.Content) == 0 || strings.TrimSpace(cr.Content[0].Text) == "" {
		return "", fmt.Errorf("empty claude response")
	}
	return strings.TrimSpace(cr.Content[0].Text), nil
}

func buildPrompt(tableFQN string, lineage models.LineageSummary, failed []models.FailedTestSummary) string {
	var sb strings.Builder
	sb.WriteString("You are a senior data engineer writing an internal data incident report.\n")
	sb.WriteString("Use ONLY the facts below. If something is unknown, say so explicitly.\n\n")
	sb.WriteString("Facts:\n")
	sb.WriteString(fmt.Sprintf("- Primary table (focal entity): %s\n", tableFQN))
	sb.WriteString(fmt.Sprintf("- Upstream tables/assets (best-effort): %s\n", strings.Join(lineage.Upstream, ", ")))
	sb.WriteString(fmt.Sprintf("- Downstream tables/assets (best-effort): %s\n", strings.Join(lineage.Downstream, ", ")))
	sb.WriteString(fmt.Sprintf("- Lineage edge counts (raw): upstream=%d downstream=%d\n", lineage.UpstreamRaw, lineage.DownstreamRaw))

	if len(failed) == 0 {
		sb.WriteString("- Data quality tests: no failing tests were returned by OpenMetadata for this table (or tests are not configured).\n")
	} else {
		sb.WriteString("- Failing / bad data quality tests:\n")
		for _, t := range failed {
			sb.WriteString(fmt.Sprintf("  - %s (%s): %s\n", t.Name, t.Status, truncate(t.Result, 500)))
		}
	}

	sb.WriteString("\nOutput format (Markdown):\n")
	sb.WriteString("1) Incident summary (2-4 sentences)\n")
	sb.WriteString("2) Root cause analysis (bullet list; tie to lineage + failing tests when possible)\n")
	sb.WriteString("3) Impact assessment (what downstream consumers may be affected)\n")
	sb.WriteString("4) Severity (LOW/MEDIUM/HIGH) with a one-line justification\n")
	sb.WriteString("5) Recommended remediation (bullet list; concrete steps)\n")
	return sb.String()
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
