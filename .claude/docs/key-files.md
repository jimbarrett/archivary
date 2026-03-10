# Key Files

## Entry Points
- `cmd/archivary/main.go` — CLI entry point. All subcommands dispatched here. Server initialization in `cmdServe()`.
- `frontend/src/main.js` — Vue app bootstrap
- `frontend/src/App.vue` — Root component (sidebar + router-view layout)

## Core Business Logic

### Backend
- `internal/store/filestore.go` — FileStore: the heart of page storage. Read, write, delete, move pages. Build directory tree. In-memory UUID→path index.
- `internal/store/store.go` — `Page` struct and `ContentStore` interface definition.
- `internal/index/indexer.go` — Indexer: manages SQLite index. Reindex, incremental index, wiki-link extraction, backlinks.
- `internal/index/search.go` — FTS5 search with BM25 ranking.
- `internal/index/db.go` — SQLite schema creation (pages, tags, links, FTS5).
- `internal/sync/manager.go` — SyncManager: git auto-commit, auto-push, pull+push sync, excluded dirs, gitignore management.
- `internal/api/handlers.go` — All HTTP handler functions.
- `internal/api/server.go` — Echo server setup, route registration, embedded frontend serving.

### Frontend
- `frontend/src/lib/api.js` — API client. All fetch calls to backend.
- `frontend/src/lib/markdown.js` — Markdown rendering with wiki-link `[[uuid]]` resolution.
- `frontend/src/lib/sync.js` — Reactive sync state management with 30s polling.
- `frontend/src/components/Sidebar.vue` — Main navigation: search, tree, new page, drag-and-drop.
- `frontend/src/components/TreeNode.vue` — Recursive tree node with folder expand/collapse and drag-and-drop.
- `frontend/src/views/PageView.vue` — Page display with rendered markdown and backlinks.
- `frontend/src/views/EditView.vue` — Markdown editor.
- `frontend/src/views/SyncView.vue` — Git sync configuration UI.
- `frontend/src/router.js` — Vue Router route definitions.

## Configuration
- `internal/config/config.go` — Config struct, workspace/data dir resolution, port finding.
- `internal/config/browser.go` — Cross-platform browser opener.
- `internal/sync/config.go` — Sync config persistence (`sync.json`).
- `Makefile` — Build orchestration (build, dev, clean targets).
- `frontend/vite.config.js` — Vite config with API proxy for development.

## Supporting Files
- `internal/markdown/frontmatter.go` — YAML frontmatter parser (hand-rolled, no YAML lib).
- `internal/git/git.go` — Git CLI wrapper.
- `internal/update/update.go` — GitHub release checker and self-updater.
- `internal/store/welcome.go` — Welcome page seeder for first launch.
- `frontend/embed.go` — `go:embed dist/*` directive.
- `frontend/src/lib/events.js` — Reactive counter for triggering sidebar tree refresh.

## Test Files
- `internal/store/filestore_test.go`
- `internal/index/index_test.go`
- `internal/markdown/frontmatter_test.go`
- `internal/git/git_test.go`
- `internal/sync/config_test.go`
- `internal/sync/manager_test.go`
