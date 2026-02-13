package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RemoteConfig describes a single synced directory and its git remote settings.
type RemoteConfig struct {
	Path                string `json:"path"`
	URL                 string `json:"url"`
	Branch              string `json:"branch"`
	AutoCommit          bool   `json:"auto_commit"`
	AutoPush            bool   `json:"auto_push"`
	PushIntervalMinutes int    `json:"push_interval_minutes"`
}

// SyncConfig holds the list of all configured sync remotes.
type SyncConfig struct {
	Remotes []RemoteConfig `json:"remotes"`
}

const syncConfigFile = "sync.json"

// LoadSyncConfig reads the sync configuration from <dataDir>/sync.json.
// Returns an empty config if the file does not exist.
func LoadSyncConfig(dataDir string) (*SyncConfig, error) {
	path := filepath.Join(dataDir, syncConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SyncConfig{}, nil
		}
		return nil, fmt.Errorf("reading sync config: %w", err)
	}

	var cfg SyncConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing sync config: %w", err)
	}
	return &cfg, nil
}

// SaveSyncConfig writes the sync configuration to <dataDir>/sync.json atomically.
func SaveSyncConfig(dataDir string, cfg *SyncConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling sync config: %w", err)
	}

	path := filepath.Join(dataDir, syncConfigFile)
	tmp := path + ".tmp"

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing sync config: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming sync config: %w", err)
	}
	return nil
}
