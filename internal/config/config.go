package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const AppName = "archivary"

type Config struct {
	// WorkspaceDir is the directory where user markdown files are stored.
	WorkspaceDir string

	// DataDir is the directory for app data (SQLite database, config, logs).
	DataDir string

	// Port is the HTTP server port.
	Port string
}

// Load returns a Config with defaults applied. Directories are created if
// they don't already exist.
func Load() (*Config, error) {
	workspace, err := defaultWorkspaceDir()
	if err != nil {
		return nil, fmt.Errorf("determining workspace dir: %w", err)
	}

	dataDir, err := defaultDataDir()
	if err != nil {
		return nil, fmt.Errorf("determining data dir: %w", err)
	}

	cfg := &Config{
		WorkspaceDir: workspace,
		DataDir:      dataDir,
		Port:         "8080",
	}

	// Ensure directories exist
	for _, dir := range []string{cfg.WorkspaceDir, cfg.DataDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	return cfg, nil
}

func defaultWorkspaceDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Archivary"), nil
}

func defaultDataDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", AppName), nil

	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, AppName), nil

	default: // linux and others
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			dataHome = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(dataHome, AppName), nil
	}
}
