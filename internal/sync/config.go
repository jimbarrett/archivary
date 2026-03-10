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
	Remotes       []RemoteConfig `json:"remotes"`
	ExcludedDirs  []string       `json:"excluded_dirs,omitempty"`
	ExcludedFiles []string       `json:"excluded_files,omitempty"`
}

// IsExcluded returns true if the given directory name is in the excluded list.
func (c *SyncConfig) IsExcluded(dir string) bool {
	for _, d := range c.ExcludedDirs {
		if d == dir {
			return true
		}
	}
	return false
}

// AddExcludedDir adds a directory to the excluded list if not already present.
func (c *SyncConfig) AddExcludedDir(dir string) {
	if !c.IsExcluded(dir) {
		c.ExcludedDirs = append(c.ExcludedDirs, dir)
	}
}

// RemoveExcludedDir removes a directory from the excluded list.
func (c *SyncConfig) RemoveExcludedDir(dir string) {
	out := c.ExcludedDirs[:0]
	for _, d := range c.ExcludedDirs {
		if d != dir {
			out = append(out, d)
		}
	}
	c.ExcludedDirs = out
}

// IsFileExcluded returns true if the given file name is in the excluded files list.
func (c *SyncConfig) IsFileExcluded(file string) bool {
	for _, f := range c.ExcludedFiles {
		if f == file {
			return true
		}
	}
	return false
}

// AddExcludedFile adds a file to the excluded files list if not already present.
func (c *SyncConfig) AddExcludedFile(file string) {
	if !c.IsFileExcluded(file) {
		c.ExcludedFiles = append(c.ExcludedFiles, file)
	}
}

// RemoveExcludedFile removes a file from the excluded files list.
func (c *SyncConfig) RemoveExcludedFile(file string) {
	out := c.ExcludedFiles[:0]
	for _, f := range c.ExcludedFiles {
		if f != file {
			out = append(out, f)
		}
	}
	c.ExcludedFiles = out
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
