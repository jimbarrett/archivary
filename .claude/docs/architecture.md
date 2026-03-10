# Architecture

## Component Breakdown

### CLI Entry Point (`cmd/archivary/main.go`)
- Parses subcommands: `start`, `stop`, `version`, `update`, `_serve` (internal)
- `start` checks for running instance via PID file, forks `_serve` as background daemon
- `_serve` initializes all components, writes PID file, runs foreground server
- `stop` sends SIGTERM to running process, waits up to 5s, then SIGKILL

### API Layer (`internal/api/`)
- **server.go**: Configures Echo, registers routes, serves embedded frontend, handles graceful shutdown
- **handlers.go**: All HTTP handlers; holds references to FileStore, Indexer, and SyncManager
- Frontend SPA fallback: non-API routes serve `index.html` for Vue Router

### FileStore (`internal/store/`)
- **store.go**: `ContentStore` interface definition and `Page` struct
- **filestore.go**: Implementation that reads/writes `.md` files on disk
  - Maintains in-memory `pageIndex` map (UUID → relative path) for O(1) lookups
  - Scans workspace on init; auto-generates UUIDs for files missing them
  - Atomic writes via temp file + `os.Rename`
  - Tree builder for sidebar (`BuildTree`)
  - Directory operations: create, rename, delete (with force option)
  - Move page between directories
- **welcome.go**: Seeds `welcome.md` on first launch if workspace is empty

### Indexer (`internal/index/`)
- **db.go**: Opens SQLite, creates schema (pages, tags, links, pages_fts)
- **indexer.go**: Full reindex and incremental index operations
  - Content hashing (SHA-256) to skip unchanged pages during reindex
  - Extracts `[[uuid]]` wiki-links and stores in `links` table
  - Backlinks query via JOIN on links table
- **search.go**: FTS5 search with BM25 ranking (title weighted 10x, body 1x)

### Sync Manager (`internal/sync/`)
- **manager.go**: Orchestrates git operations for the workspace
  - `NotifyChange()` — auto-commits on file save/delete if enabled
  - `SyncAll()` / `SyncDir()` — pull + push with reindex after pull
  - Background loop ticks every minute for auto-push
  - Manages `.gitignore` with `ARCHIVARY MANAGED` block for excluded dirs
- **config.go**: Persists sync settings as `sync.json` in data dir

### Git Wrapper (`internal/git/`)
- Wraps the `git` CLI via `exec.Command`
- Operations: init, clone, add, commit, push, pull (rebase), status, log
- Push auto-sets upstream (`-u`) on first push

### Frontend (`frontend/src/`)
- **App.vue**: Root layout — sidebar + router-view
- **Sidebar.vue**: Search, page tree, new page dialog, sync/reindex buttons, drag-and-drop
- **TreeNode.vue**: Recursive folder/file tree with drag-and-drop support
- **Views**: HomeView, PageView, EditView, SearchView, DirectoryView, SyncView, NotFoundView
- **lib/api.js**: Fetch wrapper for all `/api/*` endpoints
- **lib/markdown.js**: markdown-it renderer with `[[uuid]]` wiki-link resolution
- **lib/sync.js**: Reactive sync state with 30s polling
- **lib/events.js**: Simple reactive counter for triggering sidebar refresh

## Design Patterns

### Source-of-Truth Pattern
Markdown files on disk are the canonical source. SQLite is a derived index rebuilt on startup. This means:
- Users can edit files externally with any text editor
- Index corruption is self-healing (just restart)
- No migration strategy needed — schema is recreated each launch

### Atomic Write Pattern
All file writes go through temp file + `os.Rename` to prevent partial writes on crash.

### Interface Segregation
`ContentStore` interface in `store.go` decouples the indexer from the file system implementation, enabling in-memory test databases.

### Embedded SPA Pattern
Frontend is compiled to `frontend/dist/`, then embedded via `go:embed` into the Go binary. The server falls back to `index.html` for any non-API, non-file route to support client-side routing.

## Layering

```
CLI (cmd/archivary)
  └─▸ Config (internal/config)
  └─▸ API Server (internal/api)
        ├─▸ FileStore (internal/store)
        │     └─▸ Markdown Parser (internal/markdown)
        ├─▸ Indexer (internal/index)
        │     └─▸ SQLite (modernc.org/sqlite)
        └─▸ SyncManager (internal/sync)
              └─▸ GitRepo (internal/git)
```
