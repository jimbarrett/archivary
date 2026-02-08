package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jimbarrett/archivary/internal/store"
)

func setupTestDB(t *testing.T) (*Indexer, func()) {
	t.Helper()
	db, err := OpenMemoryDB()
	if err != nil {
		t.Fatalf("opening memory db: %v", err)
	}
	return NewIndexer(db), func() { db.Close() }
}

func makePage(id, title, content, path string) *store.Page {
	return &store.Page{
		ID:        id,
		Title:     title,
		Content:   content,
		Path:      path,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestIndexPage_AndSearch(t *testing.T) {
	idx, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	page := makePage("p1", "Getting Started", "# Getting Started\n\nThis is a guide to setting up the application.", "getting-started.md")
	if err := idx.IndexPage(ctx, page); err != nil {
		t.Fatalf("indexing page: %v", err)
	}

	results, err := idx.Search(ctx, "guide")
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "p1" {
		t.Errorf("expected ID 'p1', got %q", results[0].ID)
	}
	if results[0].Title != "Getting Started" {
		t.Errorf("expected title 'Getting Started', got %q", results[0].Title)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	idx, cleanup := setupTestDB(t)
	defer cleanup()

	results, err := idx.Search(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for empty query, got %v", results)
	}
}

func TestSearch_NoResults(t *testing.T) {
	idx, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	page := makePage("p1", "Cooking Tips", "How to boil water.", "cooking.md")
	if err := idx.IndexPage(ctx, page); err != nil {
		t.Fatal(err)
	}

	results, err := idx.Search(ctx, "quantum physics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_TitleMatchRankedHigher(t *testing.T) {
	idx, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Page with "docker" in title
	p1 := makePage("p1", "Docker Setup", "How to install and configure containers.", "docker.md")
	// Page with "docker" only in body
	p2 := makePage("p2", "Infrastructure Notes", "We use docker for deployment of our services.", "infra.md")

	idx.IndexPage(ctx, p1)
	idx.IndexPage(ctx, p2)

	results, err := idx.Search(ctx, "docker")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// BM25 with title weight 10x should rank the title match first
	if results[0].ID != "p1" {
		t.Errorf("expected title match (p1) first, got %q", results[0].ID)
	}
}

func TestSearch_MultipleResults(t *testing.T) {
	idx, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	pages := []*store.Page{
		makePage("p1", "Go Basics", "Learn the Go programming language.", "go-basics.md"),
		makePage("p2", "Go Testing", "How to write tests in Go.", "go-testing.md"),
		makePage("p3", "Python Basics", "Learn the Python programming language.", "python.md"),
	}
	for _, p := range pages {
		if err := idx.IndexPage(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	results, err := idx.Search(ctx, "Go")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'Go', got %d", len(results))
	}
}

func TestRemovePage(t *testing.T) {
	idx, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	page := makePage("p1", "Temporary", "This will be removed.", "temp.md")
	idx.IndexPage(ctx, page)

	if err := idx.RemovePage(ctx, "p1"); err != nil {
		t.Fatalf("removing page: %v", err)
	}

	results, err := idx.Search(ctx, "temporary")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after removal, got %d", len(results))
	}
}

func TestIndexPage_Update(t *testing.T) {
	idx, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	page := makePage("p1", "Original", "Original content about dogs.", "page.md")
	idx.IndexPage(ctx, page)

	// Update the content
	page.Title = "Updated"
	page.Content = "Updated content about cats."
	idx.IndexPage(ctx, page)

	// Old content should not match
	results, err := idx.Search(ctx, "dogs")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for old content, got %d", len(results))
	}

	// New content should match
	results, err = idx.Search(ctx, "cats")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for new content, got %d", len(results))
	}
}

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			"single link",
			"See [[abc-123]] for details.",
			[]string{"abc-123"},
		},
		{
			"link with label",
			"See [[abc-123|Setup Guide]] for details.",
			[]string{"abc-123"},
		},
		{
			"multiple links",
			"See [[abc]] and [[def]] and [[abc]] again.",
			[]string{"abc", "def"}, // deduplicated
		},
		{
			"no links",
			"Just plain text.",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLinks(tt.content)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d links, got %d: %v", len(tt.want), len(got), got)
			}
			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("link[%d]: expected %q, got %q", i, want, got[i])
				}
			}
		})
	}
}

func TestBacklinks(t *testing.T) {
	idx, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Page A links to Page B
	pageA := makePage("page-a", "Page A", "This links to [[page-b]].", "a.md")
	pageB := makePage("page-b", "Page B", "Standalone page.", "b.md")

	idx.IndexPage(ctx, pageA)
	idx.IndexPage(ctx, pageB)

	backlinks, err := idx.GetBacklinks(ctx, "page-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(backlinks) != 1 {
		t.Fatalf("expected 1 backlink, got %d", len(backlinks))
	}
	if backlinks[0].ID != "page-a" {
		t.Errorf("expected backlink from 'page-a', got %q", backlinks[0].ID)
	}
}

func TestReindex(t *testing.T) {
	idx, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create a mock content store using a FileStore with test files
	dir := t.TempDir()

	// Write test files
	writeFile(t, dir, "one.md", "---\nid: p1\n---\n# Page One\n\nFirst page.\n")
	writeFile(t, dir, "two.md", "---\nid: p2\n---\n# Page Two\n\nSecond page.\n")

	fs, err := store.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Initial reindex
	if err := idx.Reindex(ctx, fs); err != nil {
		t.Fatalf("reindex error: %v", err)
	}

	results, err := idx.Search(ctx, "page")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results after reindex, got %d", len(results))
	}

	// Reindex again (no changes) should be a no-op
	if err := idx.Reindex(ctx, fs); err != nil {
		t.Fatalf("second reindex error: %v", err)
	}
}

func TestContentHash(t *testing.T) {
	h1 := contentHash("hello")
	h2 := contentHash("hello")
	h3 := contentHash("world")

	if h1 != h2 {
		t.Error("same content should produce same hash")
	}
	if h1 == h3 {
		t.Error("different content should produce different hash")
	}
}

// helper to write test files
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
