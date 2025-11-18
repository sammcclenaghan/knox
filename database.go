package main

import (
	"database/sql"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Note struct {
	Path           string
	ExpiryDatetime time.Time
	TrackedAt      time.Time
}

type DB struct {
	conn *sql.DB
	mu   sync.Mutex
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS notes (
		note_path TEXT PRIMARY KEY,
		expiry_datetime TIMESTAMP NOT NULL,
		tracked_at TIMESTAMP NOT NULL
	);
	`
	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) TrackNote(note Note) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
	INSERT INTO notes (note_path, expiry_datetime, tracked_at)
	VALUES (?, ?, ?)
	ON CONFLICT(note_path) DO UPDATE SET
		expiry_datetime = excluded.expiry_datetime,
		tracked_at = excluded.tracked_at
	`

	_, err := db.conn.Exec(query, note.Path, note.ExpiryDatetime, note.TrackedAt)
	return err
}

func (db *DB) GetExpired() ([]Note, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
	SELECT note_path, expiry_datetime, tracked_at
	FROM notes
	WHERE expiry_datetime <= ?
	`

	rows, err := db.conn.Query(query, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.Path, &note.ExpiryDatetime, &note.TrackedAt); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	return notes, rows.Err()
}

func (db *DB) GetExpiringWithin(duration time.Duration) ([]Note, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
	SELECT note_path, expiry_datetime, tracked_at
	FROM notes
	WHERE expiry_datetime > ? AND expiry_datetime <= ?
	ORDER BY expiry_datetime ASC
	`

	deadline := time.Now().Add(duration)
	rows, err := db.conn.Query(query, time.Now(), deadline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.Path, &note.ExpiryDatetime, &note.TrackedAt); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	return notes, rows.Err()
}

func (db *DB) DeleteNote(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM notes WHERE note_path = ?", path)
	return err
}

func (db *DB) Close() error {
	return db.conn.Close()
}
