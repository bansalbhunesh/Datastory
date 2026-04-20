package openmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
	"github.com/bansalbhunesh/Datastory/backend/internal/errs"
)

// SearchTables returns table hits for an autocomplete query.
func (c *Client) SearchTables(ctx context.Context, q string, size int) ([]domain.TableHit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, errs.BadRequest("search: empty query")
	}
	if size <= 0 {
		size = 10
	}
	vals := url.Values{}
	vals.Set("q", q)
	vals.Set("index", "table_search_index")
	vals.Set("size", fmt.Sprintf("%d", size))

	var raw struct {
		Hits struct {
			Hits []struct {
				Source domain.TableHit `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/search/query", vals, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]domain.TableHit, 0, len(raw.Hits.Hits))
	for _, h := range raw.Hits.Hits {
		if strings.TrimSpace(h.Source.FullyQualifiedName) != "" {
			out = append(out, h.Source)
		}
	}
	return out, nil
}

// LineageJSON returns raw lineage graph for a given FQN.
func (c *Client) LineageJSON(ctx context.Context, fqn string, upstreamDepth, downstreamDepth int) (json.RawMessage, error) {
	fqn = strings.TrimSpace(fqn)
	if fqn == "" {
		return nil, errs.BadRequest("fqn empty")
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

// Ping validates the OM HTTP surface is reachable.
func (c *Client) Ping(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/version", nil)
	if tok := c.getToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return errs.Upstream("om ping", err)
	}
	defer resp.Body.Close()
	// Auth errors still imply the server is up.
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errs.Upstream(fmt.Sprintf("om ping: %s", resp.Status), nil)
	}
	return nil
}

func (c *Client) HasStaticToken() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token != "" && c.email == ""
}

func (c *Client) HasCreds() bool { return c.email != "" && c.password != "" }
