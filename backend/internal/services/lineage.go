package services

import (
	"encoding/json"
	"strings"

	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
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

// SummarizeLineage turns OM lineage JSON into an upstream/downstream FQN summary.
// Edges: From feeds To (data flows From -> To).
func SummarizeLineage(raw []byte) (domain.LineageSummary, error) {
	var g lineageGraph
	if err := json.Unmarshal(raw, &g); err != nil {
		return domain.LineageSummary{}, err
	}
	focal := strings.TrimSpace(g.Entity.FullyQualifiedName)

	sum := domain.LineageSummary{
		Focal:         focal,
		UpstreamRaw:   len(g.UpstreamEdges),
		DownstreamRaw: len(g.DownstreamEdges),
	}
	if focal == "" {
		return sum, nil
	}
	sum.Upstream = walk(focal, g.UpstreamEdges, directionUpstream)
	sum.Downstream = walk(focal, g.DownstreamEdges, directionDownstream)

	// Fallback: some OM versions put edges on the wrong side of the payload, or return
	// only `nodes`. If either side is empty, inspect ALL edges (both arrays combined)
	// for edges that directly touch the focal entity. We only classify nodes that have
	// an unambiguous directional edge to focal — we never dump generic nodes into a bucket.
	if len(sum.Upstream) == 0 || len(sum.Downstream) == 0 {
		upFallback, downFallback := classifyByEdgeSets(focal, g)
		if len(sum.Upstream) == 0 {
			sum.Upstream = upFallback
		}
		if len(sum.Downstream) == 0 {
			sum.Downstream = downFallback
		}
	}
	return sum, nil
}

type direction int

const (
	directionUpstream direction = iota
	directionDownstream
)

// walk performs BFS over edges. For upstream, we follow edges whose To == cursor (predecessors).
// For downstream, we follow edges whose From == cursor (successors).
func walk(focal string, edges []lineageEdge, d direction) []string {
	queue := []string{focal}
	seen := map[string]struct{}{focal: {}}
	out := make([]string, 0)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, e := range edges {
			from := strings.TrimSpace(e.FromEntity.FullyQualifiedName)
			to := strings.TrimSpace(e.ToEntity.FullyQualifiedName)
			if from == "" || to == "" {
				continue
			}
			var next string
			switch d {
			case directionUpstream:
				if to != cur {
					continue
				}
				next = from
			case directionDownstream:
				if from != cur {
					continue
				}
				next = to
			}
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			if next != focal {
				out = append(out, next)
			}
			queue = append(queue, next)
		}
	}
	return out
}

// classifyByEdgeSets inspects all edges across both sets; any node that appears
// strictly as a predecessor of focal → upstream; strictly successor → downstream.
// Ambiguous nodes (no edge to focal) are skipped, not blindly dumped as downstream.
func classifyByEdgeSets(focal string, g lineageGraph) (up, down []string) {
	all := append([]lineageEdge{}, g.UpstreamEdges...)
	all = append(all, g.DownstreamEdges...)

	upSet := map[string]struct{}{}
	downSet := map[string]struct{}{}
	for _, e := range all {
		from := strings.TrimSpace(e.FromEntity.FullyQualifiedName)
		to := strings.TrimSpace(e.ToEntity.FullyQualifiedName)
		if from == focal && to != "" && to != focal {
			downSet[to] = struct{}{}
		}
		if to == focal && from != "" && from != focal {
			upSet[from] = struct{}{}
		}
	}
	up = setToSlice(upSet)
	down = setToSlice(downSet)
	return
}

func setToSlice(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
