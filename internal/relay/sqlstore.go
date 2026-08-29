package relay

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type SQLStore struct {
	mu sync.Mutex
	db *sql.DB
}

type AuditEntry struct {
	DeviceID    string
	Op          string
	ExecID      string
	Decision    string
	Code        string
	DurationMS  int64
	ArgsPreview string
	ArgsDigest  string
}

type AuditStore interface {
	AppendAudit(AuditEntry)
}

func OpenSQLStore(dir string) (*SQLStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "relay.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLStore) migrate() error {
	_, err := s.db.Exec(`
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
CREATE TABLE IF NOT EXISTS devices (
  id TEXT PRIMARY KEY,
  secret_hash TEXT NOT NULL,
  grant_hash TEXT NOT NULL DEFAULT '',
  inbound INTEGER NOT NULL DEFAULT 0,
  last_seen INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS grants (
  device_id TEXT PRIMARY KEY,
  grant_hash TEXT NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS audit (
  ts INTEGER NOT NULL,
  device_id TEXT,
  op TEXT,
  exec_id TEXT,
  decision TEXT,
  code TEXT,
  duration_ms INTEGER,
  args_preview TEXT,
  args_digest TEXT
);
CREATE INDEX IF NOT EXISTS audit_ts ON audit(ts);
`)
	return err
}

func (s *SQLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func (s *SQLStore) LoadDevice(id string) (string, string, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var secret, grant string
	var inbound int
	err := s.db.QueryRow(`SELECT secret_hash, grant_hash, inbound FROM devices WHERE id=?`, id).
		Scan(&secret, &grant, &inbound)
	if err == sql.ErrNoRows {
		return "", "", false, false
	}
	if err != nil {
		return "", "", false, false
	}
	return secret, grant, inbound != 0, true
}

func (s *SQLStore) SaveDevice(id, secretHash, grantHash string, inbound bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inb := 0
	if inbound {
		inb = 1
	}
	now := time.Now().Unix()
	_, _ = s.db.Exec(`
INSERT INTO devices (id, secret_hash, grant_hash, inbound, last_seen) VALUES (?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET secret_hash=excluded.secret_hash, grant_hash=excluded.grant_hash,
  inbound=excluded.inbound, last_seen=excluded.last_seen`,
		id, secretHash, grantHash, inb, now)
	if grantHash != "" {
		_, _ = s.db.Exec(`
INSERT INTO grants (device_id, grant_hash, updated_at) VALUES (?,?,?)
ON CONFLICT(device_id) DO UPDATE SET grant_hash=excluded.grant_hash, updated_at=excluded.updated_at`,
			id, grantHash, now)
	}
}

func (s *SQLStore) DeleteDevice(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(`DELETE FROM devices WHERE id=?`, id)
	_, _ = s.db.Exec(`DELETE FROM grants WHERE device_id=?`, id)
}

func (s *SQLStore) AppendAudit(e AuditEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(
		`INSERT INTO audit (ts, device_id, op, exec_id, decision, code, duration_ms, args_preview, args_digest)
VALUES (?,?,?,?,?,?,?,?,?)`,
		time.Now().Unix(), e.DeviceID, e.Op, e.ExecID, e.Decision, e.Code, e.DurationMS, e.ArgsPreview, e.ArgsDigest,
	)
}
