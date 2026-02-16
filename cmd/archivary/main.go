package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jimbarrett/archivary/internal/api"
	"github.com/jimbarrett/archivary/internal/config"
	"github.com/jimbarrett/archivary/internal/index"
	"github.com/jimbarrett/archivary/internal/store"
	"github.com/jimbarrett/archivary/internal/sync"
	"github.com/jimbarrett/archivary/internal/update"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		port := ""
		openBrowser := true
		for _, arg := range os.Args[2:] {
			switch arg {
			case "--no-browser":
				openBrowser = false
			default:
				port = arg
			}
		}
		if err := run(port, openBrowser); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("archivary %s\n", version)
		info, err := update.Check(version)
		if err == nil && info.UpdateAvailable {
			fmt.Printf("Update available: %s (%s)\n", info.LatestVersion, info.ReleaseURL)
		} else if err == nil {
			fmt.Println("You are running the latest version.")
		}
	case "update":
		fmt.Println("Checking for updates...")
		info, err := update.Check(version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if !info.UpdateAvailable {
			fmt.Printf("Already up to date (%s).\n", version)
			return
		}
		if info.DownloadURL == "" {
			fmt.Fprintf(os.Stderr, "No binary available for your platform (%s).\n", update.AssetName(info.LatestVersion))
			os.Exit(1)
		}
		binaryPath, writable := update.CanWriteBinary()
		if !writable {
			fmt.Fprintf(os.Stderr, "Cannot write to %s (permission denied).\n", binaryPath)
			fmt.Fprintf(os.Stderr, "Run this command to update manually:\n  %s\n", update.ManualUpdateCommand(info.DownloadURL, binaryPath))
			os.Exit(1)
		}
		fmt.Printf("Downloading %s for %s...\n", info.LatestVersion, update.AssetName(info.LatestVersion))
		if err := update.Apply(info.DownloadURL); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Updated successfully. Restart archivary to use the new version.")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func run(port string, openBrowser bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if port == "" {
		// Auto-select an available port starting at the default.
		found, err := config.FindAvailablePort(config.DefaultPort)
		if err != nil {
			return fmt.Errorf("finding available port: %w", err)
		}
		port = fmt.Sprintf("%d", found)
	}
	cfg.Port = port

	fmt.Printf("Workspace: %s\n", cfg.WorkspaceDir)
	fmt.Printf("Data dir:  %s\n", cfg.DataDir)

	// Seed welcome page if workspace is empty
	if err := store.SeedWelcomePage(cfg.WorkspaceDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not seed welcome page: %v\n", err)
	}

	// Initialize file store
	fileStore, err := store.NewFileStore(cfg.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("initializing file store: %w", err)
	}

	// Initialize SQLite database and indexer
	db, err := index.OpenDB(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	indexer := index.NewIndexer(db)

	// Reindex on startup
	fmt.Print("Indexing workspace...")
	if err := indexer.Reindex(context.Background(), fileStore); err != nil {
		return fmt.Errorf("reindexing: %w", err)
	}
	fmt.Println(" done.")

	// Initialize sync manager
	syncMgr, err := sync.NewSyncManager(cfg.WorkspaceDir, cfg.DataDir, fileStore, indexer)
	if err != nil {
		return fmt.Errorf("initializing sync manager: %w", err)
	}
	syncMgr.Start()
	defer syncMgr.Stop()

	// Open browser (the OS command is non-blocking, so it's fine to call before Start)
	if openBrowser {
		url := fmt.Sprintf("http://localhost:%s", port)
		if err := config.OpenBrowser(url); err != nil {
			// Not fatal — just print a note
			fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
		}
	}

	return api.StartServer(cfg, fileStore, indexer, syncMgr)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: archivary <command>

Commands:
  serve [port] [--no-browser]  Start the web server (default: auto-select from 10200)
  version                      Show version and check for updates
  update                       Update to the latest version
`)
}
