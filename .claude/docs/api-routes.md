# API Routes

All routes are prefixed with `/api`. No authentication is required (local-only app).

## Middleware Chain
1. `middleware.Logger()` — request logging
2. `middleware.Recover()` — panic recovery
3. `middleware.CORS` — allows `http://localhost:5173` (Vite dev server)

## Error Format
All errors return JSON: `{ "error": "message" }`

## Page Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check → `{ "status": "ok" }` |
| GET | `/api/pages?dir=prefix` | List pages, optionally filtered by directory prefix |
| GET | `/api/pages/:id` | Get single page by UUID |
| GET | `/api/pages/check-path?path=...` | Check if file path exists → `{ "exists": bool }` |
| POST | `/api/pages` | Create page. Body: `{ "title", "content", "path" }` |
| PUT | `/api/pages/:id` | Update page. Body: `{ "content", "path?" }` |
| DELETE | `/api/pages/:id` | Delete page |
| POST | `/api/pages/:id/move` | Move page. Body: `{ "new_path" }` |
| GET | `/api/pages/:id/backlinks` | Get pages that link to this page |

### Page Response Shape
```json
{
  "id": "uuid-string",
  "title": "Page Title",
  "content": "# Page Title\n\nBody text...",
  "path": "folder/page.md",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

## Search & Tree

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/search?q=query` | Full-text search. Returns up to 50 results ranked by BM25. |
| GET | `/api/tree` | Get full workspace directory tree for sidebar |
| POST | `/api/reindex` | Trigger full reindex of workspace |

### Search Result Shape
```json
{
  "id": "uuid",
  "title": "Page Title",
  "path": "folder/page.md",
  "rank": -5.23,
  "snippet": "...matching <mark>text</mark>..."
}
```

## Directory Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/dirs` | Create directory. Body: `{ "path" }` |
| PUT | `/api/dirs/*` | Rename directory. Body: `{ "name" }` |
| DELETE | `/api/dirs/*?force=true` | Delete directory. `force=true` removes contents. |

## Sync Endpoints

Sync paths use `_root` as URL-safe placeholder for `"."` (workspace root).

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/sync/status` | Get sync status for all repos |
| GET | `/api/sync/status/:path` | Get sync status for specific repo |
| POST | `/api/sync/now` | Trigger sync (pull+push) for all repos |
| POST | `/api/sync/now/:path` | Trigger sync for specific repo |
| GET | `/api/sync/remotes` | List configured remotes |
| POST | `/api/sync/remotes` | Add remote. Body: `{ "url", "path", "branch", "auto_commit", "auto_push", "push_interval_minutes" }` |
| PUT | `/api/sync/remotes/:path` | Update remote config (partial update) |
| DELETE | `/api/sync/remotes/:path` | Remove remote and delete `.git` dir |
| POST | `/api/sync/commit/:path` | Manual commit. Body: `{ "message" }` |
| GET | `/api/sync/log/:path?n=20` | Get last N git commits |
| GET | `/api/sync/excluded` | List excluded directories |
| POST | `/api/sync/exclude/:path` | Exclude directory from sync |
| POST | `/api/sync/include/:path` | Re-include excluded directory |

### Sync Status Shape
```json
{
  "path": ".",
  "url": "git@github.com:user/repo.git",
  "branch": "main",
  "clean": true,
  "ahead": 0,
  "behind": 0,
  "last_sync": "2025-01-01T00:00:00Z",
  "error": ""
}
```

## Frontend Serving
All non-`/api` routes serve the embedded Vue SPA. Unknown paths fall back to `index.html` for client-side routing.
