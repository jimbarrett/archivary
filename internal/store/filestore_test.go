package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func writeTestFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	absPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNewFileStore_EmptyDir(t *testing.T) {
	dir := setupTestWorkspace(t)
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pages, err := fs.ListPages(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(pages))
	}
}

func TestNewFileStore_IndexesExistingFiles(t *testing.T) {
	dir := setupTestWorkspace(t)
	writeTestFile(t, dir, "hello.md", "---\nid: page-1\n---\n# Hello World\n\nContent here.\n")
	writeTestFile(t, dir, "guides/setup.md", "---\nid: page-2\n---\n# Setup Guide\n\nSetup content.\n")

	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pages, err := fs.ListPages(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
}

func TestNewFileStore_GeneratesIDForFileWithout(t *testing.T) {
	dir := setupTestWorkspace(t)
	writeTestFile(t, dir, "no-id.md", "# No ID Page\n\nJust content.\n")

	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pages, err := fs.ListPages(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if pages[0].ID == "" {
		t.Error("expected generated ID, got empty")
	}

	// Verify the ID was written back to the file
	data, err := os.ReadFile(filepath.Join(dir, "no-id.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "id: ") {
		t.Error("expected file to contain 'id:' after indexing")
	}
}

func TestGetPage(t *testing.T) {
	dir := setupTestWorkspace(t)
	writeTestFile(t, dir, "test.md", "---\nid: get-test\n---\n# Test Page\n\nBody content.\n")

	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	page, err := fs.GetPage(context.Background(), "get-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.ID != "get-test" {
		t.Errorf("expected ID 'get-test', got %q", page.ID)
	}
	if page.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", page.Title)
	}
	if !strings.Contains(page.Content, "Body content.") {
		t.Errorf("expected content to contain 'Body content.', got %q", page.Content)
	}
}

func TestGetPage_NotFound(t *testing.T) {
	dir := setupTestWorkspace(t)
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = fs.GetPage(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent page")
	}
}

func TestSavePage_New(t *testing.T) {
	dir := setupTestWorkspace(t)
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	page := &Page{
		ID:      "new-page",
		Content: "# Brand New\n\nFresh content.\n",
		Path:    "new.md",
	}

	if err := fs.SavePage(context.Background(), page); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify we can read it back
	got, err := fs.GetPage(context.Background(), "new-page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "Brand New" {
		t.Errorf("expected title 'Brand New', got %q", got.Title)
	}

	// Verify the file exists on disk
	data, err := os.ReadFile(filepath.Join(dir, "new.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "id: new-page") {
		t.Error("expected file to contain frontmatter with ID")
	}
}

func TestSavePage_InSubdirectory(t *testing.T) {
	dir := setupTestWorkspace(t)
	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	page := &Page{
		ID:      "sub-page",
		Content: "# Nested Page\n\nIn a subdirectory.\n",
		Path:    "guides/deep/nested.md",
	}

	if err := fs.SavePage(context.Background(), page); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify subdirectories were created
	if _, err := os.Stat(filepath.Join(dir, "guides", "deep", "nested.md")); os.IsNotExist(err) {
		t.Error("expected nested file to exist")
	}
}

func TestSavePage_Update(t *testing.T) {
	dir := setupTestWorkspace(t)
	writeTestFile(t, dir, "update.md", "---\nid: update-me\n---\n# Original\n\nOld content.\n")

	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	page := &Page{
		ID:      "update-me",
		Content: "# Updated\n\nNew content.\n",
		Path:    "update.md",
	}

	if err := fs.SavePage(context.Background(), page); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := fs.GetPage(context.Background(), "update-me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "Updated" {
		t.Errorf("expected title 'Updated', got %q", got.Title)
	}
}

func TestDeletePage(t *testing.T) {
	dir := setupTestWorkspace(t)
	writeTestFile(t, dir, "delete-me.md", "---\nid: doomed\n---\n# Gone Soon\n")

	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := fs.DeletePage(context.Background(), "doomed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not be gettable
	_, err = fs.GetPage(context.Background(), "doomed")
	if err == nil {
		t.Error("expected error after delete")
	}

	// File should be gone from disk
	if _, err := os.Stat(filepath.Join(dir, "delete-me.md")); !os.IsNotExist(err) {
		t.Error("expected file to be deleted from disk")
	}
}

func TestListPages_FilterByDir(t *testing.T) {
	dir := setupTestWorkspace(t)
	writeTestFile(t, dir, "root.md", "---\nid: p1\n---\n# Root\n")
	writeTestFile(t, dir, "guides/one.md", "---\nid: p2\n---\n# Guide One\n")
	writeTestFile(t, dir, "guides/two.md", "---\nid: p3\n---\n# Guide Two\n")
	writeTestFile(t, dir, "notes/note.md", "---\nid: p4\n---\n# Note\n")

	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All pages
	all, err := fs.ListPages(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("expected 4 pages, got %d", len(all))
	}

	// Only guides
	guides, err := fs.ListPages(context.Background(), "guides/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(guides) != 2 {
		t.Errorf("expected 2 guide pages, got %d", len(guides))
	}
}

func TestBuildTree(t *testing.T) {
	dir := setupTestWorkspace(t)
	writeTestFile(t, dir, "root.md", "---\nid: p1\n---\n# Root Page\n")
	writeTestFile(t, dir, "guides/setup.md", "---\nid: p2\n---\n# Setup\n")

	fs, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tree, err := fs.BuildTree(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tree.IsDir {
		t.Error("expected root to be a directory")
	}
	if len(tree.Children) < 2 {
		t.Errorf("expected at least 2 children (file + dir), got %d", len(tree.Children))
	}
}

func TestTitleFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"hello-world.md", "Hello world"},
		{"setup_guide.md", "Setup guide"},
		{"README.md", "README"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := TitleFromFilename(tt.filename)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
