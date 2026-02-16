package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

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
	case "start", "serve":
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
		if err := cmdStart(port, openBrowser); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "stop":
		if err := cmdStop(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "_serve":
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
		if err := cmdServe(port, openBrowser); err != nil {
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

// cmdStart checks if an instance is already running; if so, opens the browser.
// Otherwise it forks a background _serve process.
func cmdStart(port string, openBrowser bool) error {
	pid, runningPort, alive := readPidFile()
	if alive {
		fmt.Printf("Archivary is already running (pid %d)\n", pid)
		if openBrowser && runningPort != "" {
			url := fmt.Sprintf("http://localhost:%s", runningPort)
			_ = config.OpenBrowser(url)
		}
		return nil
	}

	// Build the _serve command with the same args
	args := []string{"_serve"}
	if port != "" {
		args = append(args, port)
	}
	if !openBrowser {
		args = append(args, "--no-browser")
	}

	// Ensure data dir exists for the log file
	logPath := config.LogFile()
	if dir := logPath[:len(logPath)-len("archivary.log")]; dir != "" {
		os.MkdirAll(dir, 0755)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	// Detach from parent process group so it survives terminal close
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("starting background process: %w", err)
	}
	logFile.Close()

	fmt.Printf("Archivary started (pid %d)\n", cmd.Process.Pid)
	fmt.Printf("Log file: %s\n", logPath)
	return nil
}

// cmdStop reads the PID file and sends SIGTERM to the running process.
func cmdStop() error {
	pid, _, alive := readPidFile()
	if !alive {
		fmt.Println("Archivary is not running")
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process: %w", err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Process might already be gone
		os.Remove(config.PidFile())
		fmt.Println("Archivary stopped")
		return nil
	}

	// Wait for the process to exit (up to 5 seconds)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			// Process is gone
			os.Remove(config.PidFile())
			fmt.Println("Archivary stopped")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill if still alive
	_ = proc.Kill()
	os.Remove(config.PidFile())
	fmt.Println("Archivary stopped (forced)")
	return nil
}

// cmdServe is the internal foreground server command invoked by start.
// It writes the PID file, sets up signal handling, and runs the server.
func cmdServe(port string, openBrowser bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if port == "" {
		found, err := config.FindAvailablePort(config.DefaultPort)
		if err != nil {
			return fmt.Errorf("finding available port: %w", err)
		}
		port = fmt.Sprintf("%d", found)
	}
	cfg.Port = port

	// Write PID file
	pidContent := fmt.Sprintf("%d:%s", os.Getpid(), port)
	if err := os.WriteFile(config.PidFile(), []byte(pidContent), 0644); err != nil {
		return fmt.Errorf("writing pid file: %w", err)
	}
	defer os.Remove(config.PidFile())

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

	// Open browser
	if openBrowser {
		url := fmt.Sprintf("http://localhost:%s", port)
		if err := config.OpenBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
		}
	}

	// Set up context that cancels on SIGTERM/SIGINT
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	return api.StartServer(ctx, cfg, fileStore, indexer, syncMgr)
}

// readPidFile reads the PID file and checks whether the process is still alive.
// Returns the PID, port, and whether the process is alive.
func readPidFile() (int, string, bool) {
	data, err := os.ReadFile(config.PidFile())
	if err != nil {
		return 0, "", false
	}

	parts := strings.SplitN(strings.TrimSpace(string(data)), ":", 2)
	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", false
	}

	port := ""
	if len(parts) > 1 {
		port = parts[1]
	}

	// Check if process is alive by sending signal 0
	proc, err := os.FindProcess(pid)
	if err != nil {
		return pid, port, false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		// Process is dead — stale PID file
		os.Remove(config.PidFile())
		return pid, port, false
	}

	return pid, port, true
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: archivary <command>

Commands:
  start [port] [--no-browser]  Start Archivary in the background
  stop                         Stop the running instance
  version                      Show version and check for updates
  update                       Update to the latest version
`)
}
