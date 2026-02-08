package store

import (
	"os"
	"path/filepath"
)

const welcomeContent = `---
id: welcome
---
# Welcome to Archivary

Your personal knowledge base is ready.

## Getting Started

- **Create a page** using the **+** button in the sidebar
- **Edit a page** by clicking the **Edit** button when viewing it
- **Link between pages** using the link button in the editor toolbar
- **Search** using the search box in the sidebar

## How It Works

All your pages are stored as Markdown files in your workspace directory. You can edit them here in the app, or with any text editor.

Pages are linked using wiki-style links that look like ` + "`[[page-id]]`" + ` in the raw Markdown. The app resolves these to page titles automatically.

Happy writing!
`

// SeedWelcomePage creates a welcome.md file in the workspace if the directory
// contains no .md files (including subdirectories). This gives new users
// something to see on first launch.
func SeedWelcomePage(workspaceDir string) error {
	// Check if any .md files already exist anywhere in the workspace
	found := false
	filepath.WalkDir(workspaceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".md" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if found {
		return nil
	}

	return os.WriteFile(
		filepath.Join(workspaceDir, "welcome.md"),
		[]byte(welcomeContent),
		0644,
	)
}
