package services

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bansalbhunesh/Datastory/backend/internal/domain"
)

type IncidentStore interface {
	Append(domain.IncidentLogEntry) error
	ListByTable(tableFQN string, limit int) ([]domain.IncidentLogEntry, error)
}

type fileIncidentStore struct {
	path string
	mu   sync.Mutex
}

func NewFileIncidentStore(path string) IncidentStore {
	return &fileIncidentStore{path: strings.TrimSpace(path)}
}

func (s *fileIncidentStore) Append(e domain.IncidentLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *fileIncidentStore) ListByTable(tableFQN string, limit int) ([]domain.IncidentLogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.IncidentLogEntry{}, nil
		}
		return nil, err
	}
	defer f.Close()
	if limit <= 0 {
		limit = 20
	}
	out := make([]domain.IncidentLogEntry, 0, limit)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e domain.IncidentLogEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if tableFQN != "" && !strings.EqualFold(e.TableFQN, tableFQN) {
			continue
		}
		out = append(out, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	// reverse newest-first and apply limit
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
