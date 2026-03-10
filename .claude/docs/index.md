# Archivary - Project Knowledge Base

## Executive Summary
Archivary is a local-first personal knowledge base that stores pages as Markdown files with UUID frontmatter. A Go backend (Echo) serves a Vue 3 SPA and provides full-text search via SQLite FTS5. The compiled frontend is embedded into the Go binary for single-binary deployment.

## Tech Stack
- **Backend:** Go 1.25 + Echo v4 HTTP framework
- **Frontend:** Vue 3 + Vue Router 4 + Vite 7, markdown-it for rendering
- **Database:** SQLite (modernc.org/sqlite, pure Go) with FTS5 full-text search
- **Storage:** Markdown files with YAML frontmatter (source of truth) in `~/Archivary/`
- **Sync:** Git CLI wrapper for optional remote sync (auto-commit, auto-push)

## Architecture
```
┌──────────────────────────────────────────┐
│  Vue 3 SPA (embedded in binary)          │
│  Sidebar ─ TreeNode ─ Views ─ Dialogs   │
└──────────────┬───────────────────────────┘
               │ fetch /api/*
┌──────────────▼───────────────────────────┐
│  Echo v4 HTTP Server (internal/api)      │
│  CORS · Logger · Recover middleware      │
├──────────────┬────────────┬──────────────┤
│  FileStore   │  Indexer   │  SyncManager │
│  (store)     │  (index)   │  (sync)      │
├──────────────┼────────────┼──────────────┤
│  ~/Archivary │  SQLite DB │  Git CLI     │
│  .md files   │  FTS5      │  wrapper     │
└──────────────┴────────────┴──────────────┘
```

## Directory Map
- `cmd/archivary/` — CLI entry point (start/stop/version/update commands)
- `internal/api/` — Echo server setup and HTTP handlers
- `internal/store/` — FileStore: reads/writes Markdown files, tree builder
- `internal/index/` — SQLite schema, FTS5 indexing, search, backlinks
- `internal/sync/` — Git-based sync manager, config persistence
- `internal/git/` — Git CLI wrapper (init/add/commit/push/pull)
- `internal/markdown/` — Frontmatter parser/serializer, title extraction
- `internal/config/` — App config, port finder, browser opener
- `internal/update/` — GitHub release checker, self-updater
- `frontend/src/` — Vue components, views, router, API client, markdown renderer
- `frontend/embed.go` — `go:embed` directive for bundling dist/ into binary

## Quick Reference
```bash
make build              # Build frontend + Go binary → bin/archivary
make dev-backend        # Run Go server with hot-reload (port 8080)
make dev-frontend       # Run Vite dev server (port 5173, proxies /api)
make clean              # Remove bin/ and frontend/dist/
go test ./...           # Run all Go tests
./bin/archivary start   # Start as background daemon
./bin/archivary stop    # Stop running instance
```

## Key Conventions
- Pages are identified by UUIDs stored in YAML frontmatter (`id: <uuid>`)
- Titles are derived from the first `# heading` or the filename
- Wiki-links use `[[uuid]]` or `[[uuid|label]]` syntax
- SQLite index is rebuilt on every startup from disk files
- FileStore writes use atomic temp-file + rename pattern
- Default port: 10200 (auto-increments if busy)
- Data dir: `~/.local/share/archivary/` (Linux), `~/Library/Application Support/archivary/` (macOS)
