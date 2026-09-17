package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    code       TEXT UNIQUE NOT NULL,
    url        TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_code    ON links(code);
CREATE INDEX IF NOT EXISTS idx_expires ON links(expires_at);

CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT PRIMARY KEY,
    pin        TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_session_expires ON sessions(expires_at);
`

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &Store{db: db}
	go s.cleanupLoop()
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Create stores a link and returns its short code.
func (s *Store) Create(rawURL string) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := randomCode(6)
		if err != nil {
			return "", err
		}
		_, err = s.db.Exec(
			`INSERT INTO links (code, url, expires_at) VALUES (?, ?, ?)`,
			code, rawURL, time.Now().Add(24*time.Hour),
		)
		if err == nil {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique code")
}

// Resolve returns the original URL for a given code, or "" if not found/expired.
func (s *Store) Resolve(code string) (string, error) {
	var rawURL string
	err := s.db.QueryRow(
		`SELECT url FROM links WHERE code = ? AND expires_at > CURRENT_TIMESTAMP`,
		code,
	).Scan(&rawURL)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return rawURL, err
}

// CreateSession persists a session token.
func (s *Store) CreateSession(token string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO sessions (token, pin, expires_at) VALUES (?, '', ?)`,
		token, time.Now().Add(2*time.Hour),
	)
	return err
}

// ValidateSession returns true if token matches a non-expired session.
func (s *Store) ValidateSession(token string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE token = ? AND expires_at > CURRENT_TIMESTAMP`,
		token,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteSession removes a session.
func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (s *Store) cleanupLoop() {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for range t.C {
		_, _ = s.db.Exec(`DELETE FROM links WHERE expires_at < CURRENT_TIMESTAMP`)
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`)
	}
}

func randomCode(n int) (string, error) {
	b := make([]byte, n)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		num, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = charset[num.Int64()]
	}
	return string(b), nil
}
