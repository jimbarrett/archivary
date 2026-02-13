package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jimbarrett/archivary/internal/markdown"
)

// FileStore implements ContentStore by reading and writing markdown files
// on disk. Files are the canonical source of truth.
type FileStore struct {
	// root is the workspace directory containing markdown files.
	root string

	// pageIndex maps page IDs to their relative file paths for fast lookups.
	pageIndex map[string]string
}

// NewFileStore creates a FileStore rooted at the given workspace directory
// and builds an initial in-memory index by scanning all markdown files.
func NewFileStore(root string) (*FileStore, error) {
	fs := &FileStore{
		root:      root,
		pageIndex: make(map[string]string),
	}
	if err := fs.buildIndex(); err != nil {
		return nil, fmt.Errorf("building file index: %w", err)
	}
	return fs, nil
}

func (fs *FileStore) GetPage(_ context.Context, id string) (*Page, error) {
	relPath, ok := fs.pageIndex[id]
	if !ok {
		return nil, fmt.Errorf("page not found: %s", id)
	}
	return fs.readPage(relPath)
}

func (fs *FileStore) SavePage(_ context.Context, page *Page) error {
	fm := markdown.Frontmatter{ID: page.ID}
	fm = markdown.EnsureID(fm)
	page.ID = fm.ID

	raw := markdown.Serialize(fm, page.Content)

	absPath := filepath.Join(fs.root, page.Path)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tmpFile := absPath + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(raw), 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmpFile, absPath); err != nil {
		os.Remove(tmpFile) // clean up on failure
		return fmt.Errorf("renaming temp file: %w", err)
	}

	// Update the in-memory index
	fs.pageIndex[page.ID] = page.Path
	return nil
}

func (fs *FileStore) DeletePage(_ context.Context, id string) error {
	relPath, ok := fs.pageIndex[id]
	if !ok {
		return fmt.Errorf("page not found: %s", id)
	}

	absPath := filepath.Join(fs.root, relPath)
	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}

	delete(fs.pageIndex, id)
	return nil
}

func (fs *FileStore) ListPages(_ context.Context, dirPrefix string) ([]*Page, error) {
	var pages []*Page
	for _, relPath := range fs.pageIndex {
		if dirPrefix != "" && !strings.HasPrefix(relPath, dirPrefix) {
			continue
		}
		page, err := fs.readPage(relPath)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	return pages, nil
}

// readPage reads a single markdown file and returns it as a Page.
func (fs *FileStore) readPage(relPath string) (*Page, error) {
	absPath := filepath.Join(fs.root, relPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", relPath, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat file %s: %w", relPath, err)
	}

	fm, body := markdown.Parse(string(data))
	fm = markdown.EnsureID(fm)

	title := markdown.ExtractTitle(body)
	if title == "" {
		// Fall back to filename without extension
		title = strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	}

	return &Page{
		ID:        fm.ID,
		Title:     title,
		Content:   body,
		Path:      relPath,
		UpdatedAt: info.ModTime(),
	}, nil
}

// buildIndex scans the workspace directory for all .md files and populates
// the in-memory page ID -> path index.
func (fs *FileStore) buildIndex() error {
	return filepath.WalkDir(fs.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		relPath, err := filepath.Rel(fs.root, path)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", relPath, err)
		}

		fm, _ := markdown.Parse(string(data))

		if fm.ID == "" {
			// File has no ID — generate one and write it back
			fm = markdown.EnsureID(fm)
			_, body := markdown.Parse(string(data))
			raw := markdown.Serialize(fm, body)
			if writeErr := atomicWrite(path, []byte(raw)); writeErr != nil {
				return fmt.Errorf("writing ID to %s: %w", relPath, writeErr)
			}
		}

		fs.pageIndex[fm.ID] = relPath
		return nil
	})
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// TitleFromFilename derives a display title from a filename.
func TitleFromFilename(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	// Capitalize first letter
	if len(name) > 0 {
		return strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

// WalkDir returns a nested representation of the workspace directory structure.
// This is used by the API to build the sidebar folder tree.
type DirEntry struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"is_dir"`
	Children []*DirEntry `json:"children,omitempty"`
	PageID   string      `json:"page_id,omitempty"`
}

func (fs *FileStore) BuildTree(_ context.Context) (*DirEntry, error) {
	root := &DirEntry{
		Name:  filepath.Base(fs.root),
		Path:  "",
		IsDir: true,
	}

	dirMap := map[string]*DirEntry{"": root}

	// Ensure parent dirs exist in the tree
	ensureDir := func(relDir string) *DirEntry {
		if relDir == "" || relDir == "." {
			return root
		}
		if entry, ok := dirMap[relDir]; ok {
			return entry
		}
		parts := strings.Split(filepath.ToSlash(relDir), "/")
		current := root
		builtPath := ""
		for _, part := range parts {
			if builtPath == "" {
				builtPath = part
			} else {
				builtPath = builtPath + "/" + part
			}
			if entry, ok := dirMap[builtPath]; ok {
				current = entry
				continue
			}
			entry := &DirEntry{
				Name:  part,
				Path:  builtPath,
				IsDir: true,
			}
			current.Children = append(current.Children, entry)
			dirMap[builtPath] = entry
			current = entry
		}
		return current
	}

	// Walk the filesystem to discover all directories (including empty ones)
	filepath.WalkDir(fs.root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == fs.root {
			return nil
		}
		if d.Name() == ".git" {
			return filepath.SkipDir
		}
		relDir, err := filepath.Rel(fs.root, path)
		if err != nil {
			return nil
		}
		ensureDir(filepath.ToSlash(relDir))
		return nil
	})

	for id, relPath := range fs.pageIndex {
		dir := filepath.Dir(relPath)
		if dir == "." {
			dir = ""
		}
		parent := ensureDir(dir)

		data, err := os.ReadFile(filepath.Join(fs.root, relPath))
		if err != nil {
			continue
		}
		_, body := markdown.Parse(string(data))
		title := markdown.ExtractTitle(body)
		if title == "" {
			title = TitleFromFilename(filepath.Base(relPath))
		}

		parent.Children = append(parent.Children, &DirEntry{
			Name:   title,
			Path:   relPath,
			IsDir:  false,
			PageID: id,
		})
	}

	return root, nil
}

// RenameDir renames a directory on disk and updates all pageIndex entries
// whose paths had the old directory prefix.
func (fs *FileStore) RenameDir(_ context.Context, oldPath, newName string) error {
	oldAbs := filepath.Join(fs.root, oldPath)
	info, err := os.Stat(oldAbs)
	if err != nil {
		return fmt.Errorf("directory not found: %s", oldPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", oldPath)
	}

	// Build the new path: same parent, new name
	parentDir := filepath.Dir(oldPath)
	var newPath string
	if parentDir == "." || parentDir == "" {
		newPath = newName
	} else {
		newPath = parentDir + "/" + newName
	}
	newAbs := filepath.Join(fs.root, newPath)

	// Check the target doesn't already exist
	if _, err := os.Stat(newAbs); err == nil {
		return fmt.Errorf("directory already exists: %s", newPath)
	}

	if err := os.Rename(oldAbs, newAbs); err != nil {
		return fmt.Errorf("renaming directory: %w", err)
	}

	// Update pageIndex entries with old prefix
	oldPrefix := oldPath + "/"
	newPrefix := newPath + "/"
	for id, relPath := range fs.pageIndex {
		if strings.HasPrefix(relPath, oldPrefix) {
			fs.pageIndex[id] = newPrefix + strings.TrimPrefix(relPath, oldPrefix)
		}
	}

	return nil
}

// CreateDir creates a new directory at the given relative path.
func (fs *FileStore) CreateDir(_ context.Context, dirPath string) error {
	absPath := filepath.Join(fs.root, dirPath)

	// Check it doesn't already exist
	if _, err := os.Stat(absPath); err == nil {
		return fmt.Errorf("already exists: %s", dirPath)
	}

	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	return nil
}

// DeleteDir removes an empty directory. Returns an error if the directory
// contains any markdown files.
func (fs *FileStore) DeleteDir(_ context.Context, dirPath string) error {
	absPath := filepath.Join(fs.root, dirPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("directory not found: %s", dirPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dirPath)
	}

	// Check if any pages live under this directory
	prefix := dirPath + "/"
	for _, relPath := range fs.pageIndex {
		if strings.HasPrefix(relPath, prefix) {
			return fmt.Errorf("directory is not empty: contains pages")
		}
	}

	if err := os.RemoveAll(absPath); err != nil {
		return fmt.Errorf("removing directory: %w", err)
	}

	return nil
}

// PathExists checks whether a file exists at the given relative path
// in the workspace.
func (fs *FileStore) PathExists(path string) bool {
	absPath := filepath.Join(fs.root, path)
	_, err := os.Stat(absPath)
	return err == nil
}

// MovePage moves a page file to a new path within the workspace.
// It handles the filesystem move and updates the pageIndex.
// The new path must include the filename (e.g., "new-dir/file.md").
func (fs *FileStore) MovePage(ctx context.Context, id string, newPath string) error {
	oldPath, ok := fs.pageIndex[id]
	if !ok {
		return fmt.Errorf("page not found: %s", id)
	}

	if oldPath == newPath {
		return nil // No-op
	}

	oldAbs := filepath.Join(fs.root, oldPath)
	newAbs := filepath.Join(fs.root, newPath)

	// Check if destination already exists
	if _, err := os.Stat(newAbs); err == nil {
		return fmt.Errorf("destination already exists: %s", newPath)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(newAbs), 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Move the file
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return fmt.Errorf("moving file: %w", err)
	}

	// Update pageIndex
	fs.pageIndex[id] = newPath
	return nil
}

// RebuildIndex re-scans the workspace directory and rebuilds the in-memory
// page index. Call this before Reindex to pick up external file changes.
func (fs *FileStore) RebuildIndex() error {
	fs.pageIndex = make(map[string]string)
	return fs.buildIndex()
}

// Ensure FileStore satisfies ContentStore at compile time.
var _ ContentStore = (*FileStore)(nil)
