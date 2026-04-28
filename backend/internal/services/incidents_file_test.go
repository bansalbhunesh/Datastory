package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
)

func TestFileIncidentStore_AppendList(t *testing.T) {
	dir := t.TempDir()
	store := NewFileIncidentStore(filepath.Join(dir, "inc.jsonl"))
	ctx := context.Background()

	for _, e := range []domain.IncidentLogEntry{
		{ID: "1", CreatedAt: 100, TableFQN: "a", Severity: domain.SeverityLow, Source: "deterministic"},
		{ID: "2", CreatedAt: 200, TableFQN: "a", Severity: domain.SeverityHigh, Source: "claude"},
		{ID: "3", CreatedAt: 150, TableFQN: "b", Severity: domain.SeverityMedium, Source: "deterministic"},
	} {
		if err := store.Append(ctx, e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	got, err := store.ListByTable("a", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 a entries, got %d", len(got))
	}
	// The file store reverses input order on list, so the latest-appended
	// entry for "a" comes first.
	if got[0].ID != "2" {
		t.Fatalf("expected id=2 first, got %s", got[0].ID)
	}
}

func TestFileIncidentStore_RejectsEmptyID(t *testing.T) {
	dir := t.TempDir()
	store := NewFileIncidentStore(filepath.Join(dir, "inc.jsonl"))
	if err := store.Append(context.Background(), domain.IncidentLogEntry{TableFQN: "x"}); err == nil {
		t.Fatal("expected error for empty id")
	}
}
