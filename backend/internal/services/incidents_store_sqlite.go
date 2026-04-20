package services

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
)

type sqliteIncidentStore struct {
	db *sql.DB
}

func NewSQLiteIncidentStore(path string) (IncidentStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("incident sqlite path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS incidents (
	id TEXT PRIMARY KEY,
	created_at INTEGER NOT NULL,
	table_fqn TEXT NOT NULL,
	severity TEXT NOT NULL,
	source TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_incidents_table_created
	ON incidents(table_fqn, created_at DESC);
`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &sqliteIncidentStore{db: db}, nil
}

func (s *sqliteIncidentStore) Append(e domain.IncidentLogEntry) error {
	_, err := s.db.Exec(`INSERT INTO incidents(id, created_at, table_fqn, severity, source) VALUES(?,?,?,?,?)`,
		e.ID, e.CreatedAt, e.TableFQN, string(e.Severity), e.Source)
	return err
}

func (s *sqliteIncidentStore) ListByTable(tableFQN string, limit int) ([]domain.IncidentLogEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`
SELECT id, created_at, table_fqn, severity, source
FROM incidents
WHERE table_fqn = ?
ORDER BY created_at DESC
LIMIT ?`, tableFQN, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.IncidentLogEntry, 0, limit)
	for rows.Next() {
		var e domain.IncidentLogEntry
		var sev string
		if err := rows.Scan(&e.ID, &e.CreatedAt, &e.TableFQN, &sev, &e.Source); err != nil {
			return nil, err
		}
		e.Severity = domain.Severity(sev)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
