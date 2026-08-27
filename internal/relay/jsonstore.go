package relay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// JSONStore persists device hashes in --data. Swap for SQLite later (docs/stack.md).
type JSONStore struct {
	mu   sync.Mutex
	path string
	rows map[string]storedDevice
}

type storedDevice struct {
	SecretHash string `json:"secret_hash"`
	GrantHash  string `json:"grant_hash"`
	Inbound    bool   `json:"inbound"`
}

func OpenJSONStore(dir string) (*JSONStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &JSONStore{path: filepath.Join(dir, "devices.json"), rows: map[string]storedDevice{}}
	b, err := os.ReadFile(s.path)
	if err == nil {
		_ = json.Unmarshal(b, &s.rows)
	}
	return s, nil
}

func (s *JSONStore) LoadDevice(id string) (string, string, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rows[id]
	return r.SecretHash, r.GrantHash, r.Inbound, ok
}

func (s *JSONStore) SaveDevice(id, secretHash, grantHash string, inbound bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[id] = storedDevice{secretHash, grantHash, inbound}
	s.flushLocked()
}

func (s *JSONStore) DeleteDevice(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rows, id)
	s.flushLocked()
}

func (s *JSONStore) flushLocked() {
	b, err := json.MarshalIndent(s.rows, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, b, 0o600)
}
