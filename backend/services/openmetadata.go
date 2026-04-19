package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OpenMetadataClient is a small REST client for search + entity + lineage calls.
type OpenMetadataClient struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func NewOpenMetadataClient(baseURL, bearerToken string) *OpenMetadataClient {
	return &OpenMetadataClient{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:   strings.TrimSpace(bearerToken),
		Client:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *OpenMetadataClient) SetToken(token string) { c.Token = strings.TrimSpace(token) }

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"accessToken"`
}

// Login exchanges email/password for a JWT.
func (c *OpenMetadataClient) Login(ctx context.Context, email, password string) (string, error) {
	body, err := json.Marshal(loginRequest{Email: email, Password: password})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/users/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

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
		return "", fmt.Errorf("login failed: %s: %s", resp.Status, string(b))
	}

	var lr loginResponse
	if err := json.Unmarshal(b, &lr); err != nil {
		return "", fmt.Errorf("decode login: %w", err)
	}
	if lr.AccessToken == "" {
		return "", errors.New("login: empty accessToken")
	}
	c.SetToken(lr.AccessToken)
	return lr.AccessToken, nil
}

func (c *OpenMetadataClient) authHeader(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func (c *OpenMetadataClient) doJSON(ctx context.Context, method, path string, query url.Values, reqBody []byte, out any) error {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var body io.Reader
	if len(reqBody) > 0 {
		body = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return err
	}
	if len(reqBody) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	c.authHeader(req)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %s: %s", method, u.Path, resp.Status, string(b))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("decode %s: %w", u.Path, err)
	}
	return nil
}

// SearchTables calls GET /api/v1/search/query.
func (c *OpenMetadataClient) SearchTables(ctx context.Context, q string, size int) (json.RawMessage, error) {
	if strings.TrimSpace(q) == "" {
		return nil, errors.New("search: empty query")
	}
	if size <= 0 {
		size = 10
	}
	vals := url.Values{}
	vals.Set("q", q)
	vals.Set("index", "table_search_index")
	vals.Set("size", fmt.Sprintf("%d", size))

	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/search/query", vals, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// TableSearchHit is a convenience view of common fields in search hits.
type TableSearchHit struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	FullyQualifiedName string `json:"fullyQualifiedName"`
}

// ParseTableSearchHits extracts table rows from a SearchTables JSON payload.
func ParseTableSearchHits(searchJSON json.RawMessage) ([]TableSearchHit, error) {
	var root map[string]any
	if err := json.Unmarshal(searchJSON, &root); err != nil {
		return nil, err
	}

	hitsObj, ok := root["hits"].(map[string]any)
	if !ok {
		return nil, errors.New("search: missing hits")
	}
	hitsArr, ok := hitsObj["hits"].([]any)
	if !ok {
		return []TableSearchHit{}, nil
	}

	out := make([]TableSearchHit, 0, len(hitsArr))
	for _, h := range hitsArr {
		m, ok := h.(map[string]any)
		if !ok {
			continue
		}
		src, ok := m["_source"].(map[string]any)
		if !ok {
			continue
		}
		b, err := json.Marshal(src)
		if err != nil {
			continue
		}
		var row TableSearchHit
		if err := json.Unmarshal(b, &row); err != nil {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

// GetTableByFQN calls GET /api/v1/tables/name/{fqn}.
func (c *OpenMetadataClient) GetTableByFQN(ctx context.Context, fqn string) (json.RawMessage, error) {
	fqn = strings.TrimSpace(fqn)
	if fqn == "" {
		return nil, errors.New("fqn empty")
	}
	path := "/api/v1/tables/name/" + url.PathEscape(fqn)

	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// GetTableLineageByFQN calls GET /api/v1/lineage/table/name/{fqn}.
func (c *OpenMetadataClient) GetTableLineageByFQN(ctx context.Context, fqn string, upstreamDepth, downstreamDepth int) (json.RawMessage, error) {
	fqn = strings.TrimSpace(fqn)
	if fqn == "" {
		return nil, errors.New("fqn empty")
	}
	vals := url.Values{}
	vals.Set("upstreamDepth", fmt.Sprintf("%d", upstreamDepth))
	vals.Set("downstreamDepth", fmt.Sprintf("%d", downstreamDepth))

	path := "/api/v1/lineage/table/name/" + url.PathEscape(fqn)
	var raw json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, path, vals, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// ResolveTableFQN returns a table FQN from either explicit input or the first search hit.
func (c *OpenMetadataClient) ResolveTableFQN(ctx context.Context, explicitFQN, searchQuery string) (string, json.RawMessage, error) {
	explicitFQN = strings.TrimSpace(explicitFQN)
	if explicitFQN != "" {
		return explicitFQN, nil, nil
	}
	q := strings.TrimSpace(searchQuery)
	if q == "" {
		return "", nil, errors.New("missing tableFQN and query")
	}
	raw, err := c.SearchTables(ctx, q, 5)
	if err != nil {
		return "", nil, err
	}
	hits, err := ParseTableSearchHits(raw)
	if err != nil {
		return "", nil, err
	}
	if len(hits) == 0 || strings.TrimSpace(hits[0].FullyQualifiedName) == "" {
		return "", raw, fmt.Errorf("no table hits for %q", q)
	}
	return hits[0].FullyQualifiedName, raw, nil
}
