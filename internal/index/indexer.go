package index

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/jimbarrett/archivary/internal/store"
)

// wikiLinkPattern matches [[uuid]] or [[uuid|label]] style links.
var wikiLinkPattern = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// Indexer manages the SQLite index, keeping it in sync with the FileStore.
type Indexer struct {
	db *sql.DB
}

// NewIndexer creates an Indexer backed by the given database.
func NewIndexer(db *sql.DB) *Indexer {
	return &Indexer{db: db}
}

// Reindex performs a full sync of the FileStore into the SQLite index.
// It compares content hashes to skip unchanged pages, adds new pages,
// updates changed pages, and removes pages that no longer exist on disk.
func (idx *Indexer) Reindex(ctx context.Context, cs store.ContentStore) error {
	pages, err := cs.ListPages(ctx, "")
	if err != nil {
		return fmt.Errorf("listing pages: %w", err)
	}

	// Build a set of page IDs currently on disk
	diskIDs := make(map[string]bool, len(pages))
	for _, page := range pages {
		diskIDs[page.ID] = true
	}

	// Get all currently indexed page IDs and their content hashes
	indexedHashes, err := idx.getAllHashes(ctx)
	if err != nil {
		return fmt.Errorf("reading indexed hashes: %w", err)
	}

	// Upsert pages that are new or changed
	for _, page := range pages {
		hash := contentHash(page.Content)
		if existingHash, ok := indexedHashes[page.ID]; ok && existingHash == hash {
			continue // unchanged
		}
		if err := idx.IndexPage(ctx, page); err != nil {
			return fmt.Errorf("indexing page %s: %w", page.ID, err)
		}
	}

	// Remove pages from the index that no longer exist on disk
	for id := range indexedHashes {
		if !diskIDs[id] {
			if err := idx.RemovePage(ctx, id); err != nil {
				return fmt.Errorf("removing page %s: %w", id, err)
			}
		}
	}

	return nil
}

// IndexPage inserts or updates a single page in the index.
func (idx *Indexer) IndexPage(ctx context.Context, page *store.Page) error {
	hash := contentHash(page.Content)

	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Upsert the pages table
	_, err = tx.ExecContext(ctx, `
		INSERT INTO pages (id, title, path, content_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			path = excluded.path,
			content_hash = excluded.content_hash,
			updated_at = excluded.updated_at
	`, page.ID, page.Title, page.Path, hash, page.CreatedAt, page.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upserting page row: %w", err)
	}

	// Update FTS index: delete old entry then insert new one
	_, err = tx.ExecContext(ctx, `DELETE FROM pages_fts WHERE page_id = ?`, page.ID)
	if err != nil {
		return fmt.Errorf("deleting old FTS entry: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO pages_fts (page_id, title, body) VALUES (?, ?, ?)
	`, page.ID, page.Title, page.Content)
	if err != nil {
		return fmt.Errorf("inserting FTS entry: %w", err)
	}

	// Update links: clear existing, then re-extract
	_, err = tx.ExecContext(ctx, `DELETE FROM links WHERE source_id = ?`, page.ID)
	if err != nil {
		return fmt.Errorf("clearing links: %w", err)
	}

	targets := extractLinks(page.Content)
	for _, targetID := range targets {
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO links (source_id, target_id) VALUES (?, ?)
		`, page.ID, targetID)
		if err != nil {
			return fmt.Errorf("inserting link: %w", err)
		}
	}

	return tx.Commit()
}

// RemovePage deletes a page from the index entirely.
func (idx *Indexer) RemovePage(ctx context.Context, id string) error {
	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove FTS entry
	_, err = tx.ExecContext(ctx, `DELETE FROM pages_fts WHERE page_id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting FTS entry: %w", err)
	}

	// Remove links
	_, err = tx.ExecContext(ctx, `DELETE FROM links WHERE source_id = ? OR target_id = ?`, id, id)
	if err != nil {
		return fmt.Errorf("deleting links: %w", err)
	}

	// Remove page row (tags cascade via ON DELETE CASCADE)
	_, err = tx.ExecContext(ctx, `DELETE FROM pages WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting page: %w", err)
	}

	return tx.Commit()
}

// GetBacklinks returns the IDs and titles of pages that link to the given page.
func (idx *Indexer) GetBacklinks(ctx context.Context, pageID string) ([]store.Page, error) {
	rows, err := idx.db.QueryContext(ctx, `
		SELECT p.id, p.title, p.path
		FROM links l
		JOIN pages p ON p.id = l.source_id
		WHERE l.target_id = ?
	`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []store.Page
	for rows.Next() {
		var p store.Page
		if err := rows.Scan(&p.ID, &p.Title, &p.Path); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

// getAllHashes returns a map of page ID -> content hash for all indexed pages.
func (idx *Indexer) getAllHashes(ctx context.Context) (map[string]string, error) {
	rows, err := idx.db.QueryContext(ctx, `SELECT id, content_hash FROM pages`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hashes := make(map[string]string)
	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			return nil, err
		}
		hashes[id] = hash
	}
	return hashes, rows.Err()
}

// contentHash returns a hex-encoded SHA-256 hash of the content.
func contentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}

// extractLinks finds all [[target-id]] references in the content.
func extractLinks(content string) []string {
	matches := wikiLinkPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var targets []string
	for _, m := range matches {
		id := m[1]
		if !seen[id] {
			seen[id] = true
			targets = append(targets, id)
		}
	}
	return targets
}
