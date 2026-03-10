# Gotchas & Tribal Knowledge

## Frontmatter Parser is Hand-Rolled
`internal/markdown/frontmatter.go` uses a simple line-based key:value parser — not a real YAML library. It only supports flat `key: value` pairs. Adding nested YAML or arrays will break it.

## `_serve` is an Internal Command
The `_serve` command is not documented in usage text. It's the actual foreground server, invoked by `start` via `exec.Command`. Users should never call it directly. It exists so `start` can fork a detached background process.

## SQLite Index is Ephemeral
The SQLite database is rebuilt from disk files on every startup. Do not store data in SQLite that doesn't also exist in the Markdown files — it will be lost on restart. The `tags` table exists in the schema but is not populated by the indexer.

## UUIDs are Auto-Generated Into Files
When `FileStore.buildIndex()` encounters a `.md` file without an `id:` in its frontmatter, it generates a UUID and **writes it back to the file**. This means first startup will modify all existing Markdown files that lack IDs.

## Wiki-Links Use UUIDs, Not Titles
Links between pages use `[[uuid]]` syntax, not `[[Page Title]]`. This makes links stable across renames but means the raw Markdown is not human-readable for links. The frontend resolves UUIDs to titles at render time.

## Sync Only Supports Workspace Root
Despite the config supporting a `path` field on remotes, only `path: "."` (workspace root) is actually supported. The manager logs and ignores legacy per-directory remotes.

## The `_root` URL Convention
In sync API routes, the workspace root path `"."` is represented as `_root` in URLs to avoid router path normalization issues. Both the Go handler (`syncPath()`) and JS client (`syncPath()`) translate this.

## Port Auto-Selection
Default port is 10200, not 8080. If 10200 is busy, it tries up to 100 ports sequentially (10200-10299). The actual port is recorded in the PID file.

## Drag-and-Drop Moves Files
The sidebar tree supports drag-and-drop to move pages between directories. This calls `POST /api/pages/:id/move` and triggers both delete+create sync notifications for the old and new paths.

## Content Hash for Change Detection
The indexer uses SHA-256 of page content (body only, not frontmatter) to skip unchanged pages during reindex. This means modifying only frontmatter fields won't trigger a re-index of that page.

## Atomic Writes Everywhere
Both `FileStore.SavePage()` and `SaveSyncConfig()` use temp file + `os.Rename` for atomic writes. This prevents data corruption on crashes but means the temp `.tmp` files might be left behind if the process is killed mid-write.

## Pull Uses Rebase, Not Merge
`GitRepo.Pull()` uses `git fetch` + `git rebase` (not merge). This keeps history linear but can cause issues if there are local uncommitted changes during a sync.

## No Authentication
The HTTP server has no authentication. It's designed for local-only access. The CORS config only allows `localhost:5173` (Vite dev server).
