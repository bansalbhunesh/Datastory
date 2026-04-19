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
	Entity            entityRef     `json:"entity"`
	Nodes             []entityRef   `json:"nodes"`
	UpstreamEdges     []lineageEdge `json:"upstreamEdges"`
	DownstreamEdges   []lineageEdge `json:"downstreamEdges"`
}

// SummarizeLineage turns OpenMetadata lineage JSON into ordered upstream / downstream lists.
func SummarizeLineage(lineageJSON []byte) (models.LineageSummary, error) {
	var g lineageGraph
	if err := json.Unmarshal(lineageJSON, &g); err != nil {
		return models.LineageSummary{}, err
	}

	focal := strings.TrimSpace(g.Entity.FullyQualifiedName)
	up := make([]string, 0)
	down := make([]string, 0)
	seenUp := map[string]struct{}{}
	seenDown := map[string]struct{}{}

	// Heuristic aligned with common OM payloads: upstreamEdges flow From -> To (From feeds To).
	for _, e := range g.UpstreamEdges {
		from := strings.TrimSpace(e.FromEntity.FullyQualifiedName)
		to := strings.TrimSpace(e.ToEntity.FullyQualifiedName)
		if from == "" {
			continue
		}
		if focal == "" || to == focal || strings.TrimSpace(g.Entity.FullyQualifiedName) == "" {
			if _, ok := seenUp[from]; !ok {
				seenUp[from] = struct{}{}
				up = append(up, from)
			}
		}
	}
	for _, e := range g.DownstreamEdges {
		from := strings.TrimSpace(e.FromEntity.FullyQualifiedName)
		to := strings.TrimSpace(e.ToEntity.FullyQualifiedName)
		if to == "" {
			continue
		}
		if focal == "" || from == focal {
			if _, ok := seenDown[to]; !ok {
				seenDown[to] = struct{}{}
				down = append(down, to)
			}
		}
	}

	// Fallback: if edges were empty but nodes exist, still show something useful.
	if len(up) == 0 && len(down) == 0 && len(g.Nodes) > 0 {
		for _, n := range g.Nodes {
			fqn := strings.TrimSpace(n.FullyQualifiedName)
			if fqn == "" || fqn == focal {
				continue
			}
			// Without reliable direction, treat non-focal nodes as "related".
			if _, ok := seenDown[fqn]; !ok {
				seenDown[fqn] = struct{}{}
				down = append(down, fqn)
			}
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
