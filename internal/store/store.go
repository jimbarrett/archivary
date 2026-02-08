package store

import (
	"context"
	"time"
)

// Page represents a single knowledge base page.
type Page struct {
	// ID is a stable UUID assigned via frontmatter.
	ID string `json:"id"`

	// Title is derived from the first # heading or the filename.
	Title string `json:"title"`

	// Content is the raw markdown body (excluding frontmatter).
	Content string `json:"content"`

	// Path is the relative filepath within the workspace (e.g. "guides/setup.md").
	Path string `json:"path"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ContentStore defines the interface for page storage backends.
type ContentStore interface {
	// GetPage retrieves a single page by ID.
	GetPage(ctx context.Context, id string) (*Page, error)

	// SavePage creates or updates a page. If the page ID already exists, it is
	// updated; otherwise a new file is created.
	SavePage(ctx context.Context, page *Page) error

	// DeletePage removes a page by ID.
	DeletePage(ctx context.Context, id string) error

	// ListPages returns all pages, optionally filtered by directory prefix.
	ListPages(ctx context.Context, dirPrefix string) ([]*Page, error)
}
