# Archivary

A personal knowledge base application. Write and organize technical knowledge using Markdown, with wiki-style linking, full-text search, and a clean dark-themed UI.

Runs as a single binary — no external dependencies required.

## Features

- **Markdown-first** — pages are `.md` files on disk, editable with any tool
- **Wiki-style links** — link between pages with `[[page-id]]` syntax
- **Full-text search** — powered by SQLite FTS5 with ranked results
- **Folder organization** — subdirectories in your workspace become folders in the sidebar
- **Backlinks** — see which pages link to the current page
- **Single binary** — frontend is embedded; download and run
- **Local-first** — your files stay on your machine

## Installation

Download the latest binary for your platform from the [Releases](https://github.com/jimbarrett/archivary/releases) page.

### Linux / macOS

```bash
chmod +x archivary_*
sudo mv archivary_* /usr/local/bin/archivary
```

### Windows

Rename the downloaded file to `archivary.exe` and move it to a directory in your `PATH`.

### Verify

```bash
archivary serve
```

This will:
1. Create a workspace at `~/Archivary/` (if it doesn't exist)
2. Seed a welcome page on first run
3. Open your browser to `http://localhost:8080`

## Building from Source

Requires Go 1.21+ and Node.js 18+.

```bash
git clone https://github.com/jimbarrett/archivary.git
cd archivary
make build
./bin/archivary serve
```

### Make Targets

| Target | Description |
|---|---|
| `make build` | Build frontend + Go binary |
| `make build-frontend` | Build only the Vue frontend |
| `make build-backend` | Build only the Go binary |
| `make run` | Build and run |
| `make dev-backend` | Run Go server with hot-reload |
| `make dev-frontend` | Run Vite dev server with hot-reload |
| `make clean` | Remove build artifacts |

### Development

Run the backend and frontend dev servers in separate terminals:

```bash
make dev-backend    # Go server on :8080
make dev-frontend   # Vite dev server on :5173 (proxies /api to :8080)
```

## Usage

```
archivary serve [port] [--no-browser]
```

- Default port is `8080`
- `--no-browser` skips auto-opening the browser

## How It Works

- **Workspace directory** (`~/Archivary/`) — your Markdown files live here
- **App data directory** (`~/.local/share/archivary/`) — SQLite index, not user-facing
- Each page has a UUID in its YAML frontmatter for stable linking
- The SQLite index is rebuilt from files on every startup
- Files without frontmatter get a UUID auto-assigned

## Tech Stack

- **Backend:** Go + Echo
- **Frontend:** Vue 3 + Vite
- **Storage:** Markdown files (canonical) + SQLite (index/search)
- **SQLite driver:** modernc.org/sqlite (pure Go, no CGO)

## License

MIT
