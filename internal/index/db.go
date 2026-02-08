package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// OpenDB opens (or creates) the SQLite database at the given data directory.
func OpenDB(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "archivary.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return db, nil
}

// OpenMemoryDB creates an in-memory SQLite database, useful for testing.
func OpenMemoryDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS pages (
		id           TEXT PRIMARY KEY,
		title        TEXT NOT NULL DEFAULT '',
		path         TEXT NOT NULL UNIQUE,
		content_hash TEXT NOT NULL DEFAULT '',
		created_at   DATETIME,
		updated_at   DATETIME
	);

	CREATE TABLE IF NOT EXISTS tags (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		page_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
		tag     TEXT NOT NULL,
		UNIQUE(page_id, tag)
	);

	CREATE INDEX IF NOT EXISTS idx_tags_page ON tags(page_id);
	CREATE INDEX IF NOT EXISTS idx_tags_tag ON tags(tag);

	CREATE TABLE IF NOT EXISTS links (
		source_id TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
		target_id TEXT NOT NULL,
		PRIMARY KEY (source_id, target_id)
	);

	CREATE INDEX IF NOT EXISTS idx_links_target ON links(target_id);

	CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts USING fts5(
		page_id,
		title,
		body,
		tokenize='porter unicode61'
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("executing schema: %w", err)
	}
	return nil
}
