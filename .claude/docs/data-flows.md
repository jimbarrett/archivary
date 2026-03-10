# Data Flows

## Page Create Flow

```
User clicks "+" in sidebar
       │
       ▼
NewPageDialog.vue → POST /api/pages { title, content, path }
       │
       ▼
handlers.createPage()
  1. Bind JSON request
  2. FileStore.SavePage()
     a. EnsureID() — generate UUID if missing
     b. Serialize frontmatter + body
     c. Write temp file → os.Rename (atomic)
     d. Update in-memory pageIndex
  3. FileStore.GetPage() — read back full page
  4. Indexer.IndexPage()
     a. Upsert pages table
     b. Update FTS5 index
     c. Extract [[uuid]] links → insert into links table
  5. SyncManager.NotifyChange(path, "create")
     a. Check if dir is excluded
     b. If auto-commit: git add + git commit
  6. Return 201 + saved page JSON
```

## Page View Flow

```
User clicks page in sidebar → router navigates to /page/:id
       │
       ▼
PageView.vue → GET /api/pages/:id
       │
       ▼
handlers.getPage()
  1. FileStore.GetPage(id)
     a. Look up path in pageIndex[id]
     b. Read file from disk
     c. Parse frontmatter + body
     d. Extract title from first # heading
     e. Return Page struct
  2. Return 200 + page JSON
       │
       ▼
PageView.vue receives page data
  1. GET /api/pages (all pages for link resolution)
  2. GET /api/pages/:id/backlinks
  3. renderMarkdown(content, pages)
     a. Replace [[uuid]] with <a> links using page titles
     b. markdown-it renders to HTML
  4. Display rendered HTML + backlinks panel
```

## Search Flow

```
User types in sidebar search box
       │ (200ms debounce)
       ▼
Sidebar.vue → GET /api/search?q=query
       │
       ▼
handlers.search()
  1. Indexer.Search(query)
     a. FTS5 MATCH query against pages_fts
     b. BM25 ranking (title 10x, body 1x)
     c. snippet() for context excerpts
     d. JOIN pages table for metadata
     e. LIMIT 50 results
  2. Return SearchResult[] with id, title, path, rank, snippet
       │
       ▼
Sidebar shows inline results (or user presses Enter → SearchView)
```

## Startup Flow

```
archivary start [port]
       │
       ▼
cmdStart()
  1. Check PID file → if alive, open browser
  2. Fork: exec _serve as detached background process
  3. Redirect stdout/stderr to log file
       │
       ▼
cmdServe() (_serve)
  1. config.Load() — resolve workspace + data dirs
  2. Write PID file (pid:port)
  3. store.SeedWelcomePage() — create welcome.md if empty
  4. store.NewFileStore() — scan workspace, build pageIndex
  5. index.OpenDB() — open/create SQLite with WAL mode
  6. indexer.Reindex() — full sync disk → SQLite
  7. sync.NewSyncManager() — load sync.json, open git repo
  8. syncMgr.Start() — launch background push goroutine
  9. Open browser
 10. api.StartServer() — listen on port, block until SIGTERM
```

## Sync Flow

```
POST /api/sync/now (or background auto-push timer)
       │
       ▼
SyncManager.syncRepo()
  1. updateRootGitignore() — write excluded dirs
  2. git add -A
  3. git commit "sync workspace"
  4. git fetch → git rebase upstream
  5. FileStore.RebuildIndex() — re-scan after pull
  6. Indexer.Reindex() — update SQLite after pull
  7. git push
```

## File Change Notification Flow

```
API handler saves/deletes a page
       │
       ▼
SyncManager.NotifyChange(filePath, action)
  1. Check if file's top-level dir is excluded → skip
  2. Find RemoteConfig for "."
  3. Check AutoCommit enabled
  4. git add <filePath>
  5. git commit "<action> <filename>"
```
