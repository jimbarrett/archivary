# Dependencies & External Services

## External Services

### GitHub Releases API
- **Used by:** `internal/update/update.go`
- **Purpose:** Check for new versions and download binary updates
- **Endpoint:** `https://api.github.com/repos/jimbarrett/archivary/releases/latest`
- **Auth:** None (public repo, unauthenticated API)
- **Timeout:** 10s for version check, 5min for binary download
- **Retry:** None — fails immediately on error

### Git CLI
- **Used by:** `internal/git/git.go`
- **Purpose:** Sync workspace to remote git repositories
- **Requirement:** `git` must be installed and on PATH
- **Operations:** init, clone, add, commit, push, pull (rebase), fetch, status, log
- **Auth:** Relies on user's existing git credential configuration (SSH keys, credential helpers)

## Go Dependencies (direct)

| Package | Purpose |
|---------|---------|
| `github.com/labstack/echo/v4` | HTTP framework (routing, middleware, context) |
| `modernc.org/sqlite` | Pure Go SQLite driver (no CGO) |
| `github.com/google/uuid` | UUID generation for page IDs |

## Frontend Dependencies (direct)

| Package | Purpose |
|---------|---------|
| `vue` (3.5.24) | UI framework |
| `vue-router` (4.6.4) | Client-side routing |
| `markdown-it` (14.1.0) | Markdown → HTML rendering |
| `vite` (7.2.4) | Build tool / dev server |
| `@vitejs/plugin-vue` (6.0.1) | Vue SFC compilation |

## System Dependencies

| Dependency | Required By | Notes |
|------------|-------------|-------|
| `git` | Sync feature | Optional — app works without it, sync just won't function |
| `xdg-open` / `open` / `cmd start` | Browser launch | Platform-specific, used to open UI on startup |

## Configuration Files

### `sync.json` (in data dir)
```json
{
  "remotes": [
    {
      "path": ".",
      "url": "git@github.com:user/repo.git",
      "branch": "main",
      "auto_commit": true,
      "auto_push": true,
      "push_interval_minutes": 5
    }
  ],
  "excluded_dirs": ["private"]
}
```

### PID file (in data dir)
Format: `<pid>:<port>` — used by `start`/`stop` commands.

### Log file (in data dir)
`archivary.log` — stdout/stderr from the background daemon process.
