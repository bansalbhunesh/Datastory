package services

import (
	"encoding/json"
	"strings"

	"github.com/bansalbhunesh/Datastory/backend/models"
)

type entityRef struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	FullyQualifiedName string `json:"fullyQualifiedName"`
}

type lineageEdge struct {
	FromEntity entityRef `json:"fromEntity"`
	ToEntity   entityRef `json:"toEntity"`
}

type lineageGraph struct {
	Entity          entityRef     `json:"entity"`
	Nodes           []entityRef   `json:"nodes"`
	UpstreamEdges   []lineageEdge `json:"upstreamEdges"`
	DownstreamEdges []lineageEdge `json:"downstreamEdges"`
}

// SummarizeLineage turns OpenMetadata lineage JSON into upstream / downstream FQN lists.
// OpenMetadata edges are generally directed From -> To meaning data flows From -> To.
func SummarizeLineage(lineageJSON []byte) (models.LineageSummary, error) {
	var g lineageGraph
	if err := json.Unmarshal(lineageJSON, &g); err != nil {
		return models.LineageSummary{}, err
	}

	focal := strings.TrimSpace(g.Entity.FullyQualifiedName)
	if focal == "" {
		// Extremely defensive: still return raw edge counts.
		return models.LineageSummary{
			Focal:         "",
			Upstream:      nil,
			Downstream:    nil,
			UpstreamRaw:   len(g.UpstreamEdges),
			DownstreamRaw: len(g.DownstreamEdges),
		}, nil
	}

	up := walkUpstream(focal, g.UpstreamEdges)
	down := walkDownstream(focal, g.DownstreamEdges)

	// Fallback when servers return nodes but sparse edges.
	if len(up) == 0 && len(down) == 0 && len(g.Nodes) > 0 {
		seen := map[string]struct{}{focal: {}}
		down = make([]string, 0)
		for _, n := range g.Nodes {
			fqn := strings.TrimSpace(n.FullyQualifiedName)
			if fqn == "" {
				continue
			}
			if _, ok := seen[fqn]; ok {
				continue
			}
			seen[fqn] = struct{}{}
			down = append(down, fqn)
		}
	}

	return models.LineageSummary{
		Focal:         focal,
		Upstream:      up,
		Downstream:    down,
		UpstreamRaw:   len(g.UpstreamEdges),
		DownstreamRaw: len(g.DownstreamEdges),
	}, nil
}

func fqnKey(e entityRef) string {
	return strings.TrimSpace(e.FullyQualifiedName)
}

// walkUpstream collects all upstream FQNs reachable from focal using upstreamEdges (From feeds To).
func walkUpstream(focal string, edges []lineageEdge) []string {
	focal = strings.TrimSpace(focal)
	queue := []string{focal}
	seen := map[string]struct{}{focal: {}}
	out := make([]string, 0)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, e := range edges {
			to := fqnKey(e.ToEntity)
			from := fqnKey(e.FromEntity)
			if from == "" || to == "" {
				continue
			}
			if to != cur {
				continue
			}
			if _, ok := seen[from]; ok {
				continue
			}
			seen[from] = struct{}{}
			if from != focal {
				out = append(out, from)
			}
			queue = append(queue, from)
		}
	}
	return out
}

// walkDownstream collects downstream FQNs reachable from focal using downstreamEdges (From feeds To).
func walkDownstream(focal string, edges []lineageEdge) []string {
	focal = strings.TrimSpace(focal)
	queue := []string{focal}
	seen := map[string]struct{}{focal: {}}
	out := make([]string, 0)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, e := range edges {
			from := fqnKey(e.FromEntity)
			to := fqnKey(e.ToEntity)
			if from == "" || to == "" {
				continue
			}
			if from != cur {
				continue
			}
			if _, ok := seen[to]; ok {
				continue
			}
			seen[to] = struct{}{}
			if to != focal {
				out = append(out, to)
			}
			queue = append(queue, to)
		}
	}
	return out
}
