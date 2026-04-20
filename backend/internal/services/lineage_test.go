package services

import (
	"testing"
)

func TestSummarizeLineage_EdgesDirectional(t *testing.T) {
	raw := []byte(`{
		"entity": {"fullyQualifiedName": "svc.db.public.orders"},
		"nodes": [],
		"upstreamEdges": [
			{"fromEntity": {"fullyQualifiedName": "svc.db.public.raw_orders"},
			 "toEntity":   {"fullyQualifiedName": "svc.db.public.orders"}}
		],
		"downstreamEdges": [
			{"fromEntity": {"fullyQualifiedName": "svc.db.public.orders"},
			 "toEntity":   {"fullyQualifiedName": "svc.db.public.orders_daily"}}
		]
	}`)
	sum, err := SummarizeLineage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Focal != "svc.db.public.orders" {
		t.Fatalf("focal mismatch: %s", sum.Focal)
	}
	if len(sum.Upstream) != 1 || sum.Upstream[0] != "svc.db.public.raw_orders" {
		t.Fatalf("upstream wrong: %+v", sum.Upstream)
	}
	if len(sum.Downstream) != 1 || sum.Downstream[0] != "svc.db.public.orders_daily" {
		t.Fatalf("downstream wrong: %+v", sum.Downstream)
	}
}

// Reproduces the original bug: when only `nodes` is populated (no edges on either side),
// the old code classified everything as downstream. The new fallback must not.
func TestSummarizeLineage_FallbackDoesNotDumpToDownstream(t *testing.T) {
	raw := []byte(`{
		"entity": {"fullyQualifiedName": "svc.db.public.orders"},
		"nodes": [
			{"fullyQualifiedName": "svc.db.public.raw_orders"},
			{"fullyQualifiedName": "svc.db.public.orders_daily"}
		],
		"upstreamEdges": [],
		"downstreamEdges": []
	}`)
	sum, err := SummarizeLineage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With no edges touching focal, ambiguous nodes must NOT be dumped as downstream.
	if len(sum.Downstream) != 0 {
		t.Fatalf("expected empty downstream (ambiguous nodes), got %+v", sum.Downstream)
	}
	if len(sum.Upstream) != 0 {
		t.Fatalf("expected empty upstream (ambiguous nodes), got %+v", sum.Upstream)
	}
}

func TestSummarizeLineage_FallbackClassifiesByEdges(t *testing.T) {
	// Edges present but on wrong side of payload (some OM versions) — fallback should classify correctly.
	raw := []byte(`{
		"entity": {"fullyQualifiedName": "svc.db.public.orders"},
		"nodes": [
			{"fullyQualifiedName": "svc.db.public.raw_orders"},
			{"fullyQualifiedName": "svc.db.public.orders_daily"}
		],
		"upstreamEdges": [
			{"fromEntity": {"fullyQualifiedName": "svc.db.public.raw_orders"},
			 "toEntity":   {"fullyQualifiedName": "svc.db.public.orders"}},
			{"fromEntity": {"fullyQualifiedName": "svc.db.public.orders"},
			 "toEntity":   {"fullyQualifiedName": "svc.db.public.orders_daily"}}
		],
		"downstreamEdges": []
	}`)
	sum, err := SummarizeLineage(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// First pass populates upstream via upstreamEdges; downstream derived via fallback classification.
	if !contains(sum.Upstream, "svc.db.public.raw_orders") {
		t.Fatalf("missing upstream: %+v", sum.Upstream)
	}
	if !contains(sum.Downstream, "svc.db.public.orders_daily") {
		t.Fatalf("fallback should classify orders_daily as downstream: %+v", sum.Downstream)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
