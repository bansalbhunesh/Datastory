package openmetadata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/bansalbhunesh/Datastory/backend/internal/errs"
	"github.com/bansalbhunesh/Datastory/backend/internal/syncx"
)

// Client is a thread-safe REST client for OpenMetadata.
type Client struct {
	baseURL  string
	http     *http.Client
	email    string
	password string

	mu    sync.RWMutex
	token string

	login syncx.SingleFlight
}

type Options struct {
	BaseURL  string
	Token    string
	Email    string
	Password string
	Timeout  time.Duration
}

func New(opts Options) *Client {
	if opts.Timeout == 0 {
		opts.Timeout = 45 * time.Second
	}
	return &Client{
		baseURL:  strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		token:    strings.TrimSpace(opts.Token),
		email:    strings.TrimSpace(opts.Email),
		password: opts.Password,
		http:     &http.Client{Timeout: opts.Timeout},
	}
}

func (c *Client) getToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

func (c *Client) setToken(t string) {
	c.mu.Lock()
	c.token = strings.TrimSpace(t)
	c.mu.Unlock()
}

// ensureAuth logs in if no token. Coalesces concurrent logins.
func (c *Client) ensureAuth(ctx context.Context) error {
	if c.getToken() != "" {
		return nil
	}
	if c.email == "" || c.password == "" {
		return errs.Unauthorized("set OM_TOKEN or OM_EMAIL + OM_PASSWORD")
	}
	_, err, _ := c.login.Do("login", func() (any, error) {
		return nil, c.loginLocked(ctx)
	})
	return err
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type loginResp struct {
	AccessToken string `json:"accessToken"`
}

func (c *Client) loginLocked(ctx context.Context) error {
	body, _ := json.Marshal(loginReq{Email: c.email, Password: c.password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/users/login", bytes.NewReader(body))
	if err != nil {
		return errs.Internal("build login request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return errs.Upstream("login transport", err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errs.Unauthorized(fmt.Sprintf("login failed: %s", resp.Status))
	}
	var lr loginResp
	if err := json.Unmarshal(b, &lr); err != nil || lr.AccessToken == "" {
		return errs.Unauthorized("login: empty/invalid accessToken")
	}
	c.setToken(lr.AccessToken)
	return nil
}

// doJSON performs an authed request with retries + decodes into out if non-nil.
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, reqBody []byte, out any) error {
	if err := c.ensureAuth(ctx); err != nil {
		return err
	}

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return errs.Internal("bad url", err)
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var body io.Reader
		if len(reqBody) > 0 {
			body = bytes.NewReader(reqBody)
		}
		req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
		if err != nil {
			return errs.Internal("build request", err)
		}
		if len(reqBody) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if tok := c.getToken(); tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = errs.Upstream("transport", err)
			if !retryable(ctx, attempt, maxAttempts) {
				return lastErr
			}
			backoff(ctx, attempt)
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// If auth expired, drop token & retry once.
		if resp.StatusCode == http.StatusUnauthorized && c.email != "" && attempt == 1 {
			c.setToken("")
			if err := c.ensureAuth(ctx); err != nil {
				return err
			}
			continue
		}

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = errs.Upstream(fmt.Sprintf("%s %s: %s", method, path, resp.Status), nil)
			if !retryable(ctx, attempt, maxAttempts) {
				return lastErr
			}
			backoff(ctx, attempt)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errs.Upstream(fmt.Sprintf("%s %s: %s", method, path, resp.Status), errors.New(truncate(string(respBody), 300)))
		}
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(respBody, out); err != nil {
			return errs.Upstream("decode "+path, err)
		}
		return nil
	}
	return lastErr
}

func retryable(ctx context.Context, attempt, max int) bool {
	if ctx.Err() != nil {
		return false
	}
	return attempt < max
}

func backoff(ctx context.Context, attempt int) {
	d := time.Duration(1<<uint(attempt-1)) * 200 * time.Millisecond
	jitter := time.Duration(rand.Int63n(int64(100 * time.Millisecond)))
	select {
	case <-time.After(d + jitter):
	case <-ctx.Done():
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
