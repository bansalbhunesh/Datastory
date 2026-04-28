package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bansalbhunesh/Datastory/backend/internal/errs"
)

// Client is the minimal LLM surface the service depends on.
// Keeping it small makes mocking trivial.
type Client interface {
	Rewrite(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error)
	Enabled() bool
}

// ------------------------- Anthropic impl -------------------------

type Anthropic struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewAnthropic(apiKey, model string) *Anthropic {
	if strings.TrimSpace(model) == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &Anthropic{
		apiKey: strings.TrimSpace(apiKey),
		model:  model,
		http:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (a *Anthropic) Enabled() bool { return a.apiKey != "" }

type anthropicReq struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResp struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Rewrite returns raw model text. Retries transient errors once.
func (a *Anthropic) Rewrite(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	if !a.Enabled() {
		return "", errs.Internal("llm disabled", nil)
	}
	if maxTokens <= 0 {
		maxTokens = 2048
	}

	payload, err := json.Marshal(anthropicReq{
		Model:     a.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  []anthropicMessage{{Role: "user", Content: userPrompt}},
	})
	if err != nil {
		return "", errs.Internal("encode llm request", err)
	}

	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		text, retry, err := a.once(ctx, payload)
		if err == nil {
			return text, nil
		}
		lastErr = err
		if !retry || ctx.Err() != nil {
			break
		}
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return "", lastErr
		}
	}
	return "", lastErr
}

func (a *Anthropic) once(ctx context.Context, payload []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return "", false, errs.Internal("build llm request", err)
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := a.http.Do(req)
	if err != nil {
		return "", true, errs.Upstream("llm transport", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return "", true, errs.Upstream("llm read response", readErr)
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return "", true, errs.Upstream(fmt.Sprintf("llm http %s", resp.Status), nil)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, errs.Upstream(fmt.Sprintf("llm http %s: %s", resp.Status, truncate(string(body), 300)), nil)
	}

	var cr anthropicResp
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", false, errs.Upstream("decode llm", err)
	}
	if cr.Error != nil && cr.Error.Message != "" {
		return "", false, errs.Upstream("llm: "+cr.Error.Message, nil)
	}
	if len(cr.Content) == 0 || strings.TrimSpace(cr.Content[0].Text) == "" {
		return "", false, errs.Upstream("llm: empty response", nil)
	}
	return strings.TrimSpace(cr.Content[0].Text), false, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
