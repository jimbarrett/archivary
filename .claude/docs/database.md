# Database

## Overview
SQLite database at `<data-dir>/archivary.db` using WAL mode for concurrent reads. The database is a **derived index** — it is rebuilt from Markdown files on every startup. No migrations are needed.

Driver: `modernc.org/sqlite` (pure Go, no CGO required).

## Schema

### `pages` table
Primary metadata table. Populated from disk files during reindex.

| Column | Type | Notes |
|--------|------|-------|
| `id` | TEXT PK | UUID from frontmatter |
| `title` | TEXT | From first `# heading` or filename |
| `path` | TEXT UNIQUE | Relative path within workspace |
| `content_hash` | TEXT | SHA-256 hex of body content (for change detection) |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | From file mtime |

### `tags` table
Page-to-tag associations (currently schema-only, not populated by indexer).

| Column | Type | Notes |
|--------|------|-------|
| `id` | INTEGER PK AUTO | |
| `page_id` | TEXT FK→pages | ON DELETE CASCADE |
| `tag` | TEXT | |
| | UNIQUE | (page_id, tag) |

Indexes: `idx_tags_page(page_id)`, `idx_tags_tag(tag)`

### `links` table
Wiki-link references between pages. Extracted from `[[uuid]]` patterns.

| Column | Type | Notes |
|--------|------|-------|
| `source_id` | TEXT FK→pages | ON DELETE CASCADE |
| `target_id` | TEXT | May reference non-existent pages |
| | PRIMARY KEY | (source_id, target_id) |

Index: `idx_links_target(target_id)` — used for backlink queries

### `pages_fts` (FTS5 virtual table)
Full-text search index using Porter stemmer + Unicode61 tokenizer.

| Column | Notes |
|--------|-------|
| `page_id` | UUID (not searchable, used for JOIN) |
| `title` | Page title (weighted 10x in BM25) |
| `body` | Full markdown body (weighted 1x in BM25) |

## ER Diagram

```
┌─────────────────┐       ┌─────────────────┐
│     pages        │       │     tags         │
├─────────────────┤       ├─────────────────┤
│ id (PK)         │◄──┐   │ id (PK)         │
│ title           │   │   │ page_id (FK)────┤
│ path (UNIQUE)   │   │   │ tag             │
│ content_hash    │   │   └─────────────────┘
│ created_at      │   │
│ updated_at      │   │   ┌─────────────────┐
└────────┬────────┘   │   │     links        │
         │            │   ├─────────────────┤
         │            ├───│ source_id (FK)  │
         │            │   │ target_id       │
         │            │   └─────────────────┘
         │            │
┌────────▼────────┐   │   (JOIN on p.id = fts.page_id)
│  pages_fts      │   │
├─────────────────┤   │
│ page_id ────────┼───┘
│ title           │
│ body            │
└─────────────────┘
```

## Reindex Strategy
On startup (`Indexer.Reindex`):
1. List all pages from FileStore
2. Fetch all existing content hashes from index
3. For each page on disk:
   - Compute SHA-256 of content
   - If hash matches existing → skip (unchanged)
   - Otherwise → upsert page row, update FTS entry, re-extract links
4. Remove indexed pages that no longer exist on disk

After a git pull (`SyncManager.syncRepo`):
- `FileStore.RebuildIndex()` re-scans workspace
- `Indexer.Reindex()` syncs SQLite with new disk state

## Storage Locations
- **Linux:** `~/.local/share/archivary/archivary.db`
- **macOS:** `~/Library/Application Support/archivary/archivary.db`
- **Windows:** `%APPDATA%/archivary/archivary.db`
