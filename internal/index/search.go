package index

import (
	"context"
	"database/sql"
	"fmt"
)

// SearchResult represents a single search hit.
type SearchResult struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Path  string  `json:"path"`
	Rank  float64 `json:"rank"`
	// Snippet is a short excerpt with matching terms highlighted.
	Snippet string `json:"snippet"`
}

// Search performs a full-text search across page titles and bodies.
// Results are ranked by relevance (BM25) with title matches weighted higher.
func (idx *Indexer) Search(ctx context.Context, query string) ([]SearchResult, error) {
	if query == "" {
		return nil, nil
	}

	rows, err := idx.db.QueryContext(ctx, `
		SELECT p.id, p.title, p.path,
		       bm25(pages_fts, 5.0, 10.0, 1.0) AS rank,
		       snippet(pages_fts, 2, '<mark>', '</mark>', '...', 40) AS snippet
		FROM pages_fts fts
		JOIN pages p ON p.id = fts.page_id
		WHERE pages_fts MATCH ?
		ORDER BY rank
		LIMIT 50
	`, query)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var rank sql.NullFloat64
		var snippet sql.NullString
		if err := rows.Scan(&r.ID, &r.Title, &r.Path, &rank, &snippet); err != nil {
			return nil, fmt.Errorf("scanning result: %w", err)
		}
		if rank.Valid {
			r.Rank = rank.Float64
		}
		if snippet.Valid {
			r.Snippet = snippet.String
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
