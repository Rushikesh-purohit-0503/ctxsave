package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db      *sql.DB
	rootDir string
}

func New(projectDir string) (*Store, error) {
	ctxDir := filepath.Join(projectDir, ".ctxsave")
	if err := os.MkdirAll(ctxDir, 0755); err != nil {
		return nil, fmt.Errorf("create .ctxsave dir: %w", err)
	}

	dbPath := filepath.Join(ctxDir, "context.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{db: db, rootDir: projectDir}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id         TEXT PRIMARY KEY,
		created_at DATETIME NOT NULL,
		source     TEXT NOT NULL,
		project    TEXT NOT NULL DEFAULT '',
		label      TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS entries (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		type       TEXT NOT NULL,
		content    TEXT NOT NULL,
		metadata   TEXT NOT NULL DEFAULT '',
		order_idx  INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS summaries (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id     TEXT NOT NULL REFERENCES sessions(id),
		level          TEXT NOT NULL,
		content        TEXT NOT NULL,
		token_estimate INTEGER NOT NULL DEFAULT 0,
		created_at     DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS processed_transcripts (
		file_path  TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		file_size  INTEGER NOT NULL DEFAULT 0,
		captured_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_entries_session ON entries(session_id);
	CREATE INDEX IF NOT EXISTS idx_summaries_session ON summaries(session_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) CreateSession(source, project, label string) (*Session, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(
		"INSERT INTO sessions (id, created_at, source, project, label) VALUES (?, ?, ?, ?, ?)",
		id, now, source, project, label,
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}
	return &Session{ID: id, CreatedAt: now, Source: source, Project: project, Label: label}, nil
}

func (s *Store) AddEntry(sessionID string, entryType EntryType, content, metadata string, orderIdx int) (*Entry, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		"INSERT INTO entries (session_id, type, content, metadata, order_idx, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		sessionID, string(entryType), content, metadata, orderIdx, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert entry: %w", err)
	}
	id, _ := res.LastInsertId()
	return &Entry{
		ID: id, SessionID: sessionID, Type: entryType,
		Content: content, Metadata: metadata, OrderIdx: orderIdx, CreatedAt: now,
	}, nil
}

func (s *Store) AddSummary(sessionID, level, content string, tokenEstimate int) (*Summary, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		"INSERT INTO summaries (session_id, level, content, token_estimate, created_at) VALUES (?, ?, ?, ?, ?)",
		sessionID, level, content, tokenEstimate, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert summary: %w", err)
	}
	id, _ := res.LastInsertId()
	return &Summary{
		ID: id, SessionID: sessionID, Level: level,
		Content: content, TokenEstimate: tokenEstimate, CreatedAt: now,
	}, nil
}

func (s *Store) ListSessions(limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		"SELECT id, created_at, source, project, label FROM sessions ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.CreatedAt, &sess.Source, &sess.Project, &sess.Label); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *Store) GetSession(id string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(
		"SELECT id, created_at, source, project, label FROM sessions WHERE id = ?", id,
	).Scan(&sess.ID, &sess.CreatedAt, &sess.Source, &sess.Project, &sess.Label)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) GetEntries(sessionID string) ([]Entry, error) {
	rows, err := s.db.Query(
		"SELECT id, session_id, type, content, metadata, order_idx, created_at FROM entries WHERE session_id = ? ORDER BY order_idx",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Type, &e.Content, &e.Metadata, &e.OrderIdx, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) GetSummaries(sessionID string) ([]Summary, error) {
	rows, err := s.db.Query(
		"SELECT id, session_id, level, content, token_estimate, created_at FROM summaries WHERE session_id = ? ORDER BY created_at",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []Summary
	for rows.Next() {
		var sm Summary
		if err := rows.Scan(&sm.ID, &sm.SessionID, &sm.Level, &sm.Content, &sm.TokenEstimate, &sm.CreatedAt); err != nil {
			return nil, err
		}
		summaries = append(summaries, sm)
	}
	return summaries, rows.Err()
}

func (s *Store) GetAllEntries(limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.Query(
		`SELECT e.id, e.session_id, e.type, e.content, e.metadata, e.order_idx, e.created_at
		 FROM entries e
		 JOIN sessions s ON e.session_id = s.id
		 ORDER BY s.created_at DESC, e.order_idx
		 LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Type, &e.Content, &e.Metadata, &e.OrderIdx, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) CountEntries(sessionID string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM entries WHERE session_id = ?", sessionID).Scan(&count)
	return count, err
}

func (s *Store) IsTranscriptProcessed(filePath string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM processed_transcripts WHERE file_path = ?", filePath).Scan(&count)
	return count > 0, err
}

func (s *Store) MarkTranscriptProcessed(filePath, sessionID string, fileSize int64) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO processed_transcripts (file_path, session_id, file_size, captured_at) VALUES (?, ?, ?, ?)",
		filePath, sessionID, fileSize, time.Now().UTC(),
	)
	return err
}

func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
