package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
)

func TestSQLiteIncidentStore_AppendAndList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteIncidentStore(filepath.Join(dir, "inc.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	ctx := context.Background()
	entries := []domain.IncidentLogEntry{
		{ID: "a1", CreatedAt: 1000, TableFQN: "svc.db.s.t1", Severity: domain.SeverityHigh, Source: "deterministic"},
		{ID: "a2", CreatedAt: 2000, TableFQN: "svc.db.s.t1", Severity: domain.SeverityMedium, Source: "claude"},
		{ID: "b1", CreatedAt: 1500, TableFQN: "svc.db.s.t2", Severity: domain.SeverityLow, Source: "deterministic"},
	}
	for _, e := range entries {
		if err := store.Append(ctx, e); err != nil {
			t.Fatalf("append %s: %v", e.ID, err)
		}
	}

	got, err := store.ListByTable("svc.db.s.t1", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries for t1, got %d", len(got))
	}
	// Newest first.
	if got[0].ID != "a2" || got[1].ID != "a1" {
		t.Fatalf("expected newest-first order: %+v", got)
	}
}

func TestSQLiteIncidentStore_AppendRejectsEmptyID(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteIncidentStore(filepath.Join(dir, "inc.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Append(context.Background(), domain.IncidentLogEntry{TableFQN: "x"}); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestSQLiteIncidentStore_AppendRejectsEmptyTable(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteIncidentStore(filepath.Join(dir, "inc.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Append(context.Background(), domain.IncidentLogEntry{ID: "x"}); err == nil {
		t.Fatal("expected error for empty tableFQN")
	}
}

func TestSQLiteIncidentStore_LimitApplied(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteIncidentStore(filepath.Join(dir, "inc.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	for i := 0; i < 5; i++ {
		e := domain.IncidentLogEntry{
			ID:        string(rune('a' + i)),
			CreatedAt: int64(1000 + i),
			TableFQN:  "t.t.t",
			Severity:  domain.SeverityLow,
			Source:    "deterministic",
		}
		if err := store.Append(context.Background(), e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	got, err := store.ListByTable("t.t.t", 2)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries (limit), got %d", len(got))
	}
}

func TestSQLiteIncidentStore_EmptyPathRejected(t *testing.T) {
	if _, err := NewSQLiteIncidentStore(""); err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, err := NewSQLiteIncidentStore("   "); err == nil {
		t.Fatal("expected error for whitespace-only path")
	}
}
