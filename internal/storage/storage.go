package storage

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
	"pika/internal/catalog"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	f.Close()
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=2000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		db.Close()
		return nil, err
	}
	if version > 1 {
		db.Close()
		return nil, fmt.Errorf("state database schema %d is newer than supported version 1", version)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS usage (candidate_id TEXT PRIMARY KEY, use_count INTEGER NOT NULL DEFAULT 0, last_used_at INTEGER NOT NULL); PRAGMA user_version=1;`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db}, nil
}
func (s *Store) Load() (map[string]catalog.Usage, error) {
	out := map[string]catalog.Usage{}
	rows, err := s.db.Query("SELECT candidate_id,use_count,last_used_at FROM usage")
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var u catalog.Usage
		if err = rows.Scan(&id, &u.Count, &u.LastUsed); err != nil {
			return out, err
		}
		out[id] = u
	}
	return out, rows.Err()
}
func (s *Store) Save(usage map[string]catalog.Usage) error {
	if len(usage) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare("INSERT INTO usage(candidate_id,use_count,last_used_at) VALUES(?,?,?) ON CONFLICT(candidate_id) DO UPDATE SET use_count=excluded.use_count,last_used_at=excluded.last_used_at")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for id, u := range usage {
		if _, err = stmt.Exec(id, u.Count, u.LastUsed); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) Close() error { return s.db.Close() }
