package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jimbarrett/archivary/internal/api"
	"github.com/jimbarrett/archivary/internal/config"
	"github.com/jimbarrett/archivary/internal/index"
	"github.com/jimbarrett/archivary/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		port := "8080"
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

	// Open browser (the OS command is non-blocking, so it's fine to call before Start)
	if openBrowser {
		url := fmt.Sprintf("http://localhost:%s", port)
		if err := config.OpenBrowser(url); err != nil {
			// Not fatal — just print a note
			fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
		}
	}

	return api.StartServer(cfg, fileStore, indexer)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: archivary <command>

Commands:
  serve [port] [--no-browser]  Start the web server (default port: 8080)
`)
}
