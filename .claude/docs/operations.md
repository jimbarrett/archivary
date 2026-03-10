# Operations

## Running Locally

### Development Mode (two terminals)
```bash
# Terminal 1: Go backend (port 8080)
make dev-backend

# Terminal 2: Vite frontend with hot-reload (port 5173, proxies /api to :8080)
make dev-frontend
```
Access the app at `http://localhost:5173` during development.

### Production Build
```bash
make build                  # Builds frontend + Go binary → bin/archivary
./bin/archivary start       # Start as background daemon
./bin/archivary stop        # Stop running instance
```

### First-Time Setup
```bash
cd frontend && npm install  # Install frontend dependencies
make build                  # Build everything
```

## Running Tests
```bash
go test ./...               # All Go tests
go test ./internal/store/   # FileStore tests
go test ./internal/index/   # Indexer tests
go test ./internal/sync/    # Sync manager tests
go test ./internal/git/     # Git wrapper tests
go test ./internal/markdown/ # Frontmatter parser tests
```
No frontend tests exist currently.

## Build Commands

| Command | Description |
|---------|-------------|
| `make build` | Full build (frontend + backend) |
| `make build-frontend` | Vite build only → `frontend/dist/` |
| `make build-backend` | Go binary only → `bin/archivary` (depends on build-frontend) |
| `make clean` | Remove `bin/` and `frontend/dist/` |

## CLI Commands

| Command | Description |
|---------|-------------|
| `archivary start [port] [--no-browser]` | Start daemon (or open browser if already running) |
| `archivary stop` | Stop running daemon (SIGTERM, then SIGKILL after 5s) |
| `archivary version` | Show version, check for updates |
| `archivary update` | Download and apply latest release from GitHub |

## Environment & Configuration

No environment variables are used. All configuration is derived from OS conventions:

| Setting | Linux | macOS | Windows |
|---------|-------|-------|---------|
| Workspace | `~/Archivary/` | `~/Archivary/` | `~/Archivary/` |
| Data dir | `~/.local/share/archivary/` | `~/Library/Application Support/archivary/` | `%APPDATA%/archivary/` |
| Database | `<data-dir>/archivary.db` | | |
| PID file | `<data-dir>/archivary.pid` | | |
| Log file | `<data-dir>/archivary.log` | | |
| Sync config | `<data-dir>/sync.json` | | |
| Default port | 10200 | | |

## Debugging Common Issues

### "Archivary is already running"
Check PID file: `cat ~/.local/share/archivary/archivary.pid`
If the process is dead but PID file exists, delete it manually.

### Pages not showing up after external edit
Click the reindex button (clock icon) in the sidebar, or `POST /api/reindex`.

### Sync not working
- Ensure `git` is installed and on PATH
- Check that SSH keys or credentials are configured for the remote
- View logs: `cat ~/.local/share/archivary/archivary.log`
- Check sync status: `GET /api/sync/status`

### Port conflict
The app auto-selects from 10200-10299. Check the PID file for the actual port, or check `archivary.log`.

### Database corruption
Just restart the app — the SQLite index is fully rebuilt from disk files on every startup.

## Self-Update
```bash
archivary update
```
Downloads the latest release from GitHub, replaces the running binary. Requires write permission to the binary location. Falls back to providing a manual `curl` command if permissions are insufficient.
